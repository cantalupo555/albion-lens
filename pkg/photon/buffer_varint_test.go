package photon

import (
	"testing"
)

func TestReadVarint32(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    uint32
		wantErr error
	}{
		{"zero", []byte{0x00}, 0, nil},
		{"one", []byte{0x01}, 1, nil},
		{"127", []byte{0x7F}, 127, nil},
		{"128", []byte{0x80, 0x01}, 128, nil},
		{"300", []byte{0xAC, 0x02}, 300, nil},
		{"16384", []byte{0x80, 0x80, 0x01}, 16384, nil},
		{"max_uint32", []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x0F}, 0xFFFFFFFF, nil},
		{"truncated", []byte{0x80}, 0, ErrBufferUnderflow},
		{"overflow_5bytes", []byte{0x80, 0x80, 0x80, 0x80, 0x10}, 0, ErrVarintOverflow},
		{"overflow_6bytes", []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x01}, 0, ErrVarintOverflow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewBufferReader(tt.data)
			got, err := r.ReadVarint32()
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("expected %d, got %d", tt.want, got)
			}
		})
	}
}

func TestReadVarint64(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    uint64
		wantErr error
	}{
		{"zero", []byte{0x00}, 0, nil},
		{"one", []byte{0x01}, 1, nil},
		{"300", []byte{0xAC, 0x02}, 300, nil},
		{"max_uint32", []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x0F}, 0xFFFFFFFF, nil},
		{"large_8bytes", []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x7F}, 0x00FFFFFFFFFFFFFF, nil},
		{"max_uint64", []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x01}, 0xFFFFFFFFFFFFFFFF, nil},
		{"truncated", []byte{0x80}, 0, ErrBufferUnderflow},
		{"overflow_10bytes", []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x02}, 0, ErrVarintOverflow},
		{"overflow_11bytes", []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01}, 0, ErrVarintOverflow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewBufferReader(tt.data)
			got, err := r.ReadVarint64()
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("expected %d, got %d", tt.want, got)
			}
		})
	}
}

func TestDecodeZigZag32(t *testing.T) {
	tests := []struct {
		input uint32
		want  int32
	}{
		{0, 0},
		{1, -1},
		{2, 1},
		{3, -2},
		{4, 2},
		{10, 5},
		{11, -6},
		{9, -5},
		{0xFFFFFFFF, -2147483648},
	}

	for _, tt := range tests {
		got := DecodeZigZag32(tt.input)
		if got != tt.want {
			t.Errorf("DecodeZigZag32(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestDecodeZigZag64(t *testing.T) {
	tests := []struct {
		input uint64
		want  int64
	}{
		{0, 0},
		{1, -1},
		{2, 1},
		{3, -2},
		{10, 5},
		{0xFFFFFFFFFFFFFFFF, -9223372036854775808},
	}

	for _, tt := range tests {
		got := DecodeZigZag64(tt.input)
		if got != tt.want {
			t.Errorf("DecodeZigZag64(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestReadInt16LE(t *testing.T) {
	// 0x0201 LE = 513
	r := NewBufferReader([]byte{0x01, 0x02})
	val, err := r.ReadInt16LE()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 513 {
		t.Errorf("expected 513, got %d", val)
	}

	// 0xFFFF = -1
	r = NewBufferReader([]byte{0xFF, 0xFF})
	val, _ = r.ReadInt16LE()
	if val != -1 {
		t.Errorf("expected -1, got %d", val)
	}

	// underflow
	r = NewBufferReader([]byte{0x01})
	_, err = r.ReadInt16LE()
	if err != ErrBufferUnderflow {
		t.Errorf("expected ErrBufferUnderflow, got %v", err)
	}
}

func TestReadUint16LE(t *testing.T) {
	r := NewBufferReader([]byte{0x34, 0x12})
	val, err := r.ReadUint16LE()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 0x1234 {
		t.Errorf("expected 0x1234, got 0x%04X", val)
	}
}

func TestReadFloat32LE(t *testing.T) {
	// IEEE 754 float32 1.0 = 0x3F800000, LE: 0x00, 0x00, 0x80, 0x3F
	r := NewBufferReader([]byte{0x00, 0x00, 0x80, 0x3F})
	val, err := r.ReadFloat32LE()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 1.0 {
		t.Errorf("expected 1.0, got %f", val)
	}
}

func TestReadFloat64LE(t *testing.T) {
	// IEEE 754 float64 1.0 = 0x3FF0000000000000, LE: 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF0, 0x3F
	r := NewBufferReader([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF0, 0x3F})
	val, err := r.ReadFloat64LE()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 1.0 {
		t.Errorf("expected 1.0, got %f", val)
	}
}
