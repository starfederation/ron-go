package ron

import (
	"sort"
	"unicode/utf8"
)

type jsonMember struct {
	Key        string
	ValueStart int
	ValueEnd   int
}

type jsonMemberSorter []jsonMember

func (s jsonMemberSorter) Len() int {
	return len(s)
}

func (s jsonMemberSorter) Less(i, j int) bool {
	return s[i].Key < s[j].Key
}

func (s jsonMemberSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func writeRONJSONInto(buf *jsonBytes, src []byte, prefix, indent string, canonical bool) error {
	p := parser{src: src}
	p.skipSpace()
	if p.pos < len(p.src) && p.src[p.pos] != '{' && p.src[p.pos] != '[' {
		start := p.pos
		bufStart := len(*buf)
		if err := p.writeTopLevelJSONObject(buf, prefix, indent, canonical); err == nil {
			return nil
		}
		*buf = (*buf)[:bufStart]
		p.pos = start
	}

	if err := p.writeJSONValue(buf, prefix, indent, 0, canonical); err != nil {
		return err
	}
	p.skipSpace()
	if p.pos != len(p.src) {
		return p.errorf("unexpected trailing data")
	}
	return nil
}

func (p *parser) writeTopLevelJSONObject(buf *jsonBytes, prefix, indent string, canonical bool) error {
	members := jsonMembers{Values: make([]jsonMember, 0, 4)}
	var values jsonBytes
	for {
		p.skipSpace()
		if p.pos == len(p.src) {
			return p.writeJSONObjectMembers(buf, members.Values, values, prefix, indent, 0, canonical)
		}
		if p.src[p.pos] == '{' || p.src[p.pos] == '[' {
			return p.errorf("top-level object elision not applicable")
		}

		key, err := p.parseKeyCurrent()
		if err != nil {
			return err
		}

		valueStart := len(values)
		if err := p.writeJSONValue(&values, prefix, indent, 1, canonical); err != nil {
			return err
		}
		members.Set(key, valueStart, len(values))
	}
}

func (p *parser) writeJSONValue(buf *jsonBytes, prefix, indent string, depth int, canonical bool) error {
	p.skipSpace()
	return p.writeJSONValueCurrent(buf, prefix, indent, depth, canonical)
}

func writeJSONQuoted(buf *jsonBytes, value string) {
	const hex = "0123456789abcdef"

	dst := *buf
	dst = append(dst, '"')
	start := 0
	for i := 0; i < len(value); {
		b := value[i]
		if b < utf8.RuneSelf {
			if b >= 0x20 && b != '\\' && b != '"' {
				i++
				continue
			}

			dst = append(dst, value[start:i]...)
			switch b {
			case '\\', '"':
				dst = append(dst, '\\', b)
			case '\b':
				dst = append(dst, '\\', 'b')
			case '\f':
				dst = append(dst, '\\', 'f')
			case '\n':
				dst = append(dst, '\\', 'n')
			case '\r':
				dst = append(dst, '\\', 'r')
			case '\t':
				dst = append(dst, '\\', 't')
			default:
				dst = append(dst, '\\', 'u', '0', '0', hex[b>>4], hex[b&0xf])
			}
			i++
			start = i
			continue
		}

		r, size := utf8.DecodeRuneInString(value[i:])
		if r == utf8.RuneError && size == 1 {
			dst = append(dst, value[start:i]...)
			dst = append(dst, '\\', 'u', 'f', 'f', 'f', 'd')
			i++
			start = i
			continue
		}
		i += size
	}
	dst = append(dst, value[start:]...)
	dst = append(dst, '"')
	*buf = dst
}

func (p *parser) writeJSONValueCurrent(buf *jsonBytes, prefix, indent string, depth int, canonical bool) error {
	if p.pos == len(p.src) {
		return p.errorf("expected value")
	}
	switch p.src[p.pos] {
	case '{':
		return p.writeJSONObject(buf, prefix, indent, depth, canonical)
	case '[':
		return p.writeJSONArray(buf, prefix, indent, depth, canonical)
	case ',':
		writeJSONQuoted(buf, p.parseCommaPrefixedToken())
		return nil
	case '\'':
		value, err := p.parseApostropheValue()
		if err != nil {
			return err
		}
		writeJSONQuoted(buf, value)
		return nil
	case '"':
		value, err := p.parseQuotedString()
		if err != nil {
			return err
		}
		writeJSONQuoted(buf, value)
		return nil
	}

	start, end, err := p.parseTokenSpan()
	if err != nil {
		return err
	}
	token := p.src[start:end]
	switch {
	case len(token) == 4 && token[0] == 't' && token[1] == 'r' && token[2] == 'u' && token[3] == 'e':
		writeString(buf, "true")
	case len(token) == 5 && token[0] == 'f' && token[1] == 'a' && token[2] == 'l' && token[3] == 's' && token[4] == 'e':
		writeString(buf, "false")
	case len(token) == 4 && token[0] == 'n' && token[1] == 'u' && token[2] == 'l' && token[3] == 'l':
		writeString(buf, "null")
	case looksLikeNumberBytes(token):
		writeBytes(buf, token)
	default:
		writeJSONQuoted(buf, bytesToString(token))
	}
	return nil
}

func (p *parser) writeJSONObject(buf *jsonBytes, prefix, indent string, depth int, canonical bool) error {
	p.pos++
	members := jsonMembers{Values: make([]jsonMember, 0, 8)}
	values := make(jsonBytes, 0, 128)
	for {
		p.skipWhitespace()
		if p.pos == len(p.src) {
			return p.errorf("expected }")
		}
		if p.src[p.pos] == '}' {
			p.pos++
			return p.writeJSONObjectMembers(buf, members.Values, values, prefix, indent, depth, canonical)
		}

		key, err := p.parseKeyCurrent()
		if err != nil {
			return err
		}

		p.skipWhitespace()
		valueStart := len(values)
		if err := p.writeJSONValueCurrent(&values, prefix, indent, depth+1, canonical); err != nil {
			return err
		}
		members.Set(key, valueStart, len(values))
		p.skipSeparators()
	}
}

func (p *parser) writeJSONObjectMembers(buf *jsonBytes, members []jsonMember, values []byte, prefix, indent string, depth int, canonical bool) error {
	if len(members) == 0 {
		writeString(buf, "{}")
		return nil
	}
	if canonical {
		sort.Sort(jsonMemberSorter(members))
	}
	writeByte(buf, '{')
	if indent == "" {
		for i, member := range members {
			if i > 0 {
				writeByte(buf, ',')
			}
			writeJSONQuoted(buf, member.Key)
			writeByte(buf, ':')
			writeBytes(buf, values[member.ValueStart:member.ValueEnd])
		}
		writeByte(buf, '}')
		return nil
	}
	writeByte(buf, '\n')
	for i, member := range members {
		writeJSONIndent(buf, prefix, indent, depth+1)
		writeJSONQuoted(buf, member.Key)
		writeString(buf, ": ")
		writeBytes(buf, values[member.ValueStart:member.ValueEnd])
		if i+1 < len(members) {
			writeByte(buf, ',')
		}
		writeByte(buf, '\n')
	}
	writeJSONIndent(buf, prefix, indent, depth)
	writeByte(buf, '}')
	return nil
}

func (p *parser) writeJSONArray(buf *jsonBytes, prefix, indent string, depth int, canonical bool) error {
	p.pos++
	p.skipWhitespace()
	if p.pos == len(p.src) {
		return p.errorf("expected ]")
	}
	if p.src[p.pos] == ']' {
		p.pos++
		writeString(buf, "[]")
		return nil
	}

	writeByte(buf, '[')
	if indent == "" {
		for i := 0; ; i++ {
			if i > 0 {
				writeByte(buf, ',')
			}
			if err := p.writeJSONValueCurrent(buf, prefix, indent, depth+1, canonical); err != nil {
				return err
			}
			p.skipSeparators()
			if p.pos == len(p.src) {
				return p.errorf("expected ]")
			}
			if p.src[p.pos] == ']' {
				p.pos++
				writeByte(buf, ']')
				return nil
			}
		}
	}

	writeByte(buf, '\n')
	for i := 0; ; i++ {
		writeJSONIndent(buf, prefix, indent, depth+1)
		if err := p.writeJSONValueCurrent(buf, prefix, indent, depth+1, canonical); err != nil {
			return err
		}
		p.skipSeparators()
		if p.pos == len(p.src) {
			return p.errorf("expected ]")
		}
		if p.src[p.pos] == ']' {
			p.pos++
			writeByte(buf, '\n')
			writeJSONIndent(buf, prefix, indent, depth)
			writeByte(buf, ']')
			return nil
		}
		writeByte(buf, ',')
		writeByte(buf, '\n')
	}
}

type jsonMembers struct {
	Values []jsonMember
	Index  map[string]int
}

func (m *jsonMembers) Set(key string, valueStart, valueEnd int) {
	if m.Index != nil {
		if idx, ok := m.Index[key]; ok {
			copy(m.Values[idx:], m.Values[idx+1:])
			m.Values = m.Values[:len(m.Values)-1]
			delete(m.Index, key)
			for i := idx; i < len(m.Values); i++ {
				m.Index[m.Values[i].Key] = i
			}
		}
	} else if len(m.Values) > 0 {
		for idx, member := range m.Values {
			if member.Key == key {
				copy(m.Values[idx:], m.Values[idx+1:])
				m.Values = m.Values[:len(m.Values)-1]
				break
			}
		}
		if len(m.Values) == 8 {
			m.Index = make(map[string]int, len(m.Values)+1)
			for i, member := range m.Values {
				m.Index[member.Key] = i
			}
		}
	}
	if m.Index != nil {
		m.Index[key] = len(m.Values)
	}
	m.Values = append(m.Values, jsonMember{
		Key:        key,
		ValueStart: valueStart,
		ValueEnd:   valueEnd,
	})
}
