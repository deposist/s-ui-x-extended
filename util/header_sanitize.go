package util

import (
	"strings"
	"unicode/utf8"
)

func SafeHeader(value string, max int) string {
	var b strings.Builder
	if max > 0 && len(value) > max {
		b.Grow(max)
	} else {
		b.Grow(len(value))
	}
	for _, r := range value {
		if r == 0 || r < 0x20 || r == 0x7f {
			continue
		}
		if max > 0 {
			runeLen := utf8.RuneLen(r)
			if runeLen < 0 {
				runeLen = len(string(r))
			}
			if b.Len()+runeLen > max {
				break
			}
		}
		b.WriteRune(r)
	}
	return b.String()
}
