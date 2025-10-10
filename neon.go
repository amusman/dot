//go:build goexperiment.simd

// NEON SIMD example
// Run with: GOEXPERIMENT=simd go run sample.go
package main

import (
        "fmt"
        "os"
        "simd"
)

//go:noinline
func testFloat32x4() {
        fmt.Println("=== Float32x4 Vector Addition ===")

        a := [4]float32{1.0, 2.0, 3.0, 4.0}
        b := [4]float32{5.0, 6.0, 7.0, 8.0}
        c := [4]float32{100.0, 100.0, 100.0, 100.0}

        // Load arrays into SIMD vectors
        va := simd.LoadFloat32x4(&a)
        vb := simd.LoadFloat32x4(&b)

        // Perform vector addition (all 4 elements at once)
        result := va.Add(vb)

        vc := simd.LoadFloat32x4(&c)
        resfma := vc.MulAdd(va, vb)

        // Store result back to array
        var output [4]float32
        result.Store(&output)
        var outfma [4]float32
        resfma.Store(&outfma)

        fmt.Printf("a:      %v\n", a)
        fmt.Printf("b:      %v\n", b)
        fmt.Printf("c:      %v\n", c)
        fmt.Printf("a + b:  %v\n", output)
        fmt.Printf("c + a * b:  %v\n", outfma)
}

//go:noinline
func testFloat64x2() {
        fmt.Println("\n=== Float64x2 Vector Addition ===")

        a := [2]float64{10.5, 20.5}
        b := [2]float64{2.0, 4.0}

        // Load arrays into SIMD vectors
        va := simd.LoadFloat64x2(&a)
        vb := simd.LoadFloat64x2(&b)

        // Perform vector addition (all 2 elements at once)
        result := va.Add(vb)

        // Store result back to array
        var output [2]float64
        result.Store(&output)

        fmt.Printf("a:      %v\n", a)
        fmt.Printf("b:      %v\n", b)
        fmt.Printf("a + b:  %v\n", output)
}

//go:noinline
func testInt32x4() {
        fmt.Println("\n=== Int32x4 Vector Addition ===")

        a := [4]int32{10, 20, 30, 40}
        b := [4]int32{5, 6, 7, 8}

        // Load arrays into SIMD vectors
        va := simd.LoadInt32x4(&a)
        vb := simd.LoadInt32x4(&b)

        // Perform vector addition (all 4 elements at once)
        result := va.Add(vb)
        pairsum := va.PairwiseAdd(vb)

        // Store result back to array
        var output [4]int32
        result.Store(&output)
        var pairoutput [4]int32
        pairsum.Store(&pairoutput)

        fmt.Printf("a:      %v\n", a)
        fmt.Printf("b:      %v\n", b)
        fmt.Printf("a + b:  %v\n", output)
        fmt.Printf("a + b:  %v (pairwise)\n", pairoutput)
}

//go:noinline
func testInt64x2() {
        fmt.Println("\n=== Int64x2 Vector Addition ===")

        a := [2]int64{100, 200}
        b := [2]int64{50, 75}

        // Load arrays into SIMD vectors
        va := simd.LoadInt64x2(&a)
        vb := simd.LoadInt64x2(&b)

        // Perform vector addition (all 2 elements at once)
        result := va.Add(vb)

        // Store result back to array
        var output [2]int64
        result.Store(&output)

        fmt.Printf("a:      %v\n", a)
        fmt.Printf("b:      %v\n", b)
        fmt.Printf("a + b:  %v\n", output)
}

func main() {
        testFloat32x4()
        testFloat64x2()
        testInt32x4()
        testInt64x2()

        // Test validation - return non-zero on unexpected results
        fail := false

        // Test Float32x4
        a32 := [4]float32{1.0, 2.0, 3.0, 4.0}
        b32 := [4]float32{5.0, 6.0, 7.0, 8.0}
        va32 := simd.LoadFloat32x4(&a32)
        vb32 := simd.LoadFloat32x4(&b32)
        result32 := va32.Add(vb32)
        var output32 [4]float32
        result32.Store(&output32)

        expected32 := [4]float32{6.0, 8.0, 10.0, 12.0}
        for i := range output32 {
                if output32[i] != expected32[i] {
                        fmt.Printf("Float32x4 test failed: expected %v, got %v\n", expected32, output32)
                        fail = true
                        break
                }
        }

        // Test Float64x2
        a64 := [2]float64{10.5, 20.5}
        b64 := [2]float64{2.0, 4.0}
        va64 := simd.LoadFloat64x2(&a64)
        vb64 := simd.LoadFloat64x2(&b64)
        result64 := va64.Add(vb64)
        var output64 [2]float64
        result64.Store(&output64)

        expected64 := [2]float64{12.5, 24.5}
        for i := range output64 {
                if output64[i] != expected64[i] {
                        fmt.Printf("Float64x2 test failed: expected %v, got %v\n", expected64, output64)
                        fail = true
                        break
                }
        }

        // Test Int32x4
        a_i32 := [4]int32{0b1100, 0b1010, 0b1111, 0b0000} // 12, 10, 15, 0 in binary
        b_i32 := [4]int32{0b1010, 0b1100, 0b1111, 0b1111} // 10, 12, 15, 15 in binary
        va_i32 := simd.LoadInt32x4(&a_i32)
        vb_i32 := simd.LoadInt32x4(&b_i32)

        // Test Add
        result_i32 := va_i32.Add(vb_i32)
        var output_i32 [4]int32
        result_i32.Store(&output_i32)
        expected_i32 := [4]int32{22, 22, 30, 15} // 12+10, 10+12, 15+15, 0+15
        for i := range output_i32 {
                if output_i32[i] != expected_i32[i] {
                        fmt.Printf("Int32x4 Add test failed: expected %v, got %v\n", expected_i32, output_i32)
                        fail = true
                        break
                }
        }

        addpairs_i32 := va_i32.PairwiseAdd(vb_i32)
        addpairs_i32.Store(&output_i32)
        addpairs_expected_i32 := [4]int32{22, 15, 22, 30} // 12+10, 15+0, 10+12, 15+15
        for i := range output_i32 {
                if output_i32[i] != addpairs_expected_i32[i] {
                        fmt.Printf("Int32x4 Add test failed: expected %v, got %v\n", addpairs_expected_i32, output_i32)
                        fail = true
                        break
                }
        }

        // Test And (bitwise AND)
        result_and_i32 := va_i32.And(vb_i32)
        var output_and_i32 [4]int32
        result_and_i32.Store(&output_and_i32)
        expected_and_i32 := [4]int32{0b1000, 0b1000, 0b1111, 0b0000} // 8, 8, 15, 0
        for i := range output_and_i32 {
                if output_and_i32[i] != expected_and_i32[i] {
                        fmt.Printf("Int32x4 And test failed: expected %v, got %v\n", expected_and_i32, output_and_i32)
                        fail = true
                        break
                }
        }

        // Test Or (bitwise OR)
        result_or_i32 := va_i32.Or(vb_i32)
        var output_or_i32 [4]int32
        result_or_i32.Store(&output_or_i32)
        expected_or_i32 := [4]int32{0b1110, 0b1110, 0b1111, 0b1111} // 14, 14, 15, 15
        for i := range output_or_i32 {
                if output_or_i32[i] != expected_or_i32[i] {
                        fmt.Printf("Int32x4 Or test failed: expected %v, got %v\n", expected_or_i32, output_or_i32)
                        fail = true
                        break
                }
        }

        // Test Equal (CMEQ)
        result_eq_i32 := va_i32.Equal(vb_i32).AsInt32x4()
        var output_eq_i32 [4]int32
        result_eq_i32.Store(&output_eq_i32)
        expected_eq_i32 := [4]int32{0, 0, -1, 0}
        for i := range output_eq_i32 {
                if output_eq_i32[i] != expected_eq_i32[i] {
                        fmt.Printf("Int32x4 Equal test failed: expected %v, got %v\n", expected_eq_i32, output_eq_i32)
                        fail = true
                        break
                }
        }

        select_f32 := va32.AsUint8x16().Blend(vb32.AsUint8x16(), result_eq_i32.AsUint8x16()).AsFloat32x4()
        select_f32.Store(&output32)

        expected_sel32 := [4]float32{1.0, 2.0, 7.0, 4.0}
        for i := range output32 {
                if output32[i] != expected_sel32[i] {
                        fmt.Printf("Float32x4 Blend failed: expected %v, got %v req=%v\n", expected_sel32, output32, expected_eq_i32)
                        fail = true
                        break
                }
        }

        // Test Test (CMTST)
        result_tst_i32 := va_i32.Test(vb_i32).AsInt32x4()
        var output_tst_i32 [4]int32
        result_tst_i32.Store(&output_tst_i32)
        expected_tst_i32 := [4]int32{-1, -1, -1, 0}
        for i := range output_tst_i32 {
                if output_tst_i32[i] != expected_tst_i32[i] {
                        fmt.Printf("Int32x4 Test test failed: expected %v, got %v\n", expected_tst_i32, output_tst_i32)
                        fail = true
                        break
                }
        }

        // Test Int32x4 ShiftAllLeft (SHL)
        // Reuse the loaded signed integer vectors
        shiftLeftResult := va_i32.ShiftAllLeft(2) // Shift each element left by 2 bits
        var output_shift_left [4]int32
        shiftLeftResult.Store(&output_shift_left)
        expected_shift_left := [4]int32{48, 40, 60, 0}
        for i := range output_shift_left {
                if output_shift_left[i] != expected_shift_left[i] {
                        fmt.Printf("Int32x4 ShiftAllLeft test failed: expected %v, got %v\n", expected_shift_left, output_shift_left)
                        fail = true
                        break
                }
        }

        // Test Uint32x4 ShiftAllRight (USHR)
        // Reuse the loaded value, convert to Uint32x4 for an unsigned right shift test
        va_u32 := va_i32.SetElem(3, -1).AsUint32x4()
        shiftRightResult := va_u32.ShiftAllRight(1) // Shift each element right by 1 bit
        var output_shift_right [4]uint32
        shiftRightResult.Store(&output_shift_right)
        expected_shift_right := [4]uint32{6, 5, 7, 2147483647}
        for i := range output_shift_right {
                if output_shift_right[i] != expected_shift_right[i] {
                        fmt.Printf("Uint32x4 ShiftAllRight test failed: expected %v, got %v\n", expected_shift_right, output_shift_right)
                        fail = true
                        break
                }
        }

        if result_i32.GetElem(0) != expected_i32[0] {
                fmt.Printf("Int32x4 GetElem test failed: expected %v, got %v\n", expected_i32[0], result_i32.GetElem(0))
                fail = true
        }

        if result_i32.GetElem(1) != expected_i32[1] {
                fmt.Printf("Int32x4 GetElem test failed: expected %v, got %v\n", expected_i32[1], result_i32.GetElem(1))
                fail = true
        }

        if result_i32.GetElem(2) != expected_i32[2] {
                fmt.Printf("Int32x4 GetElem test failed: expected %v, got %v\n", expected_i32[2], result_i32.GetElem(2))
                fail = true
        }

        changed_i32 := result_i32.SetElem(3, 25)

        if result_i32.GetElem(3) != expected_i32[3] {
                fmt.Printf("Int32x4 GetElem test failed: expected %v, got %v\n", expected_i32[3], result_i32.GetElem(3))
                fail = true
        }

        if changed_i32.GetElem(2) != expected_i32[2] {
                fmt.Printf("Int32x4 GetElem test failed: expected %v, got %v\n", expected_i32[2], result_i32.GetElem(2))
                fail = true
        }

        if changed_i32.GetElem(3) != 25 {
                fmt.Printf("Int32x4 GetElem test failed: expected %v, got %v\n", 25, changed_i32.GetElem(3))
                fail = true
        }

        bc := result_i32.Broadcast128()
        if bc.GetElem(3) != expected_i32[0] {
                fmt.Printf("Int32x4 Broadcast128 test failed: expected %v, got %v\n", expected_i32[0], bc.GetElem(3))
                fail = true
        }

        // Test Int64x2
        a_i64 := [2]int64{100, 200}
        b_i64 := [2]int64{50, 75}
        va_i64 := simd.LoadInt64x2(&a_i64)
        vb_i64 := simd.LoadInt64x2(&b_i64)
        result_i64 := va_i64.Add(vb_i64)
        var output_i64 [2]int64
        result_i64.Store(&output_i64)

        expected_i64 := [2]int64{150, 275}
        for i := range output_i64 {
                if output_i64[i] != expected_i64[i] {
                        fmt.Printf("Int64x2 test failed: expected %v, got %v\n", expected_i64, output_i64)
                        fail = true
                        break
                }
        }

        // Test Uint8x16 Permute (VTBL/VTBX)
        a_u8 := [16]uint8{10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160}
        indices := [16]uint8{16, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}

        va_u8 := simd.LoadUint8x16(&a_u8)
        vindices := simd.LoadUint8x16(&indices)

        result_permutxa := va_u8.PermuteX(va_u8, vindices)    // permute a, for out of range indices take originals from a
        result_permutxi := vindices.PermuteX(va_u8, vindices) // permute a, for out of range indices take originals from indices
        result_permute := va_u8.Permute(vindices)             // permute a, for out of range indices take zeroes
        var output_permute [16]uint8
        result_permute.Store(&output_permute)

        // Expected: original array in reverse order
        expected_permute := [16]uint8{0, 150, 140, 130, 120, 110, 100, 90, 80, 70, 60, 50, 40, 30, 20, 10}
        for i := range output_permute {
                if output_permute[i] != expected_permute[i] {
                        fmt.Printf("Uint8x16 Permute test failed: expected %v, got %v\n", expected_permute, output_permute)
                        fail = true
                        break
                }
        }

        // Expected to leave original (from a) for VTBX
        result_permutxa.Store(&output_permute)
        expected_permute[0] = 10
        for i := range output_permute {
                if output_permute[i] != expected_permute[i] {
                        fmt.Printf("Uint8x16 PermuteX test failed: expected %v, got %v\n", expected_permute, output_permute)
                        fail = true
                        break
                }
        }
        // Expected to leave original (from indices) for VTBX
        result_permutxi.Store(&output_permute)
        expected_permute[0] = 16
        for i := range output_permute {
                if output_permute[i] != expected_permute[i] {
                        fmt.Printf("Uint8x16 PermuteX test failed: expected %v, got %v\n", expected_permute, output_permute)
                        fail = true
                        break
                }
        }

        // Test Uint8x16 OnesCount (VCNT)
        ones_input := [16]uint8{
                0b00000000, // 0 ones
                0b00000001, // 1 one
                0b00000011, // 2 ones
                0b00000111, // 3 ones
                0b00001111, // 4 ones
                0b00011111, // 5 ones
                0b00111111, // 6 ones
                0b01111111, // 7 ones
                0b11111111, // 8 ones
                0b10101010, // 4 ones (alternating pattern)
                0b01010101, // 4 ones (alternating pattern)
                0b10000000, // 1 one
                0b11000000, // 2 ones
                0b11100000, // 3 ones
                0b11110000, // 4 ones
                0b11111000, // 5 ones
        }

        vinput := simd.LoadUint8x16(&ones_input)
        result_ones := vinput.OnesCount()
        var output_ones [16]uint8
        result_ones.Store(&output_ones)

        expected_ones := [16]uint8{
                0, // 0b00000000 has 0 ones
                1, // 0b00000001 has 1 one
                2, // 0b00000011 has 2 ones
                3, // 0b00000111 has 3 ones
                4, // 0b00001111 has 4 ones
                5, // 0b00011111 has 5 ones
                6, // 0b00111111 has 6 ones
                7, // 0b01111111 has 7 ones
                8, // 0b11111111 has 8 ones
                4, // 0b10101010 has 4 ones
                4, // 0b01010101 has 4 ones
                1, // 0b10000000 has 1 one
                2, // 0b11000000 has 2 ones
                3, // 0b11100000 has 3 ones
                4, // 0b11110000 has 4 ones
                5, // 0b11111000 has 5 ones
        }

        for i := range output_ones {
                if output_ones[i] != expected_ones[i] {
                        fmt.Printf("Uint8x16 OnesCount test failed at index %d: input 0b%08b, expected %d ones, got %d\n",
                                i, ones_input[i], expected_ones[i], output_ones[i])
                        fmt.Printf("Full result: expected %v, got %v\n", expected_ones, output_ones)
                        fail = true
                        break
                }
        }

        // Test Uint8x16 Extract (VEXT)
        first := [16]uint8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
        second := [16]uint8{16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}

        vfirst := simd.LoadUint8x16(&first)
        vsecond := simd.LoadUint8x16(&second)

        // Test extracting with constant = 4
        // This should take bytes [4..19] from the concatenated first+second vectors
        result_extract := vfirst.Extract(4, vsecond)
        var output_extract [16]uint8
        result_extract.Store(&output_extract)

        // Expected: bytes 4-15 from first, then bytes 0-3 from second
        // first[4..15] = {4,5,6,7,8,9,10,11,12,13,14,15}
        // second[0..3] = {16,17,18,19}
        expected_extract := [16]uint8{4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}

        for i := range output_extract {
                if output_extract[i] != expected_extract[i] {
                        fmt.Printf("Uint8x16 Extract test failed at index %d: expected %d, got %d\n",
                                i, expected_extract[i], output_extract[i])
                        fmt.Printf("Full result: expected %v, got %v\n", expected_extract, output_extract)
                        fail = true
                        break
                }
        }

        if fail {
                os.Exit(1)
        }
}
