package main

import (
        "simd"
        "testing"
)

// Zigzag encoding algorithm for int32 -> uint32
func zigzagEncodeGo(in []int32, out []uint32) {
        for i, v := range in {
                out[i] = uint32((v << 1) ^ (v >> 31))
        }
}

// Zigzag decoding algorithm for uint32 -> int32
func zigzagDecodeGo(in []uint32, out []int32) {
        for i, v := range in {
                out[i] = int32((v >> 1) ^ (-(v & 1)))
        }
}

// SIMD implementation of zigzag encoding
// Zigzag encode: (val << 1) ^ (val >> 31)
func zigzagEncodeSIMD(in []int32, out []uint32) {
        if len(in) != len(out) {
                panic("input and output slices must have same length")
        }

        i := 0
        // Process 4 elements at a time using SIMD
        for ; i <= len(in)-4; i += 4 {
                val := simd.LoadInt32x4((*[4]int32)(in[i : i+4]))
                left := val.ShiftAllLeft(1)
                right := val.ShiftAllRight(31)
                encoded := left.Xor(right)
                encoded.AsUint32x4().Store((*[4]uint32)(out[i : i+4]))
        }

        // Process remaining elements with scalar code
        if i < len(in) {
                zigzagEncodeGo(in[i:], out[i:])
        }
}

// SIMD implementation of zigzag decoding
// Zigzag decode: (val >> 1) ^ (0 - (val & 1))
func zigzagDecodeSIMD(in []uint32, out []int32) {
        if len(in) != len(out) {
                panic("input and output slices must have same length")
        }

        zeros := simd.LoadInt32x4(&[4]int32{0, 0, 0, 0})
        ones := zeros.SetElem(0, 1).Broadcast128()
        i := 0
        // Process 4 elements at a time using SIMD
        for ; i <= len(in)-4; i += 4 {
                val := simd.LoadUint32x4((*[4]uint32)(in[i : i+4]))
                lowbits := val.AsInt32x4().And(ones)
                shifted := val.ShiftAllRight(1).AsInt32x4()
                negated := zeros.Sub(lowbits)
                decoded := shifted.Xor(negated)
                decoded.Store((*[4]int32)(out[i : i+4]))
        }

        // Process remaining elements with scalar code
        if i < len(in) {
                zigzagDecodeGo(in[i:], out[i:])
        }
}

// Unit tests
func TestZigzagEncodeDecode(t *testing.T) {
        testCases := []struct {
                name     string
                input    []int32
                expected []int32
        }{
                {
                        name:     "zero",
                        input:    []int32{0},
                        expected: []int32{0},
                },
                {
                        name:     "positive_numbers",
                        input:    []int32{1, 2, 100, 1000, 10000},
                        expected: []int32{1, 2, 100, 1000, 10000},
                },
                {
                        name:     "negative_numbers",
                        input:    []int32{-1, -2, -100, -1000, -10000},
                        expected: []int32{-1, -2, -100, -1000, -10000},
                },
                {
                        name:     "mixed_numbers",
                        input:    []int32{0, 1, -1, 2, -2, 2147483647, -2147483648},
                        expected: []int32{0, 1, -1, 2, -2, 2147483647, -2147483648},
                },
        }

        for _, tc := range testCases {
                t.Run(tc.name+"_go", func(t *testing.T) {
                        testEncodeDecode(t, tc.input, tc.expected, zigzagEncodeGo, zigzagDecodeGo)
                })
                t.Run(tc.name+"_simd", func(t *testing.T) {
                        testEncodeDecode(t, tc.input, tc.expected, zigzagEncodeSIMD, zigzagDecodeSIMD)
                })
        }
}

func testEncodeDecode(t *testing.T, input, expected []int32, encodeFunc func([]int32, []uint32), decodeFunc func([]uint32, []int32)) {
        encoded := make([]uint32, len(input))
        decoded := make([]int32, len(input))

        // Test encoding
        encodeFunc(input, encoded)

        // Test decoding
        decodeFunc(encoded, decoded)

        // Verify result matches expected
        for i := range expected {
                if decoded[i] != expected[i] {
                        t.Errorf("At index %d: expected %d, got %d", i, expected[i], decoded[i])
                }
        }
}

// Benchmarks
func benchmarkZigzag(b *testing.B, size int, encodeFunc func([]int32, []uint32), decodeFunc func([]uint32, []int32)) {
        input := make([]int32, size)
        encoded := make([]uint32, size)
        decoded := make([]int32, size)

        // Initialize with some data
        for i := range input {
                input[i] = int32(i*3 - size/2) // Mix of positive and negative numbers
        }

        b.ResetTimer()

        b.Run("Encode", func(b *testing.B) {
                for i := 0; i < b.N; i++ {
                        encodeFunc(input, encoded)
                }
        })

        b.Run("Decode", func(b *testing.B) {
                for i := 0; i < b.N; i++ {
                        decodeFunc(encoded, decoded)
                }
        })
}

// Go implementation benchmarks
func BenchmarkZigzagGo_64(b *testing.B)    { benchmarkZigzag(b, 64, zigzagEncodeGo, zigzagDecodeGo) }
func BenchmarkZigzagGo_256(b *testing.B)   { benchmarkZigzag(b, 256, zigzagEncodeGo, zigzagDecodeGo) }
func BenchmarkZigzagGo_1024(b *testing.B)  { benchmarkZigzag(b, 1024, zigzagEncodeGo, zigzagDecodeGo) }
func BenchmarkZigzagGo_4096(b *testing.B)  { benchmarkZigzag(b, 4096, zigzagEncodeGo, zigzagDecodeGo) }
func BenchmarkZigzagGo_16384(b *testing.B) { benchmarkZigzag(b, 16384, zigzagEncodeGo, zigzagDecodeGo) }

// SIMD implementation benchmarks
func BenchmarkZigzagSIMD_64(b *testing.B) { benchmarkZigzag(b, 64, zigzagEncodeSIMD, zigzagDecodeSIMD) }
func BenchmarkZigzagSIMD_256(b *testing.B) {
        benchmarkZigzag(b, 256, zigzagEncodeSIMD, zigzagDecodeSIMD)
}
func BenchmarkZigzagSIMD_1024(b *testing.B) {
        benchmarkZigzag(b, 1024, zigzagEncodeSIMD, zigzagDecodeSIMD)
}
func BenchmarkZigzagSIMD_4096(b *testing.B) {
        benchmarkZigzag(b, 4096, zigzagEncodeSIMD, zigzagDecodeSIMD)
}
func BenchmarkZigzagSIMD_16384(b *testing.B) {
        benchmarkZigzag(b, 16384, zigzagEncodeSIMD, zigzagDecodeSIMD)
}


