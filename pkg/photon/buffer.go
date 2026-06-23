// Package photon implements the Photon Engine network protocol parser.
package photon

import (
	"encoding/binary"
	"errors"
	"math"
)

// ErrBufferUnderflow is returned when there are not enough bytes to read.
var ErrBufferUnderflow = errors.New("buffer underflow: not enough data to read")

// ErrVarintOverflow is returned when a varint exceeds the maximum valid length.
var ErrVarintOverflow = errors.New("varint overflow: value exceeds maximum size")

// BufferReader provides sequential reading of a byte buffer
// with automatic offset management and bounds checking.
type BufferReader struct {
	data   []byte
	offset int
}

// NewBufferReader creates a new BufferReader.
func NewBufferReader(data []byte) *BufferReader {
	return &BufferReader{
		data:   data,
		offset: 0,
	}
}

// ============================================
// Information methods
// ============================================

// Remaining returns how many bytes are left to read.
func (r *BufferReader) Remaining() int {
	return len(r.data) - r.offset
}

// CanRead checks if at least n bytes are available.
func (r *BufferReader) CanRead(n int) bool {
	return r.offset+n <= len(r.data)
}

// IsEmpty returns true if there are no more bytes to read.
func (r *BufferReader) IsEmpty() bool {
	return r.offset >= len(r.data)
}

// ============================================
// Navigation methods
// ============================================

// Skip advances the offset by n bytes.
func (r *BufferReader) Skip(n int) error {
	if !r.CanRead(n) {
		return ErrBufferUnderflow
	}
	r.offset += n
	return nil
}

// ============================================
// Unsigned integer reads
// ============================================

// ReadByte reads 1 byte (uint8).
func (r *BufferReader) ReadByte() (byte, error) {
	if !r.CanRead(1) {
		return 0, ErrBufferUnderflow
	}
	val := r.data[r.offset]
	r.offset++
	return val, nil
}

// ReadUint32 reads 4 bytes big-endian as uint32.
func (r *BufferReader) ReadUint32() (uint32, error) {
	if !r.CanRead(4) {
		return 0, ErrBufferUnderflow
	}
	val := binary.BigEndian.Uint32(r.data[r.offset:])
	r.offset += 4
	return val, nil
}

// ============================================
// Signed integer reads
// ============================================

// ReadInt32 reads 4 bytes big-endian as int32.
func (r *BufferReader) ReadInt32() (int32, error) {
	val, err := r.ReadUint32()
	return int32(val), err
}

// ============================================
// Bytes and strings
// ============================================

// ReadBytes reads n bytes and returns a copy.
func (r *BufferReader) ReadBytes(n int) ([]byte, error) {
	if !r.CanRead(n) {
		return nil, ErrBufferUnderflow
	}
	result := make([]byte, n)
	copy(result, r.data[r.offset:r.offset+n])
	r.offset += n
	return result, nil
}

// ReadBytesNoCopy reads n bytes without copying (slice of original buffer).
// Warning: modifying the result affects the original buffer.
func (r *BufferReader) ReadBytesNoCopy(n int) ([]byte, error) {
	if !r.CanRead(n) {
		return nil, ErrBufferUnderflow
	}
	result := r.data[r.offset : r.offset+n]
	r.offset += n
	return result, nil
}

// ============================================
// Utility methods
// ============================================

// RemainingBytes returns the remaining bytes as a slice (without advancing).
func (r *BufferReader) RemainingBytes() []byte {
	return r.data[r.offset:]
}

// ============================================
// Protocol18: Varint reads (protobuf-style)
// ============================================

// ReadVarint32 reads a protobuf-style varint as uint32.
func (r *BufferReader) ReadVarint32() (uint32, error) {
	var value uint32
	for shift := 0; shift <= 28; shift += 7 {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		if b&0x80 == 0 {
			if shift == 28 && b > 0x0F {
				return 0, ErrVarintOverflow
			}
			return value | uint32(b)<<shift, nil
		}
		value |= uint32(b&0x7F) << shift
	}
	return 0, ErrVarintOverflow
}

// ReadVarint64 reads a protobuf-style varint as uint64.
func (r *BufferReader) ReadVarint64() (uint64, error) {
	var value uint64
	for shift := 0; shift <= 63; shift += 7 {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		if b&0x80 == 0 {
			if shift == 63 && b > 1 {
				return 0, ErrVarintOverflow
			}
			return value | uint64(b)<<shift, nil
		}
		value |= uint64(b&0x7F) << shift
	}
	return 0, ErrVarintOverflow
}

// DecodeZigZag32 decodes a zigzag-encoded uint32 to int32.
func DecodeZigZag32(value uint32) int32 {
	return int32((value >> 1) ^ (0 - (value & 1)))
}

// DecodeZigZag64 decodes a zigzag-encoded uint64 to int64.
func DecodeZigZag64(value uint64) int64 {
	return int64((value >> 1) ^ (0 - (value & 1)))
}

// ============================================
// Protocol18: Little-endian reads
// ============================================

// ReadInt16LE reads 2 bytes little-endian as int16.
func (r *BufferReader) ReadInt16LE() (int16, error) {
	if !r.CanRead(2) {
		return 0, ErrBufferUnderflow
	}
	val := int16(r.data[r.offset]) | int16(r.data[r.offset+1])<<8
	r.offset += 2
	return val, nil
}

// ReadUint16LE reads 2 bytes little-endian as uint16.
func (r *BufferReader) ReadUint16LE() (uint16, error) {
	if !r.CanRead(2) {
		return 0, ErrBufferUnderflow
	}
	val := uint16(r.data[r.offset]) | uint16(r.data[r.offset+1])<<8
	r.offset += 2
	return val, nil
}

// ReadFloat32LE reads 4 bytes little-endian as float32.
func (r *BufferReader) ReadFloat32LE() (float32, error) {
	if !r.CanRead(4) {
		return 0, ErrBufferUnderflow
	}
	bits := uint32(r.data[r.offset]) | uint32(r.data[r.offset+1])<<8 |
		uint32(r.data[r.offset+2])<<16 | uint32(r.data[r.offset+3])<<24
	r.offset += 4
	return math.Float32frombits(bits), nil
}

// ReadFloat64LE reads 8 bytes little-endian as float64.
func (r *BufferReader) ReadFloat64LE() (float64, error) {
	if !r.CanRead(8) {
		return 0, ErrBufferUnderflow
	}
	bits := binary.LittleEndian.Uint64(r.data[r.offset:])
	r.offset += 8
	return math.Float64frombits(bits), nil
}
