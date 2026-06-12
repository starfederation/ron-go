package ron

import (
	"bytes"
	"sort"
	"strconv"
)

type jsonMember struct {
	Key   string
	Value []byte
}

func writeRONJSONInto(buf *jsonBytes, src []byte, prefix, indent string) error {
	p := parser{src: src}
	p.skipSpace()
	if p.pos < len(p.src) && p.src[p.pos] != '{' && p.src[p.pos] != '[' {
		start := p.pos
		var tmp jsonBytes
		if err := p.writeTopLevelJSONObject(&tmp, prefix, indent); err == nil {
			writeBytes(buf, tmp)
			return nil
		}
		p.pos = start
	}

	if err := p.writeJSONValue(buf, prefix, indent, 0); err != nil {
		return err
	}
	p.skipSpace()
	if p.pos != len(p.src) {
		return p.errorf("unexpected trailing data")
	}
	return nil
}

func (p *parser) writeTopLevelJSONObject(buf *jsonBytes, prefix, indent string) error {
	members := jsonMembers{}
	for {
		p.skipSpace()
		if p.pos == len(p.src) {
			return p.writeJSONObjectMembers(buf, members.Values, prefix, indent, 0)
		}
		if p.src[p.pos] == '{' || p.src[p.pos] == '[' {
			return p.errorf("top-level object elision not applicable")
		}

		key, err := p.parseKeyCurrent()
		if err != nil {
			return err
		}

		var value jsonBytes
		if err := p.writeJSONValue(&value, prefix, indent, 1); err != nil {
			return err
		}
		members.Set(key, value)
	}
}

func (p *parser) writeJSONValue(buf *jsonBytes, prefix, indent string, depth int) error {
	p.skipSpace()
	return p.writeJSONValueCurrent(buf, prefix, indent, depth)
}

func (p *parser) writeJSONValueCurrent(buf *jsonBytes, prefix, indent string, depth int) error {
	if p.pos == len(p.src) {
		return p.errorf("expected value")
	}
	switch p.src[p.pos] {
	case '{':
		return p.writeJSONObject(buf, prefix, indent, depth)
	case '[':
		return p.writeJSONArray(buf, prefix, indent, depth)
	case ',':
		writeString(buf, strconv.Quote(p.parseCommaPrefixedToken()))
		return nil
	case '\'':
		value, err := p.parseApostropheValue()
		if err != nil {
			return err
		}
		writeString(buf, strconv.Quote(value))
		return nil
	case '"':
		value, err := p.parseQuotedString()
		if err != nil {
			return err
		}
		writeString(buf, strconv.Quote(value))
		return nil
	}

	start, end, err := p.parseTokenSpan()
	if err != nil {
		return err
	}
	token := p.src[start:end]
	switch {
	case bytes.Equal(token, []byte("true")):
		writeString(buf, "true")
	case bytes.Equal(token, []byte("false")):
		writeString(buf, "false")
	case bytes.Equal(token, []byte("null")):
		writeString(buf, "null")
	case looksLikeNumberBytes(token):
		writeBytes(buf, token)
	default:
		writeString(buf, strconv.Quote(bytesToString(token)))
	}
	return nil
}

func (p *parser) writeJSONObject(buf *jsonBytes, prefix, indent string, depth int) error {
	p.pos++
	members := jsonMembers{}
	for {
		p.skipWhitespace()
		if p.pos == len(p.src) {
			return p.errorf("expected }")
		}
		if p.src[p.pos] == '}' {
			p.pos++
			return p.writeJSONObjectMembers(buf, members.Values, prefix, indent, depth)
		}

		key, err := p.parseKeyCurrent()
		if err != nil {
			return err
		}

		p.skipWhitespace()
		var value jsonBytes
		if err := p.writeJSONValueCurrent(&value, prefix, indent, depth+1); err != nil {
			return err
		}
		members.Set(key, value)
		p.skipSeparators()
	}
}

func (p *parser) writeJSONObjectMembers(buf *jsonBytes, members []jsonMember, prefix, indent string, depth int) error {
	if len(members) == 0 {
		writeString(buf, "{}")
		return nil
	}
	sort.Slice(members, func(i, j int) bool { return members[i].Key < members[j].Key })
	writeByte(buf, '{')
	if indent == "" {
		for i, member := range members {
			if i > 0 {
				writeByte(buf, ',')
			}
			writeString(buf, strconv.Quote(member.Key))
			writeByte(buf, ':')
			writeBytes(buf, member.Value)
		}
		writeByte(buf, '}')
		return nil
	}
	writeByte(buf, '\n')
	for i, member := range members {
		writeJSONIndent(buf, prefix, indent, depth+1)
		writeString(buf, strconv.Quote(member.Key))
		writeString(buf, ": ")
		writeBytes(buf, member.Value)
		if i+1 < len(members) {
			writeByte(buf, ',')
		}
		writeByte(buf, '\n')
	}
	writeJSONIndent(buf, prefix, indent, depth)
	writeByte(buf, '}')
	return nil
}

func (p *parser) writeJSONArray(buf *jsonBytes, prefix, indent string, depth int) error {
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
			if err := p.writeJSONValueCurrent(buf, prefix, indent, depth+1); err != nil {
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
		if err := p.writeJSONValueCurrent(buf, prefix, indent, depth+1); err != nil {
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

func (m *jsonMembers) Set(key string, value []byte) {
	if m.Index != nil {
		if idx, ok := m.Index[key]; ok {
			m.Values[idx].Value = value
			return
		}
	} else if len(m.Values) > 0 {
		for i, member := range m.Values {
			if member.Key == key {
				m.Values[i].Value = value
				return
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
	m.Values = append(m.Values, jsonMember{Key: key, Value: value})
}
