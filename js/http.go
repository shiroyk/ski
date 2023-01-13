package js

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/dop251/goja"
	"github.com/shiroyk/cloudcat/fetcher"
	"github.com/spf13/cast"
	"golang.org/x/exp/maps"
)

// jsHttp provides an interface for fetching resources (including across the network).
type jsHttp struct {
	fetch *fetcher.Fetcher
}

// handleBody process the send request body and set the content-type
func handleBody(body any, header map[string]string) (any, error) {
	switch data := body.(type) {
	case FormData:
		buf := &bytes.Buffer{}
		mpw := multipart.NewWriter(buf)
		for k, v := range data.data {
			for _, ve := range v {
				if f, ok := ve.(FileData); ok {
					// Creates a new form-data header with the provided field name and file name.
					fw, err := mpw.CreateFormFile(k, f.Filename)
					if err != nil {
						return nil, err
					}
					// Write bytes to the part
					if _, err := fw.Write(f.Data); err != nil {
						return nil, err
					}
				} else {
					// Write string value
					if err := mpw.WriteField(k, fmt.Sprintf("%v", v)); err != nil {
						return nil, err
					}
				}
			}
		}
		header["Content-Type"] = mpw.FormDataContentType()
		if err := mpw.Close(); err != nil {
			return nil, err
		}
		return buf, nil
	case URLSearchParams:
		header["Content-Type"] = "application/x-www-form-url"
		return data.encode(), nil
	case []byte, map[string]any, string, nil:
		return body, nil
	default:
		return nil, fmt.Errorf("unsupported request body type %v", body)
	}
}

// Get Make a GET request with URL and optional headers.
func (h *jsHttp) Get(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	header := cast.ToStringMapString(call.Argument(1).Export())

	res, err := h.fetch.Get(call.Argument(0).String(), header)
	if err != nil {
		panic(vm.ToValue(err))
	}

	return vm.ToValue(NewWrapResponse(res))
}

// Post Make a POST request with URL, optional body, optional headers.
// Send POST with multipart:
// http.post(url, new FormData({'bytes': new Uint8Array([0]).buffer}))
// Send POST with x-www-form-urlencoded:
// http.post(url, new URLSearchParams({'key': 'foo', 'value': 'bar'}))
// Send POST with json:
// http.post(url, {'key': 'foo'})
func (h *jsHttp) Post(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	u := call.Argument(0).String()
	body := UnWrapValue(call.Argument(1))
	header := cast.ToStringMapString(call.Argument(2).Export())

	var err error
	body, err = handleBody(body, header)
	if err != nil {
		panic(vm.ToValue(err))
	}

	res, err := h.fetch.Post(u, body, header)
	if err != nil {
		panic(vm.ToValue(err))
	}

	return vm.ToValue(NewWrapResponse(res))
}

// Head Make a HEAD request with URL and optional headers.
func (h *jsHttp) Head(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	header := cast.ToStringMapString(call.Argument(1).Export())

	res, err := h.fetch.Head(call.Argument(0).String(), header)
	if err != nil {
		panic(vm.ToValue(err))
	}

	return vm.ToValue(NewWrapResponse(res))
}

// Request Make a request with method and URL, optional body, optional headers.
func (h *jsHttp) Request(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	method := call.Argument(0).String()
	u := call.Argument(1).String()
	body := UnWrapValue(call.Argument(2))
	header := cast.ToStringMapString(call.Argument(3).Export())

	var err error
	body, err = handleBody(body, header)
	if err != nil {
		panic(vm.ToValue(err))
	}

	res, err := h.fetch.Request(method, u, body, header)
	if err != nil {
		panic(vm.ToValue(err))
	}

	return vm.ToValue(NewWrapResponse(res))
}

// Template Make a request with an HTTP template, template argument.
func (h *jsHttp) Template(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
	template := call.Argument(0).String()
	arg := cast.ToStringMap(call.Argument(1).Export())

	req, err := fetcher.NewTemplateRequest(nil, template, arg)
	if err != nil {
		panic(vm.ToValue(err))
	}

	res, err := h.fetch.DoRequest(req)
	if err != nil {
		panic(vm.ToValue(err))
	}

	return vm.ToValue(NewWrapResponse(res))
}

// WrapResponse wraps the given response
type WrapResponse struct {
	body       []byte      // response body.
	Headers    http.Header // response headers.
	Status     int         // HTTP status code.
	StatusText string      // HTTP status message corresponding to the HTTP status code.
	Ok         bool        // true if response Status in the range 200-299.
}

func NewWrapResponse(res *fetcher.Response) *WrapResponse {
	return &WrapResponse{
		body:       res.Body,
		Headers:    res.Header,
		Status:     res.StatusCode,
		StatusText: res.Status,
		Ok:         res.StatusCode >= 200 || res.StatusCode < 300,
	}
}

// String body resolves with a string. The response is always decoded using UTF-8.
func (r *WrapResponse) String(_ goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return vm.ToValue(string(r.body))
}

// Json parsing the body text as JSON.
func (r *WrapResponse) Json(_ goja.FunctionCall, vm *goja.Runtime) goja.Value {
	j := make(map[string]any)
	_ = json.Unmarshal(r.body, &j)
	return vm.ToValue(j)
}

// Bytes returns an ArrayBuffer.
func (r *WrapResponse) Bytes(_ goja.FunctionCall, vm *goja.Runtime) goja.Value {
	return vm.ToValue(vm.NewArrayBuffer(r.body))
}

// FileData wraps the file data and filename
type FileData struct {
	Data     []byte
	Filename string
}

// FormData provides a way to construct a set of key/value pairs representing form fields and their values.
// which can be sent using the http() method and encoding type were set to "multipart/form-data".
type FormData struct {
	data map[string][]any
}

func NewFormData(call goja.ConstructorCall, vm *goja.Runtime) *goja.Object {
	param := call.Argument(0)

	if goja.IsUndefined(param) {
		return vm.ToValue(FormData{data: make(map[string][]any)}).ToObject(vm)
	} else {
		if m, ok := param.Export().(map[string]any); ok {
			data := make(map[string][]any, len(m))
			for k, v := range m {
				switch ve := v.(type) {
				case goja.ArrayBuffer:
					// Default filename "blob".
					data[k] = []any{FileData{
						Data:     ve.Bytes(),
						Filename: "blob",
					}}
				case []any:
					data[k] = ve
				default:
					data[k] = []any{fmt.Sprintf("%v", ve)}
				}
			}
			return vm.ToValue(FormData{data: data}).ToObject(vm)
		} else {
			panic(vm.ToValue(fmt.Errorf("unsupported type %T", param.Export())))
		}
	}
}

// Append method of the FormData interface appends a new value onto an existing key inside a FormData object,
// or adds the key if it does not already exist.
func (f *FormData) Append(call goja.FunctionCall) (ret goja.Value) {
	name := call.Argument(0).String()
	value := call.Argument(1).Export()
	var filename string

	if goja.IsUndefined(call.Argument(2)) {
		// Default filename "blob".
		filename = "blob"
	} else {
		filename = call.Argument(2).String()
	}

	var ele []any
	var ok bool
	if ele, ok = f.data[name]; !ok {
		ele = make([]any, 0)
	}

	switch v := value.(type) {
	case goja.ArrayBuffer:
		ele = append(ele, FileData{
			Data:     v.Bytes(),
			Filename: filename,
		})
	default:
		ele = append(ele, fmt.Sprintf("%v", v))
	}

	f.data[name] = ele

	return
}

// Delete method of the FormData interface deletes a key and its value(s) from a FormData object.
func (f *FormData) Delete(call goja.FunctionCall) (ret goja.Value) {
	delete(f.data, call.Argument(0).String())
	return
}

// Entries method returns an iterator which iterates through all key/value pairs contained in the FormData.
func (f *FormData) Entries(_ goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	entries := make([][2]any, len(f.data))
	for k, v := range f.data {
		entries = append(entries, [2]any{k, v})
	}
	return vm.ToValue(entries)
}

// Get method of the FormData interface returns the first value associated
// with a given key from within a FormData object.
// If you expect multiple values and want all of them, use the getAll() method instead.
func (f *FormData) Get(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	if v, ok := f.data[call.Argument(0).String()]; ok {
		return vm.ToValue(v[0])
	}
	return
}

// GetAll method of the FormData interface returns all the values associated
// with a given key from within a FormData object.
func (f *FormData) GetAll(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	if v, ok := f.data[call.Argument(0).String()]; ok {
		return vm.ToValue(v)
	}
	return vm.ToValue([0]any{})
}

// Has method of the FormData interface returns whether a FormData object contains a certain key.
func (f *FormData) Has(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	if _, ok := f.data[call.Argument(0).String()]; ok {
		return vm.ToValue(true)
	}
	return vm.ToValue(false)
}

// Keys method returns an iterator which iterates through all keys contained in the FormData.
// The keys are strings.
func (f *FormData) Keys(_ goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	return vm.ToValue(maps.Keys(f.data))
}

// Set method of the FormData interface sets a new value for an existing key inside a FormData object,
// or adds the key/value if it does not already exist.
func (f *FormData) Set(call goja.FunctionCall) (ret goja.Value) {
	name := call.Argument(0).String()
	value := call.Argument(1).Export()
	var filename string

	if goja.IsUndefined(call.Argument(2)) {
		filename = "blob"
	} else {
		filename = call.Argument(2).String()
	}

	switch v := value.(type) {
	case goja.ArrayBuffer:
		f.data[name] = []any{
			FileData{
				Data:     v.Bytes(),
				Filename: filename,
			},
		}
	default:
		f.data[name] = []any{fmt.Sprintf("%v", v)}
	}

	return
}

// Values method returns an iterator which iterates through all values contained in the FormData.
func (f *FormData) Values(_ goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	return vm.ToValue(maps.Values(f.data))
}

// The URLSearchParams defines utility methods to work with the query string of a URL,
// which can be sent using the http() method and encoding type were set to "application/x-www-form-url".
type URLSearchParams struct {
	data map[string][]string
}

func NewURLSearchParams(call goja.ConstructorCall, vm *goja.Runtime) *goja.Object {
	param := call.Argument(0)

	if goja.IsUndefined(param) {
		return vm.ToValue(URLSearchParams{data: make(url.Values)}).ToObject(vm)
	} else {
		if m, ok := param.Export().(map[string]any); ok {
			data := make(map[string][]string, len(m))
			for k, v := range m {
				if s, ok := v.([]any); ok {
					data[k] = cast.ToStringSlice(s)
				} else {
					data[k] = []string{fmt.Sprintf("%v", v)}
				}
			}
			return vm.ToValue(URLSearchParams{data: data}).ToObject(vm)
		} else {
			panic(vm.ToValue(fmt.Errorf("unsupported type %T", param.Export())))
		}
	}
}

// encode encodes the values into “URL encoded” form
// ("bar=baz&foo=qux") sorted by key.
func (u *URLSearchParams) encode() string {
	if u.data == nil {
		return ""
	}
	var buf strings.Builder
	keys := make([]string, 0, len(u.data))
	for k := range u.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vs := u.data[k]
		keyEscaped := url.QueryEscape(k)
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(keyEscaped)
			buf.WriteByte('=')
			buf.WriteString(url.QueryEscape(v))
		}
	}
	return buf.String()
}

// Append method of the URLSearchParams interface appends a specified key/value pair as a new search parameter.
func (u *URLSearchParams) Append(call goja.FunctionCall) (ret goja.Value) {
	name := call.Argument(0).String()
	value := call.Argument(1).String()
	u.data[name] = append(u.data[name], value)
	return
}

// Delete method of the URLSearchParams interface deletes the given search parameter and all its associated values,
// from the list of all search parameters.
func (u *URLSearchParams) Delete(call goja.FunctionCall) (ret goja.Value) {
	name := call.Argument(0).String()
	delete(u.data, name)
	return
}

// Entries method of the URLSearchParams interface returns an iterator allowing iteration
// through all key/value pairs contained in this object.
// The iterator returns key/value pairs in the same order as they appear in the query string.
// The key and value of each pair are string objects.
func (u *URLSearchParams) Entries(_ goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	entries := make([][2]string, len(u.data))
	for k, v := range u.data {
		for _, ve := range v {
			entries = append(entries, [2]string{k, ve})
		}
	}
	return vm.ToValue(entries)
}

// ForEach method of the URLSearchParams interface allows iteration
// through all values contained in this object via a callback function.
func (u *URLSearchParams) ForEach(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	arg := call.Argument(0)
	if callback, ok := goja.AssertFunction(arg); ok {
		for k, v := range u.data {
			if _, err := callback(goja.Undefined(), vm.ToValue(v), vm.ToValue(k), vm.ToValue(u)); err != nil {
				panic(vm.ToValue(err))
			}
		}
	}
	return
}

// Get method of the URLSearchParams interface returns the first value associated to the given search parameter.
func (u *URLSearchParams) Get(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	if v, ok := u.data[call.Argument(0).String()]; ok {
		return vm.ToValue(v[0])
	}
	return
}

// GetAll method of the URLSearchParams interface returns all the values associated
// with a given search parameter as an array.
func (u *URLSearchParams) GetAll(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	if v, ok := u.data[call.Argument(0).String()]; ok {
		return vm.ToValue(v)
	}
	return vm.ToValue([0]string{})
}

// Has method of the URLSearchParams interface returns a boolean value that indicates whether
// a parameter with the specified name exists.
func (u *URLSearchParams) Has(call goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	if _, ok := u.data[call.Argument(0).String()]; ok {
		return vm.ToValue(true)
	}
	return vm.ToValue(false)
}

// Keys method of the URLSearchParams interface returns an iterator allowing iteration
// through all keys contained in this object. The keys are string objects.
func (u *URLSearchParams) Keys(_ goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	return vm.ToValue(maps.Keys(u.data))
}

// Set method of the URLSearchParams interface sets the value associated with a given search parameter to the given value.
// If there were several matching values, this method deletes the others.
// If the search parameter doesn't exist, this method creates it.
func (u *URLSearchParams) Set(call goja.FunctionCall) (ret goja.Value) {
	name := call.Argument(0).String()
	value := call.Argument(1).String()
	u.data[name] = []string{value}
	return
}

// Sort method sorts all key/value pairs contained in this object in place and returns undefined.
func (u *URLSearchParams) Sort(_ goja.FunctionCall) (ret goja.Value) {
	// Not implemented
	return
}

// ToString method of the URLSearchParams interface returns a query string suitable for use in a URL.
func (u *URLSearchParams) ToString(_ goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	return vm.ToValue(u.encode())
}

// Values method of the URLSearchParams interface returns an iterator allowing iteration through
// all values contained in this object. The values are string objects.
func (u *URLSearchParams) Values(_ goja.FunctionCall, vm *goja.Runtime) (ret goja.Value) {
	return vm.ToValue(maps.Values(u.data))
}
