package photon

import (
	"math"
	"testing"
)

func TestDeserialize18_SimpleTypes(t *testing.T) {
	tests := []struct {
		name     string
		typeCode byte
		data     []byte
		want     interface{}
	}{
		{"boolean_true", P18Boolean, []byte{0x01}, true},
		{"boolean_false", P18Boolean, []byte{0x00}, false},
		{"boolean_true_const", P18BooleanTrue, []byte{}, true},
		{"boolean_false_const", P18BooleanFalse, []byte{}, false},
		{"byte", P18Byte, []byte{0x42}, byte(0x42)},
		{"byte_zero", P18ByteZero, []byte{}, byte(0)},
		{"short", P18Short, []byte{0x34, 0x12}, int16(0x1234)},
		{"short_zero", P18ShortZero, []byte{}, int16(0)},
		{"float", P18Float, []byte{0x00, 0x00, 0x80, 0x3F}, float32(1.0)},
		{"float_zero", P18FloatZero, []byte{}, float32(0)},
		{"double", P18Double, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF0, 0x3F}, float64(1.0)},
		{"double_zero", P18DoubleZero, []byte{}, float64(0)},
		{"null", P18Null, []byte{}, nil},
		{"int_zero", P18IntZero, []byte{}, int32(0)},
		{"long_zero", P18LongZero, []byte{}, int64(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewBufferReader(tt.data)
			got := deserialize18(r, tt.typeCode)
			if got != tt.want {
				t.Errorf("expected %v (%T), got %v (%T)", tt.want, tt.want, got, got)
			}
		})
	}
}

func TestDeserialize18_CompressedInt(t *testing.T) {
	r := NewBufferReader([]byte{0x00})
	got := deserialize18(r, P18CompressedInt)
	if got != int32(0) {
		t.Errorf("expected 0, got %v", got)
	}

	// varint 1 -> zigzag -1
	r = NewBufferReader([]byte{0x01})
	got = deserialize18(r, P18CompressedInt)
	if got != int32(-1) {
		t.Errorf("expected -1, got %v", got)
	}

	// varint 2 -> zigzag 1
	r = NewBufferReader([]byte{0x02})
	got = deserialize18(r, P18CompressedInt)
	if got != int32(1) {
		t.Errorf("expected 1, got %v", got)
	}
}

func TestDeserialize18_SmallInts(t *testing.T) {
	// Int1: single byte value
	r := NewBufferReader([]byte{42})
	got := deserialize18(r, P18Int1)
	if got != int32(42) {
		t.Errorf("Int1: expected 42, got %v", got)
	}

	// Int1Negative: single byte negated
	r = NewBufferReader([]byte{42})
	got = deserialize18(r, P18Int1Negative)
	if got != int32(-42) {
		t.Errorf("Int1Negative: expected -42, got %v", got)
	}

	// Int2: uint16 LE
	r = NewBufferReader([]byte{0x34, 0x12})
	got = deserialize18(r, P18Int2)
	if got != int32(0x1234) {
		t.Errorf("Int2: expected %d, got %v", 0x1234, got)
	}

	// Long1
	r = NewBufferReader([]byte{7})
	got = deserialize18(r, P18Long1)
	if got != int64(7) {
		t.Errorf("Long1: expected 7, got %v", got)
	}
}

func TestDeserialize18_String(t *testing.T) {
	// "hi" - varint length 2, then "hi"
	r := NewBufferReader([]byte{0x02, 'h', 'i'})
	got := deserialize18(r, P18String)
	if got != "hi" {
		t.Errorf("expected 'hi', got %v", got)
	}

	// empty string
	r = NewBufferReader([]byte{0x00})
	got = deserialize18(r, P18String)
	if got != "" {
		t.Errorf("expected '', got %v", got)
	}
}

func TestDecodeParameterTable18(t *testing.T) {
	// 2 params:
	//   key=1, type=CompressedInt(9), value=varint 0 -> zigzag 0
	//   key=2, type=String(7), value="ok"
	data := []byte{
		0x02,                   // 2 entries
		0x01, 0x09, 0x00,       // key=1, CompressedInt, varint 0
		0x02, 0x07, 0x02, 'o', 'k', // key=2, String, len=2, "ok"
	}
	r := NewBufferReader(data)
	params := decodeParameterTable18(r)

	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(params))
	}
	if params[1] != int32(0) {
		t.Errorf("param[1]: expected int32(0), got %v (%T)", params[1], params[1])
	}
	if params[2] != "ok" {
		t.Errorf("param[2]: expected 'ok', got %v", params[2])
	}
}

func TestDecodeParameterTable18_Empty(t *testing.T) {
	r := NewBufferReader([]byte{0x00})
	params := decodeParameterTable18(r)
	if len(params) != 0 {
		t.Errorf("expected 0 params, got %d", len(params))
	}
}

func TestDeserializeShortArray18(t *testing.T) {
	// 3 shorts: [10, 20, -1] in LE
	data := []byte{
		0x03,                   // array length = 3
		0x0A, 0x00,             // 10
		0x14, 0x00,             // 20
		0xFF, 0xFF,             // -1
	}
	r := NewBufferReader(data)
	arr := deserializeShortArray18(r)
	if len(arr) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(arr))
	}
	if arr[0] != 10 || arr[1] != 20 || arr[2] != -1 {
		t.Errorf("expected [10, 20, -1], got %v", arr)
	}
}

func TestDeserializeShortArray18_OverflowProtection(t *testing.T) {
	// Crafted varint that decodes to a huge array length but with very few bytes remaining.
	// arrLen = 0x80000002 (2147483650), elemSize = 2
	// Old code: int(arrLen*2) = int(0x00000004) = 4 > remaining -> might pass or fail depending on remaining
	// New code: arrLen > remaining/2 -> 2147483650 > small_number -> rejects
	// Encode 0x80000002 as varint: byte0=0x82, byte1=0x80, byte2=0x80, byte3=0x80, byte4=0x04
	// (0x80000002 = 2147483650)
	// Varint encoding: 0x02|0x80, 0x00|0x80, 0x00|0x80, 0x00|0x80, 0x04
	// Check: (2) | (0<<7) | (0<<14) | (0<<21) | (4<<28) = 2 + 0 + 0 + 0 + 0x40000000 = 0x40000002
	// That's not right. Let me compute: we want 0x80000002
	// Bits: 10 00000 00000000 00000000 00000010
	// Group by 7 from LSB: 0000010 | 0000000 | 0000000 | 0000000 | 0000100 (5 groups)
	// Wait: 0x80000002 = 2^31 + 2
	// In 7-bit groups (LSB first): 
	//   2 & 0x7F = 0x02, shift >> 7
	//   0 & 0x7F = 0x00, shift >> 7
	//   0 & 0x7F = 0x00, shift >> 7
	//   0 & 0x7F = 0x00, shift >> 7
	//   (0x80000002 >> 28) & 0x0F = 0x08
	// So varint bytes: 0x82, 0x80, 0x80, 0x80, 0x08
	data := []byte{
		0x82, 0x80, 0x80, 0x80, 0x08, // varint = 0x80000002
		0xFF, 0xFF, // only 2 bytes of data (1 short)
	}
	r := NewBufferReader(data)
	arr := deserializeShortArray18(r)
	if arr != nil {
		t.Errorf("expected nil (overflow protection), got %v", arr)
	}
}

func TestDeserializeFloatArray18(t *testing.T) {
	// 2 floats: [1.0, -1.0] in LE
	data := []byte{
		0x02,
		0x00, 0x00, 0x80, 0x3F, // 1.0
		0x00, 0x00, 0x80, 0xBF, // -1.0
	}
	r := NewBufferReader(data)
	arr := deserializeFloatArray18(r)
	if len(arr) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(arr))
	}
	if arr[0] != 1.0 || arr[1] != -1.0 {
		t.Errorf("expected [1.0, -1.0], got %v", arr)
	}
}

func TestDeserializeFloatArray18_OverflowProtection(t *testing.T) {
	// arrLen = 0x80000002, elemSize = 4
	// old code: int(0x80000002*4) = int(0x00000008) = 8
	// new code: 0x80000002 > remaining/4
	data := []byte{
		0x82, 0x80, 0x80, 0x80, 0x08, // varint = 0x80000002
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
	r := NewBufferReader(data)
	arr := deserializeFloatArray18(r)
	if arr != nil {
		t.Errorf("expected nil (overflow protection), got %v", arr)
	}
}

func TestDeserializeDoubleArray18(t *testing.T) {
	// 1 double: [1.0] in LE
	data := []byte{
		0x01,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xF0, 0x3F,
	}
	r := NewBufferReader(data)
	arr := deserializeDoubleArray18(r)
	if len(arr) != 1 {
		t.Fatalf("expected 1 element, got %d", len(arr))
	}
	if arr[0] != 1.0 {
		t.Errorf("expected [1.0], got %v", arr)
	}
}

func TestDeserializeDoubleArray18_OverflowProtection(t *testing.T) {
	// arrLen = 0x80000002, elemSize = 8
	// old code: int(0x80000002*8) = int(0x00000010) = 16
	// new code: 0x80000002 > remaining/8
	data := []byte{
		0x82, 0x80, 0x80, 0x80, 0x08,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	r := NewBufferReader(data)
	arr := deserializeDoubleArray18(r)
	if arr != nil {
		t.Errorf("expected nil (overflow protection), got %v", arr)
	}
}

func TestDeserializeByteArray18(t *testing.T) {
	data := []byte{0x03, 0xAA, 0xBB, 0xCC}
	r := NewBufferReader(data)
	arr := deserializeByteArray18(r)
	if len(arr) != 3 {
		t.Fatalf("expected 3 bytes, got %d", len(arr))
	}
	if arr[0] != 0xAA || arr[1] != 0xBB || arr[2] != 0xCC {
		t.Errorf("expected [0xAA, 0xBB, 0xCC], got %v", arr)
	}
}

func TestDeserializeStringArray18(t *testing.T) {
	// ["ab", "c"]
	data := []byte{
		0x02,             // 2 strings
		0x02, 'a', 'b',  // "ab"
		0x01, 'c',        // "c"
	}
	r := NewBufferReader(data)
	arr := deserializeStringArray18(r)
	if len(arr) != 2 {
		t.Fatalf("expected 2 strings, got %d", len(arr))
	}
	if arr[0] != "ab" || arr[1] != "c" {
		t.Errorf("expected ['ab', 'c'], got %v", arr)
	}
}

func TestDeserializeCompressedIntArray18(t *testing.T) {
	// [0, -1, 1] -> varints: 0(zigzag 0), 1(zigzag -1), 2(zigzag 1)
	data := []byte{0x03, 0x00, 0x01, 0x02}
	r := NewBufferReader(data)
	arr := deserializeCompressedIntArray18(r)
	if len(arr) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(arr))
	}
	if arr[0] != 0 || arr[1] != -1 || arr[2] != 1 {
		t.Errorf("expected [0, -1, 1], got %v", arr)
	}
}

func TestDeserializeBooleanArray18(t *testing.T) {
	// 10 booleans packed in 2 bytes: 0b00000011, 0b00000001
	// bits: 1,1,0,0,0,0,0,0, 1,0,0,0,0,0,0,0
	data := []byte{0x0A, 0x03, 0x01}
	r := NewBufferReader(data)
	arr := deserializeBooleanArray18(r)
	if len(arr) != 10 {
		t.Fatalf("expected 10 elements, got %d", len(arr))
	}
	expected := []bool{true, true, false, false, false, false, false, false, true, false}
	for i, want := range expected {
		if arr[i] != want {
			t.Errorf("index %d: expected %v, got %v", i, want, arr[i])
		}
	}
}

func TestDeserializeDictionary18(t *testing.T) {
	// keyType=CompressedInt(9), valueType=String(7), size=2
	// entry: key=varint 0 (zigzag 0=0), value="x"
	// entry: key=varint 2 (zigzag 1), value="y"
	data := []byte{
		0x09, 0x07,       // keyType=P18CompressedInt, valueType=P18String
		0x02,             // size=2
		0x00,             // key: varint 0 -> zigzag 0
		0x01, 'x',        // value: string len=1 "x"
		0x02,             // key: varint 2 -> zigzag 1
		0x01, 'y',        // value: string len=1 "y"
	}
	r := NewBufferReader(data)
	dict := deserializeDictionary18(r)
	if dict == nil {
		t.Fatal("expected non-nil dictionary")
	}
	if len(dict) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(dict))
	}
	if dict[int32(0)] != "x" {
		t.Errorf("dict[0]: expected 'x', got %v", dict[int32(0)])
	}
	if dict[int32(1)] != "y" {
		t.Errorf("dict[1]: expected 'y', got %v", dict[int32(1)])
	}
}

func TestDeserializeHashtable18(t *testing.T) {
	// size=2, each entry: [keyType][key][valueType][value]
	data := []byte{
		0x02,                   // size=2
		0x09, 0x00,             // keyType=CompressedInt, key=varint 0 (zigzag 0)
		0x07, 0x01, 'A',        // valueType=String, "A"
		0x09, 0x02,             // keyType=CompressedInt, key=varint 2 (zigzag 1)
		0x07, 0x01, 'B',        // valueType=String, "B"
	}
	r := NewBufferReader(data)
	dict := deserializeHashtable18(r)
	if dict == nil {
		t.Fatal("expected non-nil hashtable")
	}
	if len(dict) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(dict))
	}
}

func TestDeserializeObjectArray18(t *testing.T) {
	// [1, "hi"] -> types: CompressedInt, String
	data := []byte{
		0x02,                   // 2 elements
		0x09, 0x00,             // CompressedInt, varint 0 (zigzag 0)
		0x07, 0x02, 'h', 'i',   // String, len=2 "hi"
	}
	r := NewBufferReader(data)
	arr := deserializeObjectArray18(r)
	if len(arr) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(arr))
	}
	if arr[0] != int32(0) {
		t.Errorf("element 0: expected int32(0), got %v", arr[0])
	}
	if arr[1] != "hi" {
		t.Errorf("element 1: expected 'hi', got %v", arr[1])
	}
}

func TestDeserializeCustomType18(t *testing.T) {
	// typeCode=5, length=3, data=[1,2,3]
	data := []byte{0x05, 0x03, 0x01, 0x02, 0x03}
	r := NewBufferReader(data)
	got := deserializeCustomType18(r, 0)
	m, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", got)
	}
	if m["customTypeCode"] != byte(5) {
		t.Errorf("expected typeCode 5, got %v", m["customTypeCode"])
	}
	dataBytes, ok := m["data"].([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", m["data"])
	}
	if len(dataBytes) != 3 || dataBytes[0] != 1 || dataBytes[1] != 2 || dataBytes[2] != 3 {
		t.Errorf("expected [1,2,3], got %v", dataBytes)
	}
}

func TestDeserializeCustomTypeArray18(t *testing.T) {
	// typeCode=5, arrayLen=2, each element: [length][data]
	data := []byte{
		0x02,                   // array length = 2
		0x05,                   // custom type code
		0x02, 0xAA, 0xBB,       // elem 0: len=2, data
		0x01, 0xCC,             // elem 1: len=1, data
	}
	r := NewBufferReader(data)
	arr := deserializeCustomTypeArray18(r)
	if len(arr) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(arr))
	}
}

func TestDeserializeHashtableArray18(t *testing.T) {
	// arrayLen=2, each element is a hashtable
	data := []byte{
		0x02,                   // array length = 2
		0x00,                   // hashtable 0: empty (size=0)
		0x00,                   // hashtable 1: empty (size=0)
	}
	r := NewBufferReader(data)
	arr := deserializeHashtableArray18(r)
	if arr == nil {
		t.Fatal("expected non-nil array")
	}
}

func TestDeserializeDictionaryArray18(t *testing.T) {
	// keyType=CompressedInt(9), valueType=CompressedInt(9), arrayLen=1
	// dict 0: size=1, entry: key=0, value=0
	data := []byte{
		0x09, 0x09,   // keyType, valueType
		0x01,         // array length = 1
		0x01,         // dict 0: entry count = 1
		0x00,         // key: varint 0
		0x00,         // value: varint 0
	}
	r := NewBufferReader(data)
	arr := deserializeDictionaryArray18(r)
	if arr == nil {
		t.Fatal("expected non-nil array")
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 dict, got %d", len(arr))
	}
}

func TestDeserialize18_UnknownType(t *testing.T) {
	r := NewBufferReader([]byte{0x01, 0x02})
	got := deserialize18(r, 0xFF)
	if got != nil {
		t.Errorf("expected nil for unknown type, got %v", got)
	}
}

func TestDeserialize18_CustomTypeSlim(t *testing.T) {
	// Slim code 128 -> typeCode = 128 - 128 = 0
	// Then reads length + data
	data := []byte{0x02, 0xAA, 0xBB}
	r := NewBufferReader(data)
	got := deserialize18(r, P18CustomTypeSlim)
	m, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", got)
	}
	if m["customTypeCode"] != byte(0) {
		t.Errorf("expected typeCode 0, got %v", m["customTypeCode"])
	}
}

func TestVarintRoundTrip(t *testing.T) {
	// Verify that ReadVarint32 can decode what a correct encoder would produce
	values := []uint32{0, 1, 127, 128, 16383, 16384, 2097151, 2097152, 268435455, 268435456, math.MaxUint32}
	for _, v := range values {
		encoded := encodeVarint32(v)
		r := NewBufferReader(encoded)
		got, err := r.ReadVarint32()
		if err != nil {
			t.Errorf("ReadVarint32(%d): unexpected error: %v", v, err)
			continue
		}
		if got != v {
			t.Errorf("ReadVarint32: expected %d, got %d (encoded: %v)", v, got, encoded)
		}
	}
}

func TestVarint64RoundTrip(t *testing.T) {
	values := []uint64{0, 1, 127, 128, 16384, math.MaxUint32, math.MaxInt64, math.MaxUint64}
	for _, v := range values {
		encoded := encodeVarint64(v)
		r := NewBufferReader(encoded)
		got, err := r.ReadVarint64()
		if err != nil {
			t.Errorf("ReadVarint64(%d): unexpected error: %v", v, err)
			continue
		}
		if got != v {
			t.Errorf("ReadVarint64: expected %d, got %d (encoded: %v)", v, got, encoded)
		}
	}
}

// encodeVarint32 encodes a uint32 as a protobuf-style varint.
func encodeVarint32(v uint32) []byte {
	var buf []byte
	for v >= 0x80 {
		buf = append(buf, byte(v)|0x80)
		v >>= 7
	}
	buf = append(buf, byte(v))
	return buf
}

// encodeVarint64 encodes a uint64 as a protobuf-style varint.
func encodeVarint64(v uint64) []byte {
	var buf []byte
	for v >= 0x80 {
		buf = append(buf, byte(v)|0x80)
		v >>= 7
	}
	buf = append(buf, byte(v))
	return buf
}
