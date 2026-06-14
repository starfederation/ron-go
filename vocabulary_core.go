package ron

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

const (
	// VocabularyCoreV1 is the RON core typed vocabulary URI.
	VocabularyCoreV1 = "https://ron.dev/vocab/core/v1"
)

// UUID is a core vocabulary #uid value.
type UUID = uuid.UUID

// Decimal is a core vocabulary #dec value that preserves canonical decimal text.
type Decimal string

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
		if !ok || !isLowerUUID(value) {
			return nil, newError("invalid #uid payload")
		}
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, newError("invalid #uid payload")
		}
		return id, nil
	case "#url":
		value, ok := payload.(string)
		if !ok || !isAbsoluteURL(value) {
			return nil, newError("invalid #url payload")
		}
		parsed, err := url.Parse(value)
		if err != nil {
			return nil, newError("invalid #url payload")
		}
		return parsed, nil
	case "#dec":
		value, ok := payload.(string)
		if !ok || !isCanonicalDecimal(value) {
			return nil, newError("invalid #dec payload")
		}
		return Decimal(value), nil
	case "#b64":
		value, ok := payload.(string)
		if !ok || !isBase64URLNoPadding(value) {
			return nil, newError("invalid #b64 payload")
		}
		decoded, err := base64.RawURLEncoding.DecodeString(value)
		if err != nil {
			return nil, newError("invalid #b64 payload")
		}
		return Bytes(decoded), nil
	case "#sha256":
		value, ok := payload.(string)
		if !ok || !isLowerHex(value, 64) {
			return nil, newError("invalid #sha256 payload")
		}
		decoded, err := hex.DecodeString(value)
		if err != nil {
			return nil, newError("invalid #sha256 payload")
		}
		var hash SHA256
		copy(hash[:], decoded)
		return hash, nil
	case "#":
		if !isEntityRefPayload(payload) {
			return nil, newError("invalid # payload")
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
			Value: string(value),
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

func isLowerUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i := range value {
		switch i {
		case 8, 13, 18, 23:
			if value[i] != '-' {
				return false
			}
		default:
			if !isLowerHexByte(value[i]) {
				return false
			}
		}
	}
	return true
}

func isAbsoluteURL(value string) bool {
	if value == "" || strings.ContainsAny(value, " \t\r\n") {
		return false
	}
	u, err := url.Parse(value)
	return err == nil && u.IsAbs() && u.Scheme != ""
}

func isCanonicalDecimal(value string) bool {
	if value == "" {
		return false
	}
	pos := 0
	negative := false
	if value[pos] == '-' {
		negative = true
		pos++
		if pos == len(value) {
			return false
		}
	}
	if value[pos] == '0' {
		pos++
		if negative && pos == len(value) {
			return false
		}
	} else {
		if value[pos] < '1' || value[pos] > '9' {
			return false
		}
		for pos < len(value) && value[pos] >= '0' && value[pos] <= '9' {
			pos++
		}
	}
	if pos == len(value) {
		return true
	}
	if value[pos] != '.' {
		return false
	}
	pos++
	if pos == len(value) {
		return false
	}
	fractionStart := pos
	for pos < len(value) && value[pos] >= '0' && value[pos] <= '9' {
		pos++
	}
	if pos != len(value) || value[len(value)-1] == '0' {
		return false
	}
	return pos > fractionStart
}

func isBase64URLNoPadding(value string) bool {
	if strings.Contains(value, "=") {
		return false
	}
	_, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil
}

func isLowerHex(value string, size int) bool {
	if len(value) != size {
		return false
	}
	for i := range value {
		if !isLowerHexByte(value[i]) {
			return false
		}
	}
	return true
}

func isLowerHexByte(value byte) bool {
	return (value >= '0' && value <= '9') || (value >= 'a' && value <= 'f')
}

func isEntityRefPayload(value any) bool {
	switch value := value.(type) {
	case string:
		return true
	case json.Number:
		return isIntegerLiteral(value.String())
	case ronNumber:
		return isIntegerLiteral(string(value))
	case int64, uint64:
		return true
	default:
		return false
	}
}

func isIntegerLiteral(value string) bool {
	if value == "" {
		return false
	}
	pos := 0
	if value[pos] == '-' {
		pos++
		if pos == len(value) {
			return false
		}
	}
	if value[pos] == '0' {
		return pos+1 == len(value)
	}
	if value[pos] < '1' || value[pos] > '9' {
		return false
	}
	for pos++; pos < len(value); pos++ {
		if value[pos] < '0' || value[pos] > '9' {
			return false
		}
	}
	return true
}
