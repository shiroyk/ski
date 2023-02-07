package ext

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
)

var (
	mx         sync.RWMutex //nolint:gochecknoglobals
	extensions = make(map[ExtensionType]map[string]*Extension)
)

type ExtensionType uint

const (
	JSExtension ExtensionType = iota + 1
	ParserExtension
)

func (e ExtensionType) String() string {
	var s string
	switch e {
	case JSExtension:
		s = "js"
	case ParserExtension:
		s = "parser"
	}
	return s
}

type Extension struct {
	Name, Path, Desc, Version string
	Type                      ExtensionType
	Module                    any
}

func (e Extension) String() string {
	return fmt.Sprintf("%s [%s] %s %s ", e.Name, e.Type, e.Version, e.Path)
}

func (e Extension) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"name":    e.Name,
		"path":    e.Path,
		"desc":    e.Desc,
		"version": e.Version,
		"type":    e.Type.String(),
	})
}

// Register a new extension with the given name and type. This function will
// panic if an unsupported extension type is provided, or if an extension of the
// same type and name is already registered.
func Register(name string, typ ExtensionType, mod any) {
	mx.Lock()
	defer mx.Unlock()

	if mod == nil {
		panic(errors.New("extension cannot be nil"))
	}

	exts, ok := extensions[typ]
	if !ok {
		panic(fmt.Sprintf("unsupported extension type: %T", typ))
	}

	if _, ok := exts[name]; ok {
		panic(fmt.Sprintf("extension already registered: %s", name))
	}

	path, version := extractModuleInfo(mod)

	exts[name] = &Extension{
		Name:    name,
		Type:    typ,
		Module:  mod,
		Path:    path,
		Version: version,
	}
}

// Get returns all extensions of the specified type.
func Get(typ ExtensionType) map[string]*Extension {
	mx.RLock()
	defer mx.RUnlock()

	exts, ok := extensions[typ]
	if !ok {
		panic(fmt.Sprintf("unsupported extension type: %T", typ))
	}

	result := make(map[string]*Extension, len(exts))

	for name, ext := range exts {
		result[name] = ext
	}

	return result
}

func GetAll() []*Extension {
	mx.RLock()
	defer mx.RUnlock()

	js, parser := extensions[JSExtension], extensions[ParserExtension]
	result := make([]*Extension, 0, len(js)+len(parser))

	for _, e := range js {
		result = append(result, e)
	}
	for _, e := range parser {
		result = append(result, e)
	}

	return result
}

// extractModuleInfo attempts to return the package path and version of the Go
// module that created the given value.
func extractModuleInfo(mod any) (path, version string) {
	t := reflect.TypeOf(mod)

	switch t.Kind() {
	case reflect.Ptr, reflect.Struct:
		if t.Elem() != nil {
			path = t.Elem().PkgPath()
		}
	case reflect.Func:
		path = runtime.FuncForPC(reflect.ValueOf(mod).Pointer()).Name()
	default:
		return
	}

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	for _, dep := range buildInfo.Deps {
		depPath := strings.TrimSpace(dep.Path)
		if strings.HasPrefix(path, depPath) {
			if dep.Replace != nil {
				return depPath, dep.Replace.Version
			}
			return depPath, dep.Version
		}
	}

	return
}

func init() {
	extensions[JSExtension] = make(map[string]*Extension)
	extensions[ParserExtension] = make(map[string]*Extension)
}
