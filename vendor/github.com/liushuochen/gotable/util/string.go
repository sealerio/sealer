package util

import (
	"strings"
	"unicode"
)

const (
	chineseSymbol = "！……（），。？、"
)

func Capitalize(s string) string {
	if len(s) < 1 {
		return s
	}
	firstLetter := s[0]
	if firstLetter < 65 || (firstLetter > 90 && firstLetter < 97) ||
		firstLetter > 122 {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

func Length(s string) int {
	length := 0
	for _, c := range s {
		if isChinese(c) {
			length += 2
		} else {
			length += 1
		}
	}
	return length
}

func isChinese(c int32) bool {
	if unicode.Is(unicode.Han, c) {
		return true
	}

	for _, s := range chineseSymbol {
		if c == s {
			return true
		}
	}
	return false
}
