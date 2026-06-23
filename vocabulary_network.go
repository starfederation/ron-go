package ron

import "net/netip"

const (
	// VocabularyNetworkV1 is the RON network typed vocabulary URI.
	VocabularyNetworkV1 = "https://ron.dev/vocab/network/v1"
)

// IPv4 is a network vocabulary #ip4 value.
type IPv4 struct {
	Addr netip.Addr
}

// IPv6 is a network vocabulary #ip6 value.
type IPv6 struct {
	Addr netip.Addr
}

// CIDR is a network vocabulary #cdr value.
type CIDR struct {
	Prefix netip.Prefix
}

func (opts optionState) isNetworkTag(tag string) bool {
	if !opts.vocabularyEnabled(vocabularyNetwork, VocabularyNetworkV1) {
		return false
	}
	switch tag {
	case "#ip4", "#ip6", "#cdr":
		return true
	default:
		return false
	}
}

func (opts optionState) parseNetworkPayload(tag string, payload any) (any, error) {
	value, ok := payload.(string)
	if !ok {
		return nil, newError("invalid network payload")
	}
	switch tag {
	case "#ip4":
		addr, err := netip.ParseAddr(value)
		if err != nil || !addr.Is4() || addr.String() != value {
			return nil, newError("invalid #ip4 payload")
		}
		return IPv4{Addr: addr}, nil
	case "#ip6":
		addr, err := netip.ParseAddr(value)
		if err != nil || !addr.Is6() || addr.Is4In6() || addr.String() != value {
			return nil, newError("invalid #ip6 payload")
		}
		return IPv6{Addr: addr}, nil
	case "#cdr":
		prefix, err := netip.ParsePrefix(value)
		if err != nil || prefix.Masked() != prefix || prefix.String() != value {
			return nil, newError("invalid #cdr payload")
		}
		return CIDR{Prefix: prefix}, nil
	default:
		return nil, newError("unsupported network tag")
	}
}
