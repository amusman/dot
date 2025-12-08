// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"simd/_gen/unify"
)

// Arm64Instruction represents a parsed ARM64 instruction from XML
type Arm64Instruction struct {
	XMLName           xml.Name      `xml:"instructionsection"`
	ID                string        `xml:"id,attr"`
	Title             string        `xml:"title,attr"`
	Type              string        `xml:"type,attr"`
	DocVars           []DocVar      `xml:"docvars>docvar"`
	Desc              Desc          `xml:"desc"`
	Classes           []Class       `xml:"classes>iclass"`
	Explanations      []Explanation `xml:"explanations>explanation"`
	PsSections        []PsSection   `xml:"ps_section"`
	arrangementsCache []Arrangement // Cache for arrangements
	arngShape         ArngShape
}

// DocVar represents a documentation variable from XML
type DocVar struct {
	Key   string `xml:"key,attr"`
	Value string `xml:"value,attr"`
}

// Desc represents the description section from XML
type Desc struct {
	Authored Authored `xml:"authored"`
}

// Authored represents the authored content in the description from XML
type Authored struct {
	Paragraphs []Para `xml:"para"`
}

// Para represents a paragraph in the authored content from XML
type Para struct {
	Text string `xml:",innerxml"`
}

// Class represents an instruction class from XML
type Class struct {
	Name      string     `xml:"name,attr"`
	ID        string     `xml:"id,attr"`
	DocVars   []DocVar   `xml:"docvars>docvar"`
	Encodings []Encoding `xml:"encoding"`
}

// AsmTemplate represents an assembly template in ARM64 XML
type AsmTemplate struct {
	Content []AsmTemplateContent `xml:",any"`
}

// AsmTemplateContent represents content within an assembly template from XML
type AsmTemplateContent struct {
	XMLName xml.Name
	Text    string `xml:",chardata"`
	Link    string `xml:"link,attr"`
	Hover   string `xml:"hover,attr"`
}

// Encoding represents an instruction encoding in XML
type Encoding struct {
	Name        string      `xml:"name,attr"`
	DocVars     []DocVar    `xml:"docvars>docvar"`
	AsmTemplate AsmTemplate `xml:"asmtemplate"`
}

// Explanation represents operand explanations in XML
type Explanation struct {
	Enclist     string       `xml:"enclist,attr"`
	SymbolDefs  []SymbolDef  `xml:"symboldef"`
	Definitions []Definition `xml:"definition"`
}

// SymbolDef represents a symbol definition in explanations from XML
type SymbolDef struct {
	Symbol  string  `xml:"symbol"`
	Account Account `xml:"account"`
}

// Definition represents a definition in explanations from XML
type Definition struct {
	EncodedIn string `xml:"encodedin,attr"`
	Intro     string `xml:"intro"`
	Table     Table  `xml:"table"`
}

// Account represents an account in explanations from XML
type Account struct {
	EncodedIn string `xml:"encodedin,attr"`
	Intro     string `xml:"intro"`
}

// Table represents a table in explanations from XML
type Table struct {
	Class  string `xml:"class,attr"`
	TGroup TGroup `xml:"tgroup"`
}

// TGroup represents a table group from XML
type TGroup struct {
	Cols    int     `xml:"cols,attr"`
	Headers Headers `xml:"thead"`
	Body    Body    `xml:"tbody"`
}

// Headers represents table headers from XML
type Headers struct {
	Rows []Row `xml:"row"`
}

// Body represents table body from XML
type Body struct {
	Rows []Row `xml:"row"`
}

// Row represents a table row from XML
type Row struct {
	Entries []Entry `xml:"entry"`
}

// Entry represents a table entry from XML
type Entry struct {
	Class string `xml:"class,attr"`
	Value string `xml:",chardata"`
}

// PsSection represents a pseudocode section in XML
type PsSection struct {
	HowMany string `xml:"howmany,attr"`
	Ps      []Ps   `xml:"ps"`
}

// Ps represents a pseudocode block in XML
type Ps struct {
	Name   string   `xml:"name,attr"`
	Mylink string   `xml:"mylink,attr"`
	PSText []string `xml:"pstext"`
}

// parseArm64Instruction parses an ARM64 instruction definition from XML file
func parseArm64Instruction(filePath string) (*Arm64Instruction, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open XML file %s: %v", filePath, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read XML file %s: %v", filePath, err)
	}

	var instruction Arm64Instruction
	err = xml.Unmarshal(content, &instruction)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML file %s: %v", filePath, err)
	}

	return &instruction, nil
}

// loadARM64 loads ARM64 instruction definitions from XML files at given path and returns them as unify values.
func loadARM64(path, asmFilter string, debug bool) []*unify.Value {
	var defs []*unify.Value

	// Scan all XML files in the directory
	xmlFiles, err := filepath.Glob(filepath.Join(path, "*.xml"))
	if err != nil {
		log.Printf("Warning: failed to scan XML files in %s: %v", path, err)
		return defs
	}

	// Process each XML file
	for _, xmlFile := range xmlFiles {
		instruction, err := parseArm64Instruction(xmlFile)
		if err != nil {
			continue
		}

		mnemonic := instruction.Mnemonic()
		if mnemonic == "" {
			continue
		}

		// Only process neon ADVSIMD instructions for now
		if instruction.InstrClass() != "advsimd" {
			continue
		}

		if asmFilter != "" {
			matched, err := regexp.MatchString(asmFilter, mnemonic)
			if err != nil || !matched {
				continue
			}
		}

		defs = append(defs, instruction.EmitAll()...)
	}

	if len(defs) == 0 {
		log.Printf("Warning: no ARM64 instructions found. ASM filter: '%s'", asmFilter)
	}
	if debug {
		log.Printf("Debug: added %d ARM64 instruction defs from %d XML files in %s", len(defs), len(xmlFiles), path)
	}

	return defs
}

// Arm64OperandType defines the type of an operand for ARM64 instruction generation.
type Arm64OperandType int

const (
	Arm64OperandVReg  Arm64OperandType = iota // Vector register
	Arm64OperandGReg                          // General register
	Arm64OperandImm                           // Immediate
	Arm64OperandVElem                         // Vector element (e.g., <Vm>.H[<index>]): early-lowered into immediate + vreg with same AsmPos
	Arm64OperandList                          // List operand (e.g., { <Vn>.16B, <Vn+1>.16B }): early-lowered into vreg with ListNumber
)

func (t Arm64OperandType) String() string {
	switch t {
	case Arm64OperandVReg:
		return "VReg"
	case Arm64OperandGReg:
		return "GReg"
	case Arm64OperandImm:
		return "Imm"
	case Arm64OperandVElem:
		return "VElem"
	case Arm64OperandList:
		return "List"
	default:
		return "Unknown"
	}
}

// BaseTypeSet allows to specify the type set of values independent of arrangement's size, e.g.:
// - Float (instruction used for floating point values in lanes),
// - Uint (instruction used only for unsigned integer values in lanes with any arrangement),
// - Float|Int|Uint (INS with two immediates, copy i-th lane from src vreg to j-th lane of dst vreg: basically don't care about base type).
type BaseTypeSet int

const (
	BaseTypeInt = 1 << iota
	BaseTypeUint
	BaseTypeFloat
)

func (t BaseTypeSet) String() string {
	switch t {
	case BaseTypeInt:
		return "int"
	case BaseTypeUint:
		return "uint"
	case BaseTypeFloat:
		return "float"
	default:
		return ""
	}
}

// Arm64Operand represents an arm64 instruction operand instantiated for concrete arrangement.
type Arm64Operand struct {
	Type       Arm64OperandType
	Class      string // "vreg", "greg", "immediate"
	BaseType   string // Base type ("int", "uint", "float")
	ElemBits   int    // Element bits (for vectors)
	Bits       int    // Total bits
	Lanes      int    // Number of lanes (for vectors)
	ImmMax     int    // Immediate max value (for immediates)
	Name       string // Optional operand name for documentation
	ListNumber int    // List number for reguster list (e.g., useful for TBL/TBX instructions)
	AsmPos     int    // Assembly position (usually 0 for the destination register, 1+ for inputs).
	// Input with AsmPos == 0 represents original value of the destination register for ssa form.
	// Immediate with AsmPos == subsequent register operand's AsmPos represents a vector element (the immediate specifies the lane number).
}

// Instantiate updates the operand's type information based on the given arrangement and instruction mnemonic.
// This is used when generating instruction definitions for specific vector arrangements.
func (op *Arm64Operand) Instantiate(arrangement Arrangement, ashape ArngShape, vregPos int, mnemonic string) {
	switch op.Type {
	case Arm64OperandVReg:
		switch {
		case ashape == NarrowArngs && vregPos == 0:
			op.ElemBits = arrangement.elemBits / 2
			op.Bits = arrangement.bits / 2
		case ashape == LongArngs && vregPos == 0:
			op.ElemBits = arrangement.elemBits * 2
			op.Bits = arrangement.bits * 2
		default:
			op.ElemBits = arrangement.elemBits
			op.Bits = arrangement.bits
		}
		op.Lanes = arrangement.lanes
		op.BaseType = arrangement.baseType
	case Arm64OperandImm:
		// Update immediate operands based on arrangement
		// For shift operations, set immediate max to element_bits - 1 (sometimes need element_bits?)
		// For vector element indices, immediate max should be lanes - 1
		// TODO: But we should actually handle more special cases
		if op.ImmMax == -1 {
			// Check if this is a vector element immediate (has "_i" in name)
			if strings.Contains(op.Name, "_i") {
				// Vector element index: max = lanes - 1
				op.ImmMax = arrangement.lanes - 1
			} else {
				// Shift operation: max = element_bits - 1
				op.ImmMax = arrangement.elemBits - 1
			}
		}
	case Arm64OperandGReg:
		op.BaseType = arrangement.baseType
		if mnemonic == "UMOV" {
			// Update general register width based on arrangement
			// - 2D arrangement needs 64-bit (X register)
			// - All other arrangements need 32-bit (W register)
			if arrangement.arrangement == "2D" {
				op.Bits = 64
			} else {
				op.Bits = 32
			}
		}
	case Arm64OperandVElem, Arm64OperandList:
		panic("expected this operand type to be early-lowered")
	}
}

// Template defines operand templates for instruction which get instantiated for each arrangement.
type Template struct {
	operands    []Arm64Operand
	instruction *Arm64Instruction
}

// Arrangement defines the properties of a vector arrangement.
type Arrangement struct {
	arrangement string
	baseType    string
	elemBits    int
	bits        int
	lanes       int
}

// ArngShape makes certain vreg operands half or double bits wide.
type ArngShape int

const (
	DefaultArngs = ArngShape(iota) // vreg args have same bits number
	NarrowArngs                    // destination is half bits wide, e.g. XTN/XTN2
	LongArngs                      // destination is double bits wide, e.g. UXTL/UXTL2
	WideArngs                      // arg 2 is half bits wide as arg 1 or result, e.g. UADDW
)

// extractDocVar searches for a docvar by key in the instruction's docvars
func (instruction *Arm64Instruction) extractDocVar(key string) string {
	for _, docVar := range instruction.DocVars {
		if docVar.Key == key {
			return docVar.Value
		}
	}
	return ""
}

// Mnemonic extracts the mnemonic from docvars
func (instruction *Arm64Instruction) Mnemonic() string {
	return instruction.extractDocVar("mnemonic")
}

// Bitwise returns true if the instruction is bitwise, e.g. AND
func (instruction *Arm64Instruction) Bitwise() bool {
	mnemonic := instruction.Mnemonic()
	switch mnemonic {
	case "AND", "BIC", "EOR", "NOT", "ORR", "ORN":
		return true
	}
	return false
}

// InstrClass returns the instruction class from docvars
func (instruction *Arm64Instruction) InstrClass() string {
	return instruction.extractDocVar("instr-class")
}

// ResultInArg0 determines if result shares register with first argument.
// This occurs when the destination register is also read as an input operand.
func (instruction *Arm64Instruction) ResultInArg0() bool {
	mnemonic := instruction.Mnemonic()

	// Workaround for some instructions which have a multi-instruction pseudocode: supporting them
	// would need more complex parsing and may use variables defined in other sections.
	// E.g. EOR instruction shares pseudocode with BIT and has 'case' stmt based on the opcode;
	// and for TBL/TBX the pattern reading Vd looks like "= if is_tbl ... else [d,".
	switch mnemonic {
	case "TBL", "EOR":
		return false
	case "TBX":
		return true
	}

	// Check pseudocode for the "= [d," pattern
	for _, psSection := range instruction.PsSections {
		for _, ps := range psSection.Ps {
			for _, pstext := range ps.PSText {
				if strings.Contains(strings.TrimSpace(pstext), "= [d,") {
					return true
				}
			}
		}
	}
	return false
}

// getInputOperandName assigns a sequential name for input operands (x, y, z, ...)
func getInputOperandName(index int) string {
	if index == 0 {
		return "x"
	} else if index == 1 {
		return "y"
	} else if index == 2 {
		return "z"
	}
	return fmt.Sprintf("operand%d", index)
}

// analyzeOperand analyzes an operand string and returns the operand type, whether it's a destination, and the immediate name and immMax (if any).
func analyzeOperand(operandStr string) (Arm64OperandType, bool, string, int) {
	switch {
	case strings.HasPrefix(operandStr, "{") && strings.HasSuffix(operandStr, "}"):
		// List operand: { <Vn>.16B, <Vn+1>.16B }
		return Arm64OperandList, false, "", 0
	case strings.Contains(operandStr, "<Vd>") && strings.Contains(operandStr, "[<index"):
		// Vector element destination: <Vd>.<Ts>[<index>]
		return Arm64OperandVElem, true, "", 0
	case strings.Contains(operandStr, "<Vn>") && strings.Contains(operandStr, "[<index"):
		// Vector element input: <Vn>.<Ts>[<index>]
		return Arm64OperandVElem, false, "", 0
	case strings.Contains(operandStr, "<Vm>") && strings.Contains(operandStr, "[<index"):
		// Vector element input: <Vm>.<Ts>[<index>]
		return Arm64OperandVElem, false, "", 0
	case strings.HasPrefix(operandStr, "<Vd>") || strings.HasPrefix(operandStr, "<V><d>") || strings.HasPrefix(operandStr, "<Va><d>") || strings.HasPrefix(operandStr, "<Hd>"):
		// Vector destination register
		return Arm64OperandVReg, true, "", 0
	case strings.HasPrefix(operandStr, "<Vn>") || strings.HasPrefix(operandStr, "<Vm>") || strings.HasPrefix(operandStr, "<V><n>") || strings.HasPrefix(operandStr, "<V><m>") || strings.HasPrefix(operandStr, "<Va><n>") || strings.HasPrefix(operandStr, "<Vb><n>"):
		// Vector input register
		return Arm64OperandVReg, false, "", 0
	case strings.HasPrefix(operandStr, "<Wd>") || strings.HasPrefix(operandStr, "<Xd>"):
		// General destination register (W=32-bit, X=64-bit)
		return Arm64OperandGReg, true, "", 0
	case strings.HasPrefix(operandStr, "<Wn>") || strings.HasPrefix(operandStr, "<Xn>") || strings.HasPrefix(operandStr, "<Wm>") || strings.HasPrefix(operandStr, "<Xm>"):
		// General input register
		return Arm64OperandGReg, false, "", 0
	case strings.HasPrefix(operandStr, "<R>"):
		// General register
		return Arm64OperandGReg, false, "", 0
	case strings.Contains(operandStr, "#<"):
		// Immediate operand: #<immediate_name>
		// Extract the immediate name from between #< and >
		start := strings.Index(operandStr, "#<") + 2
		end := strings.Index(operandStr[start:], ">")
		if end >= 0 {
			immediateName := operandStr[start : start+end]
			return Arm64OperandImm, false, immediateName, -1
		}
		return Arm64OperandImm, false, "immediate", -1
	case strings.Contains(operandStr, "#0"):
		return Arm64OperandImm, false, "immzero", 0
	default:
		// Default to vector register
		return Arm64OperandVReg, false, "", 0
	}
}

// Operands extracts operand information from the assembly template.
func (instruction *Arm64Instruction) Operands(asmTemplate string) []Arm64Operand {
	resultInArg0 := instruction.ResultInArg0()

	debug := false
	if debug {
		fmt.Printf("Operands for instruction %s:\n", instruction.Mnemonic())
		fmt.Printf("  asmTemplate: %q\n", asmTemplate)
		fmt.Printf("  resultInArg0: %v\n", resultInArg0)
	}

	parts := strings.Split(asmTemplate, ",")
	var operandMatches []string
	var inList bool
	var currentList strings.Builder

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check if we're entering or exiting a list operand
		if strings.HasPrefix(part, "{") && strings.Contains(part, "<V") {
			inList = true
			currentList.WriteString(part)
			continue
		}
		if strings.HasSuffix(part, "}") && inList {
			inList = false
			currentList.WriteString(",")
			currentList.WriteString(part)
			operandMatches = append(operandMatches, currentList.String())
			currentList.Reset()
			continue
		}

		if inList {
			// Inside a list operand, accumulate parts
			currentList.WriteString(",")
			currentList.WriteString(part)
		} else {
			if i == 0 {
				// For the first part, extract just the operand (skip instruction mnemonic)
				if idx := strings.Index(part, " <"); idx >= 0 {
					operandMatches = append(operandMatches, strings.TrimSpace(part[idx:]))
				}
			} else {
				// For other parts, check if we need to merge with previous operand
				// This handles optional parts like {, LSL #<amount>}
				if len(operandMatches) > 0 && (strings.HasPrefix(part, "{") || strings.HasSuffix(operandMatches[len(operandMatches)-1], "{")) {
					// This is an optional part - merge it with the previous operand
					lastOperand := operandMatches[len(operandMatches)-1]
					operandMatches[len(operandMatches)-1] = lastOperand + ", " + part
				} else {
					// Normal case: use the whole operand
					operandMatches = append(operandMatches, part)
				}
			}
		}
	}

	if debug {
		for i, operandStr := range operandMatches {
			opType, isDestination, immediateName, immMax := analyzeOperand(operandStr)
			if immediateName != "" {
				fmt.Printf("  pos %d: %q -> type=%s dest=%v name=%s immMax=%d\n", i, operandStr, opType, isDestination, immediateName, immMax)
			} else {
				fmt.Printf("  pos %d: %q -> type=%s dest=%v\n", i, operandStr, opType, isDestination)
			}
		}
	}

	var outs []Arm64Operand
	var ins []Arm64Operand
	var imms []Arm64Operand
	inputOperandCount := 0
	for i, operandStr := range operandMatches {
		var operand Arm64Operand
		opType, isDestination, immediateName, immMax := analyzeOperand(operandStr)

		switch opType {
		case Arm64OperandVElem:
			// Vector element operand: early lower as index-immediate and a vector register
			if isDestination {
				// Model element output as whole vector output + input with element index
				resultInArg0 = true

				// Vector element destination: <Vd>.<Ts>[<index>]
				indexOperand := Arm64Operand{
					Type:       Arm64OperandImm,
					Class:      "immediate",
					Name:       "destination_i",
					AsmPos:     0,
					ListNumber: -1,
					ImmMax:     -1, // Will be set based on arrangement
				}
				imms = append(imms, indexOperand)

				outs = append(outs, Arm64Operand{
					Type:       Arm64OperandVReg,
					Class:      "vreg",
					Name:       "destination",
					AsmPos:     0,
					ListNumber: -1,
				})
			} else {
				// Vector element input: <Vn>.<Ts>[<index>] or <Vm>.<Ts>[<index>]
				name := getInputOperandName(inputOperandCount)
				indexOperand := Arm64Operand{
					Type:       Arm64OperandImm,
					Class:      "immediate",
					Name:       name + "_i",
					AsmPos:     i,
					ListNumber: -1,
					ImmMax:     -1, // Will be set based on arrangement
				}
				imms = append(imms, indexOperand)

				operand = Arm64Operand{
					Type:       Arm64OperandVReg,
					Class:      "vreg",
					Name:       name,
					AsmPos:     i,
					ListNumber: -1,
				}
				ins = append(ins, operand)
				inputOperandCount++
			}
		case Arm64OperandVReg:
			operand = Arm64Operand{
				Type:       Arm64OperandVReg,
				Class:      "vreg",
				ListNumber: -1,
			}
			if isDestination {
				operand.Name = "destination"
				operand.AsmPos = 0 // Output operand
				outs = append(outs, operand)
			} else {
				operand.Name = getInputOperandName(inputOperandCount)
				operand.AsmPos = i // Input operand
				ins = append(ins, operand)
				inputOperandCount++
			}
		case Arm64OperandGReg:
			operand = Arm64Operand{
				Type:       Arm64OperandGReg,
				Class:      "greg",
				ListNumber: -1,
			}
			if isDestination {
				operand.Name = "destination"
				operand.AsmPos = 0 // Output operand
				outs = append(outs, operand)
			} else {
				operand.Name = getInputOperandName(inputOperandCount)
				operand.AsmPos = i // Input operand
				ins = append(ins, operand)
				inputOperandCount++
			}
		case Arm64OperandImm:
			operand = Arm64Operand{
				Type:       Arm64OperandImm,
				Class:      "immediate",
				Name:       immediateName,
				AsmPos:     i,
				ListNumber: -1,
				ImmMax:     immMax,
			}
			imms = append(imms, operand)
			inputOperandCount++
		case Arm64OperandList:
			operand = Arm64Operand{
				Type:       Arm64OperandVReg,
				Class:      "vreg",
				Name:       getInputOperandName(inputOperandCount),
				AsmPos:     i, // Input operand
				ListNumber: 0, // List number for table lookup instructions
			}
			ins = append(ins, operand)
			inputOperandCount++
		default:
			panic(fmt.Sprintf("unknown op: %s\n", operandStr))
		}
	}

	operands := append(outs, imms...)
	// Add "original" operand for original ssa value passed into resultInArg0 instruction
	if resultInArg0 && len(operands) > 0 {
		original := outs[0]
		original.Name = "original"
		operands = append(operands, original)
	}
	operands = append(operands, ins...)

	return operands
}

// BaseTypeSet determines if an instruction operates on integers or floats
func (instruction *Arm64Instruction) BaseTypeSet() BaseTypeSet {
	mnemonic := instruction.Mnemonic()
	if mnemonic == "DUP" || mnemonic == "INS" {
		return BaseTypeInt | BaseTypeUint | BaseTypeFloat
	}

	floatPattern := regexp.MustCompile(`-?(?:half|single|double)`)

	for _, docVar := range instruction.DocVars {
		if floatPattern.MatchString(docVar.Value) {
			return BaseTypeFloat
		}
	}

	for _, class := range instruction.Classes {
		for _, docVar := range class.DocVars {
			if floatPattern.MatchString(docVar.Value) {
				return BaseTypeFloat
			}
		}
	}

	// TODO: Detect unsigned-only, signed-only
	return BaseTypeInt | BaseTypeUint
}

// extractUMOVArrangements handles special case for UMOV instruction
// UMOV has specific arrangements based on element size and vector register size
func (instruction *Arm64Instruction) extractUMOVArrangements() []string {
	var arrangements []string

	// Check if this is the 32-bit or 64-bit variant by looking at the assembly template
	var has32Bit, has64Bit bool
	if len(instruction.Classes) > 0 && len(instruction.Classes[0].Encodings) > 0 {
		for _, encoding := range instruction.Classes[0].Encodings {
			asmTemplate := asmTemplateToString(encoding.AsmTemplate)
			if strings.Contains(asmTemplate, "<Wd>") {
				has32Bit = true
			}
			if strings.Contains(asmTemplate, "<Xd>") {
				has64Bit = true
			}
		}
	}

	// Generate arrangements based on available variants
	if has32Bit {
		// 32-bit UMOV variants (Wd destination)
		arrangements = append(arrangements, "8B", "4H", "2S")
	}
	if has64Bit {
		// 64-bit UMOV variants (Xd destination)
		arrangements = append(arrangements, "16B", "8H", "4S", "2D")
	}

	return arrangements
}

// Arrangements collects valid arrangement/type specifiers for the instruction
func (instruction *Arm64Instruction) Arrangements() ([]Arrangement, ArngShape) {
	if instruction.arrangementsCache != nil {
		return instruction.arrangementsCache, instruction.arngShape
	}
	baseTypeSet := instruction.BaseTypeSet()
	stringArrangements, ashape := instruction.arrangementStrings()
	bitwise := instruction.Bitwise()
	var arrangements []Arrangement
	for ty := BaseTypeSet(BaseTypeInt); ty <= BaseTypeFloat; ty <<= 1 {
		if ty&baseTypeSet == 0 {
			continue
		}
		for _, arrStr := range stringArrangements {
			elemBits, bits, lanes := parseArrangement(arrStr)
			if elemBits == 0 {
				continue
			}
			if ty == BaseTypeFloat && elemBits != 32 && elemBits != 64 {
				continue
			}
			maxElemBits := elemBits
			if bitwise {
				maxElemBits = bits >> 1
			}
			l := lanes
			for e := elemBits; e <= maxElemBits; e = e * 2 {
				arrangements = append(arrangements, Arrangement{
					arrangement: arrStr,
					baseType:    ty.String(),
					elemBits:    e,
					bits:        bits,
					lanes:       l,
				})
				l = l >> 1
			}
		}
	}
	instruction.arrangementsCache = arrangements
	instruction.arngShape = ashape
	return arrangements, ashape
}

// arrangementStrings extracts arrangement specifiers from XML explanations as strings
func (instruction *Arm64Instruction) arrangementStrings() ([]string, ArngShape) {
	var arrangements []string

	mnemonic := instruction.Mnemonic()
	ashape := DefaultArngs
	switch mnemonic {
	case "AESE", "AESMC":
		arrangements = append(arrangements, "16B")
		return arrangements, DefaultArngs

	case "UMOV":
		return instruction.extractUMOVArrangements(), DefaultArngs

	case "INS":
		// INS instruction inserts general register values into vector elements
		// It supports all arrangements: B, H, S, D with 128-bit vector registers
		arrangements = append(arrangements, "16B", "8H", "4S", "2D")
		return arrangements, DefaultArngs

	case "USHLL", "SSHLL":
		ashape = LongArngs

	case "SHRN", "SQXTN", "UQXTN", "XTN":
		ashape = NarrowArngs
	}

	for _, explanation := range instruction.Explanations {
		for _, definition := range explanation.Definitions {
			if definition.Table.TGroup.Body.Rows != nil {
				for _, row := range definition.Table.TGroup.Body.Rows {
					for _, entry := range row.Entries {
						if entry.Class == "symbol" {
							arrangements = append(arrangements, strings.TrimSpace(entry.Value))
						}
					}
				}
			}
		}
	}

	return removeDuplicates(arrangements), ashape
}

// removeDuplicates removes duplicate strings from a slice
func removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	result := []string{}

	for _, item := range slice {
		if _, value := keys[item]; !value {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

// parseArrangement gets element bits and lanes number from arrangement string like "4S", "2D", "16B"
func parseArrangement(arrangement string) (elemBits, bits, lanes int) {
	if len(arrangement) < 2 {
		return 0, 0, 0
	}

	lanesStr := arrangement[:len(arrangement)-1]
	elemType := arrangement[len(arrangement)-1:]

	lanes, err := strconv.Atoi(lanesStr)
	if err != nil {
		return 0, 0, 0
	}

	switch elemType {
	case "B": // Byte
		elemBits = 8
	case "H": // Halfword
		elemBits = 16
	case "S": // Single word
		elemBits = 32
	case "D": // Double word
		elemBits = 64
	default:
		return 0, 0, 0
	}

	return elemBits, elemBits * lanes, lanes
}

// asmTemplateToString converts an AsmTemplate structure to a string
func asmTemplateToString(template AsmTemplate) string {
	var result strings.Builder
	for _, content := range template.Content {
		switch content.XMLName.Local {
		case "text":
			result.WriteString(content.Text)
		case "a":
			result.WriteString(content.Text)
		}
	}
	return result.String()
}

// Templates returns operand templates
func (instruction *Arm64Instruction) Templates() []Template {
	var operands []Arm64Operand
	asmTemplate := ""
searchTemplate:
	for _, class := range instruction.Classes {
		for _, encoding := range class.Encodings {
			curAsmTemplate := asmTemplateToString(encoding.AsmTemplate)
			if strings.Contains(curAsmTemplate, ">.") && curAsmTemplate != asmTemplate {
				asmTemplate = curAsmTemplate
				break searchTemplate
			}
		}
	}
	operands = instruction.Operands(asmTemplate)
	return []Template{Template{operands: operands, instruction: instruction}}
}

// Documentation extracts detailed instruction documentation from the XML
func (instruction *Arm64Instruction) Documentation() string {
	documentation := instruction.Title
	if len(instruction.Desc.Authored.Paragraphs) > 0 {
		documentation = instruction.Desc.Authored.Paragraphs[0].Text
	}
	return documentation
}

// asComment formats text as a comment
func asComment(text string, width int) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "\n", " ")
	words := strings.Fields(text)
	var lines []string
	line := ""
	for _, w := range words {
		if line != "" {
			line = line + " "
		}
		line = line + w
		if len(line) >= width {
			lines = append(lines, "// "+line)
			line = ""
		}
	}
	if len(line) > 0 {
		lines = append(lines, "// "+line)
	}
	return strings.Join(lines, "\n")
}

// Emit generates the unify.Value representation of this operand
func (op *Arm64Operand) Emit() *unify.Value {
	var opDb unify.DefBuilder
	opDb.Add("class", unify.NewValue(unify.NewStringExact(op.Class)))

	if op.BaseType != "" {
		opDb.Add("base", unify.NewValue(unify.NewStringExact(op.BaseType)))
	}

	if op.Bits > 0 {
		opDb.Add("bits", unify.NewValue(unify.NewStringExact(fmt.Sprint(op.Bits))))
	}

	if op.ElemBits > 0 {
		opDb.Add("elemBits", unify.NewValue(unify.NewStringExact(fmt.Sprint(op.ElemBits))))
	}

	if op.Lanes > 0 {
		opDb.Add("lanes", unify.NewValue(unify.NewStringExact(fmt.Sprint(op.Lanes))))
	}

	if op.Type == Arm64OperandImm {
		opDb.Add("bits", unify.NewValue(unify.NewStringExact("8")))
		if op.ImmMax == 0 {
			opDb.Add("const", unify.NewValue(unify.NewStringExact("0")))
		} else {
			opDb.Add("immOffset", unify.NewValue(unify.NewStringExact("0")))
		}
		if op.ImmMax > 0 {
			opDb.Add("immMax", unify.NewValue(unify.NewStringExact(fmt.Sprint(op.ImmMax))))
		}
	}

	if op.Name != "" {
		opDb.Add("name", unify.NewValue(unify.NewStringExact(op.Name)))
	}

	if op.ListNumber >= 0 {
		opDb.Add("listNumber", unify.NewValue(unify.NewStringExact(fmt.Sprint(op.ListNumber))))
	}

	opDb.Add("asmPos", unify.NewValue(unify.NewStringExact(fmt.Sprint(op.AsmPos))))

	return unify.NewValue(opDb.Build())
}

// Emit generates a single instruction definition for the given arrangement.
func (template *Template) Emit(arrangement string) *unify.Value {
	var db unify.DefBuilder

	// Map mnemonic to Go assembly
	mnemonic := template.instruction.Mnemonic()
	switch mnemonic {
	case "INS", "UMOV":
		arrangement = arrangement[len(arrangement)-1:]
		mnemonic = "VMOV"
	case "USHLL", "SSHLL":
		if len(template.operands) == 2 {
			// TODO: correct name depending on operands have same bits number
			mnemonic = "V" + mnemonic[0:1] + "XTL"
		} else {
			mnemonic = "V" + mnemonic
		}
	case "DUP":
		arrangement = arrangement[len(arrangement)-1:]
		mnemonic = "V" + mnemonic
	case "AESE", "AESMC":
		// these instructions do not start with "V"
	default:
		mnemonic = "V" + mnemonic
	}

	db.Add("asm", unify.NewValue(unify.NewStringExact(mnemonic)))
	db.Add("arrangement", unify.NewValue(unify.NewStringExact(arrangement)))
	db.Add("goarch", unify.NewValue(unify.NewStringExact("arm64")))
	db.Add("cpuFeature", unify.NewValue(unify.NewStringExact("NEON"))) // TODO: features
	db.Add("inVariant", unify.NewValue(unify.NewTuple()))

	if doc := template.instruction.Documentation(); doc != "" {
		db.Add("details", unify.NewValue(unify.NewStringExact(asComment(doc, 80))))
	}

	var inVals, outVals []*unify.Value
	for _, op := range template.operands {
		if op.Name == "destination" {
			outVals = append(outVals, op.Emit())
		} else {
			inVals = append(inVals, op.Emit())
		}
	}

	db.Add("in", unify.NewValue(unify.NewTuple(inVals...)))
	db.Add("out", unify.NewValue(unify.NewTuple(outVals...)))

	return unify.NewValue(db.Build())
}

// EmitAll generates instruction definitions for all arrangements of this instruction.
func (instruction *Arm64Instruction) EmitAll() []*unify.Value {
	var defs []*unify.Value

	mnemonic := instruction.Mnemonic()
	templates := instruction.Templates()
	arrangements, ashape := instruction.Arrangements()
	for _, template := range templates {
		for _, arr := range arrangements {
			// Instatiate instruction's operands based on arrangement and emit it
			updatedTemplate := template
			updatedTemplate.operands = make([]Arm64Operand, len(template.operands))
			copy(updatedTemplate.operands, template.operands)
			// Track vreg position for special arrangement shapes
			// (e.g. Narrow arrangement shape needs to "narrow" operand 0 only).
			vregPos := 0
			for i := range updatedTemplate.operands {
				updatedTemplate.operands[i].Instantiate(arr, ashape, vregPos, mnemonic)
				if updatedTemplate.operands[i].Type == Arm64OperandVReg {
					vregPos++
				}
			}
			defs = append(defs, updatedTemplate.Emit(arr.arrangement))
		}
	}

	return defs
}
