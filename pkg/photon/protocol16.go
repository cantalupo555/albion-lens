package photon

import (
	"encoding/binary"
	"math"
)

// Protocol16 data types
const (
	TypeUnknown        = 0
	TypeNull           = 42  // '*'
	TypeDictionary     = 68  // 'D'
	TypeStringArray    = 97  // 'a'
	TypeByte           = 98  // 'b'
	TypeDouble         = 100 // 'd'
	TypeEventData      = 101 // 'e'
	TypeFloat          = 102 // 'f'
	TypeInteger        = 105 // 'i'
	TypeHashtable      = 104 // 'h'
	TypeShort          = 107 // 'k'
	TypeLong           = 108 // 'l'
	TypeIntegerArray   = 110 // 'n'
	TypeBoolean        = 111 // 'o'
	TypeOperationResp  = 112 // 'p'
	TypeOperationReq   = 113 // 'q'
	TypeString         = 115 // 's'
	TypeByteArray      = 120 // 'x'
	TypeArray          = 121 // 'y'
	TypeObjectArray    = 122 // 'z'
)

// decodeParameterTable decodes a Protocol16 parameter table
func decodeParameterTable(data []byte) map[byte]interface{} {
	params := make(map[byte]interface{})

	if len(data) < 2 {
		return params
	}

	offset := 0

	// Read parameter count
	paramCount := int(binary.BigEndian.Uint16(data[offset:]))
	offset += 2

	for i := 0; i < paramCount && offset < len(data); i++ {
		if offset >= len(data) {
			break
		}

		// Read parameter key
		paramKey := data[offset]
		offset++

		if offset >= len(data) {
			break
		}

		// Read parameter type
		paramType := data[offset]
		offset++

		// Read parameter value
		value, newOffset := readValue(data, offset, paramType)
		params[paramKey] = value
		offset = newOffset
	}

	return params
}

// readValue reads a Protocol16 typed value
func readValue(data []byte, offset int, paramType byte) (interface{}, int) {
	if offset >= len(data) {
		return nil, offset
	}

	switch paramType {
	case 0, TypeNull:
		return nil, offset

	case TypeByte:
		if offset >= len(data) {
			return nil, offset
		}
		return data[offset], offset + 1

	case TypeBoolean:
		if offset >= len(data) {
			return nil, offset
		}
		return data[offset] == 1, offset + 1

	case TypeShort, 7: // 7 is also used for short in some cases
		if offset+2 > len(data) {
			return nil, offset
		}
		val := int16(binary.BigEndian.Uint16(data[offset:]))
		return val, offset + 2

	case TypeInteger:
		if offset+4 > len(data) {
			return nil, offset
		}
		val := int32(binary.BigEndian.Uint32(data[offset:]))
		return val, offset + 4

	case TypeLong:
		if offset+8 > len(data) {
			return nil, offset
		}
		val := int64(binary.BigEndian.Uint64(data[offset:]))
		return val, offset + 8

	case TypeFloat:
		if offset+4 > len(data) {
			return nil, offset
		}
		bits := binary.BigEndian.Uint32(data[offset:])
		val := math.Float32frombits(bits)
		return val, offset + 4

	case TypeDouble:
		if offset+8 > len(data) {
			return nil, offset
		}
		bits := binary.BigEndian.Uint64(data[offset:])
		val := math.Float64frombits(bits)
		return val, offset + 8

	case TypeString:
		if offset+2 > len(data) {
			return nil, offset
		}
		length := int(binary.BigEndian.Uint16(data[offset:]))
		offset += 2
		if offset+length > len(data) {
			return "", offset
		}
		str := string(data[offset : offset+length])
		return str, offset + length

	case TypeByteArray:
		if offset+4 > len(data) {
			return nil, offset
		}
		length := int(binary.BigEndian.Uint32(data[offset:]))
		offset += 4
		if offset+length > len(data) {
			return nil, offset
		}
		arr := make([]byte, length)
		copy(arr, data[offset:offset+length])
		return arr, offset + length

	case TypeArray:
		if offset+3 > len(data) {
			return nil, offset
		}
		length := int(binary.BigEndian.Uint16(data[offset:]))
		offset += 2
		elemType := data[offset]
		offset++

		arr := make([]interface{}, length)
		for i := 0; i < length && offset < len(data); i++ {
			val, newOffset := readValue(data, offset, elemType)
			arr[i] = val
			offset = newOffset
		}
		return arr, offset

	case TypeIntegerArray:
		if offset+4 > len(data) {
			return nil, offset
		}
		length := int(binary.BigEndian.Uint32(data[offset:]))
		offset += 4

		arr := make([]int32, length)
		for i := 0; i < length && offset+4 <= len(data); i++ {
			arr[i] = int32(binary.BigEndian.Uint32(data[offset:]))
			offset += 4
		}
		return arr, offset

	case TypeStringArray:
		if offset+2 > len(data) {
			return nil, offset
		}
		length := int(binary.BigEndian.Uint16(data[offset:]))
		offset += 2

		arr := make([]string, length)
		for i := 0; i < length && offset < len(data); i++ {
			str, newOffset := readValue(data, offset, TypeString)
			if s, ok := str.(string); ok {
				arr[i] = s
			}
			offset = newOffset
		}
		return arr, offset

	case TypeDictionary, TypeHashtable:
		if offset+4 > len(data) {
			return nil, offset
		}

		keyType := data[offset]
		offset++
		valueType := data[offset]
		offset++
		length := int(binary.BigEndian.Uint16(data[offset:]))
		offset += 2

		dict := make(map[interface{}]interface{})
		for i := 0; i < length && offset < len(data); i++ {
			// Read key
			var key interface{}
			if keyType == 0 || keyType == TypeUnknown {
				// Unknown type, read type first
				if offset >= len(data) {
					break
				}
				actualKeyType := data[offset]
				offset++
				key, offset = readValue(data, offset, actualKeyType)
			} else {
				key, offset = readValue(data, offset, keyType)
			}

			// Read value
			var val interface{}
			if valueType == 0 || valueType == TypeUnknown {
				// Unknown type, read type first
				if offset >= len(data) {
					break
				}
				actualValueType := data[offset]
				offset++
				val, offset = readValue(data, offset, actualValueType)
			} else {
				val, offset = readValue(data, offset, valueType)
			}

			dict[key] = val
		}
		return dict, offset

	case TypeObjectArray:
		if offset+2 > len(data) {
			return nil, offset
		}
		length := int(binary.BigEndian.Uint16(data[offset:]))
		offset += 2

		arr := make([]interface{}, length)
		for i := 0; i < length && offset < len(data); i++ {
			// Each element has its own type
			if offset >= len(data) {
				break
			}
			elemType := data[offset]
			offset++
			val, newOffset := readValue(data, offset, elemType)
			arr[i] = val
			offset = newOffset
		}
		return arr, offset

	default:
		// Unknown type, skip
		return nil, offset
	}
}
