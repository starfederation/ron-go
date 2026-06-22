package ron

import (
	stdregexp "regexp"
	"strconv"
	"strings"

	fastregexp "github.com/coregx/coregex"
)

// RegExp is a core vocabulary #rx JavaScript regular expression compiled for fast pure-Go matching.
// Source and Flags preserve the canonical RON payload. Matching uses a RE2-compatible Go pattern.
type RegExp struct {
	Source string
	Flags  string

	re    *fastregexp.Regexp
	stdre *stdregexp.Regexp
}

// ParseRegExp validates a JavaScript RegExp source and canonical flags, then compiles a fast pure-Go matcher.
func ParseRegExp(source, flags string) (RegExp, error) {
	pattern, err := jsRegExpGoPattern(source, flags)
	if err != nil {
		return RegExp{}, err
	}
	stdre, err := stdregexp.Compile(pattern)
	if err != nil {
		return RegExp{}, err
	}
	re, err := fastregexp.Compile(pattern)
	if err != nil {
		return RegExp{}, err
	}

	return RegExp{
		Source: source,
		Flags:  flags,
		re:     re,
		stdre:  stdre,
	}, nil
}

// GoPattern returns the RE2-compatible Go pattern used for matching.
func (r RegExp) GoPattern() (string, error) {
	return jsRegExpGoPattern(r.Source, r.Flags)
}

// Regexp returns the compiled fast RE2 matcher.
func (r RegExp) Regexp() (*fastregexp.Regexp, error) {
	if r.re != nil {
		return r.re, nil
	}
	parsed, err := ParseRegExp(r.Source, r.Flags)
	if err != nil {
		return nil, err
	}
	return parsed.re, nil
}

// MatchString reports whether the regular expression matches s.
func (r RegExp) MatchString(s string) bool {
	if strings.ContainsRune(r.Flags, 'i') {
		if r.stdre != nil {
			return r.stdre.MatchString(s)
		}
		pattern, err := r.GoPattern()
		if err != nil {
			return false
		}
		stdre, err := stdregexp.Compile(pattern)
		if err != nil {
			return false
		}
		return stdre.MatchString(s)
	}

	re, err := r.Regexp()
	if err != nil {
		return false
	}
	return re.MatchString(s)
}

func parseRegExpPayload(payload any) (RegExp, error) {
	array, ok := payload.([]any)
	if !ok || len(array) < 1 || len(array) > 2 {
		return RegExp{}, newError("invalid #rx payload")
	}
	source, ok := array[0].(string)
	if !ok {
		return RegExp{}, newError("invalid #rx payload")
	}

	flags := ""
	if len(array) == 2 {
		var ok bool
		flags, ok = array[1].(string)
		if !ok {
			return RegExp{}, newError("invalid #rx payload")
		}
	}
	if !validJSRegExpFlags(flags) {
		return RegExp{}, newError("invalid #rx flags")
	}

	value, err := ParseRegExp(source, flags)
	if err != nil {
		return RegExp{}, newError("invalid #rx source")
	}
	return value, nil
}

func regExpTaggedMember(value RegExp) objectMember {
	payload := []any{value.Source}
	if value.Flags != "" {
		payload = append(payload, value.Flags)
	}
	return objectMember{
		Key:   "#rx",
		Value: payload,
	}
}

func jsRegExpGoPattern(source, flags string) (string, error) {
	if !validJSRegExpFlags(flags) {
		return "", newError("invalid #rx flags")
	}

	converted, err := convertJSRegExpSource(source)
	if err != nil {
		return "", err
	}

	goFlags := strings.Builder{}
	for i := range flags {
		switch flags[i] {
		case 'i', 'm', 's':
			goFlags.WriteByte(flags[i])
		}
	}
	if goFlags.Len() == 0 {
		return converted, nil
	}
	return "(?" + goFlags.String() + ":" + converted + ")", nil
}

func validJSRegExpFlags(flags string) bool {
	const order = "dgimsuvy"
	last := -1
	seenU := false
	seenV := false
	for _, flag := range flags {
		index := strings.IndexRune(order, flag)
		if index < 0 || index <= last {
			return false
		}
		last = index
		switch flag {
		case 'u':
			seenU = true
		case 'v':
			seenV = true
		}
	}
	return !seenU || !seenV
}

func convertJSRegExpSource(source string) (string, error) {
	var converted strings.Builder
	converted.Grow(len(source))
	inClass := false
	for i := 0; i < len(source); {
		if source[i] != '\\' {
			if source[i] == '[' && !inClass {
				inClass = true
			} else if source[i] == ']' && inClass {
				inClass = false
			}
			converted.WriteByte(source[i])
			i++
			continue
		}
		if i+1 == len(source) {
			converted.WriteByte(source[i])
			i++
			continue
		}

		next := source[i+1]
		switch {
		case next == 'u':
			end, err := appendUnicodeEscape(&converted, source, i+2)
			if err != nil {
				return "", err
			}
			i = end
		case next == 'c' && i+2 < len(source) && isASCIIAlpha(source[i+2]):
			appendHexEscape(&converted, source[i+2]&0x1f)
			i += 3
		case next == 'b' && inClass:
			appendHexEscape(&converted, 0x08)
			i += 2
		default:
			converted.WriteByte(source[i])
			converted.WriteByte(next)
			i += 2
		}
	}
	return converted.String(), nil
}

func appendUnicodeEscape(dst *strings.Builder, source string, pos int) (int, error) {
	if pos < len(source) && source[pos] == '{' {
		end := pos + 1
		for end < len(source) && source[end] != '}' {
			if !isHexByte(source[end]) {
				return 0, newError("invalid #rx source")
			}
			end++
		}
		if end == pos+1 || end == len(source) {
			return 0, newError("invalid #rx source")
		}
		value, err := strconv.ParseUint(source[pos+1:end], 16, 32)
		if err != nil || value > 0x10ffff {
			return 0, newError("invalid #rx source")
		}
		dst.WriteString(`\x{`)
		dst.WriteString(source[pos+1 : end])
		dst.WriteByte('}')
		return end + 1, nil
	}

	if pos+4 > len(source) {
		return 0, newError("invalid #rx source")
	}
	for _, value := range source[pos : pos+4] {
		if !isHexByte(byte(value)) {
			return 0, newError("invalid #rx source")
		}
	}
	dst.WriteString(`\x{`)
	dst.WriteString(source[pos : pos+4])
	dst.WriteByte('}')
	return pos + 4, nil
}

func appendHexEscape(dst *strings.Builder, value byte) {
	dst.WriteString(`\x{`)
	if value < 0x10 {
		dst.WriteByte('0')
	}
	dst.WriteString(strconv.FormatUint(uint64(value), 16))
	dst.WriteByte('}')
}

func isHexByte(value byte) bool {
	return (value >= '0' && value <= '9') || (value >= 'A' && value <= 'F') || (value >= 'a' && value <= 'f')
}

func isASCIIAlpha(value byte) bool {
	return (value >= 'A' && value <= 'Z') || (value >= 'a' && value <= 'z')
}
