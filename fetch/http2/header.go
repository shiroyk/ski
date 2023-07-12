package http2

import (
	"net/http"
	"sort"
	"strings"
	"sync"

	"golang.org/x/net/http/httpguts"
)

// validWireHeaderFieldName reports whether v is a valid header field
// name (key). See httpguts.ValidHeaderName for the base rules.
//
// Further, http2 says:
//
//	"Just as in HTTP/1.x, header field names are strings of ASCII
//	characters that are compared in a case-insensitive
//	fashion. However, header field names MUST be converted to
//	lowercase prior to their encoding in HTTP/2. "
func validWireHeaderFieldName(v string) bool {
	if len(v) == 0 {
		return false
	}
	for _, r := range v {
		if !httpguts.IsTokenRune(r) {
			return false
		}
		if 'A' <= r && r <= 'Z' {
			return false
		}
	}
	return true
}

type keyValues struct {
	key    string
	values []string
}

// A headerSorter implements sort.Interface by sorting a []keyValues
// by the given order, if not nil, or by Key otherwise.
// It's used as a pointer, so it can fit in a sort.Interface
// value without allocation.
type headerSorter struct {
	kvs   []keyValues
	order map[string]int
}

func (s *headerSorter) Len() int      { return len(s.kvs) }
func (s *headerSorter) Swap(i, j int) { s.kvs[i], s.kvs[j] = s.kvs[j], s.kvs[i] }
func (s *headerSorter) Less(i, j int) bool {
	// If the order isn't defined, sort lexicographically.
	if len(s.order) == 0 {
		return s.kvs[i].key < s.kvs[j].key
	}
	si, iok := s.order[strings.ToLower(s.kvs[i].key)]
	sj, jok := s.order[strings.ToLower(s.kvs[j].key)]
	if !iok && !jok {
		return s.kvs[i].key < s.kvs[j].key
	} else if !iok && jok {
		return false
	} else if iok && !jok {
		return true
	}
	return si < sj
}

var headerSorterPool = sync.Pool{
	New: func() interface{} { return new(headerSorter) },
}

func sortedKeyValues(header http.Header) (kvs []keyValues) {
	sorter := headerSorterPool.Get().(*headerSorter)
	if cap(sorter.kvs) < len(header) {
		sorter.kvs = make([]keyValues, 0, len(header))
	}
	kvs = sorter.kvs[:0]
	for k, vv := range header {
		kvs = append(kvs, keyValues{k, vv})
	}
	sorter.kvs = kvs
	sort.Sort(sorter)
	return kvs
}

func sortedKeyValuesBy(header http.Header, headerOrder []string) (kvs []keyValues) {
	sorter := headerSorterPool.Get().(*headerSorter)
	if cap(sorter.kvs) < len(header) {
		sorter.kvs = make([]keyValues, 0, len(header))
	}
	kvs = sorter.kvs[:0]
	for k, vv := range header {
		kvs = append(kvs, keyValues{k, vv})
	}
	sorter.kvs = kvs
	sorter.order = make(map[string]int)
	for i, v := range headerOrder {
		sorter.order[v] = i
	}
	sort.Sort(sorter)
	return kvs
}
