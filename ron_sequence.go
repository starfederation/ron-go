package ron

import (
	"bufio"
	"fmt"
	"io"
)

// RonSequenceEncoder writes RS-prefixed, LF-terminated RON values.
// Records default to a 1 MiB size limit and 100 levels of nesting.
type RonSequenceEncoder struct {
	writer   io.Writer
	settings streamSettings
}

// NewRonSequenceEncoder returns an encoder for application/ron-seq streams.
func NewRonSequenceEncoder(writer io.Writer, options ...Option) *RonSequenceEncoder {
	return &RonSequenceEncoder{
		writer:   writer,
		settings: newStreamSettings(options),
	}
}

// Encode writes one RS-prefixed RON value followed by LF.
func (e *RonSequenceEncoder) Encode(value any) error {
	record, err := Marshal(value, e.settings.options...)
	if err != nil {
		return err
	}
	if err := validateRonStreamRecord(record, e.settings); err != nil {
		return err
	}

	framed := make([]byte, len(record)+2)
	framed[0] = 0x1e
	copy(framed[1:], record)
	framed[len(framed)-1] = '\n'
	return writeStreamBytes(e.writer, framed)
}

// RonSequenceDecoder reads RS-prefixed RON values and recovers at each RS.
// Invalid elements are consumed, so callers should continue after non-EOF errors.
type RonSequenceDecoder struct {
	reader   *bufio.Reader
	settings streamSettings
	started  bool
	eof      bool
}

// NewRonSequenceDecoder returns a decoder for application/ron-seq streams.
func NewRonSequenceDecoder(reader io.Reader, options ...Option) *RonSequenceDecoder {
	return &RonSequenceDecoder{
		reader:   bufio.NewReader(reader),
		settings: newStreamSettings(options),
	}
}

// Decode reads the next valid RON sequence value into value using encoding/json semantics.
func (d *RonSequenceDecoder) Decode(value any) error {
	if d.eof {
		return io.EOF
	}

	readLimit := d.settings.maxRecordSize
	if readLimit < int(^uint(0)>>1) {
		readLimit++
	}
	if !d.started {
		preamble, foundRecordSeparator, err := readBoundedUntil(d.reader, 0x1e, readLimit)
		d.started = foundRecordSeparator
		d.eof = !foundRecordSeparator
		if err != nil {
			return err
		}
		if !foundRecordSeparator {
			if len(preamble) == 0 {
				return io.EOF
			}
			return ErrRonSequencePreamble
		}
		if len(preamble) > 0 {
			return ErrRonSequencePreamble
		}
	}

	for {
		candidate, foundRecordSeparator, err := readBoundedUntil(d.reader, 0x1e, readLimit)
		if !foundRecordSeparator {
			d.eof = true
		}
		if err != nil {
			return err
		}
		if len(candidate) == 0 {
			if d.eof {
				return io.EOF
			}
			continue
		}

		record := candidate
		if record[len(record)-1] == '\n' {
			record = record[:len(record)-1]
		} else {
			p := parser{src: record}
			p.skipSpace()
			if p.pos == len(record) {
				return ErrTruncatedRonSequence
			}
			switch record[p.pos] {
			case '{', '[', '\'', '"':
			default:
				return ErrTruncatedRonSequence
			}
		}
		if len(record) > d.settings.maxRecordSize {
			return fmt.Errorf("%w: limit %d bytes", ErrRecordTooLarge, d.settings.maxRecordSize)
		}
		return decodeRonStreamValue(record, value, d.settings)
	}
}
