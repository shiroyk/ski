package cache

// Shortener is URL shortener to reduce a long link and headers.
type Shortener interface {
	// Set returns to shorten identifier for the given HTTP request.
	Set(http string) string
	// Get returns the HTTP request for the given identifier.
	Get(shortener string) (http string, ok bool)
}
