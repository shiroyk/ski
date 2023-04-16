package fetch

import (
	"io"
	"net/http"

	"github.com/shiroyk/cloudcat/core"
)

// DoString do request and read response body as string
func DoString(fetch core.Fetch, req *http.Request) (string, error) {
	res, err := fetch.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
