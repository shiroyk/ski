package buffer

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf16"

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
	_ = ctor.Set("copyBytesFrom", b.copyBytesFrom)
	_ = ctor.Set("from", b.from)
	_ = ctor.Set("isBuffer", b.isBuffer)
	_ = ctor.Set("isEncoding", b.isEncoding)
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
	_ = p.Set("includes", b.includes)
	_ = p.Set("indexOf", b.indexOf)
	_ = p.Set("lastIndexOf", b.lastIndexOf)
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
func (b *Buffer) from(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	arg := call.Argument(0)
	// a symbol exports as an empty byte slice, node does not convert it
	if types.ValueType(arg) == "symbol" {
		panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
			`The first argument must be of type string or an instance of Buffer,`+
				`ArrayBuffer, or Array or an Array-like Object. Received type symbol`))
	}

	switch arg.ExportType() {
	case types.TypeBytes:
		return newBuffer(rt, arg.Export().([]byte))
	case types.TypeArrayBuffer:
		return newBuffer(rt, arg.Export().(sobek.ArrayBuffer).Bytes())
	case types.TypeString:
		// node allows an invalid encoding when the value is a string
		encoding := call.Argument(1)
		if !types.IsString(encoding) {
			encoding = sobek.Undefined()
		}
		buf := newBuffer(rt, decode(rt, arg.String(), encoding))
		return buf
	default:
		// node only converts objects, a function is not a valid value even
		// though it has a length property
		obj, ok := arg.(*sobek.Object)
		if !ok || types.ValueType(arg) != "object" {
			break
		}
		// node converts the value with valueOf() first, e.g. a String object
		if valueOf, ok := sobek.AssertFunction(obj.Get("valueOf")); ok {
			value, err := valueOf(obj)
			if err != nil {
				js.Throw(rt, err)
			}
			if !types.IsNil(value) && value != arg {
				switch value.ExportType() {
				case types.TypeString:
					return newBuffer(rt, decode(rt, value.String(), call.Argument(1)))
				case types.TypeBytes:
					if data, ok := value.Export().([]byte); ok {
						return newBuffer(rt, bytes.Clone(data))
					}
				case types.TypeArrayBuffer:
					return newBuffer(rt, value.Export().(sobek.ArrayBuffer).Bytes())
				}
			}
		}
		// node converts the json representation of a buffer
		if kind := obj.Get("type"); !types.IsNil(kind) && kind.String() == "Buffer" {
			if data, ok := obj.Get("data").(*sobek.Object); ok && data.ClassName() == "Array" {
				return b.from(sobek.FunctionCall{This: call.This, Arguments: []sobek.Value{data}}, rt)
			}
		}
		if v := obj.Get("length"); !types.IsNil(v) {
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
		// node converts the value with the "string" hint of Symbol.toPrimitive
		if primitive := toPrimitive(rt, obj, "string"); primitive != nil {
			return b.from(sobek.FunctionCall{This: call.This, Arguments: []sobek.Value{primitive}}, rt)
		}
	}
	panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
		`The first argument must be of type string or an instance of Buffer,`+
			`ArrayBuffer, or Array or an Array-like Object. Received type %s`, types.ValueType(arg)))
}

// toPrimitive returns the primitive value of the object using the hint, it is
// the ToPrimitive abstract operation, nil means that the object cannot be
// converted to a primitive.
func toPrimitive(rt *sobek.Runtime, object *sobek.Object, hint string) sobek.Value {
	toPrimitive, ok := sobek.AssertFunction(object.GetSymbol(sobek.SymToPrimitive))
	if !ok {
		return nil
	}
	value, err := toPrimitive(object, rt.ToValue(hint))
	if err != nil {
		js.Throw(rt, err)
	}
	if _, ok := value.(*sobek.Object); ok {
		return nil
	}
	return value
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

// copyBytesFrom copies the bytes of a typed array view into a new Buffer, the
// offset and the length are counted in elements of the view.
func (*Buffer) copyBytesFrom(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	view := call.Argument(0)
	if view == nil || !types.IsTypedArray(rt, view) {
		panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
			`The "view" argument must be an instance of TypedArray. Received type %s`, types.ValueType(view)))
	}
	object := view.ToObject(rt)
	element := max(object.Get("BYTES_PER_ELEMENT").ToInteger(), 1)
	length := object.Get("length").ToInteger()

	offset := numberAtLeast(rt, "offset", call.Argument(1), 0, 0)
	count := max(length-offset, 0)
	if v := call.Argument(2); !sobek.IsUndefined(v) {
		count = numberAtLeast(rt, "length", v, 0, count)
	}
	count = min(count, max(length-offset, 0))

	data := allocBytes(rt, count*element)
	if count > 0 {
		if buf, ok := GetBuffer(rt, view); ok {
			copy(data, buf[offset*element:offset*element+count*element])
		}
	}
	return newBuffer(rt, data)
}

// isBuffer returns true if the value is a buffer
func (*Buffer) isBuffer(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(IsBuffer(rt, call.Argument(0)))
}

// isEncoding returns true if the encoding is supported
func (*Buffer) isEncoding(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(normalizeEncoding(call.Argument(0).String()) != "")
}

// byteLength returns the byte length of a string when encoded using encoding.
func (*Buffer) byteLength(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	value := call.Argument(0)
	if data, ok := GetBuffer(rt, value); ok {
		return rt.ToValue(len(data))
	}
	if types.IsString(value) {
		encoding := call.Argument(1)
		// node assumes utf8 for an unrecognized encoding
		if !sobek.IsUndefined(encoding) {
			switch name := normalizeEncoding(encoding.String()); name {
			case "base64", "base64url":
				// node estimates the length of the base64 string
				return rt.ToValue(len(strings.TrimRight(value.String(), "=")) * 3 / 4)
			case "":
				encoding = rt.ToValue("utf8")
			}
		}
		return rt.ToValue(len(decode(rt, value.String(), encoding)))
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
		totalLength = numberInRange(rt, "length", v, 0, maxLength, int64(data.Len()))
	}

	result := allocBytes(rt, totalLength)
	copy(result, data.Bytes())

	return newBuffer(rt, result)
}

// toString decodes buf to a string according to the specified character encoding in encoding.
func (*Buffer) toString(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	return rt.ToValue(encode(rt, this.slice(call.Argument(1), call.Argument(2)), call.Argument(0)))
}

// slice returns the range of the buffer, node clamps the start and the end.
func (b buffer) slice(start, end sobek.Value) buffer {
	from := int64(0)
	to := int64(len(b))
	if !sobek.IsUndefined(start) {
		from = clampOffset(start.ToFloat(), int64(len(b)))
	}
	if !sobek.IsUndefined(end) {
		to = clampOffset(end.ToFloat(), int64(len(b)))
	}
	if from >= to {
		return nil
	}
	return b[from:to]
}

// clampOffset converts the value to a buffer offset, node clamps the values
// that are out of the buffer bounds and treats NaN as zero.
func clampOffset(value float64, length int64) int64 {
	switch {
	case math.IsNaN(value) || value < 0:
		return 0
	case value > float64(length):
		return length
	}
	return int64(value)
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

// indexOf returns the index of the first occurrence of value, -1 when the
// value is not found.
func (b *Buffer) indexOf(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(b.search(rt, call, false))
}

// lastIndexOf returns the index of the last occurrence of value, -1 when the
// value is not found.
func (b *Buffer) lastIndexOf(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(b.search(rt, call, true))
}

// includes returns true if the value is found in the buffer.
func (b *Buffer) includes(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	return rt.ToValue(b.search(rt, call, false) != -1)
}

// search implements the indexOf, lastIndexOf and includes of node, the value
// may be a number, a string, a Buffer or a Uint8Array.
func (b *Buffer) search(rt *sobek.Runtime, call sobek.FunctionCall, backwards bool) int64 {
	this := toBuffer(rt, call.This)
	offsetArg, endArg, encodingArg := call.Argument(1), call.Argument(2), call.Argument(3)
	// node treats a string in the offset or in the end position as the encoding
	switch {
	case types.IsString(offsetArg):
		encodingArg, offsetArg = offsetArg, sobek.Undefined()
	case types.IsString(endArg):
		encodingArg, endArg = endArg, sobek.Undefined()
	}

	value, ok := needle(rt, call.Argument(0), encodingArg)
	if !ok {
		panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
			`The "value" argument must be one of type number or string or an instance of`+
				` Buffer or Uint8Array. Received type %s`, types.ValueType(call.Argument(0))))
	}
	// node requires whole code units when a buffer is searched as utf16le
	aligned := !types.IsNumber(call.Argument(0)) && normalizeEncoding(encodingArg.String()) == "utf16le"
	if aligned && len(value)%2 == 1 {
		return -1
	}

	offset := searchOffset(offsetArg, int64(len(this)), backwards)
	end := int64(len(this))
	if !sobek.IsUndefined(endArg) {
		end = clampOffset(endArg.ToFloat(), int64(len(this)))
	}
	if len(value) == 0 {
		if backwards {
			return min(max(offset, 0), end)
		}
		return min(max(offset, 0), end)
	}
	if backwards {
		window := max(min(min(offset+int64(len(value)), end), int64(len(this))), 0)
		if aligned {
			return int64(alignedLastIndex(this[:window], value))
		}
		return int64(bytes.LastIndex(this[:window], value))
	}
	if end > int64(len(this)) {
		end = int64(len(this))
	}
	if offset >= end {
		return -1
	}
	if aligned {
		if index := alignedIndex(this[offset:end], value); index >= 0 {
			return offset + int64(index)
		}
		return -1
	}
	index := bytes.Index(this[offset:end], value)
	if index < 0 {
		return -1
	}
	return offset + int64(index)
}

// alignedIndex returns the index of the first match on a code unit boundary,
// -1 when the value is not found, utf16le only matches the whole code units.
func alignedIndex(data, value []byte) int {
	for i := 0; i+len(value) <= len(data); i += 2 {
		if bytes.Equal(data[i:i+len(value)], value) {
			return i
		}
	}
	return -1
}

// alignedLastIndex returns the index of the last match on a code unit boundary.
func alignedLastIndex(data, value []byte) int {
	last := len(data) - len(value)
	if last%2 != 0 {
		last--
	}
	for i := last; i >= 0; i -= 2 {
		if bytes.Equal(data[i:i+len(value)], value) {
			return i
		}
	}
	return -1
}

// needle returns the bytes of an indexOf value, false when the value is not a
// number, a string, a Buffer or a Uint8Array.
func needle(rt *sobek.Runtime, value sobek.Value, encoding sobek.Value) ([]byte, bool) {
	switch {
	case types.IsNumber(value):
		// node truncates the number to an unsigned 8-bit integer
		return []byte{byte(value.ToInteger())}, true
	case types.IsString(value):
		return decode(rt, value.String(), encoding), true
	}
	return viewBytes(rt, value)
}

// searchOffset returns the byte offset of an indexOf value, node counts a
// negative offset from the end of the buffer.
func searchOffset(value sobek.Value, length int64, backwards bool) int64 {
	if sobek.IsUndefined(value) {
		if backwards {
			return length
		}
		return 0
	}
	number := value.ToFloat()
	switch {
	case math.IsNaN(number):
		if backwards {
			return length
		}
		return 0
	case math.IsInf(number, 1):
		return length
	case math.IsInf(number, -1):
		if backwards {
			return -length - 1
		}
		return 0
	}
	offset := int64(number)
	if offset < 0 {
		offset = length + offset
	}
	if offset > length {
		return length
	}
	return offset
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
		source := toBuffer(rt, call.This)
		if len(call.Arguments) == 1 {
			return rt.ToValue(bytes.Compare(source, target))
		}
		// node does not coerce the offset arguments of the prototype method
		targetStart := numberAtLeast(rt, "targetStart", call.Argument(1), 0, 0)
		targetEnd := numberAtMost(rt, "targetEnd", call.Argument(2), int64(len(target)), int64(len(target)))
		sourceStart := numberAtLeast(rt, "sourceStart", call.Argument(3), 0, 0)
		sourceEnd := numberAtMost(rt, "sourceEnd", call.Argument(4), int64(len(source)), int64(len(source)))
		if targetStart >= targetEnd {
			if sourceEnd > sourceStart {
				return rt.ToValue(1)
			}
			return rt.ToValue(0)
		}
		if sourceStart >= sourceEnd {
			return rt.ToValue(-1)
		}
		return rt.ToValue(bytes.Compare(source[sourceStart:sourceEnd], target[targetStart:targetEnd]))
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

	targetStart := numberAtLeast(rt, "targetStart", call.Argument(1), 0, 0)
	sourceStart := numberInRange(rt, "sourceStart", call.Argument(2), 0, int64(len(this)), 0)
	// node clamps a sourceEnd that is out of the source bounds
	sourceEnd := numberAtLeast(rt, "sourceEnd", call.Argument(3), 0, int64(len(this)))
	sourceEnd = min(sourceEnd, int64(len(this)))
	sourceEnd = max(sourceEnd, sourceStart)

	data := make([]byte, sourceEnd-sourceStart)
	buffer(this).readAt(data, sourceStart)

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

	offset := numberInRange(rt, "offset", offsetArg, 0, int64(len(this)), 0)

	buf := decode(rt, call.Argument(0).String(), encodingArg)
	name := "utf8"
	if !sobek.IsUndefined(encodingArg) {
		name = normalizeEncoding(encodingArg.String())
	}
	length := int64(len(this)) - offset

	if v := lengthArg; !sobek.IsUndefined(v) {
		length = numberInRange(rt, "length", v, 0, int64(len(buf)), int64(len(buf)))
	}

	// node does not write a partial character
	length = writeLength(name, buf, length)
	written := copy(this[offset:], buf[:length])

	// node returns the number of bytes written
	return rt.ToValue(written)
}

// writeLength returns the number of bytes to write, node never writes a partial
// character: utf16le stops at an odd byte and utf8 stops before the character
// that does not fit into the buffer.
func writeLength(encoding string, data []byte, length int64) int64 {
	if length >= int64(len(data)) {
		return int64(len(data))
	}
	switch encoding {
	case "utf16le":
		return length - length%2
	case "utf8":
		for length > 0 {
			start := length - 1
			for start > 0 && data[start]&0xc0 == 0x80 {
				start--
			}
			if start+int64(utf8Size(data[start])) <= length {
				return length
			}
			length = start
		}
	}
	return length
}

// utf8Size returns the byte count of the utf8 sequence that starts with char.
func utf8Size(char byte) int {
	switch {
	case char&0x80 == 0:
		return 1
	case char&0xe0 == 0xc0:
		return 2
	case char&0xf0 == 0xe0:
		return 3
	case char&0xf8 == 0xf0:
		return 4
	}
	return 1
}

// fill fills buf with the specified value.
func (*Buffer) fill(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toBuffer(rt, call.This)
	value := call.Argument(0)
	max := int64(len(this))
	offset := int(numberInRange(rt, "offset", call.Argument(1), 0, max, 0))
	end := int(numberInRange(rt, "end", call.Argument(2), 0, max, max))

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

	this.fill(offset, end, buf)

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

// bufferSize returns the size argument of the allocation methods, node requires
// a non-negative integer that is not larger than kMaxLength.
func bufferSize(rt *sobek.Runtime, value sobek.Value) int64 {
	size := numberValue(rt, "size", value)
	if size < 0 || size > maxLength {
		throwRangeError(rt, "size", 0, maxLength, size)
	}
	return size
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
	length := numberValue(rt, "byteLength", v)
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
	switch normalizeEncoding(encoding) {
	case "utf8":
		return []byte(src)
	case "hex":
		return decodeHex(src)
	case "base64":
		return decodeBase64(src, false)
	case "base64url":
		return decodeBase64(src, true)
	case "latin1":
		// node writes the code unit truncated to 8 bits
		decoded := make([]byte, 0, len(src))
		for _, char := range src {
			decoded = append(decoded, byte(char))
		}
		return decoded
	case "ascii":
		// node encodes ascii like latin1
		return decode(rt, src, rt.ToValue("latin1"))
	case "utf16le":
		units := utf16.Encode([]rune(src))
		decoded := make([]byte, 0, len(units)*2)
		for _, unit := range units {
			decoded = append(decoded, byte(unit), byte(unit>>8))
		}
		return decoded
	default:
		panic(common.NewTypeErrorf(rt, common.ErrUnknownEncoding, "Unknown encoding: %s", encoding))
	}
}

// encodingNames maps the node encoding aliases to the canonical name, the
// lookup is case insensitive.
var encodingNames = map[string]string{
	"utf8":      "utf8",
	"utf-8":     "utf8",
	"utf16le":   "utf16le",
	"utf-16le":  "utf16le",
	"ucs2":      "utf16le",
	"ucs-2":     "utf16le",
	"latin1":    "latin1",
	"binary":    "latin1",
	"ascii":     "ascii",
	"base64":    "base64",
	"base64url": "base64url",
	"hex":       "hex",
}

// normalizeEncoding returns the canonical name of the encoding, the empty
// string means that the encoding is not supported.
func normalizeEncoding(name string) string {
	return encodingNames[strings.ToLower(name)]
}

// decodeBase64 decodes the base64 string like node, the characters that are
// not part of the alphabet and the padding are ignored.
func decodeBase64(src string, url bool) []byte {
	decoded := make([]byte, 0, len(src)*3/4)
	var buf uint32
	bits := 0
	for i := 0; i < len(src); i++ {
		value, ok := base64Value(src[i], url)
		if !ok {
			continue
		}
		buf = buf<<6 | uint32(value)
		bits += 6
		if bits >= 8 {
			bits -= 8
			decoded = append(decoded, byte(buf>>bits))
		}
	}
	return decoded
}

// base64Value returns the value of a base64 digit.
func base64Value(char byte, url bool) (byte, bool) {
	switch {
	case char >= 'A' && char <= 'Z':
		return char - 'A', true
	case char >= 'a' && char <= 'z':
		return char - 'a' + 26, true
	case char >= '0' && char <= '9':
		return char - '0' + 52, true
	case char == '+' || (url && char == '-'):
		return 62, true
	case char == '/' || (url && char == '_'):
		return 63, true
	}
	return 0, false
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
	switch normalizeEncoding(encoding) {
	case "utf8":
		return string(buf)
	case "hex":
		return hex.EncodeToString(buf)
	case "base64":
		return base64.StdEncoding.EncodeToString(buf)
	case "base64url":
		return base64.RawURLEncoding.EncodeToString(buf)
	case "latin1":
		var builder strings.Builder
		builder.Grow(len(buf))
		for _, b := range buf {
			builder.WriteRune(rune(b))
		}
		return builder.String()
	case "ascii":
		// node masks the high bit when decoding ascii
		var builder strings.Builder
		builder.Grow(len(buf))
		for _, b := range buf {
			builder.WriteRune(rune(b & 0x7f))
		}
		return builder.String()
	case "utf16le":
		units := make([]uint16, 0, len(buf)/2)
		for i := 0; i+1 < len(buf); i += 2 {
			units = append(units, uint16(buf[i])|uint16(buf[i+1])<<8)
		}
		return string(utf16.Decode(units))
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
	v := bigIntValue(rt, value)
	if v.Cmp(maxBigint) <= 0 && v.Cmp(minBigint) >= 0 {
		return uint64(v.Int64())
	}
	panic(common.NewRangeErrorf(rt, common.ErrOutOfRange,
		`The value of "value" is out of range. It must be >= -(2n ** 63n) and < 2n ** 63. Received %sn`, v))
}

// toBigUint converts a BigInt to an uint
func toBigUint(rt *sobek.Runtime, value sobek.Value) uint64 {
	v := bigIntValue(rt, value)
	if v.Cmp(maxBigUint) <= 0 && v.Cmp(big.NewInt(0)) >= 0 {
		return v.Uint64()
	}
	panic(common.NewRangeErrorf(rt, common.ErrOutOfRange,
		`The value of "value" is out of range. It must be >= 0n and < 2n ** 64n. Received %sn`, v))
}

// bigIntValue returns the big int of a BigInt argument.
func bigIntValue(rt *sobek.Runtime, value sobek.Value) *big.Int {
	v, ok := value.Export().(*big.Int)
	if !ok {
		panic(rt.NewTypeError("Cannot mix BigInt and other types, use explicit conversions"))
	}
	return v
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
	offset := numberArg(rt, "offset", v, 0)
	if offset < 0 || offset > int64(len(b))-numBytes {
		throwBoundsError(rt, "offset", offset, int64(len(b))-numBytes)
	}
	return offset
}

// offsetStrict returns the offset of the methods that require it, the byteLength
// methods of node do not default an undefined offset to zero.
func (b buffer) offsetStrict(rt *sobek.Runtime, v sobek.Value, numBytes int64) int64 {
	if sobek.IsUndefined(v) {
		throwInvalidArgType(rt, "offset", v)
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

// fill fills the buffer with data, the offset and the end are validated by the
// caller.
func (b buffer) fill(offset, end int, buf []byte) {
	if len(buf) > len(b) {
		b.writeAt(buf[:len(b)], int64(offset))
	} else {
		for i := offset; i < end; {
			i += copy(b[i:], buf)
		}
	}
}

// numberArg returns the value of an integer argument, the default is returned
// when the argument is undefined, a type error is thrown when the value is not
// a number and a range error when it is not an integer.
func numberArg(rt *sobek.Runtime, name string, value sobek.Value, def int64) int64 {
	if sobek.IsUndefined(value) {
		return def
	}
	if !types.IsNumber(value) {
		throwInvalidArgType(rt, name, value)
	}
	number := value.ToFloat()
	if !isInteger(number) {
		throwNotInteger(rt, name, value)
	}
	return int64(number)
}

// numberValue returns the value of an integer argument that must be provided.
func numberValue(rt *sobek.Runtime, name string, value sobek.Value) int64 {
	if sobek.IsUndefined(value) {
		throwInvalidArgType(rt, name, value)
	}
	if !types.IsNumber(value) {
		throwInvalidArgType(rt, name, value)
	}
	number := value.ToFloat()
	if !isInteger(number) {
		throwNotInteger(rt, name, value)
	}
	return int64(number)
}

// numberInRange returns the value of an integer argument within the inclusive
// range, the message follows the offset and the size errors of node.
func numberInRange(rt *sobek.Runtime, name string, value sobek.Value, min, max, def int64) int64 {
	number := numberArg(rt, name, value, def)
	if number < min || number > max {
		throwRangeError(rt, name, min, max, number)
	}
	return number
}

// numberAtLeast returns the value of an integer argument that is >= min.
func numberAtLeast(rt *sobek.Runtime, name string, value sobek.Value, min, def int64) int64 {
	number := numberArg(rt, name, value, def)
	if number < min {
		panic(common.NewRangeErrorf(rt, common.ErrOutOfRange,
			`The value of "%s" is out of range. It must be >= %d. Received %d`, name, min, number))
	}
	return number
}

// numberAtMost returns the value of an integer argument that is <= max.
func numberAtMost(rt *sobek.Runtime, name string, value sobek.Value, max, def int64) int64 {
	number := numberArg(rt, name, value, def)
	if number > max {
		panic(common.NewRangeErrorf(rt, common.ErrOutOfRange,
			`The value of "%s" is out of range. It must be <= %d. Received %d`, name, max, number))
	}
	return number
}

// throwInvalidArgType throws the node error of an argument that is not a number.
func throwInvalidArgType(rt *sobek.Runtime, name string, value sobek.Value) {
	panic(common.NewTypeErrorf(rt, common.ErrInvalidArgType,
		`The "%s" argument must be of type number. Received type %s`, name, types.ValueType(value)))
}

// throwNotInteger throws the node error of an argument that is not an integer.
func throwNotInteger(rt *sobek.Runtime, name string, value sobek.Value) {
	panic(common.NewRangeErrorf(rt, common.ErrOutOfRange,
		`The value of "%s" is out of range. It must be an integer. Received %s`, name, value.String()))
}

// isInteger reports whether the value is a javascript integer.
func isInteger(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v == math.Trunc(v)
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
