package ron

import (
	"bytes"
	"fmt"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

type parseError struct {
	msg string
	pos int
}

func (e *parseError) Error() string {
	return fmt.Sprintf("ron: %s at byte %d", e.msg, e.pos)
}

func newError(msg string) error {
	return &parseError{
		msg: msg,
	}
}

type parser struct {
	src                    []byte
	pos                    int
	maxNestingDepth        int
	nestingDepth           int
	jsonScratch            []bytes.Buffer
	jsonScratchDepth       int
	jsonMemberScratch      [][]jsonMember
	jsonMemberScratchDepth int
}

type ronNumber string

var asciiDelimiter = [utf8.RuneSelf]bool{
	'{':  true,
	'}':  true,
	'[':  true,
	']':  true,
	'"':  true,
	'\'': true,
	',':  true,
	' ':  true,
	'\t': true,
	'\n': true,
	'\r': true,
}

func parse(src []byte) (any, error) {
	return parseWithMaxDepth(src, 0)
}

func parseWithMaxDepth(src []byte, maxNestingDepth int) (any, error) {
	p := parser{
		src:             src,
		maxNestingDepth: maxNestingDepth,
	}
	p.skipSpace()
	if p.pos < len(p.src) && p.src[p.pos] != '{' && p.src[p.pos] != '[' {
		p.nestingDepth = 1
		p.pos = 0
		p.skipSpace()
		key, err := p.parseKeyCurrent()
		if err == nil {
			p.skipWhitespace()
			value, err := p.parseValueCurrent()
			if err == ErrNestingTooDeep {
				return nil, err
			}
			if err == nil {
				object := map[string]any{
					key: value,
				}
				ok := true
				p.skipSpace()
				for p.pos < len(p.src) {
					key, err := p.parseKeyCurrent()
					if err != nil {
						ok = false
						break
					}

					p.skipWhitespace()
					value, err := p.parseValueCurrent()
					if err == ErrNestingTooDeep {
						return nil, err
					}
					if err != nil {
						ok = false
						break
					}

					object[key] = value
					p.skipSpace()
				}
				if ok {
					return object, nil
				}
			}
		}
	}

	p.pos = 0
	p.nestingDepth = 0
	value, err := p.parseValue()
	if err != nil {
		return nil, err
	}

	p.skipSpace()
	if p.pos != len(p.src) {
		return nil, p.errorf("unexpected trailing data")
	}
	return value, nil
}

func (p *parser) parseValue() (any, error) {
	p.skipSpace()
	return p.parseValueCurrent()
}

func (p *parser) parseValueCurrent() (any, error) {
	if p.pos == len(p.src) {
		return nil, p.errorf("expected value")
	}
	switch p.src[p.pos] {
	case '{':
		if err := p.enterContainer(); err != nil {
			return nil, err
		}
		defer p.leaveContainer()
		p.pos++
		object := make(map[string]any, 4)
		for {
			p.skipWhitespace()
			if p.pos == len(p.src) {
				return nil, p.errorf("expected }")
			}
			if p.src[p.pos] == '}' {
				p.pos++
				return object, nil
			}

			key, err := p.parseKeyCurrent()
			if err != nil {
				return nil, err
			}

			p.skipWhitespace()
			value, err := p.parseValueCurrent()
			if err != nil {
				return nil, err
			}

			object[key] = value
			p.skipSeparators()
		}
	case '[':
		if err := p.enterContainer(); err != nil {
			return nil, err
		}
		defer p.leaveContainer()
		p.pos++
		array := make([]any, 0, 4)
		for {
			p.skipWhitespace()
			if p.pos == len(p.src) {
				return nil, p.errorf("expected ]")
			}
			if p.src[p.pos] == ']' {
				p.pos++
				return array, nil
			}

			value, err := p.parseValueCurrent()
			if err != nil {
				return nil, err
			}

			array = append(array, value)
			p.skipSeparators()
		}
	case ',':
		return p.parseCommaPrefixedToken()
	case '\'':
		return p.parseApostropheValue()
	case '"':
		return p.parseQuotedString()
	}

	start, end, err := p.parseTokenSpan()
	if err != nil {
		return nil, err
	}

	token := p.src[start:end]
	switch {
	case len(token) == 4 && token[0] == 't' && token[1] == 'r' && token[2] == 'u' && token[3] == 'e':
		return true, nil
	case len(token) == 5 && token[0] == 'f' && token[1] == 'a' && token[2] == 'l' && token[3] == 's' && token[4] == 'e':
		return false, nil
	case len(token) == 4 && token[0] == 'n' && token[1] == 'u' && token[2] == 'l' && token[3] == 'l':
		return nil, nil
	}
	if looksLikeNumberBytes(token) {
		return ronNumber(bytesToString(token)), nil
	}
	return p.decodeStringSpan(start, end)
}

func (p *parser) parseKeyCurrent() (string, error) {
	if p.pos == len(p.src) {
		return "", p.errorf("expected object key")
	}
	switch p.src[p.pos] {
	case ',':
		return p.parseCommaPrefixedToken()
	case '\'':
		return p.parseApostropheValue()
	case '"':
		return p.parseQuotedString()
	case '{', '}', '[', ']':
		return "", p.errorf("expected object key")
	}

	start, end, err := p.parseTokenSpan()
	if err != nil {
		return "", err
	}
	return p.decodeStringSpan(start, end)
}

func (p *parser) parseApostropheValue() (string, error) {
	apostropheIsToken := false
	if p.pos+1 == len(p.src) {
		apostropheIsToken = true
	} else if p.src[p.pos+1] == ' ' || p.src[p.pos+1] == '\t' || p.src[p.pos+1] == '\n' || p.src[p.pos+1] == '\r' {
		apostropheIsToken = true
		for pos := p.pos + 2; pos < len(p.src); pos++ {
			switch p.src[pos] {
			case '\'':
				apostropheIsToken = false
				pos = len(p.src)
			case '{', '}', '[', ']':
				pos = len(p.src)
			}
		}
	}
	if apostropheIsToken {
		p.pos++
		return "'", nil
	}

	start := p.pos
	value, err := p.parseQuotedString()
	if err == nil {
		return value, nil
	}
	p.pos = start
	if p.pos+1 == len(p.src) || isDelimiter(p.src[p.pos+1]) {
		p.pos++
		return "'", nil
	}
	return "", err
}

func (p *parser) parseQuotedString() (string, error) {
	quote := p.src[p.pos]
	count := 0
	for p.pos+count < len(p.src) && p.src[p.pos+count] == quote {
		count++
	}
	if p.pos+count == len(p.src) || isDelimiter(p.src[p.pos+count]) {
		if count%2 == 0 {
			p.pos += count
			return "", nil
		}
		if quote == '\'' && count >= 5 && (count-2)%3 == 0 {
			p.pos += count
			return bytesToString(bytes.Repeat([]byte{quote}, (count-2)/3)), nil
		}
	}

	p.pos += count
	start := p.pos
	for p.pos < len(p.src) {
		b := p.src[p.pos]
		if b == '\\' {
			_, next, err := p.decodeEscape(p.pos)
			if err != nil {
				return "", err
			}
			p.pos = next
			continue
		}
		if b < 0x20 {
			return "", p.errorf("unescaped control character in string")
		}
		if b == quote {
			run := 0
			for p.pos+run < len(p.src) && p.src[p.pos+run] == quote {
				run++
			}
			if run >= count {
				end := p.pos
				p.pos += count
				return p.decodeStringSpan(start, end)
			}
			p.pos += run
			continue
		}
		if b < utf8.RuneSelf {
			p.pos++
			continue
		}

		_, size := utf8.DecodeRune(p.src[p.pos:])
		if size == 1 {
			return "", p.errorf("invalid UTF-8 in string")
		}
		p.pos += size
	}
	return "", p.errorf("unterminated string")
}

func (p *parser) parseCommaPrefixedToken() (string, error) {
	start := p.pos
	p.pos++
	if err := p.scanTokenEnd(); err != nil {
		return "", err
	}
	return p.decodeStringSpan(start, p.pos)
}

func (p *parser) parseTokenSpan() (int, int, error) {
	start := p.pos
	if err := p.scanTokenEnd(); err != nil {
		return 0, 0, err
	}
	if start == p.pos {
		return 0, 0, p.errorf("expected token")
	}
	return start, p.pos, nil
}

func (p *parser) scanTokenEnd() error {
	for p.pos < len(p.src) {
		b := p.src[p.pos]
		if b == '\\' {
			_, next, err := p.decodeEscape(p.pos)
			if err != nil {
				return err
			}
			p.pos = next
			continue
		}
		if b < utf8.RuneSelf {
			if asciiDelimiter[b] {
				return nil
			}
			if b < 0x20 {
				return p.errorf("unescaped control character in string")
			}
			p.pos++
			continue
		}

		r, size := utf8.DecodeRune(p.src[p.pos:])
		if size == 1 {
			return p.errorf("invalid UTF-8 in string")
		}
		if unicode.IsSpace(r) {
			return nil
		}
		p.pos += size
	}
	return nil
}

func (p *parser) decodeStringSpan(start, end int) (string, error) {
	firstEscape := bytes.IndexByte(p.src[start:end], '\\')
	if firstEscape < 0 {
		return bytesToString(p.src[start:end]), nil
	}

	decoded := make([]byte, 0, end-start)
	pos := start
	for pos < end {
		nextEscape := bytes.IndexByte(p.src[pos:end], '\\')
		if nextEscape < 0 {
			decoded = append(decoded, p.src[pos:end]...)
			break
		}

		escapePos := pos + nextEscape
		decoded = append(decoded, p.src[pos:escapePos]...)
		r, next, err := p.decodeEscape(escapePos)
		if err != nil {
			return "", err
		}
		decoded = utf8.AppendRune(decoded, r)
		pos = next
	}
	return string(decoded), nil
}

func (p *parser) decodeEscape(pos int) (rune, int, error) {
	if pos+1 >= len(p.src) {
		return 0, 0, p.errorAt("truncated escape", pos)
	}

	switch p.src[pos+1] {
	case '"', '\\', '/':
		return rune(p.src[pos+1]), pos + 2, nil
	case 'b':
		return '\b', pos + 2, nil
	case 'f':
		return '\f', pos + 2, nil
	case 'n':
		return '\n', pos + 2, nil
	case 'r':
		return '\r', pos + 2, nil
	case 't':
		return '\t', pos + 2, nil
	case 'u':
	default:
		return 0, 0, p.errorAt("unknown escape", pos)
	}

	decodeHex4 := func(start int) (uint16, int) {
		var value uint16
		for i := start; i < start+4; i++ {
			var digit byte
			switch b := p.src[i]; {
			case b >= '0' && b <= '9':
				digit = b - '0'
			case b >= 'a' && b <= 'f':
				digit = b - 'a' + 10
			case b >= 'A' && b <= 'F':
				digit = b - 'A' + 10
			default:
				return 0, i
			}
			value = value<<4 | uint16(digit)
		}
		return value, -1
	}

	if pos+6 > len(p.src) {
		return 0, 0, p.errorAt("truncated unicode escape", pos)
	}
	value, invalid := decodeHex4(pos + 2)
	if invalid >= 0 {
		return 0, 0, p.errorAt("non-hex unicode escape", invalid)
	}
	next := pos + 6
	if value >= 0xd800 && value <= 0xdbff {
		if next+6 > len(p.src) || p.src[next] != '\\' || p.src[next+1] != 'u' {
			return 0, 0, p.errorAt("unpaired high surrogate", pos)
		}
		low, invalid := decodeHex4(next + 2)
		if invalid >= 0 {
			return 0, 0, p.errorAt("non-hex unicode escape", invalid)
		}
		if low < 0xdc00 || low > 0xdfff {
			return 0, 0, p.errorAt("unpaired high surrogate", pos)
		}
		return utf16.DecodeRune(rune(value), rune(low)), next + 6, nil
	}
	if value >= 0xdc00 && value <= 0xdfff {
		return 0, 0, p.errorAt("unpaired low surrogate", pos)
	}
	return rune(value), next, nil
}

func (p *parser) skipSpace() {
	pos := p.pos
	src := p.src
	for pos < len(src) {
		if b := src[pos]; b < utf8.RuneSelf {
			switch b {
			case ' ', '\t', '\n', '\r', ',':
				pos++
				continue
			default:
				p.pos = pos
				return
			}
		}
		r, size := utf8.DecodeRune(src[pos:])
		if !unicode.IsSpace(r) {
			p.pos = pos
			return
		}
		pos += size
	}
	p.pos = pos
}

func (p *parser) skipWhitespace() {
	pos := p.pos
	src := p.src
	for pos < len(src) {
		if b := src[pos]; b < utf8.RuneSelf {
			switch b {
			case ' ', '\t', '\n', '\r':
				pos++
				continue
			default:
				p.pos = pos
				return
			}
		}
		r, size := utf8.DecodeRune(src[pos:])
		if !unicode.IsSpace(r) {
			p.pos = pos
			return
		}
		pos += size
	}
	p.pos = pos
}

func (p *parser) skipSeparators() {
	for {
		p.skipWhitespace()
		if p.pos == len(p.src) || p.src[p.pos] != ',' {
			return
		}
		p.pos++
	}
}

func (p *parser) enterContainer() error {
	if p.maxNestingDepth > 0 && p.nestingDepth >= p.maxNestingDepth {
		return ErrNestingTooDeep
	}
	p.nestingDepth++
	return nil
}

func (p *parser) leaveContainer() {
	p.nestingDepth--
}

func (p *parser) errorf(msg string) error {
	return p.errorAt(msg, p.pos)
}

func (p *parser) errorAt(msg string, pos int) error {
	return &parseError{
		msg: msg,
		pos: pos,
	}
}

func isDelimiter(b byte) bool {
	return b < utf8.RuneSelf && asciiDelimiter[b]
}

func bytesToString(value []byte) string {
	if len(value) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(value), len(value))
}

func looksLikeNumberBytes(token []byte) bool {
	if len(token) == 0 {
		return false
	}

	pos := 0
	if token[pos] == '-' {
		pos++
		if pos == len(token) {
			return false
		}
	}
	switch token[pos] {
	case '0':
		pos++
	default:
		if token[pos] < '1' || token[pos] > '9' {
			return false
		}
		for pos < len(token) && token[pos] >= '0' && token[pos] <= '9' {
			pos++
		}
	}
	if pos < len(token) && token[pos] == '.' {
		pos++
		if pos == len(token) || token[pos] < '0' || token[pos] > '9' {
			return false
		}
		for pos < len(token) && token[pos] >= '0' && token[pos] <= '9' {
			pos++
		}
	}
	if pos < len(token) && (token[pos] == 'e' || token[pos] == 'E') {
		pos++
		if pos < len(token) && (token[pos] == '+' || token[pos] == '-') {
			pos++
		}
		if pos == len(token) || token[pos] < '0' || token[pos] > '9' {
			return false
		}
		for pos < len(token) && token[pos] >= '0' && token[pos] <= '9' {
			pos++
		}
	}
	if pos != len(token) {
		return false
	}
	return true
}
