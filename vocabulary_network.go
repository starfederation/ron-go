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
	if _, ok := opts.vocabularies[VocabularyNetworkV1]; !ok {
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

func networkTaggedMember(value any) (objectMember, bool) {
	switch value := value.(type) {
	case IPv4:
		if !value.Addr.Is4() {
			return objectMember{}, false
		}
		return objectMember{
			Key:   "#ip4",
			Value: value.Addr.String(),
		}, true
	case IPv6:
		if !value.Addr.Is6() || value.Addr.Is4In6() {
			return objectMember{}, false
		}
		return objectMember{
			Key:   "#ip6",
			Value: value.Addr.String(),
		}, true
	case CIDR:
		if value.Prefix != value.Prefix.Masked() {
			return objectMember{}, false
		}
		return objectMember{
			Key:   "#cdr",
			Value: value.Prefix.String(),
		}, true
	default:
		return objectMember{}, false
	}
}
