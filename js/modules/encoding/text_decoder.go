package encoding

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/grafana/sobek"
	"github.com/shiroyk/ski/js"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
)

func init() {
	js.Register("TextDecoder", new(TextDecoder))
	js.Register("TextEncoder", new(TextEncoder))
}

// TextDecoder a decoder for a specific text encoding, such as
// UTF-8, ISO-8859-2, KOI8-R, GBK, etc. A decoder takes a stream of bytes
// as input and emits a stream of code points.
// https://developer.mozilla.org/en-US/docs/Web/API/TextDecoder
type TextDecoder struct{}

func (t *TextDecoder) prototype(rt *sobek.Runtime) *sobek.Object {
	p := rt.NewObject()
	_ = p.DefineAccessorProperty("encoding", rt.ToValue(t.encoding), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("fatal", rt.ToValue(t.fatal), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.DefineAccessorProperty("ignoreBOM", rt.ToValue(t.ignoreBOM), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE)
	_ = p.Set("decode", t.decode)
	_ = p.SetSymbol(sobek.SymToStringTag, func(sobek.ConstructorCall) sobek.Value { return rt.ToValue("TextDecoder") })
	return p
}

func (t *TextDecoder) constructor(call sobek.ConstructorCall, rt *sobek.Runtime) *sobek.Object {
	var (
		label     = "utf-8"
		fatal     = false
		ignoreBOM = false
	)

	if v := call.Argument(0); !sobek.IsUndefined(v) {
		label = strings.ToLower(v.String())
	}
	if v := call.Argument(1); !sobek.IsUndefined(v) {
		opts := v.ToObject(rt)
		if v := opts.Get("fatal"); v != nil {
			fatal = v.ToBoolean()
		}
		if v := opts.Get("ignoreBOM"); v != nil {
			ignoreBOM = v.ToBoolean()
		}
	}

	enc, ok := encodings[label]
	if !ok {
		panic(rt.NewTypeError("unsupported encoding: %s", label))
	}

	instance := &textDecoder{
		encoding:  label,
		decoder:   enc.NewDecoder(),
		fatal:     fatal,
		ignoreBOM: ignoreBOM,
	}

	obj := rt.ToValue(instance).(*sobek.Object)
	_ = obj.SetPrototype(call.This.Prototype())
	return obj
}

func (*TextDecoder) encoding(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toTextDecoder(rt, call.This)
	return rt.ToValue(this.encoding)
}

func (*TextDecoder) fatal(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toTextDecoder(rt, call.This)
	return rt.ToValue(this.fatal)
}

func (*TextDecoder) ignoreBOM(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toTextDecoder(rt, call.This)
	return rt.ToValue(this.ignoreBOM)
}

func (*TextDecoder) decode(call sobek.FunctionCall, rt *sobek.Runtime) sobek.Value {
	this := toTextDecoder(rt, call.This)
	var input []byte

	if v := call.Argument(0); !sobek.IsUndefined(v) {
		switch t := v.Export().(type) {
		case []byte:
			input = t
		case sobek.ArrayBuffer:
			input = t.Bytes()
		default:
			panic(rt.NewTypeError("unsupported input type: %T", t))
		}
	}

	if len(input) == 0 {
		return rt.ToValue("")
	}

	if this.encoding == "utf-8" {
		if !this.ignoreBOM && len(input) >= 3 && bytes.Equal(input[:3], []byte{0xEF, 0xBB, 0xBF}) {
			input = input[3:]
		}
		if !utf8.Valid(input) && this.fatal {
			js.Throw(rt, fmt.Errorf("invalid UTF-8 sequence"))
		}
		return rt.ToValue(string(input))
	}

	result, err := this.decoder.Bytes(input)
	if err != nil && this.fatal {
		js.Throw(rt, err)
	}
	return rt.ToValue(string(result))
}

type textDecoder struct {
	encoding  string
	decoder   *encoding.Decoder
	fatal     bool
	ignoreBOM bool
}

var (
	typeTextDecoder = reflect.TypeOf((*textDecoder)(nil))
	encodings       = map[string]encoding.Encoding{
		"utf-8":        unicode.UTF8,
		"utf8":         unicode.UTF8,
		"utf-16le":     unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM),
		"utf-16be":     unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM),
		"gbk":          simplifiedchinese.GBK,
		"gb2312":       simplifiedchinese.GBK,
		"gb18030":      simplifiedchinese.GB18030,
		"big5":         traditionalchinese.Big5,
		"euc-jp":       japanese.EUCJP,
		"iso-2022-jp":  japanese.ISO2022JP,
		"shift-jis":    japanese.ShiftJIS,
		"euc-kr":       korean.EUCKR,
		"iso-8859-1":   charmap.ISO8859_1,
		"iso-8859-2":   charmap.ISO8859_2,
		"iso-8859-3":   charmap.ISO8859_3,
		"iso-8859-4":   charmap.ISO8859_4,
		"iso-8859-5":   charmap.ISO8859_5,
		"iso-8859-6":   charmap.ISO8859_6,
		"iso-8859-7":   charmap.ISO8859_7,
		"iso-8859-8":   charmap.ISO8859_8,
		"iso-8859-9":   charmap.ISO8859_9,
		"iso-8859-10":  charmap.ISO8859_10,
		"iso-8859-13":  charmap.ISO8859_13,
		"iso-8859-14":  charmap.ISO8859_14,
		"iso-8859-15":  charmap.ISO8859_15,
		"iso-8859-16":  charmap.ISO8859_16,
		"windows-1250": charmap.Windows1250,
		"windows-1251": charmap.Windows1251,
		"windows-1252": charmap.Windows1252,
		"windows-1253": charmap.Windows1253,
		"windows-1254": charmap.Windows1254,
		"windows-1255": charmap.Windows1255,
		"windows-1256": charmap.Windows1256,
		"windows-1257": charmap.Windows1257,
		"windows-1258": charmap.Windows1258,
		"koi8-r":       charmap.KOI8R,
		"koi8-u":       charmap.KOI8U,
	}
)

func toTextDecoder(rt *sobek.Runtime, value sobek.Value) *textDecoder {
	if value.ExportType() == typeTextDecoder {
		return value.Export().(*textDecoder)
	}
	panic(rt.NewTypeError(`Value of "this" must be of type TextDecoder`))
}

func (t *TextDecoder) Instantiate(rt *sobek.Runtime) (sobek.Value, error) {
	proto := t.prototype(rt)
	ctor := rt.ToValue(t.constructor).(*sobek.Object)
	_ = proto.DefineDataProperty("constructor", ctor, sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_FALSE)
	_ = ctor.Set("prototype", proto)
	return ctor, nil
}

func (*TextDecoder) Global() {}
