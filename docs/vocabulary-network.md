# Network vocabulary

URI: `https://ron.dev/vocab/network/v1`

Network vocabulary support is enabled by default. It can also be named explicitly in a vocabulary profile:

```go
ron.EnableVocabularies(ron.VocabularyNetworkV1)
```

Vocabulary-aware parsing validates matching network tags and maps them to Go stdlib `net/netip` values. Rendering native network values emits the matching tagged RON form.

| Tag | Meaning | Go type | External library |
| --- | --- | --- | --- |
| `#ip4` | IPv4 address | `ron.IPv4` wrapping `netip.Addr` | Go stdlib `net/netip` |
| `#ip6` | IPv6 address | `ron.IPv6` wrapping `netip.Addr` | Go stdlib `net/netip` |
| `#cdr` | CIDR prefix | `ron.CIDR` wrapping `netip.Prefix` | Go stdlib `net/netip` |

## Type notes

### `ron.IPv4`

`ron.IPv4.Addr` must be a valid IPv4 `netip.Addr`. Rendering uses `Addr.String()`.

### `ron.IPv6`

`ron.IPv6.Addr` must be a valid IPv6 `netip.Addr`, excluding IPv4-mapped IPv6 addresses. Rendering uses RFC 5952 text from `Addr.String()`.

### `ron.CIDR`

`ron.CIDR.Prefix` must be masked. Unmasked prefixes such as `192.0.2.1/24` are rejected.
