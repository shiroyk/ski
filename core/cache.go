package core

import (
	"time"
)

// A Cache interface is used to store bytes.
type Cache interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte)
	SetWithTimeout(key string, value []byte, timeout time.Duration)
	Del(key string)
}
