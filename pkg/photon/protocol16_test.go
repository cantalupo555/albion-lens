package photon

import (
	"encoding/binary"
	"math"
	"testing"
)

// Helper to create a BufferReader from bytes
func newTestReader(data []byte) *BufferReader {
	return NewBufferReader(data)
}

// TestReadValueNull tests null value reading
func TestReadValueNull(t *testing.T) {
	r := newTestReader([]byte{})

	result := readValue(r, TypeNull)
	if result != nil {
		t.Errorf("expected nil for TypeNull, got %v", result)
	}

	result = readValue(r, 0)
	if result != nil {
		t.Errorf("expected nil for type 0, got %v", result)
	}
}

// TestReadValueByte tests byte value reading
func TestReadValueByte(t *testing.T) {
	r := newTestReader([]byte{0x42})

	result := readValue(r, TypeByte)
	if result != byte(0x42) {
		t.Errorf("expected 0x42, got %v", result)
	}
}

// TestReadValueByteEmpty tests byte reading from empty buffer
func TestReadValueByteEmpty(t *testing.T) {
	r := newTestReader([]byte{})

	result := readValue(r, TypeByte)
	if result != nil {
		t.Errorf("expected nil for empty buffer, got %v", result)
	}
}

// TestReadValueBoolean tests boolean value reading
func TestReadValueBoolean(t *testing.T) {
	// Test true
	r := newTestReader([]byte{1})
	result := readValue(r, TypeBoolean)
	if result != true {
		t.Errorf("expected true, got %v", result)
	}

	// Test false
	r = newTestReader([]byte{0})
	result = readValue(r, TypeBoolean)
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}

// TestReadValueShort tests int16 value reading
func TestReadValueShort(t *testing.T) {
	data := make([]byte, 2)
	binary.BigEndian.PutUint16(data, uint16(0x1234))

	r := newTestReader(data)
	result := readValue(r, TypeShort)

	if result != int16(0x1234) {
		t.Errorf("expected 0x1234, got %v", result)
	}
}

// TestReadValueShortAlternative tests int16 reading with type 7
func TestReadValueShortAlternative(t *testing.T) {
	data := make([]byte, 2)
	binary.BigEndian.PutUint16(data, uint16(0x5678))

	r := newTestReader(data)
	result := readValue(r, 7) // Alternative short type

	if result != int16(0x5678) {
		t.Errorf("expected 0x5678, got %v", result)
	}
}

// TestReadValueInteger tests int32 value reading
func TestReadValueInteger(t *testing.T) {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, uint32(0x12345678))

	r := newTestReader(data)
	result := readValue(r, TypeInteger)

	if result != int32(0x12345678) {
		t.Errorf("expected 0x12345678, got %v", result)
	}
}

// TestReadValueLong tests int64 value reading
func TestReadValueLong(t *testing.T) {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, uint64(0x123456789ABCDEF0))

	r := newTestReader(data)
	result := readValue(r, TypeLong)

	if result != int64(0x123456789ABCDEF0) {
		t.Errorf("expected 0x123456789ABCDEF0, got %v", result)
	}
}

// TestReadValueFloat tests float32 value reading
func TestReadValueFloat(t *testing.T) {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, math.Float32bits(3.14))

	r := newTestReader(data)
	result := readValue(r, TypeFloat)

	if val, ok := result.(float32); !ok || math.Abs(float64(val-3.14)) > 0.001 {
		t.Errorf("expected ~3.14, got %v", result)
	}
}

// TestReadValueDouble tests float64 value reading
func TestReadValueDouble(t *testing.T) {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, math.Float64bits(3.14159265359))

	r := newTestReader(data)
	result := readValue(r, TypeDouble)

	if val, ok := result.(float64); !ok || math.Abs(val-3.14159265359) > 0.0000001 {
		t.Errorf("expected ~3.14159265359, got %v", result)
	}
}

// TestReadValueString tests string value reading
func TestReadValueString(t *testing.T) {
	str := "Hello"
	data := make([]byte, 2+len(str))
	binary.BigEndian.PutUint16(data[0:2], uint16(len(str)))
	copy(data[2:], str)

	r := newTestReader(data)
	result := readValue(r, TypeString)

	if result != str {
		t.Errorf("expected '%s', got '%v'", str, result)
	}
}

// TestReadValueStringEmpty tests empty string reading
func TestReadValueStringEmpty(t *testing.T) {
	data := make([]byte, 2)
	binary.BigEndian.PutUint16(data, 0)

	r := newTestReader(data)
	result := readValue(r, TypeString)

	if result != "" {
		t.Errorf("expected empty string, got '%v'", result)
	}
}

// TestReadValueByteArray tests byte array reading
func TestReadValueByteArray(t *testing.T) {
	arr := []byte{0x01, 0x02, 0x03, 0x04}
	data := make([]byte, 4+len(arr))
	binary.BigEndian.PutUint32(data[0:4], uint32(len(arr)))
	copy(data[4:], arr)

	r := newTestReader(data)
	result := readValue(r, TypeByteArray)

	if resultArr, ok := result.([]byte); !ok {
		t.Errorf("expected []byte, got %T", result)
	} else if len(resultArr) != len(arr) {
		t.Errorf("expected length %d, got %d", len(arr), len(resultArr))
	} else {
		for i, b := range arr {
			if resultArr[i] != b {
				t.Errorf("byte %d: expected 0x%02x, got 0x%02x", i, b, resultArr[i])
			}
		}
	}
}

// TestReadValueTypedArray tests typed array reading
func TestReadValueTypedArray(t *testing.T) {
	// Array of 3 bytes
	data := []byte{
		0x00, 0x03, // Length: 3
		TypeByte,   // Element type
		0x0A, 0x0B, 0x0C, // Values
	}

	r := newTestReader(data)
	result := readValue(r, TypeArray)

	arr, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", result)
	}

	if len(arr) != 3 {
		t.Fatalf("expected length 3, got %d", len(arr))
	}

	expected := []byte{0x0A, 0x0B, 0x0C}
	for i, v := range expected {
		if arr[i] != v {
			t.Errorf("element %d: expected 0x%02x, got %v", i, v, arr[i])
		}
	}
}

// TestReadValueIntegerArray tests integer array reading
func TestReadValueIntegerArray(t *testing.T) {
	data := make([]byte, 4+3*4) // 4 bytes for length + 3 int32s
	binary.BigEndian.PutUint32(data[0:4], 3)
	binary.BigEndian.PutUint32(data[4:8], 100)
	binary.BigEndian.PutUint32(data[8:12], 200)
	binary.BigEndian.PutUint32(data[12:16], 300)

	r := newTestReader(data)
	result := readValue(r, TypeIntegerArray)

	arr, ok := result.([]int32)
	if !ok {
		t.Fatalf("expected []int32, got %T", result)
	}

	expected := []int32{100, 200, 300}
	if len(arr) != len(expected) {
		t.Fatalf("expected length %d, got %d", len(expected), len(arr))
	}

	for i, v := range expected {
		if arr[i] != v {
			t.Errorf("element %d: expected %d, got %d", i, v, arr[i])
		}
	}
}

// TestReadValueStringArray tests string array reading
func TestReadValueStringArray(t *testing.T) {
	// Build string array: ["Hi", "Go"]
	data := []byte{
		0x00, 0x02, // Length: 2 strings
		0x00, 0x02, 'H', 'i', // "Hi"
		0x00, 0x02, 'G', 'o', // "Go"
	}

	r := newTestReader(data)
	result := readValue(r, TypeStringArray)

	arr, ok := result.([]string)
	if !ok {
		t.Fatalf("expected []string, got %T", result)
	}

	expected := []string{"Hi", "Go"}
	if len(arr) != len(expected) {
		t.Fatalf("expected length %d, got %d", len(expected), len(arr))
	}

	for i, v := range expected {
		if arr[i] != v {
			t.Errorf("element %d: expected '%s', got '%s'", i, v, arr[i])
		}
	}
}

// TestReadValueObjectArray tests object array reading
func TestReadValueObjectArray(t *testing.T) {
	// Array with mixed types: byte(0x42), bool(true)
	data := []byte{
		0x00, 0x02,  // Length: 2
		TypeByte, 0x42, // First element: byte 0x42
		TypeBoolean, 0x01, // Second element: bool true
	}

	r := newTestReader(data)
	result := readValue(r, TypeObjectArray)

	arr, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", result)
	}

	if len(arr) != 2 {
		t.Fatalf("expected length 2, got %d", len(arr))
	}

	if arr[0] != byte(0x42) {
		t.Errorf("element 0: expected byte 0x42, got %v (%T)", arr[0], arr[0])
	}

	if arr[1] != true {
		t.Errorf("element 1: expected true, got %v (%T)", arr[1], arr[1])
	}
}

// TestReadValueDictionary tests dictionary reading with known types
func TestReadValueDictionary(t *testing.T) {
	// Dictionary with byte keys and int32 values: {1: 100, 2: 200}
	data := []byte{
		TypeByte,    // Key type
		TypeInteger, // Value type
		0x00, 0x02,  // Length: 2 entries
		0x01,                   // Key 1
		0x00, 0x00, 0x00, 0x64, // Value 100
		0x02,                   // Key 2
		0x00, 0x00, 0x00, 0xC8, // Value 200
	}

	r := newTestReader(data)
	result := readValue(r, TypeDictionary)

	dict, ok := result.(map[interface{}]interface{})
	if !ok {
		t.Fatalf("expected map[interface{}]interface{}, got %T", result)
	}

	if len(dict) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(dict))
	}

	if dict[byte(1)] != int32(100) {
		t.Errorf("dict[1]: expected 100, got %v", dict[byte(1)])
	}

	if dict[byte(2)] != int32(200) {
		t.Errorf("dict[2]: expected 200, got %v", dict[byte(2)])
	}
}

// TestReadValueHashtable tests hashtable reading (same as dictionary)
func TestReadValueHashtable(t *testing.T) {
	// Hashtable with byte keys and byte values: {1: 10}
	data := []byte{
		TypeByte, // Key type
		TypeByte, // Value type
		0x00, 0x01, // Length: 1 entry
		0x01, // Key 1
		0x0A, // Value 10
	}

	r := newTestReader(data)
	result := readValue(r, TypeHashtable)

	dict, ok := result.(map[interface{}]interface{})
	if !ok {
		t.Fatalf("expected map[interface{}]interface{}, got %T", result)
	}

	if dict[byte(1)] != byte(10) {
		t.Errorf("dict[1]: expected 10, got %v", dict[byte(1)])
	}
}

// TestReadValueDictionaryUnknownTypes tests dictionary with unknown key/value types
func TestReadValueDictionaryUnknownTypes(t *testing.T) {
	// Dictionary with unknown types (type=0): each entry has inline type
	data := []byte{
		0x00, // Unknown key type
		0x00, // Unknown value type
		0x00, 0x01, // Length: 1 entry
		TypeByte, 0x01, // Key: byte 1
		TypeBoolean, 0x01, // Value: bool true
	}

	r := newTestReader(data)
	result := readValue(r, TypeDictionary)

	dict, ok := result.(map[interface{}]interface{})
	if !ok {
		t.Fatalf("expected map[interface{}]interface{}, got %T", result)
	}

	if dict[byte(1)] != true {
		t.Errorf("dict[1]: expected true, got %v", dict[byte(1)])
	}
}

// TestReadValueUnknownType tests unknown type handling
func TestReadValueUnknownType(t *testing.T) {
	r := newTestReader([]byte{0x01, 0x02, 0x03})

	result := readValue(r, 0xFF) // Unknown type

	if result != nil {
		t.Errorf("expected nil for unknown type, got %v", result)
	}
}

// TestReadValueEmptyBuffer tests reading from empty buffer
func TestReadValueEmptyBuffer(t *testing.T) {
	r := newTestReader([]byte{})

	result := readValue(r, TypeInteger)
	if result != nil {
		t.Errorf("expected nil for empty buffer, got %v", result)
	}
}

// TestDecodeParameterTable tests parameter table decoding
func TestDecodeParameterTable(t *testing.T) {
	// Parameter table with 2 entries:
	// Key 1 (byte): int32(100)
	// Key 2 (byte): string "Hi"
	data := []byte{
		0x00, 0x02, // Param count: 2
		0x01, TypeInteger, 0x00, 0x00, 0x00, 0x64, // Key 1, type int, value 100
		0x02, TypeString, 0x00, 0x02, 'H', 'i', // Key 2, type string, value "Hi"
	}

	r := newTestReader(data)
	params := decodeParameterTable(r)

	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(params))
	}

	if params[1] != int32(100) {
		t.Errorf("params[1]: expected 100, got %v", params[1])
	}

	if params[2] != "Hi" {
		t.Errorf("params[2]: expected 'Hi', got '%v'", params[2])
	}
}

// TestDecodeParameterTableEmpty tests empty parameter table
func TestDecodeParameterTableEmpty(t *testing.T) {
	data := []byte{0x00, 0x00} // Count: 0

	r := newTestReader(data)
	params := decodeParameterTable(r)

	if len(params) != 0 {
		t.Errorf("expected 0 params, got %d", len(params))
	}
}

// TestDecodeParameterTableInsufficientData tests handling of truncated data
func TestDecodeParameterTableInsufficientData(t *testing.T) {
	// Not enough data for the count field
	r := newTestReader([]byte{0x00})
	params := decodeParameterTable(r)
	if len(params) != 0 {
		t.Errorf("expected empty map for insufficient data, got %d params", len(params))
	}

	// Empty buffer
	r = newTestReader([]byte{})
	params = decodeParameterTable(r)
	if len(params) != 0 {
		t.Errorf("expected empty map for empty buffer, got %d params", len(params))
	}
}

// TestDecodeParameterTableTruncatedEntry tests handling of truncated entries
func TestDecodeParameterTableTruncatedEntry(t *testing.T) {
	// Says 2 params but only has data for 1
	data := []byte{
		0x00, 0x02, // Param count: 2
		0x01, TypeByte, 0x42, // Key 1, type byte, value 0x42
		// Missing second entry
	}

	r := newTestReader(data)
	params := decodeParameterTable(r)

	// Should have parsed at least the first entry
	if params[1] != byte(0x42) {
		t.Errorf("params[1]: expected 0x42, got %v", params[1])
	}
}

// TestTypeConstants tests that type constants have expected values
func TestTypeConstants(t *testing.T) {
	testCases := []struct {
		name     string
		constant byte
		expected byte
	}{
		{"TypeNull", TypeNull, '*'},
		{"TypeDictionary", TypeDictionary, 'D'},
		{"TypeStringArray", TypeStringArray, 'a'},
		{"TypeByte", TypeByte, 'b'},
		{"TypeDouble", TypeDouble, 'd'},
		{"TypeEventData", TypeEventData, 'e'},
		{"TypeFloat", TypeFloat, 'f'},
		{"TypeHashtable", TypeHashtable, 'h'},
		{"TypeInteger", TypeInteger, 'i'},
		{"TypeShort", TypeShort, 'k'},
		{"TypeLong", TypeLong, 'l'},
		{"TypeIntegerArray", TypeIntegerArray, 'n'},
		{"TypeBoolean", TypeBoolean, 'o'},
		{"TypeOperationResp", TypeOperationResp, 'p'},
		{"TypeOperationReq", TypeOperationReq, 'q'},
		{"TypeString", TypeString, 's'},
		{"TypeByteArray", TypeByteArray, 'x'},
		{"TypeArray", TypeArray, 'y'},
		{"TypeObjectArray", TypeObjectArray, 'z'},
	}

	for _, tc := range testCases {
		if tc.constant != tc.expected {
			t.Errorf("%s: expected '%c' (0x%02x), got '%c' (0x%02x)",
				tc.name, tc.expected, tc.expected, tc.constant, tc.constant)
		}
	}
}

// TestReadValueNestedArray tests array containing arrays
func TestReadValueNestedArray(t *testing.T) {
	// Array of 2 typed arrays (each containing bytes)
	// This tests recursive readValue calls
	data := []byte{
		0x00, 0x02, // Outer array length: 2
		TypeArray,          // Element type: array
		0x00, 0x01, TypeByte, 0x0A, // First inner array: [0x0A]
		0x00, 0x01, TypeByte, 0x0B, // Second inner array: [0x0B]
	}

	r := newTestReader(data)
	result := readValue(r, TypeArray)

	arr, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", result)
	}

	if len(arr) != 2 {
		t.Fatalf("expected outer length 2, got %d", len(arr))
	}
}

// TestDecodeParameterTableWithAllTypes tests parameter table with various types
func TestDecodeParameterTableWithAllTypes(t *testing.T) {
	// Build a complex parameter table
	data := []byte{
		0x00, 0x04, // Param count: 4

		// Param 1: byte
		0x01, TypeByte, 0xFF,

		// Param 2: boolean
		0x02, TypeBoolean, 0x01,

		// Param 3: int16
		0x03, TypeShort, 0x12, 0x34,

		// Param 4: null
		0x04, TypeNull,
	}

	r := newTestReader(data)
	params := decodeParameterTable(r)

	if len(params) != 4 {
		t.Fatalf("expected 4 params, got %d", len(params))
	}

	if params[1] != byte(0xFF) {
		t.Errorf("params[1]: expected 0xFF, got %v", params[1])
	}

	if params[2] != true {
		t.Errorf("params[2]: expected true, got %v", params[2])
	}

	if params[3] != int16(0x1234) {
		t.Errorf("params[3]: expected 0x1234, got %v", params[3])
	}

	if params[4] != nil {
		t.Errorf("params[4]: expected nil, got %v", params[4])
	}
}
