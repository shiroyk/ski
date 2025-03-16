package url

import "strings"

// ref: https://url.spec.whatwg.org/#concept-urlencoded-byte-serializer
//
//	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, A, B, C, D, E, F
var noEscape = [128]byte{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 0x00 - 0x0F
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 0x10 - 0x1F
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 1, 1, 0, // 0x20 - 0x2F
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, // 0x30 - 0x3F
	0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 0x40 - 0x4F
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 1, // 0x50 - 0x5F
	0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 0x60 - 0x6F
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, // 0x70 - 0x7F
}

const upperhex = "0123456789ABCDEF"

func ishex(c byte) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'f':
		return true
	case 'A' <= c && c <= 'F':
		return true
	}
	return false
}

func unhex(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

func queryEscape(s string) string {
	escape := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c > 127 || noEscape[c] == 0 {
			escape += 2
		}
	}

	if escape == 0 {
		return s
	}

	var builder strings.Builder
	builder.Grow(len(s) + escape)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == ' ':
			builder.WriteByte('+')
		case c > 127 || noEscape[c] == 0:
			builder.WriteByte('%')
			builder.WriteByte(upperhex[c>>4])
			builder.WriteByte(upperhex[c&0x0F])
		default:
			builder.WriteByte(c)
		}
	}
	return builder.String()
}

func queryUnescape(s string) string {
	i := strings.IndexAny(s, "%+")
	if i == -1 {
		return s
	}

	var builder strings.Builder
	builder.Grow(len(s))
	builder.WriteString(s[:i])

	for ; i < len(s); i++ {
		switch s[i] {
		case '%':
			if i+2 < len(s) && ishex(s[i+1]) && ishex(s[i+2]) {
				decoded := unhex(s[i+1])<<4 | unhex(s[i+2])
				builder.WriteByte(decoded)
				i += 2
			} else {
				builder.WriteByte('%')
			}
		case '+':
			builder.WriteByte(' ')
		default:
			builder.WriteByte(s[i])
		}
	}
	return builder.String()
}
