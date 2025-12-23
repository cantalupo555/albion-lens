package photon

// Protocol16 data types - ASCII character codes defined by Photon protocol
const (
	TypeUnknown        = 0    // Unknown type
	TypeNull           = '*'  // 42 - Null value
	TypeDictionary     = 'D'  // 68 - Typed dictionary
	TypeStringArray    = 'a'  // 97 - Array of strings
	TypeByte           = 'b'  // 98 - Byte (uint8)
	TypeDouble         = 'd'  // 100 - Float64
	TypeEventData      = 'e'  // 101 - Event data
	TypeFloat          = 'f'  // 102 - Float32
	TypeHashtable      = 'h'  // 104 - Hashtable (dynamic map)
	TypeInteger        = 'i'  // 105 - Int32
	TypeShort          = 'k'  // 107 - Int16
	TypeLong           = 'l'  // 108 - Int64
	TypeIntegerArray   = 'n'  // 110 - Array of int32
	TypeBoolean        = 'o'  // 111 - Boolean
	TypeOperationResp  = 'p'  // 112 - Operation response
	TypeOperationReq   = 'q'  // 113 - Operation request
	TypeString         = 's'  // 115 - UTF-8 string
	TypeByteArray      = 'x'  // 120 - Array of bytes
	TypeArray          = 'y'  // 121 - Typed array
	TypeObjectArray    = 'z'  // 122 - Array of objects
)

// decodeParameterTable decodes a Protocol16 parameter table using BufferReader
func decodeParameterTable(r *BufferReader) map[byte]interface{} {
	params := make(map[byte]interface{})

	if r.Remaining() < 2 {
		return params
	}

	// Read parameter count
	paramCount, err := r.ReadUint16()
	if err != nil {
		return params
	}

	for i := 0; i < int(paramCount) && !r.IsEmpty(); i++ {
		// Read parameter key
		paramKey, err := r.ReadByte()
		if err != nil {
			break
		}

		// Read parameter type
		paramType, err := r.ReadByte()
		if err != nil {
			break
		}

		// Read parameter value
		value := readValue(r, paramType)
		params[paramKey] = value
	}

	return params
}

// readValue reads a Protocol16 typed value using BufferReader
func readValue(r *BufferReader, paramType byte) interface{} {
	if r.IsEmpty() {
		return nil
	}

	switch paramType {
	case 0, TypeNull:
		return nil

	case TypeByte:
		val, err := r.ReadByte()
		if err != nil {
			return nil
		}
		return val

	case TypeBoolean:
		val, err := r.ReadBool()
		if err != nil {
			return nil
		}
		return val

	case TypeShort, 7: // 7 is also used for short in some cases
		val, err := r.ReadInt16()
		if err != nil {
			return nil
		}
		return val

	case TypeInteger:
		val, err := r.ReadInt32()
		if err != nil {
			return nil
		}
		return val

	case TypeLong:
		val, err := r.ReadInt64()
		if err != nil {
			return nil
		}
		return val

	case TypeFloat:
		val, err := r.ReadFloat32()
		if err != nil {
			return nil
		}
		return val

	case TypeDouble:
		val, err := r.ReadFloat64()
		if err != nil {
			return nil
		}
		return val

	case TypeString:
		val, err := r.ReadString()
		if err != nil {
			return ""
		}
		return val

	case TypeByteArray:
		length, err := r.ReadUint32()
		if err != nil {
			return nil
		}
		arr, err := r.ReadBytes(int(length))
		if err != nil {
			return nil
		}
		return arr

	case TypeArray:
		length, err := r.ReadUint16()
		if err != nil {
			return nil
		}
		elemType, err := r.ReadByte()
		if err != nil {
			return nil
		}

		arr := make([]interface{}, length)
		for i := 0; i < int(length) && !r.IsEmpty(); i++ {
			arr[i] = readValue(r, elemType)
		}
		return arr

	case TypeIntegerArray:
		length, err := r.ReadUint32()
		if err != nil {
			return nil
		}

		arr := make([]int32, length)
		for i := 0; i < int(length); i++ {
			val, err := r.ReadInt32()
			if err != nil {
				break
			}
			arr[i] = val
		}
		return arr

	case TypeStringArray:
		length, err := r.ReadUint16()
		if err != nil {
			return nil
		}

		arr := make([]string, length)
		for i := 0; i < int(length) && !r.IsEmpty(); i++ {
			str := readValue(r, TypeString)
			if s, ok := str.(string); ok {
				arr[i] = s
			}
		}
		return arr

	case TypeDictionary, TypeHashtable:
		keyType, err := r.ReadByte()
		if err != nil {
			return nil
		}
		valueType, err := r.ReadByte()
		if err != nil {
			return nil
		}
		length, err := r.ReadUint16()
		if err != nil {
			return nil
		}

		dict := make(map[interface{}]interface{})
		for i := 0; i < int(length) && !r.IsEmpty(); i++ {
			// Read key
			var key interface{}
			if keyType == 0 || keyType == TypeUnknown {
				// Unknown type, read type first
				actualKeyType, err := r.ReadByte()
				if err != nil {
					break
				}
				key = readValue(r, actualKeyType)
			} else {
				key = readValue(r, keyType)
			}

			// Read value
			var val interface{}
			if valueType == 0 || valueType == TypeUnknown {
				// Unknown type, read type first
				actualValueType, err := r.ReadByte()
				if err != nil {
					break
				}
				val = readValue(r, actualValueType)
			} else {
				val = readValue(r, valueType)
			}

			dict[key] = val
		}
		return dict

	case TypeObjectArray:
		length, err := r.ReadUint16()
		if err != nil {
			return nil
		}

		arr := make([]interface{}, length)
		for i := 0; i < int(length) && !r.IsEmpty(); i++ {
			// Each element has its own type
			elemType, err := r.ReadByte()
			if err != nil {
				break
			}
			arr[i] = readValue(r, elemType)
		}
		return arr

	default:
		// Unknown type, skip
		return nil
	}
}
