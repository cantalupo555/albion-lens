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

	if r.Remaining() != 3 {
		t.Errorf("Expected Remaining()=3 after Skip(2), got %d", r.Remaining())
	}

	err = r.Skip(10)
	if err != ErrBufferUnderflow {
		t.Errorf("Expected ErrBufferUnderflow, got %v", err)
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
