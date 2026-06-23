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
			pos := i + 2
			if pos < len(source) && source[pos] == '{' {
				end := pos + 1
				for end < len(source) && source[end] != '}' {
					if !isHexByte(source[end]) {
						return "", newError("invalid #rx source")
					}
					end++
				}
				if end == pos+1 || end == len(source) {
					return "", newError("invalid #rx source")
				}
				value, err := strconv.ParseUint(source[pos+1:end], 16, 32)
				if err != nil || value > 0x10ffff {
					return "", newError("invalid #rx source")
				}
				converted.WriteString(`\x{`)
				converted.WriteString(source[pos+1 : end])
				converted.WriteByte('}')
				i = end + 1
				break
			}

			if pos+4 > len(source) {
				return "", newError("invalid #rx source")
			}
			for _, value := range source[pos : pos+4] {
				if !isHexByte(byte(value)) {
					return "", newError("invalid #rx source")
				}
			}
			converted.WriteString(`\x{`)
			converted.WriteString(source[pos : pos+4])
			converted.WriteByte('}')
			i = pos + 4
		case next == 'c' && i+2 < len(source) && ((source[i+2] >= 'A' && source[i+2] <= 'Z') || (source[i+2] >= 'a' && source[i+2] <= 'z')):
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

	goFlags := strings.Builder{}
	for i := range flags {
		switch flags[i] {
		case 'i', 'm', 's':
			goFlags.WriteByte(flags[i])
		}
	}
	if goFlags.Len() == 0 {
		return converted.String(), nil
	}
	return "(?" + goFlags.String() + ":" + converted.String() + ")", nil
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
