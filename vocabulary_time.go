package ron

import (
	"strconv"
	"strings"
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
		duration, err := parseDayTimeDuration(value)
		if err != nil {
			return nil, newError("invalid #dur payload")
		}
		return duration, nil
	default:
		return nil, newError("unsupported time tag")
	}
}

func timeTaggedMember(value any) (objectMember, bool) {
	switch value := value.(type) {
	case time.Time:
		if value.Location() != time.UTC {
			value = value.UTC()
		}
		return objectMember{
			Key:   "#utc",
			Value: value.Format(time.RFC3339Nano),
		}, true
	case time.Duration:
		return objectMember{
			Key:   "#dur",
			Value: formatDayTimeDuration(value),
		}, true
	default:
		return objectMember{}, false
	}
}

func parseDayTimeDuration(value string) (time.Duration, error) {
	if value == "" {
		return 0, newError("invalid #dur payload")
	}
	negative := false
	if value[0] == '-' {
		negative = true
		value = value[1:]
		if value == "" {
			return 0, newError("invalid #dur payload")
		}
	}
	if len(value) < 2 || value[0] != 'P' {
		return 0, newError("invalid #dur payload")
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
				return 0, err
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
			return 0, newError("invalid #dur payload")
		}
		if negative {
			return -total, nil
		}
		return total, nil
	}
	if value[pos] != 'T' {
		return 0, newError("invalid #dur payload")
	}
	pos++
	if pos == len(value) {
		return 0, newError("invalid #dur payload")
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
			return 0, newError("invalid #dur payload")
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
				return 0, newError("invalid #dur payload")
			}
			fraction = value[fractionStart:pos]
		}
		if pos == len(value) {
			return 0, newError("invalid #dur payload")
		}
		unit := value[pos]
		pos++
		number, err := strconv.ParseInt(integer, 10, 64)
		if err != nil {
			return 0, err
		}
		switch unit {
		case 'H':
			if fraction != "" || seenHour || seenMinute || seenSecond {
				return 0, newError("invalid #dur payload")
			}
			total += time.Duration(number) * time.Hour
			seenHour = true
		case 'M':
			if fraction != "" || seenMinute || seenSecond {
				return 0, newError("invalid #dur payload")
			}
			total += time.Duration(number) * time.Minute
			seenMinute = true
		case 'S':
			if seenSecond {
				return 0, newError("invalid #dur payload")
			}
			total += time.Duration(number) * time.Second
			if fraction != "" {
				for len(fraction) < 9 {
					fraction += "0"
				}
				nanos, err := strconv.ParseInt(fraction, 10, 64)
				if err != nil {
					return 0, err
				}
				total += time.Duration(nanos)
			}
			seenSecond = true
		default:
			return 0, newError("invalid #dur payload")
		}
		sawComponent = true
	}
	if !sawComponent {
		return 0, newError("invalid #dur payload")
	}
	if negative {
		return -total, nil
	}
	return total, nil
}

func formatDayTimeDuration(value time.Duration) string {
	if value == 0 {
		return "PT0S"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	days := value / (24 * time.Hour)
	value -= days * 24 * time.Hour
	hours := value / time.Hour
	value -= hours * time.Hour
	minutes := value / time.Minute
	value -= minutes * time.Minute
	seconds := value / time.Second
	nanos := value - seconds*time.Second

	var b strings.Builder
	if negative {
		b.WriteByte('-')
	}
	b.WriteByte('P')
	if days > 0 {
		b.WriteString(strconv.FormatInt(int64(days), 10))
		b.WriteByte('D')
	}
	if hours > 0 || minutes > 0 || seconds > 0 || nanos > 0 || days == 0 {
		b.WriteByte('T')
		if hours > 0 {
			b.WriteString(strconv.FormatInt(int64(hours), 10))
			b.WriteByte('H')
		}
		if minutes > 0 {
			b.WriteString(strconv.FormatInt(int64(minutes), 10))
			b.WriteByte('M')
		}
		if seconds > 0 || nanos > 0 || (hours == 0 && minutes == 0 && days == 0) {
			b.WriteString(strconv.FormatInt(int64(seconds), 10))
			if nanos > 0 {
				fraction := strconv.FormatInt(int64(nanos)+1_000_000_000, 10)[1:]
				fraction = strings.TrimRight(fraction, "0")
				b.WriteByte('.')
				b.WriteString(fraction)
			}
			b.WriteByte('S')
		}
	}
	return b.String()
}
