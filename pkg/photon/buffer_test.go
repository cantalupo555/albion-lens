package photon

import (
	"testing"
)

func TestNewBufferReader(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	r := NewBufferReader(data)

	if r == nil {
		t.Fatal("NewBufferReader returned nil")
	}

	if r.Len() != 5 {
		t.Errorf("Expected Len()=5, got %d", r.Len())
	}

	if r.Offset() != 0 {
		t.Errorf("Expected Offset()=0, got %d", r.Offset())
	}

	if r.Remaining() != 5 {
		t.Errorf("Expected Remaining()=5, got %d", r.Remaining())
	}
}

func TestBufferReaderCanRead(t *testing.T) {
	data := []byte{1, 2, 3}
	r := NewBufferReader(data)

	if !r.CanRead(3) {
		t.Error("CanRead(3) should be true")
	}

	if r.CanRead(4) {
		t.Error("CanRead(4) should be false")
	}

	if r.IsEmpty() {
		t.Error("IsEmpty() should be false")
	}
}

func TestBufferReaderSkip(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	r := NewBufferReader(data)

	err := r.Skip(2)
	if err != nil {
		t.Errorf("Skip(2) failed: %v", err)
	}

	if r.Offset() != 2 {
		t.Errorf("Expected Offset()=2, got %d", r.Offset())
	}

	err = r.Skip(10)
	if err != ErrBufferUnderflow {
		t.Errorf("Expected ErrBufferUnderflow, got %v", err)
	}
}

func TestBufferReaderReset(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	r := NewBufferReader(data)

	r.Skip(3)
	r.Reset()

	if r.Offset() != 0 {
		t.Errorf("After Reset(), expected Offset()=0, got %d", r.Offset())
	}
}

func TestBufferReaderSeek(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	r := NewBufferReader(data)

	err := r.Seek(3)
	if err != nil {
		t.Errorf("Seek(3) failed: %v", err)
	}

	if r.Offset() != 3 {
		t.Errorf("Expected Offset()=3, got %d", r.Offset())
	}

	err = r.Seek(10)
	if err != ErrBufferUnderflow {
		t.Errorf("Expected ErrBufferUnderflow for Seek(10), got %v", err)
	}
}

func TestBufferReaderReadByte(t *testing.T) {
	data := []byte{0xAB, 0xCD}
	r := NewBufferReader(data)

	val, err := r.ReadByte()
	if err != nil {
		t.Errorf("ReadByte failed: %v", err)
	}
	if val != 0xAB {
		t.Errorf("Expected 0xAB, got 0x%X", val)
	}

	val, err = r.ReadByte()
	if err != nil {
		t.Errorf("ReadByte failed: %v", err)
	}
	if val != 0xCD {
		t.Errorf("Expected 0xCD, got 0x%X", val)
	}

	_, err = r.ReadByte()
	if err != ErrBufferUnderflow {
		t.Errorf("Expected ErrBufferUnderflow, got %v", err)
	}
}

func TestBufferReaderReadUint16(t *testing.T) {
	// Big-endian: 0x0102 = 258
	data := []byte{0x01, 0x02}
	r := NewBufferReader(data)

	val, err := r.ReadUint16()
	if err != nil {
		t.Errorf("ReadUint16 failed: %v", err)
	}
	if val != 258 {
		t.Errorf("Expected 258, got %d", val)
	}
}

func TestBufferReaderReadUint32(t *testing.T) {
	// Big-endian: 0x01020304 = 16909060
	data := []byte{0x01, 0x02, 0x03, 0x04}
	r := NewBufferReader(data)

	val, err := r.ReadUint32()
	if err != nil {
		t.Errorf("ReadUint32 failed: %v", err)
	}
	if val != 16909060 {
		t.Errorf("Expected 16909060, got %d", val)
	}
}

func TestBufferReaderReadUint64(t *testing.T) {
	data := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00}
	r := NewBufferReader(data)

	val, err := r.ReadUint64()
	if err != nil {
		t.Errorf("ReadUint64 failed: %v", err)
	}
	if val != 256 {
		t.Errorf("Expected 256, got %d", val)
	}
}

func TestBufferReaderReadInt16(t *testing.T) {
	// Big-endian: 0xFFFF = -1 as int16
	data := []byte{0xFF, 0xFF}
	r := NewBufferReader(data)

	val, err := r.ReadInt16()
	if err != nil {
		t.Errorf("ReadInt16 failed: %v", err)
	}
	if val != -1 {
		t.Errorf("Expected -1, got %d", val)
	}
}

func TestBufferReaderReadInt32(t *testing.T) {
	// Big-endian: 0xFFFFFFFF = -1 as int32
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	r := NewBufferReader(data)

	val, err := r.ReadInt32()
	if err != nil {
		t.Errorf("ReadInt32 failed: %v", err)
	}
	if val != -1 {
		t.Errorf("Expected -1, got %d", val)
	}
}

func TestBufferReaderReadFloat32(t *testing.T) {
	// IEEE 754 float32: 1.0 = 0x3F800000
	data := []byte{0x3F, 0x80, 0x00, 0x00}
	r := NewBufferReader(data)

	val, err := r.ReadFloat32()
	if err != nil {
		t.Errorf("ReadFloat32 failed: %v", err)
	}
	if val != 1.0 {
		t.Errorf("Expected 1.0, got %f", val)
	}
}

func TestBufferReaderReadFloat64(t *testing.T) {
	// IEEE 754 float64: 1.0 = 0x3FF0000000000000
	data := []byte{0x3F, 0xF0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	r := NewBufferReader(data)

	val, err := r.ReadFloat64()
	if err != nil {
		t.Errorf("ReadFloat64 failed: %v", err)
	}
	if val != 1.0 {
		t.Errorf("Expected 1.0, got %f", val)
	}
}

func TestBufferReaderReadBytes(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	r := NewBufferReader(data)

	bytes, err := r.ReadBytes(3)
	if err != nil {
		t.Errorf("ReadBytes(3) failed: %v", err)
	}
	if len(bytes) != 3 || bytes[0] != 1 || bytes[1] != 2 || bytes[2] != 3 {
		t.Errorf("Expected [1,2,3], got %v", bytes)
	}

	// Verify it's a copy
	bytes[0] = 99
	if r.data[0] != 1 {
		t.Error("ReadBytes should return a copy, not a reference")
	}
}

func TestBufferReaderReadBytesNoCopy(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	r := NewBufferReader(data)

	bytes, err := r.ReadBytesNoCopy(3)
	if err != nil {
		t.Errorf("ReadBytesNoCopy(3) failed: %v", err)
	}
	if len(bytes) != 3 {
		t.Errorf("Expected length 3, got %d", len(bytes))
	}
}

func TestBufferReaderReadString(t *testing.T) {
	// Length prefix (2 bytes, big-endian) + string data
	// Length = 5, string = "hello"
	data := []byte{0x00, 0x05, 'h', 'e', 'l', 'l', 'o'}
	r := NewBufferReader(data)

	str, err := r.ReadString()
	if err != nil {
		t.Errorf("ReadString failed: %v", err)
	}
	if str != "hello" {
		t.Errorf("Expected 'hello', got '%s'", str)
	}
}

func TestBufferReaderReadBool(t *testing.T) {
	data := []byte{0x00, 0x01, 0xFF}
	r := NewBufferReader(data)

	val, _ := r.ReadBool()
	if val != false {
		t.Error("Expected false for 0x00")
	}

	val, _ = r.ReadBool()
	if val != true {
		t.Error("Expected true for 0x01")
	}

	val, _ = r.ReadBool()
	if val != true {
		t.Error("Expected true for 0xFF")
	}
}

func TestBufferReaderPeek(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	r := NewBufferReader(data)

	peeked, err := r.Peek(3)
	if err != nil {
		t.Errorf("Peek(3) failed: %v", err)
	}
	if len(peeked) != 3 {
		t.Errorf("Expected 3 bytes, got %d", len(peeked))
	}

	// Offset should not have changed
	if r.Offset() != 0 {
		t.Errorf("Peek should not advance offset, but got %d", r.Offset())
	}
}

func TestBufferReaderPeekByte(t *testing.T) {
	data := []byte{0xAB, 0xCD}
	r := NewBufferReader(data)

	val, err := r.PeekByte()
	if err != nil {
		t.Errorf("PeekByte failed: %v", err)
	}
	if val != 0xAB {
		t.Errorf("Expected 0xAB, got 0x%X", val)
	}

	// Offset should not have changed
	if r.Offset() != 0 {
		t.Errorf("PeekByte should not advance offset, but got %d", r.Offset())
	}
}

func TestBufferReaderSlice(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	r := NewBufferReader(data)

	slice, err := r.Slice(3)
	if err != nil {
		t.Errorf("Slice(3) failed: %v", err)
	}

	if slice.Len() != 3 {
		t.Errorf("Expected slice length 3, got %d", slice.Len())
	}

	// Original reader should have advanced
	if r.Offset() != 3 {
		t.Errorf("Expected offset 3 after Slice, got %d", r.Offset())
	}
}

func TestBufferReaderRemainingBytes(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	r := NewBufferReader(data)

	r.Skip(2)

	remaining := r.RemainingBytes()
	if len(remaining) != 3 {
		t.Errorf("Expected 3 remaining bytes, got %d", len(remaining))
	}
	if remaining[0] != 3 {
		t.Errorf("Expected first remaining byte to be 3, got %d", remaining[0])
	}
}

func TestBufferReaderEmptyBuffer(t *testing.T) {
	r := NewBufferReader([]byte{})

	if !r.IsEmpty() {
		t.Error("Empty buffer should return IsEmpty()=true")
	}

	_, err := r.ReadByte()
	if err != ErrBufferUnderflow {
		t.Errorf("Expected ErrBufferUnderflow for empty buffer, got %v", err)
	}
}
