package ron

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
)

const (
	// VocabularyCoreV1 is the RON core typed vocabulary URI.
	VocabularyCoreV1 = "https://ron.dev/vocab/core/v1"
)

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

func (opts optionState) validateCorePayload(tag string, payload any) error {
	switch tag {
	case "#uid":
		value, ok := payload.(string)
		if !ok || !isLowerUUID(value) {
			return newError("invalid #uid payload")
		}
	case "#url":
		value, ok := payload.(string)
		if !ok || !isAbsoluteURL(value) {
			return newError("invalid #url payload")
		}
	case "#dec":
		value, ok := payload.(string)
		if !ok || !isCanonicalDecimal(value) {
			return newError("invalid #dec payload")
		}
	case "#b64":
		value, ok := payload.(string)
		if !ok || !isBase64URLNoPadding(value) {
			return newError("invalid #b64 payload")
		}
	case "#sha256":
		value, ok := payload.(string)
		if !ok || !isLowerHex(value, 64) {
			return newError("invalid #sha256 payload")
		}
	case "#":
		if !isEntityRefPayload(payload) {
			return newError("invalid # payload")
		}
	case "#tag":
		array, ok := payload.([]any)
		if !ok || len(array) != 2 {
			return newError("invalid #tag payload")
		}
		return opts.validateVocabularyValue(array[1])
	default:
		return newError("unsupported core tag")
	}
	return nil
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
