package ron

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/cockroachdb/apd/v3"
	"github.com/google/uuid"
)

const (
	// VocabularyCoreV1 is the RON core typed vocabulary URI.
	VocabularyCoreV1 = "https://ron.dev/vocab/core/v1"
)

// UUID is a core vocabulary #uid value.
type UUID = uuid.UUID

// Decimal is a core vocabulary #dec arbitrary-precision decimal.
type Decimal = apd.Decimal

// Bytes is a core vocabulary #b64 value decoded from base64url without padding.
type Bytes []byte

// SHA256 is a core vocabulary #sha256 value.
type SHA256 [32]byte

// EntityRef is a core vocabulary # value for integer or string entity references.
type EntityRef struct {
	Value any
}

// OpaqueTag is a core vocabulary #tag value with implementation-defined payload.
type OpaqueTag struct {
	Tag     any
	Payload any
}

func (opts optionState) isCoreTag(tag string) bool {
	if _, ok := opts.vocabularies[VocabularyCoreV1]; !ok {
		return false
	}
	switch tag {
	case "#uid", "#url", "#dec", "#b64", "#sha256", "#", "#tag":
		return true
	default:
		return false
	}
}

func (opts optionState) parseCorePayload(tag string, payload any) (any, error) {
	switch tag {
	case "#uid":
		value, ok := payload.(string)
		if !ok {
			return nil, newError("invalid #uid payload")
		}
		id, err := uuid.Parse(value)
		if err != nil || id.String() != value {
			return nil, newError("invalid #uid payload")
		}
		return id, nil
	case "#url":
		value, ok := payload.(string)
		if !ok || value == "" || strings.ContainsAny(value, " \t\r\n") {
			return nil, newError("invalid #url payload")
		}
		parsed, err := url.Parse(value)
		if err != nil || !parsed.IsAbs() || parsed.Scheme == "" {
			return nil, newError("invalid #url payload")
		}
		return parsed, nil
	case "#dec":
		value, ok := payload.(string)
		if !ok || value == "" {
			return nil, newError("invalid #dec payload")
		}
		pos := 0
		negative := false
		if value[pos] == '-' {
			negative = true
			pos++
			if pos == len(value) {
				return nil, newError("invalid #dec payload")
			}
		}
		if value[pos] == '0' {
			pos++
			if negative && pos == len(value) {
				return nil, newError("invalid #dec payload")
			}
		} else {
			if value[pos] < '1' || value[pos] > '9' {
				return nil, newError("invalid #dec payload")
			}
			for pos < len(value) && value[pos] >= '0' && value[pos] <= '9' {
				pos++
			}
		}
		if pos < len(value) {
			if value[pos] != '.' {
				return nil, newError("invalid #dec payload")
			}
			pos++
			if pos == len(value) {
				return nil, newError("invalid #dec payload")
			}
			fractionStart := pos
			for pos < len(value) && value[pos] >= '0' && value[pos] <= '9' {
				pos++
			}
			if pos != len(value) || pos == fractionStart || value[len(value)-1] == '0' {
				return nil, newError("invalid #dec payload")
			}
		}
		var decimal Decimal
		if _, _, err := decimal.SetString(value); err != nil {
			return nil, newError("invalid #dec payload")
		}
		return &decimal, nil
	case "#b64":
		value, ok := payload.(string)
		if !ok || strings.Contains(value, "=") {
			return nil, newError("invalid #b64 payload")
		}
		decoded, err := base64.RawURLEncoding.DecodeString(value)
		if err != nil {
			return nil, newError("invalid #b64 payload")
		}
		return Bytes(decoded), nil
	case "#sha256":
		value, ok := payload.(string)
		if !ok || len(value) != 64 {
			return nil, newError("invalid #sha256 payload")
		}
		for i := range value {
			if !isLowerHexByte(value[i]) {
				return nil, newError("invalid #sha256 payload")
			}
		}
		decoded, err := hex.DecodeString(value)
		if err != nil {
			return nil, newError("invalid #sha256 payload")
		}
		var hash SHA256
		copy(hash[:], decoded)
		return hash, nil
	case "#":
		var literal string
		switch value := payload.(type) {
		case string:
			return EntityRef{Value: payload}, nil
		case json.Number:
			literal = value.String()
		case ronNumber:
			literal = string(value)
		case int64, uint64:
			return EntityRef{Value: payload}, nil
		default:
			return nil, newError("invalid # payload")
		}
		if literal == "" {
			return nil, newError("invalid # payload")
		}
		pos := 0
		if literal[pos] == '-' {
			pos++
			if pos == len(literal) {
				return nil, newError("invalid # payload")
			}
		}
		if literal[pos] == '0' {
			if pos+1 != len(literal) {
				return nil, newError("invalid # payload")
			}
		} else {
			if literal[pos] < '1' || literal[pos] > '9' {
				return nil, newError("invalid # payload")
			}
			for pos++; pos < len(literal); pos++ {
				if literal[pos] < '0' || literal[pos] > '9' {
					return nil, newError("invalid # payload")
				}
			}
		}
		return EntityRef{Value: payload}, nil
	case "#tag":
		array, ok := payload.([]any)
		if !ok || len(array) != 2 {
			return nil, newError("invalid #tag payload")
		}
		parsed, err := opts.parseVocabularyValue(array[1])
		if err != nil {
			return nil, err
		}
		return OpaqueTag{
			Tag:     array[0],
			Payload: parsed,
		}, nil
	default:
		return nil, newError("unsupported core tag")
	}
}

func coreTaggedMember(value any) (objectMember, bool) {
	switch value := value.(type) {
	case uuid.UUID:
		return objectMember{
			Key:   "#uid",
			Value: value.String(),
		}, true
	case *url.URL:
		if value == nil {
			return objectMember{}, false
		}
		return objectMember{
			Key:   "#url",
			Value: value.String(),
		}, true
	case Decimal:
		return objectMember{
			Key:   "#dec",
			Value: canonicalDecimalString(&value),
		}, true
	case *Decimal:
		if value == nil {
			return objectMember{}, false
		}
		return objectMember{
			Key:   "#dec",
			Value: canonicalDecimalString(value),
		}, true
	case Bytes:
		return objectMember{
			Key:   "#b64",
			Value: base64.RawURLEncoding.EncodeToString(value),
		}, true
	case SHA256:
		return objectMember{
			Key:   "#sha256",
			Value: hex.EncodeToString(value[:]),
		}, true
	case EntityRef:
		return objectMember{
			Key:   "#",
			Value: value.Value,
		}, true
	case OpaqueTag:
		return objectMember{
			Key: "#tag",
			Value: []any{
				value.Tag,
				value.Payload,
			},
		}, true
	default:
		return objectMember{}, false
	}
}

func canonicalDecimalString(value *Decimal) string {
	var reduced Decimal
	reduced.Reduce(value)
	return reduced.Text('f')
}

func isLowerHexByte(value byte) bool {
	return (value >= '0' && value <= '9') || (value >= 'a' && value <= 'f')
}
