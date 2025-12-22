// Package photon implements the Photon Engine network protocol parser.
package photon

import (
	"encoding/binary"
	"errors"
	"math"
)

// ErrBufferUnderflow is returned when there are not enough bytes to read.
var ErrBufferUnderflow = errors.New("buffer underflow: not enough data to read")

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

// Len returns the total size of the buffer.
func (r *BufferReader) Len() int {
	return len(r.data)
}

// Offset returns the current read position.
func (r *BufferReader) Offset() int {
	return r.offset
}

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

// Reset moves the offset back to the beginning.
func (r *BufferReader) Reset() {
	r.offset = 0
}

// Seek moves the offset to a specific position.
func (r *BufferReader) Seek(pos int) error {
	if pos < 0 || pos > len(r.data) {
		return ErrBufferUnderflow
	}
	r.offset = pos
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

// ReadUint16 reads 2 bytes big-endian as uint16.
func (r *BufferReader) ReadUint16() (uint16, error) {
	if !r.CanRead(2) {
		return 0, ErrBufferUnderflow
	}
	val := binary.BigEndian.Uint16(r.data[r.offset:])
	r.offset += 2
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

// ReadUint64 reads 8 bytes big-endian as uint64.
func (r *BufferReader) ReadUint64() (uint64, error) {
	if !r.CanRead(8) {
		return 0, ErrBufferUnderflow
	}
	val := binary.BigEndian.Uint64(r.data[r.offset:])
	r.offset += 8
	return val, nil
}

// ============================================
// Signed integer reads
// ============================================

// ReadInt8 reads 1 byte as int8.
func (r *BufferReader) ReadInt8() (int8, error) {
	val, err := r.ReadByte()
	return int8(val), err
}

// ReadInt16 reads 2 bytes big-endian as int16.
func (r *BufferReader) ReadInt16() (int16, error) {
	val, err := r.ReadUint16()
	return int16(val), err
}

// ReadInt32 reads 4 bytes big-endian as int32.
func (r *BufferReader) ReadInt32() (int32, error) {
	val, err := r.ReadUint32()
	return int32(val), err
}

// ReadInt64 reads 8 bytes big-endian as int64.
func (r *BufferReader) ReadInt64() (int64, error) {
	val, err := r.ReadUint64()
	return int64(val), err
}

// ============================================
// Float reads
// ============================================

// ReadFloat32 reads 4 bytes big-endian as float32.
func (r *BufferReader) ReadFloat32() (float32, error) {
	bits, err := r.ReadUint32()
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(bits), nil
}

// ReadFloat64 reads 8 bytes big-endian as float64.
func (r *BufferReader) ReadFloat64() (float64, error) {
	bits, err := r.ReadUint64()
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(bits), nil
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

// ReadString reads a string with a 2-byte big-endian length prefix.
// Format: [length uint16][data bytes]
func (r *BufferReader) ReadString() (string, error) {
	length, err := r.ReadUint16()
	if err != nil {
		return "", err
	}
	if !r.CanRead(int(length)) {
		return "", ErrBufferUnderflow
	}
	str := string(r.data[r.offset : r.offset+int(length)])
	r.offset += int(length)
	return str, nil
}

// ReadBool reads 1 byte as boolean (0 = false, != 0 = true).
func (r *BufferReader) ReadBool() (bool, error) {
	val, err := r.ReadByte()
	if err != nil {
		return false, err
	}
	return val != 0, nil
}

// ============================================
// Utility methods
// ============================================

// Peek returns the next n bytes without advancing the offset.
func (r *BufferReader) Peek(n int) ([]byte, error) {
	if !r.CanRead(n) {
		return nil, ErrBufferUnderflow
	}
	return r.data[r.offset : r.offset+n], nil
}

// PeekByte returns the next byte without advancing the offset.
func (r *BufferReader) PeekByte() (byte, error) {
	if !r.CanRead(1) {
		return 0, ErrBufferUnderflow
	}
	return r.data[r.offset], nil
}

// Slice returns a new BufferReader with the next n bytes.
func (r *BufferReader) Slice(n int) (*BufferReader, error) {
	if !r.CanRead(n) {
		return nil, ErrBufferUnderflow
	}
	slice := NewBufferReader(r.data[r.offset : r.offset+n])
	r.offset += n
	return slice, nil
}

// RemainingBytes returns the remaining bytes as a slice (without advancing).
func (r *BufferReader) RemainingBytes() []byte {
	return r.data[r.offset:]
}
