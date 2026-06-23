package ron

import (
	"bytes"
	"sort"
	"sync"
	"unicode/utf8"
)

type jsonMember struct {
	Key        string
	ValueStart int
	ValueEnd   int
}

type jsonScratchState struct {
	buffers []bytes.Buffer
	members [][]jsonMember
}

var jsonScratchPool = sync.Pool{
	New: func() any {
		return new(jsonScratchState)
	},
}

func getJSONScratchState() *jsonScratchState {
	return jsonScratchPool.Get().(*jsonScratchState)
}

func putJSONScratchState(state *jsonScratchState) {
	for i := range state.buffers {
		if state.buffers[i].Cap() > 1<<20 {
			state.buffers[i] = bytes.Buffer{}
		} else {
			state.buffers[i].Reset()
		}
	}
	for i := range state.members {
		if cap(state.members[i]) > 1024 {
			state.members[i] = nil
		} else {
			state.members[i] = state.members[i][:0]
		}
	}
	jsonScratchPool.Put(state)
}

type jsonMemberSorter []jsonMember

func (s jsonMemberSorter) Len() int {
	return len(s)
}

func (s jsonMemberSorter) Less(i, j int) bool {
	return rfc8785StringLess(s[i].Key, s[j].Key)
}

func (s jsonMemberSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (p *parser) writeJSONValue(buf *bytes.Buffer, prefix, indent string, depth int, canonical bool) error {
	p.skipSpace()
	return p.writeJSONValueCurrent(buf, prefix, indent, depth, canonical)
}

func (p *parser) nextJSONScratch() *bytes.Buffer {
	idx := p.jsonScratchDepth
	p.jsonScratchDepth++
	if idx == len(p.jsonScratch) {
		p.jsonScratch = append(p.jsonScratch, bytes.Buffer{})
	}
	p.jsonScratch[idx].Reset()
	return &p.jsonScratch[idx]
}

func (p *parser) releaseJSONScratch() {
	p.jsonScratchDepth--
}

func (p *parser) nextJSONMembers() (int, []jsonMember) {
	idx := p.jsonMemberScratchDepth
	p.jsonMemberScratchDepth++
	if idx == len(p.jsonMemberScratch) {
		p.jsonMemberScratch = append(p.jsonMemberScratch, make([]jsonMember, 0, 8))
	}
	return idx, p.jsonMemberScratch[idx][:0]
}

func (p *parser) releaseJSONMembers(idx int, members []jsonMember) {
	p.jsonMemberScratch[idx] = members[:0]
	p.jsonMemberScratchDepth--
}

func writeJSONQuoted(buf *bytes.Buffer, value string) {
	const hex = "0123456789abcdef"

	asciiSafe := true
	for i := 0; i < len(value); i++ {
		b := value[i]
		if b < 0x20 || b == '\\' || b == '"' || b >= utf8.RuneSelf {
			asciiSafe = false
			break
		}
	}
	if asciiSafe {
		buf.WriteByte('"')
		buf.WriteString(value)
		buf.WriteByte('"')
		return
	}

	buf.WriteByte('"')
	start := 0
	for i := 0; i < len(value); {
		b := value[i]
		if b < utf8.RuneSelf {
			if b >= 0x20 && b != '\\' && b != '"' {
				i++
				continue
			}

			buf.WriteString(value[start:i])
			switch b {
			case '\\', '"':
				buf.WriteByte('\\')
				buf.WriteByte(b)
			case '\b':
				buf.WriteString("\\b")
			case '\f':
				buf.WriteString("\\f")
			case '\n':
				buf.WriteString("\\n")
			case '\r':
				buf.WriteString("\\r")
			case '\t':
				buf.WriteString("\\t")
			default:
				buf.WriteString("\\u00")
				buf.WriteByte(hex[b>>4])
				buf.WriteByte(hex[b&0xf])
			}
			i++
			start = i
			continue
		}

		r, size := utf8.DecodeRuneInString(value[i:])
		if r == utf8.RuneError && size == 1 {
			buf.WriteString(value[start:i])
			buf.WriteString("\\ufffd")
			i++
			start = i
			continue
		}
		i += size
	}
	buf.WriteString(value[start:])
	buf.WriteByte('"')
}

func (p *parser) writeJSONValueCurrent(buf *bytes.Buffer, prefix, indent string, depth int, canonical bool) error {
	if p.pos == len(p.src) {
		return p.errorf("expected value")
	}
	switch p.src[p.pos] {
	case '{':
		p.pos++
		memberScratch, memberValues := p.nextJSONMembers()
		members := jsonMembers{Values: memberValues}
		values := p.nextJSONScratch()
		for {
			p.skipWhitespace()
			if p.pos == len(p.src) {
				p.releaseJSONScratch()
				p.releaseJSONMembers(memberScratch, members.Values)
				return p.errorf("expected }")
			}
			if p.src[p.pos] == '}' {
				p.pos++
				err := p.writeJSONObjectMembers(buf, members.Values, values.Bytes(), prefix, indent, depth, canonical && members.NeedsSort)
				p.releaseJSONScratch()
				p.releaseJSONMembers(memberScratch, members.Values)
				return err
			}

			key, err := p.parseKeyCurrent()
			if err != nil {
				p.releaseJSONScratch()
				p.releaseJSONMembers(memberScratch, members.Values)
				return err
			}

			p.skipWhitespace()
			valueStart := values.Len()
			if err := p.writeJSONValueCurrent(values, prefix, indent, depth+1, canonical); err != nil {
				p.releaseJSONScratch()
				p.releaseJSONMembers(memberScratch, members.Values)
				return err
			}
			members.Set(key, valueStart, values.Len())
			p.skipSeparators()
		}
	case '[':
		p.pos++
		p.skipWhitespace()
		if p.pos == len(p.src) {
			return p.errorf("expected ]")
		}
		if p.src[p.pos] == ']' {
			p.pos++
			buf.WriteString("[]")
			return nil
		}

		buf.WriteByte('[')
		if indent == "" {
			for i := 0; ; i++ {
				if i > 0 {
					buf.WriteByte(',')
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
					buf.WriteByte(']')
					return nil
				}
			}
		}

		buf.WriteByte('\n')
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
				buf.WriteByte('\n')
				writeJSONIndent(buf, prefix, indent, depth)
				buf.WriteByte(']')
				return nil
			}
			buf.WriteByte(',')
			buf.WriteByte('\n')
		}
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
		buf.WriteString("true")
	case len(token) == 5 && token[0] == 'f' && token[1] == 'a' && token[2] == 'l' && token[3] == 's' && token[4] == 'e':
		buf.WriteString("false")
	case len(token) == 4 && token[0] == 'n' && token[1] == 'u' && token[2] == 'l' && token[3] == 'l':
		buf.WriteString("null")
	case looksLikeNumberBytes(token):
		buf.Write(token)
	default:
		writeJSONQuoted(buf, bytesToString(token))
	}
	return nil
}

func (p *parser) writeJSONObjectMembers(buf *bytes.Buffer, members []jsonMember, values []byte, prefix, indent string, depth int, sortMembers bool) error {
	if len(members) == 0 {
		buf.WriteString("{}")
		return nil
	}
	if sortMembers {
		sort.Sort(jsonMemberSorter(members))
	}
	buf.WriteByte('{')
	if indent == "" {
		for i, member := range members {
			if i > 0 {
				buf.WriteByte(',')
			}
			writeJSONQuoted(buf, member.Key)
			buf.WriteByte(':')
			buf.Write(values[member.ValueStart:member.ValueEnd])
		}
		buf.WriteByte('}')
		return nil
	}
	buf.WriteByte('\n')
	for i, member := range members {
		writeJSONIndent(buf, prefix, indent, depth+1)
		writeJSONQuoted(buf, member.Key)
		buf.WriteString(": ")
		buf.Write(values[member.ValueStart:member.ValueEnd])
		if i+1 < len(members) {
			buf.WriteByte(',')
		}
		buf.WriteByte('\n')
	}
	writeJSONIndent(buf, prefix, indent, depth)
	buf.WriteByte('}')
	return nil
}

type jsonMembers struct {
	Values    []jsonMember
	Index     map[string]int
	NeedsSort bool
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
	if len(m.Values) > 0 && !rfc8785StringLess(m.Values[len(m.Values)-1].Key, key) {
		m.NeedsSort = true
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

func writeJSONIndent(buf *bytes.Buffer, prefix, indent string, depth int) {
	buf.WriteString(prefix)
	for i := 0; i < depth; i++ {
		buf.WriteString(indent)
	}
}
