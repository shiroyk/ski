package cache

// Shortener is URL shortener to reduce a long link and headers.
type Shortener interface {
	// Set returns to shorten identifier for the given URL and headers.
	Set(url string, headers map[string]string) string
	// Get returns the original URL and headers for the given identifier.
	Get(shortener string) (url string, headers map[string]string, ok bool)
}
