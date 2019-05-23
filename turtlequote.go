package turtlequote

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

func Escape(s string) string {
	needsQuoting := false
	singleQuotable := true

	for _, c := range s {
		switch {
		case c == '\'', c == '\\':
			singleQuotable = false
			needsQuoting = true
		case c == '"', c == ' ', c == '&':
			// single-quotable still if it's just double quotes and spaces
			needsQuoting = true
		case unicode.IsSpace(c), unicode.In(c, unicode.Zl, unicode.Zp, unicode.Zs, unicode.Other):
			needsQuoting = true
			singleQuotable = false
		}
		if needsQuoting && !singleQuotable {
			break
		}
	}

	if !needsQuoting {
		return s
	}

	if singleQuotable {
		return "'" + s + "'"
	}

	ret := `"`
	for _, c := range s {
		switch {
		case c == '"':
			ret += `\"`
		case c == '\\':
			ret += "\\\\"
		case c == ' ':
			// avoid escaping ' ' as unicode
			ret += " "
		case c == '$':
			ret += `\$`
		case c == '`':
			ret += "\\`"
		case unicode.IsSpace(c), unicode.In(c, unicode.Zl, unicode.Zp, unicode.Zs, unicode.Other):
			ret += escapeUnicode(c)
		default:
			ret += string(c)
		}
	}
	return ret + `"`
}

func escapeUnicode(c rune) string {
	switch c {
	case '\x07':
		return `\a`
	case '\x08':
		return `\b`
	case '\x0b':
		return `\v`
	case '\x0c':
		return `\f`
	case '\x1b':
		return `\e`
	case '\t':
		return `\t`
	case '\r':
		return `\r`
	case '\n':
		return `\n`
	default:
		s := strconv.QuoteRune(c)
		// QuoteRune puts single quotes around unicode chars :/
		s = strings.Trim(s, `'`)
		// go formats as \u123, convert to \u{123} since that's what snailquote
		// picked for formatting escapes
		s = `\u{` + s[2:] + "}"
		return s
	}
}

func Unescape(s string) (string, error) {
	inDoubleQuote := false
	inSingleQuote := false

	var res string

	sr := []rune(s)
	for i := 0; i < len(sr); i++ {
		c := sr[i]
		switch {
		case inSingleQuote && c == '\'':
			inSingleQuote = false
		case inSingleQuote:
			res += string(c)
		case inDoubleQuote && c == '"':
			inDoubleQuote = false
		case inDoubleQuote && c == '\\':
			if i == len(s)-1 {
				return "", fmt.Errorf("invalid backslash escape at character %d", i)
			}
			i++
			switch sr[i] {
			case 'a':
				res += "\a"
			case 'b':
				res += "\b"
			case 'v':
				res += "\x0B"
			case 'f':
				res += "\x0C"
			case 'n':
				res += "\n"
			case 'r':
				res += "\r"
			case 't':
				res += "\t"
			case 'e', 'E':
				res += "\x1B"
			case '\\':
				res += `\`
			case '\'':
				res += "'"
			case '"':
				res += `"`
			case '$':
				res += "$"
			case '`':
				res += "`"
			case ' ':
				res += " "
			case 'u':
				// \u{123} escape
				charParsed, newI, err := parseUnicodeSeq(i, sr)
				if err != nil {
					return "", err
				}
				i = newI
				res += charParsed
			default:
				return "", fmt.Errorf("invalid escape at character %d", i)
			}
		case inDoubleQuote:
			res += string(c)
		case c == '\'':
			inSingleQuote = true
		case c == '"':
			inDoubleQuote = true
		default:
			res += string(c)
		}
	}
	return res, nil
}

func parseUnicodeSeq(i int, s []rune) (string, int, error) {
	start := i
	i++ // pass the u
	if i >= len(s)-1 {
		return "", i, fmt.Errorf("invalid unicode escape at char %d", i)
	}
	fragment := ""
	if s[i] != '{' {
		return "", i, fmt.Errorf("unicode escape must be of the form \\u{hex}, { was expected at char %d", i)
	}
	for i++; i < len(s) && s[i] != '}'; i++ {
		fragment += string(s[i])
	}
	if i == len(s) {
		return "", i, fmt.Errorf("could not find closing } for unicode escape starting at %d", start)
	}

	// UnquoteChar likes left padding to be present
	for len(fragment) < 8 {
		fragment = "0" + fragment
	}
	val, _, tail, err := strconv.UnquoteChar(`\U`+fragment, 0)
	if tail != "" {
		return "", i, fmt.Errorf("invalid escape fragment %q in unicode escape starting at %d", fragment, i)
	}
	if err != nil {
		return "", i, fmt.Errorf("invalid escape fragment %q in unicode escape starting at %d: %v", fragment, i, err)
	}
	return string(val), i, nil
}
