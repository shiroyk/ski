package buffer

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"math"
	"math/big"
	"strconv"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/types"
	"github.com/shiroyk/ski/modules/common"
)

// Buffer used to represent a fixed-length sequence of bytes.
// https://nodejs.org/api/buffer.html#buffer
type Buffer struct{}

func (b *Buffer) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	ctor := rt.ToValue(b.constructor).ToObject(rt)
	_ = ctor.DefineDataProperty("name", rt.ToValue("Buffer"), sobek.FLAG_FALSE, sobek.FLAG_TRUE, sobek.FLAG_FALSE)
	proto := b.prototype(rt)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_TRUE, sobek.FLAG_TRUE, sobek.FLAG_FALSE)
	_ = ctor.SetPrototype(rt.Get("Uint8Array").ToObject(rt))
	_ = ctor.Set("prototype", proto)

	// Static methods
	_ = ctor.Set("alloc", b.alloc)
	_ = ctor.Set("allocUnsafe", b.allocUnsafe)
	_ = ctor.Set("allocUnsafeSlow", b.allocUnsafeSlow)
	_ = ctor.Set("byteLength", b.byteLength)
	_ = ctor.Set("compare", b.compare)
	_ = ctor.Set("concat", b.concat)
	_ = ctor.Set("from", b.from)
	_ = ctor.Set("isBuffer", b.isBuffer)
	_ = ctor.Set("poolSize", 8192)

	return ctor, nil
}

func (b *Buffer) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.SetPrototype(types.Prototype(rt, "Uint8Array"))

	setAlias := func(name, alias string, method any) {
		value := rt.ToValue(method)
		_ = p.Set(name, value)
		_ = p.Set(alias, value)
	}

	_ = p.Set("toString", b.toString)
	_ = p.Set("toJSON", b.toJSON)
	_ = p.Set("equals", b.equals)
	_ = p.Set("compare", b.compare)
	_ = p.Set("copy", b.copy)
	_ = p.Set("write", b.write)
	_ = p.Set("fill", b.fill)
	_ = p.Set("keys", b.keys)
	_ = p.Set("entries", b.entries)

	// Read methods
	_ = p.Set("readBigInt64BE", b.readBigInt64BE)
	_ = p.Set("readBigInt64LE", b.readBigInt64LE)

	setAlias("readBigUInt64BE", "readBigUint64BE", b.readBigUInt64BE)
	setAlias("readBigUInt64LE", "readBigUint64LE", b.readBigUInt64LE)

	_ = p.Set("readDoubleBE", b.readDoubleBE)
	_ = p.Set("readDoubleLE", b.readDoubleLE)
	_ = p.Set("readFloatBE", b.readFloatBE)
	_ = p.Set("readFloatLE", b.readFloatLE)

	_ = p.Set("readInt8", b.readInt8)
	_ = p.Set("readInt16BE", b.readInt16BE)
	_ = p.Set("readInt16LE", b.readInt16LE)
	_ = p.Set("readInt32BE", b.readInt32BE)
	_ = p.Set("readInt32LE", b.readInt32LE)

	_ = p.Set("readIntBE", b.readIntBE)
	_ = p.Set("readIntLE", b.readIntLE)

	setAlias("readUInt8", "readUint8", b.readUInt8)
	setAlias("readUInt16BE", "readUint16BE", b.readUInt16BE)
	setAlias("readUInt16LE", "readUint16LE", b.readUInt16LE)
	setAlias("readUInt32BE", "readUint32BE", b.readUInt32BE)
	setAlias("readUInt32LE", "readUint32LE", b.readUInt32LE)

	setAlias("readUIntBE", "readUintBE", b.readUIntBE)
	setAlias("readUIntLE", "readUintLE", b.readUIntLE)

	// Write methods
	_ = p.Set("writeBigInt64BE", b.writeBigInt64BE)
	_ = p.Set("writeBigInt64LE", b.writeBigInt64LE)

	setAlias("writeBigUInt64BE", "writeBigUint64BE", b.writeBigUInt64BE)
	setAlias("writeBigUInt64LE", "writeBigUint64LE", b.writeBigUInt64LE)

	_ = p.Set("writeDoubleBE", b.writeDoubleBE)
	_ = p.Set("writeDoubleLE", b.writeDoubleLE)
	_ = p.Set("writeFloatBE", b.writeFloatBE)
	_ = p.Set("writeFloatLE", b.writeFloatLE)

	_ = p.Set("writeInt8", b.writeInt8)
	_ = p.Set("writeInt16BE", b.writeInt16BE)
	_ = p.Set("writeInt16LE", b.writeInt16LE)
	_ = p.Set("writeInt32BE", b.writeInt32BE)
	_ = p.Set("writeInt32LE", b.writeInt32LE)

	_ = p.Set("writeIntBE", b.writeIntBE)
	_ = p.Set("writeIntLE", b.writeIntLE)

	setAlias("writeUInt8", "writeUint8", b.writeUInt8)
	setAlias("writeUInt16BE", "writeUint16BE", b.writeUInt16BE)
	setAlias("writeUInt16LE", "writeUint16LE", b.writeUInt16LE)
	setAlias("writeUInt32BE", "writeUint32BE", b.writeUInt32BE)
	setAlias("writeUInt32LE", "writeUint32LE", b.writeUInt32LE)

	setAlias("writeUIntBE", "writeUintBE", b.writeUIntBE)
	setAlias("writeUIntLE", "writeUintLE", b.writeUIntLE)

	return p
}

func (b *Buffer) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	arg := call.Argument(0)
	if types.IsNumber(arg) {
		if !sobek.IsUndefined(call.Argument(1)) {
			panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
				`The "string" argument must be of type string. Received type %s`, types.ValueType(arg)))
		}
		return newBuffer(rt, allocBytes(rt, bufferSize(rt, arg)))
	} else {
		return b.from(sobek.FunctionCall{This: call.This, Arguments: call.Arguments}, rt).(*sobek.Object)
	}
}

// from return a new Buffer from the given argument
func (*Buffer) from(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	arg := call.Argument(0)

	switch arg.ExportType() {
	case types.TypeBytes:
		return newBuffer(rt, arg.Export().([]byte))
	case types.TypeArrayBuffer:
		return newBuffer(rt, arg.Export().(sobek.ArrayBuffer).Bytes())
	case types.TypeString:
		buf := newBuffer(rt, decode(rt, arg.String(), call.Argument(1)))
		return buf
	default:
		obj, ok := arg.(*sobek.Object)
		if ok {
			if v := obj.Get("length"); v != nil {
				length := v.ToInteger()
				src := allocBytes(rt, length)
				for i := range length {
					item := obj.Get(strconv.FormatInt(i, 10))
					if item != nil {
						src[i] = byte(item.ToInteger())
					}
				}
				return newBuffer(rt, src)
			}
		}
	}
	panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
		`The first argument must be of type string or an instance of Buffer,`+
			`ArrayBuffer, or Array or an Array-like Object. Received type %s`, types.ValueType(arg)))
}

// alloc a new Buffer of size bytes. If fill is undefined, the Buffer will be zero-filled.
func (*Buffer) alloc(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	buf := newBuffer(rt, allocBytes(rt, bufferSize(rt, call.Argument(0))))

	if fill := call.Argument(1); !sobek.IsUndefined(fill) {
		fn, ok := sobek.AssertFunction(buf.Get("fill"))
		if !ok {
			panic(rt.NewTypeError("Buffer.fill method not exist"))
		}
		_, err := fn(buf, fill, sobek.Undefined(), sobek.Undefined(), call.Argument(2))
		if err != nil {
			js.Throw(rt, err)
		}
	}

	return buf
}

// allocUnsafe allocates a new Buffer of size bytes, the ski implementation
// always returns a zero-filled buffer.
func (b *Buffer) allocUnsafe(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return b.alloc(call, rt)
}

// allocUnsafeSlow allocates a new Buffer of size bytes, the ski implementation
// always returns a zero-filled buffer.
func (b *Buffer) allocUnsafeSlow(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return b.alloc(call, rt)
}

// isBuffer returns true if the value is a buffer
func (*Buffer) isBuffer(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(IsBuffer(rt, call.Argument(0)))
}

// byteLength returns the byte length of a string when encoded using encoding.
func (*Buffer) byteLength(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	value := call.Argument(0)
	if data, ok := GetBuffer(rt, value); ok {
		return rt.ToValue(len(data))
	}
	if types.IsString(value) {
		return rt.ToValue(len(decode(rt, value.String(), call.Argument(1))))
	}
	panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
		`The "string" argument must be of type string or an instance of Buffer or ArrayBuffer.`+
			` Received type %s`, types.ValueType(value)))
}

// concat returns a new Buffer which is the result of concatenating all the Buffer instances in the list together.
func (*Buffer) concat(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	list := call.Argument(0)

	obj, ok := list.(*sobek.Object)
	// node requires an array, a Buffer is array-like but is not a valid list
	if !ok || obj.ClassName() != "Array" {
		panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
			`The "list" argument must be an instance of Array. Received type %s`, types.ValueType(list)))
	}
	l := obj.Get("length")

	length := int(l.ToInteger())
	if length == 0 {
		// node returns an empty buffer for an empty list
		return newBuffer(rt, nil)
	}

	data := new(bytes.Buffer)
	for i := range length {
		item := obj.Get(strconv.Itoa(i))
		if item != nil {
			if buf, ok := bytesOf(rt, item); ok {
				data.Grow(len(buf))
				data.Write(buf)
				continue
			} else if types.IsUint8Array(rt, item) {
				buf, ok := item.Export().([]byte)
				if !ok {
					panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
						`The "list[%d]" argument must be an instance of Buffer or Uint8Array. Received type %s`, i, types.ValueType(item)))
				}
				data.Write(buf)
				continue
			}
		}

		panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
			`The "list[%d]" argument must be an instance of Buffer or Uint8Array. Received type %s`, i, types.ValueType(item)))
	}

	totalLength := int64(data.Len())
	if v := call.Argument(1); !sobek.IsUndefined(v) {
		if !types.IsNumber(v) {
			panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
				`The "length" argument must be of type number. Received type %s`, types.ValueType(v)))
		}
		if number := v.ToFloat(); math.IsNaN(number) || number != math.Trunc(number) {
			panic(common.NewRangeErrorf(rt, common.ErrOutOfRange,
				`The value of "length" is out of range. It must be an integer. Received %s`, v.String()))
		}
		totalLength = v.ToInteger()
		if totalLength < 0 || totalLength > maxLength {
			throwRangeError(rt, "length", 0, maxLength, totalLength)
		}
	}

	result := allocBytes(rt, totalLength)
	copy(result, data.Bytes())

	return newBuffer(rt, result)
}

// toString decodes buf to a string according to the specified character encoding in encoding.
func (*Buffer) toString(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	return rt.ToValue(encode(rt, this, call.Argument(0)))
}

// toJSON returns a JSON representation of buffer.
func (*Buffer) toJSON(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	buf := toBuffer(rt, call.This)
	data := make([]any, len(buf))
	for i, b := range buf {
		data[i] = b
	}
	ret := rt.NewObject()
	_ = ret.Set("type", "Buffer")
	_ = ret.Set("data", rt.NewArray(data...))
	return ret
}

// equals returns true if both buf and otherBuffer have exactly the same bytes
func (*Buffer) equals(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	buf1 := toBuffer(rt, call.This)
	buf2, ok := viewBytes(rt, call.Argument(0))
	if !ok {
		panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
			`The "otherBuffer" argument must be an instance of Buffer or Uint8Array. Received type %s`,
			types.ValueType(call.Argument(0))))
	}

	return rt.ToValue(bytes.Equal(buf1, buf2))
}

// compare compares buf with target and returns a number indicating whether buf comes before, after,
// or is the same as target in sort order
func (*Buffer) compare(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	// node accepts a Uint8Array as the target of the prototype method
	if IsBuffer(rt, call.This) {
		target, ok := viewBytes(rt, call.Argument(0))
		if !ok {
			panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
				`The "target" argument must be an instance of Buffer or Uint8Array. Received type %s`,
				types.ValueType(call.Argument(0))))
		}
		return rt.ToValue(bytes.Compare(toBuffer(rt, call.This), target))
	}

	bytes1, ok := bytesOf(rt, call.Argument(0))
	if !ok {
		bytes1, ok = viewBytes(rt, call.Argument(0))
	}
	if !ok {
		panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
			`The "buf1" argument must be an instance of Buffer or Uint8Array. Received type %s`,
			types.ValueType(call.Argument(0))))
	}
	bytes2, ok := viewBytes(rt, call.Argument(1))
	if !ok {
		panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
			`The "buf2" argument must be an instance of Buffer or Uint8Array. Received type %s`,
			types.ValueType(call.Argument(1))))
	}
	return rt.ToValue(bytes.Compare(bytes1, bytes2))
}

// copy copies data from a region of buf to a region in target, even if the target memory region overlaps with buf.
func (*Buffer) copy(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	// node accepts a Uint8Array as the source and as the target
	this, ok := viewBytes(rt, call.This)
	if !ok {
		panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
			`Value of "this" must be of type Buffer or Uint8Array. Received type %s`, types.ValueType(call.This)))
	}
	target, ok := viewBytes(rt, call.Argument(0))
	if !ok {
		panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
			`The "target" argument must be an instance of Buffer or Uint8Array. Received type %s`,
			types.ValueType(call.Argument(0))))
	}

	targetStart := 0
	sourceStart := 0
	sourceEnd := len(this)

	if v := call.Argument(1); !sobek.IsUndefined(v) {
		targetStart = int(v.ToInteger())
	}
	if v := call.Argument(2); !sobek.IsUndefined(v) {
		sourceStart = int(v.ToInteger())
	}
	if v := call.Argument(3); !sobek.IsUndefined(v) {
		sourceEnd = int(v.ToInteger())
	}

	if sourceStart < 0 || sourceStart > len(this) {
		throwRangeError(rt, "sourceStart", 0, len(this), sourceStart)
	}
	// node clamps a sourceEnd that is out of the source bounds
	if sourceEnd < 0 {
		panic(common.NewRangeErrorf(rt, common.ErrOutOfRange,
			`The value of "sourceEnd" is out of range. It must be >= 0. Received %d`, sourceEnd))
	}
	if sourceEnd > len(this) {
		sourceEnd = len(this)
	}
	if sourceEnd < sourceStart {
		sourceEnd = sourceStart
	}
	if targetStart < 0 {
		panic(common.NewRangeErrorf(rt, common.ErrOutOfRange,
			`The value of "targetStart" is out of range. It must be >= 0. Received %d`, targetStart))
	}

	data := make([]byte, sourceEnd-sourceStart)
	buffer(this).readAt(data, int64(sourceStart))

	// node returns the number of bytes copied
	return rt.ToValue(copy(target[targetStart:], data))
}

// write writes string to buf at offset according to the character encoding in encoding.
func (*Buffer) write(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	offsetArg, lengthArg, encodingArg := call.Argument(1), call.Argument(2), call.Argument(3)
	// node treats a string in the offset or length position as the encoding
	switch {
	case types.IsString(offsetArg):
		encodingArg, offsetArg, lengthArg = offsetArg, rt.ToValue(0), sobek.Undefined()
	case types.IsString(lengthArg):
		encodingArg, lengthArg = lengthArg, sobek.Undefined()
	}

	argument := offsetArg
	if value := argument.ToFloat(); !sobek.IsUndefined(argument) &&
		(!types.IsNumber(argument) || math.IsNaN(value) || value != math.Trunc(value)) {
		if !types.IsNumber(argument) {
			panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
				`The "offset" argument must be of type number. Received type %s`, types.ValueType(argument)))
		}
		panic(common.NewRangeErrorf(rt, common.ErrOutOfRange,
			`The value of "offset" is out of range. It must be an integer. Received %s`, argument.String()))
	}
	offset := argument.ToInteger()
	if offset < 0 || offset > int64(len(this)) {
		throwRangeError(rt, "offset", 0, len(this), offset)
	}

	buf := decode(rt, call.Argument(0).String(), encodingArg)
	length := int64(len(this)) - offset

	if v := lengthArg; !sobek.IsUndefined(v) {
		if !types.IsNumber(v) {
			panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
				`The "length" argument must be of type number. Received type %s`, types.ValueType(v)))
		}
		if number := v.ToFloat(); math.IsNaN(number) || number != math.Trunc(number) {
			panic(common.NewRangeErrorf(rt, common.ErrOutOfRange,
				`The value of "length" is out of range. It must be an integer. Received %s`, v.String()))
		}
		length = v.ToInteger()
		if length < 0 || length > int64(len(buf)) {
			throwRangeError(rt, "length", 0, len(buf), length)
		}
	}

	if length > int64(len(buf)) {
		length = int64(len(buf))
	}
	written := copy(this[offset:], buf[:length])

	// node returns the number of bytes written
	return rt.ToValue(written)
}

// fill fills buf with the specified value.
func (*Buffer) fill(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := call.Argument(0)
	offset := int(call.Argument(1).ToInteger())
	end := len(this)

	if v := call.Argument(2); !sobek.IsUndefined(v) {
		end = int(v.ToInteger())
	}

	var buf []byte
	switch value.ExportType() {
	case types.TypeArrayBuffer:
		buf = value.Export().(sobek.ArrayBuffer).Bytes()
	case types.TypeString:
		buf = decode(rt, value.String(), call.Argument(3))
	case types.TypeInt, types.TypeFloat:
		buf = []byte{byte(value.ToInteger())}
	default:
		if data, ok := bytesOf(rt, value); ok {
			buf = data
		} else if types.IsTypedArray(rt, value) {
			buf, _ = value.Export().([]byte)
		}
	}

	this.fill(rt, offset, end, buf)

	return call.This
}

// keys returns an iterator for index
func (*Buffer) keys(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	return types.Iterator(rt, func(yield func(any) bool) {
		for i := range this {
			if !yield(i) {
				break
			}
		}
	})
}

// entries returns an iterator for [index, value] pairs
func (*Buffer) entries(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	return types.Iterator(rt, func(yield func(any) bool) {
		for i, b := range this {
			if !yield(rt.NewArray(i, int64(b))) {
				break
			}
		}
	})
}

// readBigInt64BE reads an int64 value in big-endian byte order
func (*Buffer) readBigInt64BE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	data := this.readOff(rt, call.Argument(0), 8)
	return rt.ToValue(big.NewInt(int64(binary.BigEndian.Uint64(data))))
}

// readBigInt64LE reads an int64 value in little-endian byte order
func (*Buffer) readBigInt64LE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	data := this.readOff(rt, call.Argument(0), 8)
	return rt.ToValue(big.NewInt(int64(binary.LittleEndian.Uint64(data))))
}

// readBigUInt64BE reads an uint64 value in big-endian byte order
func (*Buffer) readBigUInt64BE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	data := this.readOff(rt, call.Argument(0), 8)
	return rt.ToValue(new(big.Int).SetUint64(binary.BigEndian.Uint64(data)))
}

// readBigUInt64LE reads an uint64 value in little-endian byte order
func (*Buffer) readBigUInt64LE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	data := this.readOff(rt, call.Argument(0), 8)
	return rt.ToValue(new(big.Int).SetUint64(binary.LittleEndian.Uint64(data)))
}

// readDoubleBE reads an float64 value in big-endian byte order
func (*Buffer) readDoubleBE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	data := this.readOff(rt, call.Argument(0), 8)
	return rt.ToValue(math.Float64frombits(binary.BigEndian.Uint64(data)))
}

// readDoubleLE reads a float64 value in little-endian byte order
func (*Buffer) readDoubleLE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	data := this.readOff(rt, call.Argument(0), 8)
	return rt.ToValue(math.Float64frombits(binary.LittleEndian.Uint64(data)))
}

// readFloatBE reads a float32 value in big-endian byte order
func (*Buffer) readFloatBE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	data := this.readOff(rt, call.Argument(0), 4)
	return rt.ToValue(math.Float32frombits(binary.BigEndian.Uint32(data)))
}

// readFloatLE reads a float32 value in little-endian byte order
func (*Buffer) readFloatLE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	data := this.readOff(rt, call.Argument(0), 4)
	return rt.ToValue(math.Float32frombits(binary.LittleEndian.Uint32(data)))
}

// readInt16BE reads an int16 value in big-endian byte order
func (*Buffer) readInt16BE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	data := this.readOff(rt, call.Argument(0), 2)
	return rt.ToValue(int16(binary.BigEndian.Uint16(data)))
}

// readInt16LE reads an int16 value in little-endian byte order
func (*Buffer) readInt16LE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	data := this.readOff(rt, call.Argument(0), 2)
	return rt.ToValue(int16(binary.LittleEndian.Uint16(data)))
}

// readInt32BE reads an int32 value in big-endian byte order
func (*Buffer) readInt32BE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	data := this.readOff(rt, call.Argument(0), 4)
	return rt.ToValue(int32(binary.BigEndian.Uint32(data)))
}

// readInt32LE reads an int32 value in little-endian byte order
func (*Buffer) readInt32LE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	data := this.readOff(rt, call.Argument(0), 4)
	return rt.ToValue(int32(binary.LittleEndian.Uint32(data)))
}

// readInt8 reads an int8 value
func (*Buffer) readInt8(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	data := this.readOff(rt, call.Argument(0), 1)
	return rt.ToValue(int8(data[0]))
}

// readIntBE reads an int64 value in big-endian byte order
func (*Buffer) readIntBE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	byteLength := getByteLength(rt, call.Argument(1))
	offset := this.offsetStrict(rt, call.Argument(0), byteLength)

	data := make([]byte, byteLength)
	this.readAt(data, offset)

	var value int64
	for _, b := range data {
		value = (value << 8) | int64(b)
	}
	value = (value << (64 - 8*byteLength)) >> (64 - 8*byteLength)

	return rt.ToValue(value)
}

// readIntLE reads an int64 value in little-endian byte order
func (*Buffer) readIntLE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	byteLength := getByteLength(rt, call.Argument(1))
	offset := this.offsetStrict(rt, call.Argument(0), byteLength)

	data := make([]byte, byteLength)
	this.readAt(data, offset)

	var value int64
	for i := len(data) - 1; i >= 0; i-- {
		value = (value << 8) | int64(data[i])
	}
	value = (value << (64 - 8*byteLength)) >> (64 - 8*byteLength)

	return rt.ToValue(value)
}

// readUInt16BE reads an uint16 value in big-endian byte order
func (*Buffer) readUInt16BE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	data := this.readOff(rt, call.Argument(0), 2)
	return rt.ToValue(binary.BigEndian.Uint16(data))
}

// readUInt16LE reads an uint16 value in little-endian byte order
func (*Buffer) readUInt16LE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	data := this.readOff(rt, call.Argument(0), 2)
	return rt.ToValue(binary.LittleEndian.Uint16(data))
}

// readUInt32BE reads an uint32 value in big-endian byte order
func (*Buffer) readUInt32BE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	data := this.readOff(rt, call.Argument(0), 4)
	return rt.ToValue(binary.BigEndian.Uint32(data))
}

// readUInt32LE reads an uint32 value in little-endian byte order
func (*Buffer) readUInt32LE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	data := this.readOff(rt, call.Argument(0), 4)
	return rt.ToValue(binary.LittleEndian.Uint32(data))
}

// readUInt8 reads an uint8 value
func (*Buffer) readUInt8(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	data := this.readOff(rt, call.Argument(0), 1)
	return rt.ToValue(data[0])
}

// readUIntBE reads an uint64 value in big-endian byte order
func (*Buffer) readUIntBE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	byteLength := getByteLength(rt, call.Argument(1))
	offset := this.offsetStrict(rt, call.Argument(0), byteLength)

	data := make([]byte, byteLength)
	this.readAt(data, offset)

	var value uint64
	for _, b := range data {
		value = (value << 8) | uint64(b)
	}

	return rt.ToValue(value)
}

// readUIntLE reads an uint64 value in little-endian byte order
func (*Buffer) readUIntLE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	byteLength := getByteLength(rt, call.Argument(1))
	offset := this.offsetStrict(rt, call.Argument(0), byteLength)

	data := make([]byte, byteLength)
	this.readAt(data, offset)

	var value uint64
	for i := len(data) - 1; i >= 0; i-- {
		value = (value << 8) | uint64(data[i])
	}

	return rt.ToValue(value)
}

// writeBigInt64BE writes an int64 value in big-endian byte order
func (*Buffer) writeBigInt64BE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := toBigInt(rt, call.Argument(0))
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, value)
	result := this.writeOff(rt, call.Argument(1), 8, data)
	return rt.ToValue(result)
}

// writeBigInt64LE writes an int64 value in little-endian byte order
func (*Buffer) writeBigInt64LE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := toBigInt(rt, call.Argument(0))
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, value)
	result := this.writeOff(rt, call.Argument(1), 8, data)
	return rt.ToValue(result)
}

// writeBigUInt64BE writes an uint64 value in big-endian byte order
func (*Buffer) writeBigUInt64BE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := toBigUint(rt, call.Argument(0))
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, value)
	result := this.writeOff(rt, call.Argument(1), 8, data)
	return rt.ToValue(result)
}

// writeBigUInt64LE writes an uint64 value in little-endian byte order
func (*Buffer) writeBigUInt64LE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := toBigUint(rt, call.Argument(0))
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, value)
	result := this.writeOff(rt, call.Argument(1), 8, data)
	return rt.ToValue(result)
}

// float64Bits returns the IEEE 754 bits of the value, node writes the platform
// independent quiet NaN while go uses a payload bit.
func float64Bits(value float64) uint64 {
	if math.IsNaN(value) {
		return 0x7ff8000000000000
	}
	return math.Float64bits(value)
}

// float32Bits returns the IEEE 754 bits of the value.
func float32Bits(value float32) uint32 {
	if math.IsNaN(float64(value)) {
		return 0x7fc00000
	}
	return math.Float32bits(value)
}

// writeDoubleBE writes a float64 value in big-endian byte order
func (*Buffer) writeDoubleBE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := call.Argument(0).ToFloat()
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, float64Bits(value))
	result := this.writeOff(rt, call.Argument(1), 8, data)
	return rt.ToValue(result)
}

// writeDoubleLE writes a float64 value in little-endian byte order
func (*Buffer) writeDoubleLE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := call.Argument(0).ToFloat()
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, float64Bits(value))
	result := this.writeOff(rt, call.Argument(1), 8, data)
	return rt.ToValue(result)
}

// writeFloatBE writes a float32 value in big-endian byte order
func (*Buffer) writeFloatBE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := call.Argument(0).ToFloat()
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, float32Bits(float32(value)))
	result := this.writeOff(rt, call.Argument(1), 4, data)
	return rt.ToValue(result)
}

// writeFloatLE writes a float32 value in little-endian byte order
func (*Buffer) writeFloatLE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := call.Argument(0).ToFloat()
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, float32Bits(float32(value)))
	result := this.writeOff(rt, call.Argument(1), 4, data)
	return rt.ToValue(result)
}

// writeInt16BE writes an int16 value in big-endian byte order
func (*Buffer) writeInt16BE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := toInteger[int16](rt, call.Argument(0))
	data := make([]byte, 2)
	binary.BigEndian.PutUint16(data, uint16(value))
	result := this.writeOff(rt, call.Argument(1), 2, data)
	return rt.ToValue(result)
}

// writeInt16LE writes an int16 value in little-endian byte order
func (*Buffer) writeInt16LE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := toInteger[int16](rt, call.Argument(0))
	data := make([]byte, 2)
	binary.LittleEndian.PutUint16(data, uint16(value))
	result := this.writeOff(rt, call.Argument(1), 2, data)
	return rt.ToValue(result)
}

// writeInt32BE writes an int32 value in big-endian byte order
func (*Buffer) writeInt32BE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := toInteger[int32](rt, call.Argument(0))
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, uint32(value))
	result := this.writeOff(rt, call.Argument(1), 4, data)
	return rt.ToValue(result)
}

// writeInt32LE writes an int32 value in little-endian byte order
func (*Buffer) writeInt32LE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := toInteger[int32](rt, call.Argument(0))
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(value))
	result := this.writeOff(rt, call.Argument(1), 4, data)
	return rt.ToValue(result)
}

// writeInt8 writes an int8 value
func (*Buffer) writeInt8(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := toInteger[int8](rt, call.Argument(0))
	result := this.writeOff(rt, call.Argument(1), 0, []byte{byte(value)})
	return rt.ToValue(result)
}

// writeIntBE writes an integer value in big-endian byte order
func (*Buffer) writeIntBE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := call.Argument(0).ToInteger()
	byteLength := getByteLength(rt, call.Argument(2))
	offset := this.offsetStrict(rt, call.Argument(1), byteLength)

	checkIntBytes(rt, value, byteLength)
	data := make([]byte, byteLength)

	if value < 0 {
		for i := range data {
			data[i] = 0xFF
		}
	}

	for i := byteLength - 1; i >= 0; i-- {
		data[i] = byte(value)
		value >>= 8
	}

	this.writeAt(data, offset)

	return rt.ToValue(offset + byteLength)
}

// writeIntLE writes an integer value in little-endian byte order
func (*Buffer) writeIntLE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := call.Argument(0).ToInteger()
	byteLength := getByteLength(rt, call.Argument(2))
	offset := this.offsetStrict(rt, call.Argument(1), byteLength)

	checkIntBytes(rt, value, byteLength)
	data := make([]byte, byteLength)

	if value < 0 {
		for i := range data {
			data[i] = 0xFF
		}
	}

	for i := range byteLength {
		data[i] = byte(value)
		value >>= 8
	}

	this.writeAt(data, offset)

	return rt.ToValue(offset + byteLength)
}

// writeUInt16BE writes an uint16 value in big-endian byte order
func (*Buffer) writeUInt16BE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := toInteger[uint16](rt, call.Argument(0))
	data := make([]byte, 2)
	binary.BigEndian.PutUint16(data, value)
	result := this.writeOff(rt, call.Argument(1), 2, data)
	return rt.ToValue(result)
}

// writeUInt16LE writes an uint16 value in little-endian byte order
func (*Buffer) writeUInt16LE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := toInteger[uint16](rt, call.Argument(0))
	data := make([]byte, 2)
	binary.LittleEndian.PutUint16(data, value)
	result := this.writeOff(rt, call.Argument(1), 2, data)
	return rt.ToValue(result)
}

// writeUInt32BE writes an uint32 value in big-endian byte order
func (*Buffer) writeUInt32BE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := toInteger[uint32](rt, call.Argument(0))
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, value)
	result := this.writeOff(rt, call.Argument(1), 4, data)
	return rt.ToValue(result)
}

// writeUInt32LE writes an uint32 value in little-endian byte order
func (*Buffer) writeUInt32LE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := toInteger[uint32](rt, call.Argument(0))
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, value)
	result := this.writeOff(rt, call.Argument(1), 4, data)
	return rt.ToValue(result)
}

// writeUInt8 writes an uint8 value
func (*Buffer) writeUInt8(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := toInteger[uint8](rt, call.Argument(0))
	result := this.writeOff(rt, call.Argument(1), 0, []byte{value})
	return rt.ToValue(result)
}

// writeUIntBE writes an unsigned integer in big-endian byte order
func (*Buffer) writeUIntBE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	number := call.Argument(0).ToInteger()
	byteLength := getByteLength(rt, call.Argument(2))
	offset := this.offsetStrict(rt, call.Argument(1), byteLength)

	checkUintBytes(rt, number, byteLength)
	value := uint64(number)
	data := make([]byte, byteLength)

	for i := byteLength - 1; i >= 0; i-- {
		data[i] = byte(value)
		value >>= 8
	}

	this.writeAt(data, offset)

	return rt.ToValue(offset + byteLength)
}

// writeUIntLE writes an unsigned integer in little-endian byte order
func (*Buffer) writeUIntLE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	number := call.Argument(0).ToInteger()
	byteLength := getByteLength(rt, call.Argument(2))
	offset := this.offsetStrict(rt, call.Argument(1), byteLength)

	checkUintBytes(rt, number, byteLength)
	value := uint64(number)
	data := make([]byte, byteLength)

	for i := range byteLength {
		data[i] = byte(value)
		value >>= 8
	}

	this.writeAt(data, offset)

	return rt.ToValue(offset + byteLength)
}

// maxLength is buffer.kMaxLength, the maximum length of a Buffer.
const maxLength = int64(1<<53 - 1)

// bufferSize returns the size argument of the allocation methods, it follows
// the node validation: the value must be a non-negative safe integer.
func bufferSize(rt *sobek.Runtime, value sobek.Value) int64 {
	if !types.IsNumber(value) {
		panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
			`The "size" argument must be of type number. Received type %s`, types.ValueType(value)))
	}
	size := value.ToFloat()
	if math.IsNaN(size) || size != math.Trunc(size) || size < 0 || size > float64(maxLength) {
		throwRangeError(rt, "size", 0, maxLength, size)
	}
	return int64(size)
}

// allocBytes allocates the storage of a buffer, it throws a RangeError when the
// size is out of range or the allocation fails.
func allocBytes(rt *sobek.Runtime, size int64) (data []byte) {
	if size < 0 {
		throwRangeError(rt, "size", 0, maxLength, size)
	}
	if size > maxLength {
		throwRangeError(rt, "size", 0, maxLength, size)
	}
	return make([]byte, size)
}

// bytesOf returns the underlying bytes of a Buffer, false when the value only
// inherits from Buffer.prototype without the buffer storage.
func bytesOf(rt *sobek.Runtime, value sobek.Value) ([]byte, bool) {
	if !IsBuffer(rt, value) {
		return nil, false
	}
	data, ok := value.Export().([]byte)
	return data, ok
}

// viewBytes returns the underlying bytes of a Buffer or a Uint8Array, node
// accepts a Uint8Array where a Buffer argument is expected.
func viewBytes(rt *sobek.Runtime, value sobek.Value) ([]byte, bool) {
	if data, ok := bytesOf(rt, value); ok {
		return data, true
	}
	// node accepts any ArrayBufferView, e.g. a Uint16Array target of copy()
	return GetBuffer(rt, value)
}

// newBuffer creates a new buffer with the given data
func newBuffer(rt *sobek.Runtime, data []byte) *sobek.Object {
	buf := types.New(rt, "Uint8Array", rt.ToValue(rt.NewArrayBuffer(data)))
	_ = buf.SetPrototype(types.Prototype(rt, "Buffer"))
	return buf
}

// toBuffer converts a value to a buffer
func toBuffer(rt *sobek.Runtime, value sobek.Value) buffer {
	if data, ok := bytesOf(rt, value); ok {
		return data
	}
	panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
		`Value of "this" must be of type Buffer. Received type %s`, types.ValueType(value)))
}

// getByteLength returns the byte length
func getByteLength(rt *sobek.Runtime, v sobek.Value) int64 {
	if !types.IsNumber(v) {
		panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
			`The "byteLength" argument must be of type number. Received type %s`, types.ValueType(v)))
	}
	length := v.ToFloat()
	if math.IsNaN(length) || length != math.Trunc(length) {
		panic(common.NewRangeErrorf(rt, common.ErrOutOfRange,
			`The value of "byteLength" is out of range. It must be an integer. Received %s`, v.String()))
	}
	if length < 1 || length > 6 {
		throwRangeError(rt, "byteLength", 1, 6, v.String())
	}
	return int64(length)
}

// decode decodes a string into a byte array
func decode(rt *sobek.Runtime, src string, enc sobek.Value) []byte {
	encoding := "utf8"
	if !sobek.IsUndefined(enc) {
		encoding = enc.String()
	}
	switch encoding {
	case "utf8", "utf-8":
		return []byte(src)
	case "hex":
		return decodeHex(src)
	case "base64":
		decoded, err := base64.StdEncoding.DecodeString(src)
		if err != nil {
			panic(rt.NewTypeError("Invalid base64 string"))
		}
		return decoded
	default:
		panic(common.NewTypeErrorf(rt, common.ErrUnknownEncoding, "Unknown encoding: %s", encoding))
	}
}

// decodeHex decodes the hex string like node does, the decoding stops at the
// first pair that is not a valid hexadecimal, an odd trailing nibble is dropped.
func decodeHex(src string) []byte {
	if len(src)%2 == 1 {
		src = src[:len(src)-1]
	}
	decoded := make([]byte, 0, len(src)/2)
	for i := 0; i+1 < len(src); i += 2 {
		high, ok := hexValue(src[i])
		if !ok {
			break
		}
		low, ok := hexValue(src[i+1])
		if !ok {
			break
		}
		decoded = append(decoded, high<<4|low)
	}
	return decoded
}

// hexValue returns the value of a hexadecimal digit.
func hexValue(char byte) (byte, bool) {
	switch {
	case char >= '0' && char <= '9':
		return char - '0', true
	case char >= 'a' && char <= 'f':
		return char - 'a' + 10, true
	case char >= 'A' && char <= 'F':
		return char - 'A' + 10, true
	}
	return 0, false
}

// encode encodes a byte array into a string
func encode(rt *sobek.Runtime, buf []byte, enc sobek.Value) string {
	encoding := "utf8"
	if !sobek.IsUndefined(enc) {
		encoding = enc.String()
	}
	switch encoding {
	case "utf8", "utf-8":
		return string(buf)
	case "hex":
		return hex.EncodeToString(buf)
	case "base64":
		return base64.StdEncoding.EncodeToString(buf)
	default:
		panic(common.NewTypeErrorf(rt, common.ErrUnknownEncoding, "Unknown encoding: %s", encoding))
	}
}

var (
	maxBigint  = big.NewInt(math.MaxInt64)
	minBigint  = big.NewInt(math.MinInt64)
	maxBigUint = new(big.Int).SetUint64(math.MaxUint64)
)

// toBigInt converts a BigInt to an int
func toBigInt(rt *sobek.Runtime, value sobek.Value) uint64 {
	v, ok := value.Export().(*big.Int)
	if !ok {
		panic(rt.NewTypeError("Cannot mix BigInt and other types, use explicit conversions"))
	}
	if v.Cmp(maxBigint) <= 0 && v.Cmp(minBigint) >= 0 {
		return uint64(v.Int64())
	}
	panic(common.NewRangeErrorf(rt, common.ErrOutOfRange,
		`The value of "value" is out of range. It must be >= -(2n ** 63n) and < 2n ** 63. Received %sn`, v))
}

// toBigUint converts a BigInt to an uint
func toBigUint(rt *sobek.Runtime, value sobek.Value) uint64 {
	v, ok := value.Export().(*big.Int)
	if !ok {
		panic(rt.NewTypeError("Cannot mix BigInt and other types, use explicit conversions"))
	}
	if v.Cmp(maxBigUint) <= 0 && v.Cmp(big.NewInt(0)) >= 0 {
		return v.Uint64()
	}
	panic(common.NewRangeErrorf(rt, common.ErrOutOfRange,
		`The value of "value" is out of range. It must be >= 0n and < 2n ** 64n. Received %sn`, v))
}

type integer interface {
	int8 | int16 | int32 | int64 |
		uint8 | uint16 | uint32 | uint64
}

// toInteger converts a value to an integer
func toInteger[N integer](rt *sobek.Runtime, value sobek.Value) (ret N) {
	v := value.ToInteger()
	var minInt, maxInt int64
	switch any(ret).(type) {
	case int8:
		minInt, maxInt = math.MinInt8, math.MaxInt8
	case int16:
		minInt, maxInt = math.MinInt16, math.MaxInt16
	case int32:
		minInt, maxInt = math.MinInt32, math.MaxInt32
	case uint8:
		minInt, maxInt = 0, math.MaxUint8
	case uint16:
		minInt, maxInt = 0, math.MaxUint16
	case uint32:
		minInt, maxInt = 0, math.MaxUint32
	case int64, uint64:
		return N(v)
	default:
		panic("unreached")
	}
	if v > maxInt || v < minInt {
		throwRangeError(rt, "value", minInt, maxInt, v)
	}
	return N(v)
}

// checkIntBytes checks the value of the write methods that take a byteLength,
// node prints the range as a power of two when the byteLength is larger than 4.
func checkIntBytes(rt *sobek.Runtime, value, byteLength int64) {
	bits := byteLength*8 - 1
	min, max := -(int64(1) << bits), int64(1)<<bits-1
	if value >= min && value <= max {
		return
	}
	if bits > 32 {
		panic(common.NewRangeErrorf(rt, common.ErrOutOfRange,
			`The value of "value" is out of range. It must be >= -(2 ** %d) and < 2 ** %d. Received %d`,
			bits, bits, value))
	}
	throwRangeError(rt, "value", min, max, value)
}

// checkUintBytes checks the value of the unsigned write methods that take a
// byteLength, node prints the range as a power of two when the byteLength is
// larger than 4.
func checkUintBytes(rt *sobek.Runtime, value, byteLength int64) {
	bits := byteLength * 8
	max := int64(1)<<bits - 1
	if value >= 0 && value <= max {
		return
	}
	if bits > 32 {
		panic(common.NewRangeErrorf(rt, common.ErrOutOfRange,
			`The value of "value" is out of range. It must be >= 0 and < 2 ** %d. Received %d`,
			bits, value))
	}
	throwRangeError(rt, "value", 0, max, value)
}

type buffer []byte

// offset returns the offset
func (b buffer) offset(rt *sobek.Runtime, v sobek.Value, numBytes int64) int64 {
	if sobek.IsUndefined(v) {
		v = rt.ToValue(0)
	}
	if !types.IsNumber(v) {
		panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
			`The "offset" argument must be of type number. Received type %s`, types.ValueType(v)))
	}
	value := v.ToFloat()
	offset := int64(value)
	switch {
	case math.IsNaN(value) || value != math.Trunc(value):
		panic(common.NewRangeErrorf(rt, common.ErrOutOfRange,
			`The value of "offset" is out of range. It must be an integer. Received %s`, v.String()))
	case math.IsInf(value, 0):
		throwBoundsError(rt, "offset", v.String(), int64(len(b))-numBytes)
	}
	if offset < 0 || offset > int64(len(b))-numBytes {
		throwBoundsError(rt, "offset", offset, int64(len(b))-numBytes)
	}
	return offset
}

// offsetStrict returns the offset of the methods that require it, the byteLength
// methods of node do not default an undefined offset to zero.
func (b buffer) offsetStrict(rt *sobek.Runtime, v sobek.Value, numBytes int64) int64 {
	if sobek.IsUndefined(v) {
		panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
			`The "offset" argument must be of type number. Received undefined`))
	}
	return b.offset(rt, v, numBytes)
}

// readOff reads data from the buffer with offset, numBytes
func (b buffer) readOff(rt *sobek.Runtime, offset sobek.Value, numBytes int64) []byte {
	data := make([]byte, numBytes)
	b.readAt(data, b.offset(rt, offset, numBytes))
	return data
}

// readAt reads data from the buffer
func (b buffer) readAt(p []byte, offset int64) int64 {
	return int64(copy(p, b[offset:]))
}

// writeOff writes data to the buffer with offset, numBytes
func (b buffer) writeOff(rt *sobek.Runtime, off sobek.Value, numBytes int64, data []byte) int64 {
	return b.writeAt(data, b.offset(rt, off, numBytes)) + numBytes
}

// writeAt writes data to the buffer
func (b buffer) writeAt(data []byte, offset int64) int64 {
	if offset < 0 || offset > int64(len(b)) {
		return 0
	}
	copy(b[offset:], data)
	return offset
}

// fill fills the buffer with data
func (b buffer) fill(rt *sobek.Runtime, offset, end int, buf []byte) {
	if offset < 0 || offset > len(b) {
		throwRangeError(rt, "offset", 0, len(b), offset)
	}
	if end < 0 || end > len(b) {
		throwRangeError(rt, "end", 0, len(b), end)
	}
	if len(buf) > len(b) {
		b.writeAt(buf[:len(b)], int64(offset))
	} else {
		for i := offset; i < end; {
			i += copy(b[i:], buf)
		}
	}
}

// throwRangeError throws a RangeError
func throwRangeError(rt *sobek.Runtime, field string, min, max, received any) {
	panic(common.NewRangeErrorf(rt, common.ErrOutOfRange,
		`The value of "%s" is out of range. It must be >= %v && <= %v. Received %v`, field, min, max, received))
}

// throwBoundsError throws the node bounds error of an offset, the error code is
// ERR_BUFFER_OUT_OF_BOUNDS when the requested size does not fit at all.
func throwBoundsError(rt *sobek.Runtime, field string, offset any, max int64) {
	if max < 0 {
		panic(common.NewRangeErrorf(rt, common.ErrBufferOutOfBounds,
			"Attempt to access memory outside buffer bounds"))
	}
	panic(common.NewRangeErrorf(rt, common.ErrOutOfRange,
		`The value of "%s" is out of range. It must be >= 0 and <= %d. Received %v`, field, max, offset))
}
