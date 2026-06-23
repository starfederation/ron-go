package ron

import (
	"strconv"
	"time"
)

const (
	// VocabularyTimeV1 is the RON time typed vocabulary URI.
	VocabularyTimeV1 = "https://ron.dev/vocab/time/v1"
)

// Instant is a time vocabulary #utc value.
type Instant = time.Time

// Duration is a time vocabulary #dur value.
type Duration = time.Duration

func (opts optionState) isTimeTag(tag string) bool {
	if _, ok := opts.vocabularies[VocabularyTimeV1]; !ok {
		return false
	}
	switch tag {
	case "#utc", "#dur":
		return true
	default:
		return false
	}
}

func (opts optionState) parseTimePayload(tag string, payload any) (any, error) {
	value, ok := payload.(string)
	if !ok {
		return nil, newError("invalid time payload")
	}
	switch tag {
	case "#utc":
		instant, err := time.Parse(time.RFC3339Nano, value)
		if err != nil || instant.Location() != time.UTC || instant.Format(time.RFC3339Nano) != value {
			return nil, newError("invalid #utc payload")
		}
		return instant, nil
	case "#dur":
		if value == "" {
			return nil, newError("invalid #dur payload")
		}
		negative := false
		if value[0] == '-' {
			negative = true
			value = value[1:]
			if value == "" {
				return nil, newError("invalid #dur payload")
			}
		}
		if len(value) < 2 || value[0] != 'P' {
			return nil, newError("invalid #dur payload")
		}

		pos := 1
		total := time.Duration(0)
		sawComponent := false
		if numberStart := pos; pos < len(value) && value[pos] >= '0' && value[pos] <= '9' {
			for pos < len(value) && value[pos] >= '0' && value[pos] <= '9' {
				pos++
			}
			if pos < len(value) && value[pos] == 'D' {
				days, err := strconv.ParseInt(value[numberStart:pos], 10, 64)
				if err != nil {
					return nil, err
				}
				total += time.Duration(days) * 24 * time.Hour
				sawComponent = true
				pos++
			} else {
				pos = numberStart
			}
		}
		if pos == len(value) {
			if !sawComponent {
				return nil, newError("invalid #dur payload")
			}
			if negative {
				return -total, nil
			}
			return total, nil
		}
		if value[pos] != 'T' {
			return nil, newError("invalid #dur payload")
		}
		pos++
		if pos == len(value) {
			return nil, newError("invalid #dur payload")
		}

		seenHour := false
		seenMinute := false
		seenSecond := false
		for pos < len(value) {
			start := pos
			for pos < len(value) && value[pos] >= '0' && value[pos] <= '9' {
				pos++
			}
			if start == pos {
				return nil, newError("invalid #dur payload")
			}
			integer := value[start:pos]
			fraction := ""
			if pos < len(value) && value[pos] == '.' {
				pos++
				fractionStart := pos
				for pos < len(value) && value[pos] >= '0' && value[pos] <= '9' {
					pos++
				}
				if fractionStart == pos || pos-fractionStart > 9 {
					return nil, newError("invalid #dur payload")
				}
				fraction = value[fractionStart:pos]
			}
			if pos == len(value) {
				return nil, newError("invalid #dur payload")
			}
			unit := value[pos]
			pos++
			number, err := strconv.ParseInt(integer, 10, 64)
			if err != nil {
				return nil, err
			}
			switch unit {
			case 'H':
				if fraction != "" || seenHour || seenMinute || seenSecond {
					return nil, newError("invalid #dur payload")
				}
				total += time.Duration(number) * time.Hour
				seenHour = true
			case 'M':
				if fraction != "" || seenMinute || seenSecond {
					return nil, newError("invalid #dur payload")
				}
				total += time.Duration(number) * time.Minute
				seenMinute = true
			case 'S':
				if seenSecond {
					return nil, newError("invalid #dur payload")
				}
				total += time.Duration(number) * time.Second
				if fraction != "" {
					for len(fraction) < 9 {
						fraction += "0"
					}
					nanos, err := strconv.ParseInt(fraction, 10, 64)
					if err != nil {
						return nil, err
					}
					total += time.Duration(nanos)
				}
				seenSecond = true
			default:
				return nil, newError("invalid #dur payload")
			}
			sawComponent = true
		}
		if !sawComponent {
			return nil, newError("invalid #dur payload")
		}
		if negative {
			return -total, nil
		}
		return total, nil
	default:
		return nil, newError("unsupported time tag")
	}
}
