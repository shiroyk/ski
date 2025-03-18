package buffer

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"github.com/shiroyk/ski/js/types"
)

// Buffer used to represent a fixed-length sequence of bytes.
// https://nodejs.org/api/buffer.html#buffer
type Buffer struct{}

func (b *Buffer) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	ctor := rt.ToValue(b.constructor).ToObject(rt)
	proto := b.prototype(rt)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.SetPrototype(proto)
	_ = ctor.Set("prototype", proto)

	// Static methods
	_ = ctor.Set("alloc", b.alloc)
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

	u8 := rt.Get("Uint8Array")
	if u8 == nil {
		panic(rt.NewTypeError("Uint8Array is not defined"))
	}
	_ = p.SetPrototype(u8.ToObject(rt).Prototype())
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.FunctionCall) sobek.Value { return rt.ToValue("Buffer") })

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

	_ = p.Set("readBigUInt64BE", b.readBigUInt64BE)
	_ = p.Set("readBigUint64BE", b.readBigUInt64BE)
	_ = p.Set("readBigUInt64LE", b.readBigUInt64LE)
	_ = p.Set("readBigUint64LE", b.readBigUInt64LE)

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

	_ = p.Set("readUInt8", b.readUInt8)
	_ = p.Set("readUint8", b.readUInt8)
	_ = p.Set("readUInt16BE", b.readUInt16BE)
	_ = p.Set("readUint16BE", b.readUInt16BE)
	_ = p.Set("readUInt16LE", b.readUInt16LE)
	_ = p.Set("readUint16LE", b.readUInt16LE)
	_ = p.Set("readUInt32BE", b.readUInt32BE)
	_ = p.Set("readUint32BE", b.readUInt32BE)
	_ = p.Set("readUInt32LE", b.readUInt32LE)
	_ = p.Set("readUint32LE", b.readUInt32LE)

	_ = p.Set("readUIntBE", b.readUIntBE)
	_ = p.Set("readUintBE", b.readUIntBE)
	_ = p.Set("readUIntLE", b.readUIntLE)
	_ = p.Set("readUintLE", b.readUIntLE)

	// Write methods
	_ = p.Set("writeBigInt64BE", b.writeBigInt64BE)
	_ = p.Set("writeBigInt64LE", b.writeBigInt64LE)

	_ = p.Set("writeBigUInt64BE", b.writeBigUInt64BE)
	_ = p.Set("writeBigUint64BE", b.writeBigUInt64BE)
	_ = p.Set("writeBigUInt64LE", b.writeBigUInt64LE)
	_ = p.Set("writeBigUint64LE", b.writeBigUInt64LE)

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

	_ = p.Set("writeUInt8", b.writeUInt8)
	_ = p.Set("writeUint8", b.writeUInt8)
	_ = p.Set("writeUInt16BE", b.writeUInt16BE)
	_ = p.Set("writeUint16BE", b.writeUInt16BE)
	_ = p.Set("writeUInt16LE", b.writeUInt16LE)
	_ = p.Set("writeUint16LE", b.writeUInt16LE)
	_ = p.Set("writeUInt32BE", b.writeUInt32BE)
	_ = p.Set("writeUint32BE", b.writeUInt32BE)
	_ = p.Set("writeUInt32LE", b.writeUInt32LE)
	_ = p.Set("writeUint32LE", b.writeUInt32LE)

	_ = p.Set("writeUIntBE", b.writeUIntBE)
	_ = p.Set("writeUintBE", b.writeUIntBE)
	_ = p.Set("writeUIntLE", b.writeUIntLE)
	_ = p.Set("writeUintLE", b.writeUIntLE)

	return p
}

func (b *Buffer) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	arg := call.Argument(0)
	if types.IsNumber(arg) {
		size := int(arg.ToInteger())
		if size < 0 {
			panic(rt.NewTypeError("Buffer size must be a non-negative integer"))
		}
		return newBuffer(rt, call.This, make([]byte, size))
	} else {
		return b.from(sobek.FunctionCall{This: call.This, Arguments: call.Arguments}, rt).(*sobek.Object)
	}
}

// from return a new Buffer from the given argument
func (*Buffer) from(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	arg := call.Argument(0)

	switch arg.ExportType() {
	case types.TypeBytes:
		return newBuffer(rt, call.This, arg.Export().([]byte))
	case types.TypeArrayBuffer:
		return newBuffer(rt, call.This, arg.Export().(sobek.ArrayBuffer).Bytes())
	case types.TypeString:
		buf := newBuffer(rt, call.This, decode(rt, arg.String(), call.Argument(1)))
		return buf
	default:
		obj, ok := arg.(*sobek.Object)
		if ok {
			if v := obj.Get("length"); v != nil {
				length := int(v.ToInteger())
				src := make([]byte, length)
				for i := 0; i < length; i++ {
					item := obj.Get(strconv.Itoa(i))
					if item != nil {
						src[i] = byte(item.ToInteger())
					}
				}
				return newBuffer(rt, call.This, src)
			}
		}
	}
	panic(rt.NewTypeError("First argument must be a string, Buffer, ArrayBuffer or Array or Array-like object."))
}

// alloc a new Buffer of size bytes. If fill is undefined, the Buffer will be zero-filled.
func (*Buffer) alloc(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	size := int(call.Argument(0).ToInteger())
	if size < 0 {
		panic(rt.NewTypeError("Buffer size must be a non-negative integer"))
	}

	buf := newBuffer(rt, call.This, make([]byte, size))

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

// isBuffer returns true if the value is a buffer
func (*Buffer) isBuffer(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(rt.InstanceOf(call.Argument(0), call.This.(*sobek.Object)))
}

// byteLength returns the byte length of a string when encoded using encoding.
func (*Buffer) byteLength(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	value := call.Argument(0)
	if data, ok := GetBuffer(rt, value); ok {
		return rt.ToValue(len(data))
	}
	return rt.ToValue(len(decode(rt, value.String(), call.Argument(1))))
}

// concat returns a new Buffer which is the result of concatenating all the Buffer instances in the list together.
func (*Buffer) concat(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	list := call.Argument(0)

	obj, ok := list.(*sobek.Object)
	if !ok {
		panic(rt.NewTypeError(`The "list" argument must be an instance of Array.`))
	}
	l := obj.Get("length")
	if l == nil {
		panic(rt.NewTypeError(`The "list" argument must be an instance of Array.`))
	}

	length := int(l.ToInteger())

	data := new(bytes.Buffer)
	for i := 0; i < length; i++ {
		item := obj.Get(strconv.Itoa(i))
		if item != nil {
			if IsBuffer(rt, item) {
				buf := item.Export().([]byte)
				data.Grow(len(buf))
				data.Write(buf)
				continue
			} else if types.IsUint8Array(rt, item) {
				buf := item.Export().([]byte)
				data.Write(buf)
				continue
			}
		}

		panic(rt.NewTypeError(`The "list[%s]" argument must be an instance of Buffer or Uint8Array`, i))
	}

	totalLength := data.Len()
	if v := call.Argument(1); !sobek.IsUndefined(v) {
		totalLength = int(uint64(v.ToInteger()))
	}

	return newBuffer(rt, call.This, data.Bytes()[:totalLength])
}

// toString decodes buf to a string according to the specified character encoding in encoding.
func (*Buffer) toString(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	return rt.ToValue(encode(rt, this, call.Argument(0)))
}

// toJSON returns a JSON representation of buffer.
func (*Buffer) toJSON(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	b, _ := json.Marshal(map[string]any{
		"type": "Buffer",
		"data": this,
	})
	return rt.ToValue(string(b))
}

// equals returns true if both buf and otherBuffer have exactly the same bytes
func (*Buffer) equals(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	buf1 := toBuffer(rt, call.This)
	if !IsBuffer(rt, call.Argument(0)) {
		return rt.ToValue(false)
	}
	buf2 := call.Argument(0).Export().([]byte)

	return rt.ToValue(bytes.Equal(buf1, buf2))
}

// compare compares buf with target and returns a number indicating whether buf comes before, after,
// or is the same as target in sort order
func (*Buffer) compare(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	var buf1, buf2 []byte
	b1 := call.Argument(0)
	if !IsBuffer(rt, b1) {
		panic(rt.NewTypeError("Argument must be a Buffer"))
	}
	if IsBuffer(rt, call.This) {
		buf1 = call.This.Export().([]byte)
		buf2 = b1.Export().([]byte)
	} else {
		buf1 = b1.Export().([]byte)
		b2 := call.Argument(1)
		if !IsBuffer(rt, b2) {
			panic(rt.NewTypeError("Argument must be a Buffer"))
		}
		buf2 = b2.Export().([]byte)
	}
	return rt.ToValue(bytes.Compare(buf1, buf2))
}

// copy copies data from a region of buf to a region in target, even if the target memory region overlaps with buf.
func (*Buffer) copy(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	var target buffer
	if v := call.Argument(0); IsBuffer(rt, v) {
		target = v.Export().([]byte)
	} else {
		panic(rt.NewTypeError("Argument must be a Buffer"))
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

	if sourceStart < 0 || sourceEnd > len(this) || targetStart < 0 {
		panic(rt.NewTypeError("Out of bounds index"))
	}

	data := make([]byte, sourceEnd-sourceStart)
	this.readAt(data, int64(sourceStart))
	target.writeAt(data, int64(targetStart))

	return rt.ToValue(len(data))
}

// write writes string to buf at offset according to the character encoding in encoding.
func (*Buffer) write(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	src := call.Argument(0).String()
	offset := call.Argument(1).ToInteger()
	length := int64(len(this)) - offset

	buf := decode(rt, src, call.Argument(3))

	if v := call.Argument(2); !sobek.IsUndefined(v) {
		length = v.ToInteger()
		if length < 0 || length > int64(len(buf)) {
			throwRangeError(rt, "length", 0, len(buf), length)
		}
	}

	this.writeAt(buf[:length], offset)

	return call.This
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
		if IsBuffer(rt, value) || types.IsTypedArray(rt, value) {
			buf = value.Export().([]byte)
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
	offset := this.offset(rt, call.Argument(0), 1)
	byteLength := getByteLength(rt, call.Argument(1))

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
	offset := this.offset(rt, call.Argument(0), 1)
	byteLength := getByteLength(rt, call.Argument(1))

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
	offset := this.offset(rt, call.Argument(0), 1)
	byteLength := getByteLength(rt, call.Argument(1))

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
	offset := this.offset(rt, call.Argument(0), 1)
	byteLength := getByteLength(rt, call.Argument(1))

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

// writeDoubleBE writes a float64 value in big-endian byte order
func (*Buffer) writeDoubleBE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := call.Argument(0).ToFloat()
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, math.Float64bits(value))
	result := this.writeOff(rt, call.Argument(1), 8, data)
	return rt.ToValue(result)
}

// writeDoubleLE writes a float64 value in little-endian byte order
func (*Buffer) writeDoubleLE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := call.Argument(0).ToFloat()
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, math.Float64bits(value))
	result := this.writeOff(rt, call.Argument(1), 8, data)
	return rt.ToValue(result)
}

// writeFloatBE writes a float32 value in big-endian byte order
func (*Buffer) writeFloatBE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := call.Argument(0).ToFloat()
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, math.Float32bits(float32(value)))
	result := this.writeOff(rt, call.Argument(1), 4, data)
	return rt.ToValue(result)
}

// writeFloatLE writes a float32 value in little-endian byte order
func (*Buffer) writeFloatLE(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := call.Argument(0).ToFloat()
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, math.Float32bits(float32(value)))
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
	offset := this.offset(rt, call.Argument(1), 1)
	byteLength := getByteLength(rt, call.Argument(2))

	checkInt(rt, value, byteLength)
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
	offset := this.offset(rt, call.Argument(1), 1)
	byteLength := getByteLength(rt, call.Argument(2))

	checkInt(rt, value, byteLength)
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
	value := uint64(call.Argument(0).ToInteger())
	offset := this.offset(rt, call.Argument(1), 1)
	byteLength := getByteLength(rt, call.Argument(2))

	checkUint(rt, value, byteLength)
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
	value := uint64(call.Argument(0).ToInteger())
	offset := this.offset(rt, call.Argument(1), 1)
	byteLength := getByteLength(rt, call.Argument(2))

	checkUint(rt, value, byteLength)
	data := make([]byte, byteLength)

	for i := range byteLength {
		data[i] = byte(value)
		value >>= 8
	}

	this.writeAt(data, offset)

	return rt.ToValue(offset + byteLength)
}

// newBuffer creates a new buffer with the given data
func newBuffer(rt *sobek.Runtime, this sobek.Value, data []byte) *sobek.Object {
	u8 := rt.Get("Uint8Array")
	if u8 == nil {
		panic(rt.NewTypeError("Uint8Array is undefined"))
	}
	ctor, ok := sobek.AssertConstructor(u8)
	if !ok {
		panic(rt.NewTypeError("Uint8Array is not a constructor"))
	}
	bufCtor := this.(*sobek.Object).Get("constructor").(*sobek.Object)
	object, err := ctor(bufCtor, rt.ToValue(rt.NewArrayBuffer(data)))
	if err != nil {
		panic(err)
	}
	_ = object.SetSymbol(symBuffer, symBuffer)
	return object
}

// toBuffer converts a value to a buffer
func toBuffer(rt *sobek.Runtime, value sobek.Value) buffer {
	if IsBuffer(rt, value) {
		return value.Export().([]byte)
	}
	panic(rt.NewTypeError(`Value of "this" must be of type Buffer`))
}

// getByteLength returns the byte length
func getByteLength(rt *sobek.Runtime, v sobek.Value) int64 {
	var length int64
	if types.IsNumber(v) {
		length = v.ToInteger()
	} else {
		panic(rt.NewTypeError(`The value of "byteLength" must be of type number`))
	}

	if length < 1 || length > 6 {
		throwRangeError(rt, "byteLength", 1, 6, length)
	}
	return length
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
		decoded, err := hex.DecodeString(src)
		if err != nil {
			panic(rt.NewTypeError("Invalid hex string"))
		}
		return decoded
	case "base64":
		decoded, err := base64.StdEncoding.DecodeString(src)
		if err != nil {
			panic(rt.NewTypeError("Invalid base64 string"))
		}
		return decoded
	default:
		panic(rt.NewTypeError(fmt.Sprintf("Unknown encoding: %s", encoding)))
	}
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
		panic(rt.NewTypeError(fmt.Sprintf("Unknown encoding: %s", encoding)))
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
	msg := fmt.Sprintf(`The value of "value" is out of range. It must be >= -(2n ** 63n) and < 2n ** 63. Received %sn`, v.String())
	panic(types.New(rt, "RangeError", rt.ToValue(msg)))
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
	msg := fmt.Sprintf(`The value of "value" is out of range. It must be >= 0n and < 2n ** 64n. Received %sn`, v.String())
	panic(types.New(rt, "RangeError", rt.ToValue(msg)))
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

// checkInt checks the value is a valid int between byteLength
func checkInt(rt *sobek.Runtime, v, byteLength int64) {
	var minInt, maxInt int64
	switch byteLength {
	case 1:
		minInt, maxInt = -0x80, 0x7f
	case 2:
		minInt, maxInt = -0x8000, 0x7fff
	case 3:
		minInt, maxInt = -0x800000, 0x7fffff
	case 4:
		minInt, maxInt = -0x80000000, 0x7fffffff
	case 5:
		minInt, maxInt = -0x8000000000, 0x7fffffffff
	case 6:
		minInt, maxInt = -0x800000000000, 0x7fffffffffff
	}
	if v <= maxInt && v >= minInt {
		return
	}
	throwRangeError(rt, "value", minInt, maxInt, v)
}

// checkUint checks the value is a valid uint between byteLength
func checkUint(rt *sobek.Runtime, v uint64, byteLength int64) {
	var maxUint uint64
	switch byteLength {
	case 1:
		maxUint = 0xff
	case 2:
		maxUint = 0xffff
	case 3:
		maxUint = 0xffffff
	case 4:
		maxUint = 0xffffffff
	case 5:
		maxUint = 0xffffffffff
	case 6:
		maxUint = 0xffffffffffff
	}
	if v <= maxUint && v >= 0 {
		return
	}
	throwRangeError(rt, "value", 0, maxUint, v)
}

var symBuffer = sobek.NewSymbol("Symbol.__buffer__")

type buffer []byte

// offset returns the offset
func (b buffer) offset(rt *sobek.Runtime, v sobek.Value, numBytes int64) int64 {
	var offset int64
	if types.IsNumber(v) {
		offset = v.ToInteger()
	} else {
		panic(rt.NewTypeError(`The value of "offset" must be of type number`))
	}

	if offset < 0 || offset+numBytes > int64(len(b)) {
		throwRangeError(rt, "offset", 0, int64(len(b)), offset)
	}
	return offset
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
	copy(b[offset:], data)
	return offset
}

// fill fills the buffer with data
func (b buffer) fill(rt *sobek.Runtime, offset, end int, buf []byte) {
	if len(buf) > len(b) {
		b.writeAt(buf[:len(b)], int64(offset))
	} else {
		if end < 0 || end > len(b) {
			throwRangeError(rt, "end", 0, len(b), end)
		}
		for i := offset; i < end; {
			i += copy(b[i:], buf)
		}
	}
}

// throwRangeError throws a RangeError
func throwRangeError(rt *sobek.Runtime, field string, min, max, received any) {
	msg := fmt.Sprintf(`The value of "%s" is out of range. It must be >= %d && <= %d. Received %d`, field, min, max, received)
	panic(types.New(rt, "RangeError", rt.ToValue(msg)))
}
