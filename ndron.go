package ron

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// NdronEncoder writes one compact RON value followed by LF per Encode call.
// Records default to a 1 MiB size limit and 100 levels of nesting.
type NdronEncoder struct {
	writer   io.Writer
	settings streamSettings
}

// NewNdronEncoder returns an encoder for application/x-ndron streams.
func NewNdronEncoder(writer io.Writer, options ...Option) *NdronEncoder {
	settings := newStreamSettings(options)
	settings.options = append(settings.options, IsPretty(false))
	return &NdronEncoder{
		writer:   writer,
		settings: settings,
	}
}

// Encode writes one compact RON record followed by LF.
func (e *NdronEncoder) Encode(value any) error {
	record, err := Marshal(value, e.settings.options...)
	if err != nil {
		return err
	}
	if err := validateRonStreamRecord(record, e.settings); err != nil {
		return err
	}
	if bytes.ContainsAny(record, "\r\n") {
		return fmt.Errorf("ron: NDRON record contains a raw line ending")
	}
	return writeStreamBytes(e.writer, append(record, '\n'))
}

// NdronDecoder reads one LF- or CRLF-terminated RON value per Decode call.
// Invalid records are consumed, so callers may continue after non-EOF errors.
type NdronDecoder struct {
	reader   *bufio.Reader
	settings streamSettings
}

// NewNdronDecoder returns a decoder for application/x-ndron streams.
// Empty lines are errors unless IgnoreEmptyNdronLines(true) is supplied.
func NewNdronDecoder(reader io.Reader, options ...Option) *NdronDecoder {
	return &NdronDecoder{
		reader:   bufio.NewReader(reader),
		settings: newStreamSettings(options),
	}
}

// Decode reads the next NDRON value into value using encoding/json semantics.
func (d *NdronDecoder) Decode(value any) error {
	for {
		readLimit := d.settings.maxRecordSize
		if readLimit < int(^uint(0)>>1) {
			readLimit++
		}
		record, terminated, err := readBoundedUntil(d.reader, '\n', readLimit)
		if err != nil {
			return err
		}
		if !terminated {
			if len(record) == 0 {
				return io.EOF
			}
			return ErrUnterminatedNdronRecord
		}
		if len(record) > 0 && record[len(record)-1] == '\r' {
			record = record[:len(record)-1]
		}
		if len(record) > d.settings.maxRecordSize {
			return fmt.Errorf("%w: limit %d bytes", ErrRecordTooLarge, d.settings.maxRecordSize)
		}
		if bytes.IndexByte(record, '\r') >= 0 {
			return fmt.Errorf("ron: NDRON record contains a raw carriage return")
		}
		if len(record) == 0 {
			if d.settings.ignoreEmptyLines {
				continue
			}
			return ErrEmptyNdronRecord
		}
		return decodeRonStreamValue(record, value, d.settings)
	}
}
