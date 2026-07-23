package ron

import (
	"bufio"
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
	maxRecordSize    int
	ignoreEmptyLines bool
}

func newStreamSettings(options []Option) streamSettings {
	const (
		defaultMaxRecordSize   = 1 << 20
		defaultMaxNestingDepth = 100
	)

	copiedOptions := append([]Option(nil), options...)
	state := optionState{}
	for _, option := range copiedOptions {
		option(&state)
	}

	maxRecordSize := state.maxRecordSize
	if maxRecordSize <= 0 {
		maxRecordSize = defaultMaxRecordSize
	}
	if state.maxNestingDepth <= 0 {
		copiedOptions = append(copiedOptions, MaxNestingDepth(defaultMaxNestingDepth))
	}
	return streamSettings{
		options:          copiedOptions,
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

func decodeRonStreamValue(record []byte, value any, settings streamSettings) error {
	jsonBody, err := ToJSON(record, settings.options...)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonBody, value)
}

func validateRonStreamRecord(record []byte, settings streamSettings) error {
	if len(record) > settings.maxRecordSize {
		return fmt.Errorf("%w: limit %d bytes", ErrRecordTooLarge, settings.maxRecordSize)
	}
	_, err := ToJSON(record, settings.options...)
	return err
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
