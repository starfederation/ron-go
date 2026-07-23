package ron

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrRecordTooLarge reports a stream record larger than MaxRecordSize.
	ErrRecordTooLarge = errors.New("ron: stream record exceeds maximum size")
	// ErrNestingTooDeep reports a value deeper than MaxNestingDepth.
	ErrNestingTooDeep = errors.New("ron: maximum nesting depth exceeded")
	// ErrEmptyNdronRecord reports an empty line when empty-line skipping is disabled.
	ErrEmptyNdronRecord = errors.New("ron: empty NDRON record")
	// ErrUnterminatedNdronRecord reports a final NDRON record without LF.
	ErrUnterminatedNdronRecord = errors.New("ron: unterminated final NDRON record")
	// ErrRonSequencePreamble reports bytes before the first RON sequence RS marker.
	ErrRonSequencePreamble = errors.New("ron: invalid RON sequence preamble")
	// ErrTruncatedRonSequence reports an LF-less element that is not self-delimited.
	ErrTruncatedRonSequence = errors.New("ron: potentially truncated RON sequence element")
)

type streamSettings struct {
	options          []Option
	jsonOptions      optionState
	maxRecordSize    int
	ignoreEmptyLines bool
}

func newStreamSettings(options []Option) streamSettings {
	const (
		defaultMaxRecordSize   = 1 << 20
		defaultMaxNestingDepth = 100
	)

	copiedOptions := append([]Option(nil), options...)
	state := toJSONOptionState(copiedOptions)
	maxRecordSize := state.maxRecordSize
	if maxRecordSize <= 0 {
		maxRecordSize = defaultMaxRecordSize
	}
	if state.maxNestingDepth <= 0 {
		copiedOptions = append(copiedOptions, MaxNestingDepth(defaultMaxNestingDepth))
		state.maxNestingDepth = defaultMaxNestingDepth
	}
	return streamSettings{
		options:          copiedOptions,
		jsonOptions:      state,
		maxRecordSize:    maxRecordSize,
		ignoreEmptyLines: state.ignoreEmptyNdronLines,
	}
}

func readBoundedUntil(reader *bufio.Reader, delimiter byte, maxSize int) ([]byte, bool, error) {
	var data []byte
	tooLarge := false
	for {
		chunk, err := reader.ReadSlice(delimiter)
		found := err == nil
		if found {
			chunk = chunk[:len(chunk)-1]
		}
		if len(data) == 0 && !tooLarge && (err == nil || err == io.EOF) {
			if len(chunk) > maxSize {
				return nil, found, ErrRecordTooLarge
			}
			return chunk, found, nil
		}
		if !tooLarge {
			if len(chunk) > maxSize-len(data) {
				tooLarge = true
				data = nil
			} else {
				data = append(data, chunk...)
			}
		}

		switch err {
		case nil:
			if tooLarge {
				return nil, true, ErrRecordTooLarge
			}
			return data, true, nil
		case bufio.ErrBufferFull:
			continue
		case io.EOF:
			if tooLarge {
				return nil, false, ErrRecordTooLarge
			}
			return data, false, nil
		default:
			return nil, false, err
		}
	}
}

func decodeRonStreamValue(record []byte, value any, settings streamSettings, buf *bytes.Buffer) error {
	buf.Reset()
	jsonBody, err := toJSONIntoState(buf, record, settings.jsonOptions)
	if err != nil {
		return err
	}
	if raw, ok := value.(*json.RawMessage); ok && raw != nil {
		*raw = append((*raw)[:0], jsonBody...)
		return nil
	}
	return json.Unmarshal(jsonBody, value)
}

func validateStreamRecordSize(record []byte, settings streamSettings) error {
	if len(record) > settings.maxRecordSize {
		return fmt.Errorf("%w: limit %d bytes", ErrRecordTooLarge, settings.maxRecordSize)
	}
	return nil
}

func writeStreamBytes(writer io.Writer, body []byte) error {
	written, err := writer.Write(body)
	if err != nil {
		return err
	}
	if written != len(body) {
		return io.ErrShortWrite
	}
	return nil
}
