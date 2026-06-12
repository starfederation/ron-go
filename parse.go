package ron

import (
	"bytes"
	"fmt"
	"unicode"
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
	src []byte
	pos int
}

type ronNumber string

var asciiSpace = [utf8.RuneSelf]bool{
	' ':  true,
	'\t': true,
	'\n': true,
	'\r': true,
	',':  true,
}

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
	p := parser{src: src}
	p.skipSpace()
	if p.pos < len(p.src) && p.src[p.pos] != '{' && p.src[p.pos] != '[' {
		p.pos = 0
		if value, err := p.parseTopLevelObject(); err == nil {
			return value, nil
		}
	}

	p.pos = 0
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

func (p *parser) parseTopLevelObject() (any, error) {
	p.skipSpace()
	if p.pos == len(p.src) {
		return nil, p.errorf("expected value")
	}
	switch p.src[p.pos] {
	case '{', '[':
		return nil, p.errorf("top-level object elision not applicable")
	}

	key, err := p.parseKeyCurrent()
	if err != nil {
		return nil, err
	}

	value, err := p.parseValue()
	if err != nil {
		return nil, err
	}

	object := map[string]any{
		key: value,
	}
	p.skipSpace()
	for p.pos < len(p.src) {
		key, err := p.parseKeyCurrent()
		if err != nil {
			return nil, err
		}

		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}

		object[key] = value
		p.skipSpace()
	}
	return object, nil
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
		return p.parseObject()
	case '[':
		return p.parseArray()
	case ',':
		return p.parseCommaPrefixedToken(), nil
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
	return bytesToString(token), nil
}

func (p *parser) scanValue() error {
	p.skipSpace()
	return p.scanValueCurrent()
}

func (p *parser) scanValueCurrent() error {
	if p.pos == len(p.src) {
		return p.errorf("expected value")
	}
	switch p.src[p.pos] {
	case '{':
		return p.scanObject()
	case '[':
		return p.scanArray()
	case ',':
		p.parseCommaPrefixedToken()
		return nil
	case '\'':
		return p.scanApostropheValue()
	case '"':
		return p.scanQuoted()
	}
	return p.scanToken()
}

func (p *parser) parseObject() (any, error) {
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
}

func (p *parser) scanObject() error {
	p.pos++
	for {
		p.skipWhitespace()
		if p.pos == len(p.src) {
			return p.errorf("expected }")
		}
		if p.src[p.pos] == '}' {
			p.pos++
			return nil
		}
		if _, err := p.parseKeyCurrent(); err != nil {
			return err
		}
		p.skipWhitespace()
		if err := p.scanValueCurrent(); err != nil {
			return err
		}
		p.skipSeparators()
	}
}

func (p *parser) parseArray() (any, error) {
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
}

func (p *parser) scanArray() error {
	p.pos++
	for {
		p.skipWhitespace()
		if p.pos == len(p.src) {
			return p.errorf("expected ]")
		}
		if p.src[p.pos] == ']' {
			p.pos++
			return nil
		}
		if err := p.scanValueCurrent(); err != nil {
			return err
		}
		p.skipSeparators()
	}
}

func (p *parser) parseKey() (string, error) {
	p.skipSpace()
	return p.parseKeyCurrent()
}

func (p *parser) parseKeyCurrent() (string, error) {
	if p.pos == len(p.src) {
		return "", p.errorf("expected object key")
	}
	switch p.src[p.pos] {
	case ',':
		return p.parseCommaPrefixedToken(), nil
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
	return bytesToString(p.src[start:end]), nil
}

func (p *parser) parseApostropheValue() (string, error) {
	if p.apostropheIsToken() {
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

func (p *parser) scanApostropheValue() error {
	if p.apostropheIsToken() {
		p.pos++
		return nil
	}

	start := p.pos
	if err := p.scanQuoted(); err == nil {
		return nil
	}
	p.pos = start
	if p.pos+1 == len(p.src) || isDelimiter(p.src[p.pos+1]) {
		p.pos++
		return nil
	}
	return p.scanQuoted()
}

func (p *parser) apostropheIsToken() bool {
	if p.pos+1 == len(p.src) {
		return true
	}
	switch p.src[p.pos+1] {
	case ' ', '\t', '\n', '\r':
		for pos := p.pos + 2; pos < len(p.src); pos++ {
			switch p.src[pos] {
			case '\'':
				return false
			case '{', '}', '[', ']':
				return true
			}
		}
		return true
	}
	return false
}

func (p *parser) parseQuotedBytes() ([]byte, error) {
	quote := p.src[p.pos]
	count := 0
	for p.pos+count < len(p.src) && p.src[p.pos+count] == quote {
		count++
	}
	if p.pos+count == len(p.src) || isDelimiter(p.src[p.pos+count]) {
		if count%2 == 0 {
			p.pos += count
			return nil, nil
		}
		if count >= 5 && (count-2)%3 == 0 {
			p.pos += count
			return bytes.Repeat([]byte{quote}, (count-2)/3), nil
		}
	}

	p.pos += count
	start := p.pos
	for {
		if p.pos == len(p.src) {
			return nil, p.errorf("unterminated string")
		}

		next := bytes.IndexByte(p.src[p.pos:], quote)
		if next < 0 {
			p.pos = len(p.src)
			return nil, p.errorf("unterminated string")
		}

		p.pos += next
		run := 0
		for p.pos+run < len(p.src) && p.src[p.pos+run] == quote {
			run++
		}
		if run >= count {
			value := p.src[start:p.pos]
			p.pos += count
			return value, nil
		}

		p.pos += run
	}
}

func (p *parser) parseQuotedString() (string, error) {
	value, err := p.parseQuotedBytes()
	if err != nil {
		return "", err
	}
	return bytesToString(value), nil
}

func (p *parser) scanQuoted() error {
	quote := p.src[p.pos]
	count := 0
	for p.pos+count < len(p.src) && p.src[p.pos+count] == quote {
		count++
	}
	if p.pos+count == len(p.src) || isDelimiter(p.src[p.pos+count]) {
		if count%2 == 0 || (count >= 5 && (count-2)%3 == 0) {
			p.pos += count
			return nil
		}
	}

	p.pos += count
	for {
		if p.pos == len(p.src) {
			return p.errorf("unterminated string")
		}

		next := bytes.IndexByte(p.src[p.pos:], quote)
		if next < 0 {
			p.pos = len(p.src)
			return p.errorf("unterminated string")
		}

		p.pos += next
		run := 0
		for p.pos+run < len(p.src) && p.src[p.pos+run] == quote {
			run++
		}
		if run >= count {
			p.pos += count
			return nil
		}

		p.pos += run
	}
}

func (p *parser) parseCommaPrefixedToken() string {
	start := p.pos
	p.pos++
	for p.pos < len(p.src) && !isDelimiter(p.src[p.pos]) {
		p.pos++
	}
	return bytesToString(p.src[start:p.pos])
}

func (p *parser) parseToken() (string, error) {
	start, end, err := p.parseTokenSpan()
	if err != nil {
		return "", err
	}
	return bytesToString(p.src[start:end]), nil
}

func (p *parser) parseTokenSpan() (int, int, error) {
	start := p.pos
	for p.pos < len(p.src) {
		if b := p.src[p.pos]; b < utf8.RuneSelf && asciiDelimiter[b] {
			break
		}
		p.pos++
	}
	if start == p.pos {
		return 0, 0, p.errorf("expected token")
	}
	return start, p.pos, nil
}

func (p *parser) scanToken() error {
	start := p.pos
	for p.pos < len(p.src) {
		if b := p.src[p.pos]; b < utf8.RuneSelf && asciiDelimiter[b] {
			break
		}
		p.pos++
	}
	if start == p.pos {
		return p.errorf("expected token")
	}
	return nil
}

func (p *parser) skipSpace() {
	pos := p.pos
	src := p.src
	for pos < len(src) {
		if b := src[pos]; b < utf8.RuneSelf {
			if asciiSpace[b] {
				pos++
				continue
			}
			p.pos = pos
			return
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

func (p *parser) errorf(msg string) error {
	return &parseError{
		msg: msg,
		pos: p.pos,
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

func looksLikeNumber(token string) bool {
	return looksLikeNumberBytes([]byte(token))
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
