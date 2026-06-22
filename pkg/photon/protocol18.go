package photon

// Protocol18 type codes (from Protocol18Type.cs)
const (
	P18Unknown           = 0
	P18Boolean           = 2
	P18Byte              = 3
	P18Short             = 4
	P18Float             = 5
	P18Double            = 6
	P18String            = 7
	P18Null              = 8
	P18CompressedInt     = 9
	P18CompressedLong    = 10
	P18Int1              = 11
	P18Int1Negative      = 12
	P18Int2              = 13
	P18Int2Negative      = 14
	P18Long1             = 15
	P18Long1Negative     = 16
	P18Long2             = 17
	P18Long2Negative     = 18
	P18Custom            = 19
	P18Dictionary        = 20
	P18Hashtable         = 21
	P18ObjectArray       = 23
	P18OperationRequest  = 24
	P18OperationResponse = 25
	P18EventData         = 26
	P18BooleanFalse      = 27
	P18BooleanTrue       = 28
	P18ShortZero         = 29
	P18IntZero           = 30
	P18LongZero          = 31
	P18FloatZero         = 32
	P18DoubleZero        = 33
	P18ByteZero          = 34
	P18Array             = 64
	P18BooleanArray      = 66
	P18ByteArray         = 67
	P18ShortArray        = 68
	P18FloatArray        = 69
	P18DoubleArray       = 70
	P18StringArray       = 71
	P18CompressedIntArray  = 73
	P18CompressedLongArray = 74
	P18CustomTypeArray   = 83
	P18DictionaryArray   = 84
	P18HashtableArray    = 85
	P18CustomTypeSlim    = 128
)

// decodeParameterTable18 decodes a Protocol18 parameter table.
// Format: [count:1byte] then for each entry: [key:1byte][typeCode:1byte][value]
func decodeParameterTable18(r *BufferReader) map[byte]interface{} {
	if r.Remaining() < 1 {
		return make(map[byte]interface{})
	}

	size, _ := r.ReadByte()
	params := make(map[byte]interface{}, size)

	for i := 0; i < int(size); i++ {
		if r.Remaining() < 2 {
			break
		}
		key, _ := r.ReadByte()
		typeCode, _ := r.ReadByte()
		params[key] = deserialize18(r, typeCode)
	}

	return params
}

// deserialize18 dispatches based on Protocol18 type code.
func deserialize18(r *BufferReader, typeCode byte) interface{} {
	if typeCode >= P18CustomTypeSlim {
		return deserializeCustomType18(r, typeCode)
	}

	switch typeCode {
	case P18Boolean:
		val, _ := r.ReadByte()
		return val != 0

	case P18Byte, P18ByteZero:
		if typeCode == P18ByteZero {
			return byte(0)
		}
		val, _ := r.ReadByte()
		return val

	case P18Short:
		val, _ := r.ReadInt16LE()
		return val

	case P18Float:
		val, _ := r.ReadFloat32LE()
		return val

	case P18Double:
		val, _ := r.ReadFloat64LE()
		return val

	case P18String:
		return readString18(r)

	case P18Null, P18Unknown:
		return nil

	case P18CompressedInt:
		raw, _ := r.ReadVarint32()
		return DecodeZigZag32(raw)

	case P18CompressedLong:
		raw, _ := r.ReadVarint64()
		return DecodeZigZag64(raw)

	case P18Int1:
		val, _ := r.ReadByte()
		return int32(val)

	case P18Int1Negative:
		val, _ := r.ReadByte()
		return -int32(val)

	case P18Int2:
		val, _ := r.ReadUint16LE()
		return int32(val)

	case P18Int2Negative:
		val, _ := r.ReadUint16LE()
		return -int32(val)

	case P18Long1:
		val, _ := r.ReadByte()
		return int64(val)

	case P18Long1Negative:
		val, _ := r.ReadByte()
		return -int64(val)

	case P18Long2:
		val, _ := r.ReadUint16LE()
		return int64(val)

	case P18Long2Negative:
		val, _ := r.ReadUint16LE()
		return -int64(val)

	case P18Dictionary:
		return deserializeDictionary18(r)

	case P18Hashtable:
		return deserializeHashtable18(r)

	case P18ObjectArray, P18Array:
		return deserializeObjectArray18(r)

	case P18CustomTypeArray:
		return deserializeCustomTypeArray18(r)

	case P18DictionaryArray:
		return deserializeDictionaryArray18(r)

	case P18HashtableArray:
		return deserializeHashtableArray18(r)

	case P18BooleanFalse:
		return false

	case P18BooleanTrue:
		return true

	case P18ShortZero:
		return int16(0)

	case P18IntZero:
		return int32(0)

	case P18LongZero:
		return int64(0)

	case P18FloatZero:
		return float32(0)

	case P18DoubleZero:
		return float64(0)

	case P18ByteArray:
		return deserializeByteArray18(r)

	case P18ShortArray:
		return deserializeShortArray18(r)

	case P18FloatArray:
		return deserializeFloatArray18(r)

	case P18DoubleArray:
		return deserializeDoubleArray18(r)

	case P18StringArray:
		return deserializeStringArray18(r)

	case P18CompressedIntArray:
		return deserializeCompressedIntArray18(r)

	case P18CompressedLongArray:
		return deserializeCompressedLongArray18(r)

	case P18BooleanArray:
		return deserializeBooleanArray18(r)

	case P18Custom:
		return deserializeCustomType18(r, 0)

	default:
		return nil
	}
}

func readString18(r *BufferReader) string {
	strLen, err := r.ReadVarint32()
	if err != nil || strLen == 0 {
		return ""
	}
	if strLen > uint32(r.Remaining()) {
		_ = r.Skip(r.Remaining())
		return ""
	}
	b, err := r.ReadBytes(int(strLen))
	if err != nil {
		return ""
	}
	return string(b)
}

func deserializeDictionary18(r *BufferReader) map[interface{}]interface{} {
	if r.Remaining() < 2 {
		return nil
	}
	keyType, _ := r.ReadByte()
	valueType, _ := r.ReadByte()
	return deserializeDictEntries18(r, keyType, valueType)
}

func deserializeDictEntries18(r *BufferReader, keyType, valueType byte) map[interface{}]interface{} {
	size, err := r.ReadVarint32()
	if err != nil || size == 0 || size > uint32(r.Remaining()) {
		return make(map[interface{}]interface{})
	}

	dict := make(map[interface{}]interface{}, size)

	for i := uint32(0); i < size && !r.IsEmpty(); i++ {
		var key, val interface{}
		if keyType == P18Unknown {
			if r.IsEmpty() {
				break
			}
			kt, _ := r.ReadByte()
			key = deserialize18(r, kt)
		} else {
			key = deserialize18(r, keyType)
		}
		if valueType == P18Unknown {
			if r.IsEmpty() {
				break
			}
			vt, _ := r.ReadByte()
			val = deserialize18(r, vt)
		} else {
			val = deserialize18(r, valueType)
		}
		if key != nil {
			dict[key] = val
		}
	}

	return dict
}

func deserializeHashtable18(r *BufferReader) map[interface{}]interface{} {
	size, err := r.ReadVarint32()
	if err != nil || size == 0 || size > uint32(r.Remaining()) {
		return make(map[interface{}]interface{})
	}

	dict := make(map[interface{}]interface{}, size)

	for i := uint32(0); i < size && !r.IsEmpty(); i++ {
		if r.IsEmpty() {
			break
		}
		kt, _ := r.ReadByte()
		key := deserialize18(r, kt)
		if r.IsEmpty() {
			break
		}
		vt, _ := r.ReadByte()
		val := deserialize18(r, vt)
		if key != nil {
			dict[key] = val
		}
	}

	return dict
}

func deserializeObjectArray18(r *BufferReader) []interface{} {
	size, err := r.ReadVarint32()
	if err != nil || size > uint32(r.Remaining()) {
		return nil
	}

	arr := make([]interface{}, size)
	for i := uint32(0); i < size && !r.IsEmpty(); i++ {
		elemType, _ := r.ReadByte()
		arr[i] = deserialize18(r, elemType)
	}
	return arr
}

func deserializeByteArray18(r *BufferReader) []byte {
	arrLen, err := r.ReadVarint32()
	if err != nil || arrLen > uint32(r.Remaining()) {
		return nil
	}
	b, err := r.ReadBytes(int(arrLen))
	if err != nil {
		return nil
	}
	return b
}

func deserializeShortArray18(r *BufferReader) []int16 {
	arrLen, err := r.ReadVarint32()
	if err != nil || arrLen > uint32(r.Remaining())/2 {
		return nil
	}
	arr := make([]int16, arrLen)
	for i := uint32(0); i < arrLen; i++ {
		val, err := r.ReadInt16LE()
		if err != nil {
			break
		}
		arr[i] = val
	}
	return arr
}

func deserializeFloatArray18(r *BufferReader) []float32 {
	arrLen, err := r.ReadVarint32()
	if err != nil || arrLen > uint32(r.Remaining())/4 {
		return nil
	}
	arr := make([]float32, arrLen)
	for i := uint32(0); i < arrLen; i++ {
		val, err := r.ReadFloat32LE()
		if err != nil {
			break
		}
		arr[i] = val
	}
	return arr
}

func deserializeDoubleArray18(r *BufferReader) []float64 {
	arrLen, err := r.ReadVarint32()
	if err != nil || arrLen > uint32(r.Remaining())/8 {
		return nil
	}
	arr := make([]float64, arrLen)
	for i := uint32(0); i < arrLen; i++ {
		val, err := r.ReadFloat64LE()
		if err != nil {
			break
		}
		arr[i] = val
	}
	return arr
}

func deserializeStringArray18(r *BufferReader) []string {
	arrLen, err := r.ReadVarint32()
	if err != nil || arrLen > uint32(r.Remaining()) {
		return nil
	}
	arr := make([]string, arrLen)
	for i := uint32(0); i < arrLen && !r.IsEmpty(); i++ {
		arr[i] = readString18(r)
	}
	return arr
}

func deserializeCompressedIntArray18(r *BufferReader) []int32 {
	arrLen, err := r.ReadVarint32()
	if err != nil || arrLen > uint32(r.Remaining()) {
		return nil
	}
	arr := make([]int32, arrLen)
	for i := uint32(0); i < arrLen && !r.IsEmpty(); i++ {
		raw, err := r.ReadVarint32()
		if err != nil {
			break
		}
		arr[i] = DecodeZigZag32(raw)
	}
	return arr
}

func deserializeCompressedLongArray18(r *BufferReader) []int64 {
	arrLen, err := r.ReadVarint32()
	if err != nil || arrLen > uint32(r.Remaining()) {
		return nil
	}
	arr := make([]int64, arrLen)
	for i := uint32(0); i < arrLen && !r.IsEmpty(); i++ {
		raw, err := r.ReadVarint64()
		if err != nil {
			break
		}
		arr[i] = DecodeZigZag64(raw)
	}
	return arr
}

func deserializeBooleanArray18(r *BufferReader) []bool {
	arrLen, err := r.ReadVarint32()
	if err != nil || arrLen > uint32(r.Remaining())*8 {
		return nil
	}
	arr := make([]bool, arrLen)
	fullBytes := arrLen / 8
	idx := uint32(0)
	for i := uint32(0); i < fullBytes; i++ {
		b, _ := r.ReadByte()
		for bit := 0; bit < 8 && idx < arrLen; bit++ {
			arr[idx] = b&(1<<bit) != 0
			idx++
		}
	}
	for idx < arrLen {
		b, _ := r.ReadByte()
		bit := uint32(0)
		for idx < arrLen && bit < 8 {
			arr[idx] = b&(1<<bit) != 0
			idx++
			bit++
		}
	}
	return arr
}

func deserializeCustomType18(r *BufferReader, slimCode byte) interface{} {
	var typeCode byte
	if slimCode == 0 {
		typeCode, _ = r.ReadByte()
	} else {
		typeCode = slimCode - P18CustomTypeSlim
	}
	length, err := r.ReadVarint32()
	if err != nil || length > uint32(r.Remaining()) {
		_ = r.Skip(r.Remaining())
		return nil
	}
	data, _ := r.ReadBytes(int(length))
	return map[string]interface{}{
		"customTypeCode": typeCode,
		"data":           data,
	}
}

func deserializeCustomTypeArray18(r *BufferReader) []interface{} {
	arrLen, err := r.ReadVarint32()
	if err != nil || arrLen > uint32(r.Remaining()) {
		return nil
	}
	typeCode, _ := r.ReadByte()
	arr := make([]interface{}, arrLen)
	for i := uint32(0); i < arrLen && !r.IsEmpty(); i++ {
		length, err := r.ReadVarint32()
		if err != nil || length > uint32(r.Remaining()) {
			_ = r.Skip(r.Remaining())
			break
		}
		data, _ := r.ReadBytes(int(length))
		arr[i] = map[string]interface{}{
			"customTypeCode": typeCode,
			"data":           data,
		}
	}
	return arr
}

func deserializeDictionaryArray18(r *BufferReader) []map[interface{}]interface{} {
	if r.Remaining() < 2 {
		return nil
	}
	keyType, _ := r.ReadByte()
	valueType, _ := r.ReadByte()
	arrLen, err := r.ReadVarint32()
	if err != nil || arrLen > uint32(r.Remaining()) {
		return nil
	}
	arr := make([]map[interface{}]interface{}, arrLen)
	for i := uint32(0); i < arrLen && !r.IsEmpty(); i++ {
		arr[i] = deserializeDictEntries18(r, keyType, valueType)
	}
	return arr
}

func deserializeHashtableArray18(r *BufferReader) []map[interface{}]interface{} {
	arrLen, err := r.ReadVarint32()
	if err != nil || arrLen > uint32(r.Remaining()) {
		return nil
	}
	arr := make([]map[interface{}]interface{}, arrLen)
	for i := uint32(0); i < arrLen && !r.IsEmpty(); i++ {
		arr[i] = deserializeHashtable18(r)
	}
	return arr
}
