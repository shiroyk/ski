package cloudcat

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestZeroOr(t *testing.T) {
	assert.Equal(t, 1, ZeroOr(0, 1))
}

func TestEmptyOr(t *testing.T) {
	assert.Equal(t, []int{1}, EmptyOr([]int{}, []int{1}))
}

func TestParseCookie(t *testing.T) {
	var parseCookiesTests = []struct {
		String  string
		Cookies []*http.Cookie
	}{
		{
			"Cookie-1=v$1",
			[]*http.Cookie{{Name: "Cookie-1", Value: "v$1", Raw: "Cookie-1=v$1"}},
		},
		{
			"NID=99=MaIh2c9H-Mzwz-; expires=Wed, 07-Jun-2023 19:52:03 GMT; path=/; domain=.google.com; HttpOnly",
			[]*http.Cookie{{
				Name:       "NID",
				Value:      "99=MaIh2c9H-Mzwz-",
				Path:       "/",
				Domain:     ".google.com",
				HttpOnly:   true,
				Expires:    time.Date(2023, 6, 7, 19, 52, 3, 0, time.UTC),
				RawExpires: "Wed, 07-Jun-2023 19:52:03 GMT",
				Raw:        "NID=99=MaIh2c9H-Mzwz-; expires=Wed, 07-Jun-2023 19:52:03 GMT; path=/; domain=.google.com; HttpOnly",
			}},
		},
		{
			".ASPXAUTH=7E3AA; expires=Wed, 07-Jun-2023 19:58:08 GMT; path=/; HttpOnly",
			[]*http.Cookie{{
				Name:       ".ASPXAUTH",
				Value:      "7E3AA",
				Path:       "/",
				Expires:    time.Date(2023, 6, 7, 19, 58, 8, 0, time.UTC),
				RawExpires: "Wed, 07-Jun-2023 19:58:08 GMT",
				HttpOnly:   true,
				Raw:        ".ASPXAUTH=7E3AA; expires=Wed, 07-Jun-2023 19:58:08 GMT; path=/; HttpOnly",
			}},
		},
	}
	for _, tt := range parseCookiesTests {
		assert.Equal(t, tt.Cookies, ParseCookie(tt.String))
	}
}
