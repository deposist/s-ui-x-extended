package service

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
)

// AmneziaWG obfuscation parameter validation and generation.
//
// Reserved header values are the vanilla WireGuard message types from
// amnezia-vpn/amneziawg-go/device/noise-protocol.go:
//
//	MessageUnknownType     = 0
//	MessageInitiationType  = 1
//	MessageResponseType    = 2
//	MessageCookieReplyType = 3
//	MessageTransportType   = 4
//
// The official Amnezia client generator (awgInstaller.cpp) picks headers from
// [5, INT32_MAX); AWG 2.0 additionally requires the four H ranges to be
// pairwise non-overlapping because the receive path disambiguates packet
// types by matching the 32-bit header against the configured ranges.
const (
	awgHeaderReservedMax = 4 // 0-4 are reserved (vanilla WireGuard message types)
	awgHeaderMin         = awgHeaderReservedMax + 1
	awgHeaderMax         = math.MaxUint32
	awgGenHeaderMax      = math.MaxInt32 // official client generates within int32 range

	awgMaxJunkCount = 128  // installer flag bound (--jc)
	awgMaxJunkSize  = 1280 // installer flag bound (--jmin/--jmax)
	awgMaxPadding   = 1280

	// Base wire sizes from amneziawg-go device/noise-protocol.go. Padded
	// handshake/cookie/transport packets must stay pairwise distinguishable
	// by size or the connection silently fails to come up.
	awgMessageInitiationSize  = 148
	awgMessageResponseSize    = 92
	awgMessageCookieReplySize = 64
	awgMessageTransportSize   = 32
)

type awgHeaderRange struct {
	from uint64
	to   uint64
}

func (r awgHeaderRange) overlaps(other awgHeaderRange) bool {
	return r.from <= other.to && other.from <= r.to
}

func (r awgHeaderRange) String() string {
	if r.from == r.to {
		return strconv.FormatUint(r.from, 10)
	}
	return strconv.FormatUint(r.from, 10) + "-" + strconv.FormatUint(r.to, 10)
}

// parseAWGHeaderValue parses one H1-H4 option. The panel stores headers as
// `any`: a JSON number for a single value or a string for a "from-to" range
// (sing-box badoption.Range syntax). A nil result with nil error means the
// field is absent.
func parseAWGHeaderValue(field string, value any) (*awgHeaderRange, error) {
	switch v := value.(type) {
	case nil:
		return nil, nil
	case float64:
		if v != math.Trunc(v) || v < 0 || v > awgHeaderMax {
			return nil, fmt.Errorf("amnezia %s must be an integer between 0 and %d", field, uint64(awgHeaderMax))
		}
		n := uint64(v)
		return &awgHeaderRange{from: n, to: n}, nil
	case int:
		if v < 0 || uint64(v) > awgHeaderMax {
			return nil, fmt.Errorf("amnezia %s must be an integer between 0 and %d", field, uint64(awgHeaderMax))
		}
		return &awgHeaderRange{from: uint64(v), to: uint64(v)}, nil
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return nil, nil
		}
		// Mirror sing-box badoption.Range parsing: split on "-", no inner
		// trimming, one or two parts. Anything the core would reject must be
		// rejected here or the saved endpoint would fail to start.
		parts := strings.Split(trimmed, "-")
		if len(parts) > 2 {
			return nil, fmt.Errorf("amnezia %s must be a single number or a from-to range", field)
		}
		from, err := strconv.ParseUint(parts[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("amnezia %s must be a single number or a from-to range", field)
		}
		to := from
		if len(parts) == 2 {
			to, err = strconv.ParseUint(parts[1], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("amnezia %s must be a single number or a from-to range", field)
			}
		}
		if to < from {
			return nil, fmt.Errorf("amnezia %s range start must not exceed its end", field)
		}
		return &awgHeaderRange{from: from, to: to}, nil
	default:
		return nil, fmt.Errorf("amnezia %s must be a single number or a from-to range", field)
	}
}

type awgAmneziaObfuscation struct {
	JC   int `json:"jc"`
	JMin int `json:"jmin"`
	JMax int `json:"jmax"`
	S1   int `json:"s1"`
	S2   int `json:"s2"`
	S3   int `json:"s3"`
	S4   int `json:"s4"`
	H1   any `json:"h1"`
	H2   any `json:"h2"`
	H3   any `json:"h3"`
	H4   any `json:"h4"`
}

// ValidateAmneziaOptions rejects Amnezia obfuscation parameter combinations
// that make the tunnel silently fail to come up. It is intentionally a hard
// block on endpoint save: with incompatible parameters the handshake never
// completes and there is no error on either side.
func ValidateAmneziaOptions(options json.RawMessage) error {
	if len(options) == 0 {
		return nil
	}
	var wrapper struct {
		Amnezia *awgAmneziaObfuscation `json:"amnezia"`
	}
	if err := json.Unmarshal(options, &wrapper); err != nil {
		return fmt.Errorf("amnezia options are invalid: %v", err)
	}
	if wrapper.Amnezia == nil {
		return nil
	}
	return validateAmneziaObfuscation(*wrapper.Amnezia)
}

func validateAmneziaObfuscation(a awgAmneziaObfuscation) error {
	if a.JC < 0 || a.JC > awgMaxJunkCount {
		return fmt.Errorf("amnezia Jc must be between 0 and %d", awgMaxJunkCount)
	}
	if a.JMin < 0 || a.JMin > awgMaxJunkSize || a.JMax < 0 || a.JMax > awgMaxJunkSize {
		return fmt.Errorf("amnezia Jmin/Jmax must be between 0 and %d", awgMaxJunkSize)
	}
	if a.JMin > a.JMax {
		return fmt.Errorf("amnezia Jmin (%d) must not exceed Jmax (%d)", a.JMin, a.JMax)
	}
	for _, s := range []struct {
		name  string
		value int
	}{{"S1", a.S1}, {"S2", a.S2}, {"S3", a.S3}, {"S4", a.S4}} {
		if s.value < 0 || s.value > awgMaxPadding {
			return fmt.Errorf("amnezia %s must be between 0 and %d", s.name, awgMaxPadding)
		}
	}
	// Padded packet sizes must be pairwise distinct (official Amnezia client
	// check, awgProtocolConfig.cpp::isPacketSizeEqual). The classic special
	// case is S1 + 56 == S2: init (148+S1) and response (92+S2) packets get
	// the same size and the peer cannot tell them apart, so the handshake
	// never completes.
	sizes := []struct {
		name string
		size int
	}{
		{"S1", awgMessageInitiationSize + a.S1},
		{"S2", awgMessageResponseSize + a.S2},
		{"S3", awgMessageCookieReplySize + a.S3},
		{"S4", awgMessageTransportSize + a.S4},
	}
	for i := 0; i < len(sizes); i++ {
		for j := i + 1; j < len(sizes); j++ {
			if sizes[i].size == sizes[j].size {
				return fmt.Errorf(
					"amnezia %s and %s produce equal packet sizes (%d bytes); the peers cannot distinguish packet types and the connection will not come up (S1+148, S2+92, S3+64, S4+32 must all differ)",
					sizes[i].name, sizes[j].name, sizes[i].size)
			}
		}
	}
	headers := make([]struct {
		name  string
		value *awgHeaderRange
	}, 0, 4)
	for _, h := range []struct {
		name  string
		value any
	}{{"H1", a.H1}, {"H2", a.H2}, {"H3", a.H3}, {"H4", a.H4}} {
		parsed, err := parseAWGHeaderValue(h.name, h.value)
		if err != nil {
			return err
		}
		if parsed == nil {
			continue
		}
		if parsed.from <= awgHeaderReservedMax {
			return fmt.Errorf(
				"amnezia %s (%s) uses reserved values 0-%d (vanilla WireGuard message types); use %d or higher or the traffic stays trivially detectable and mixed setups break",
				h.name, parsed, awgHeaderReservedMax, awgHeaderMin)
		}
		headers = append(headers, struct {
			name  string
			value *awgHeaderRange
		}{h.name, parsed})
	}
	for i := 0; i < len(headers); i++ {
		for j := i + 1; j < len(headers); j++ {
			if headers[i].value.overlaps(*headers[j].value) {
				return fmt.Errorf(
					"amnezia %s (%s) and %s (%s) overlap; the receiver identifies packet types by header ranges, so overlapping ranges break the handshake",
					headers[i].name, headers[i].value, headers[j].name, headers[j].value)
			}
		}
	}
	return nil
}

// AmneziaRandomParams is the server-generated obfuscation parameter set.
// H1-H4 are always generated; Jc/Jmin/Jmax only for the Balanced preset
// (project decision: Randomize touches junk parameters only on Balanced).
type AmneziaRandomParams struct {
	H1   string `json:"h1"`
	H2   string `json:"h2"`
	H3   string `json:"h3"`
	H4   string `json:"h4"`
	JC   int    `json:"jc,omitempty"`
	JMin int    `json:"jmin,omitempty"`
	JMax int    `json:"jmax,omitempty"`
}

// GenerateAmneziaParams builds four pairwise disjoint H1-H4 ranges within
// [5, 2^31-1] using crypto/rand (matching the official Amnezia client
// generator bounds). With includeJunk it also generates Jc/Jmin/Jmax from
// the Balanced preset distribution (Jc 3-6, Jmin 40-89, Jmax Jmin+50..250,
// per the amneziawg-installer ADVANCED.md default preset).
func GenerateAmneziaParams(includeJunk bool) (AmneziaRandomParams, error) {
	var params AmneziaRandomParams
	ranges, err := generateDisjointHeaderRanges()
	if err != nil {
		return params, err
	}
	params.H1 = ranges[0].String()
	params.H2 = ranges[1].String()
	params.H3 = ranges[2].String()
	params.H4 = ranges[3].String()
	if includeJunk {
		jc, err := awgCryptoRandInt(3, 6)
		if err != nil {
			return params, err
		}
		jmin, err := awgCryptoRandInt(40, 89)
		if err != nil {
			return params, err
		}
		spread, err := awgCryptoRandInt(50, 250)
		if err != nil {
			return params, err
		}
		// Bounds are small compile-time constants (max 89+250), so the uint64
		// to int conversions cannot overflow.
		params.JC = int(jc)              //nolint:gosec // bounded by awgCryptoRandInt(3, 6)
		params.JMin = int(jmin)          //nolint:gosec // bounded by awgCryptoRandInt(40, 89)
		params.JMax = int(jmin + spread) //nolint:gosec // bounded by 89+250
	}
	return params, nil
}

func generateDisjointHeaderRanges() ([4]awgHeaderRange, error) {
	var result [4]awgHeaderRange
	// Draw 8 distinct values in [5, 2^31-1], sort, pair them into 4 disjoint
	// ranges, then shuffle the assignment to H1..H4. Collisions across a
	// 2^31 space are vanishingly rare; the retry loop is a formality.
	for attempt := 0; attempt < 100; attempt++ {
		values := make([]uint64, 8)
		for i := range values {
			v, err := awgCryptoRandInt(awgHeaderMin, awgGenHeaderMax)
			if err != nil {
				return result, err
			}
			values[i] = v
		}
		sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
		distinct := true
		for i := 1; i < len(values); i++ {
			if values[i] == values[i-1] {
				distinct = false
				break
			}
		}
		if !distinct {
			continue
		}
		for i := 0; i < 4; i++ {
			result[i] = awgHeaderRange{from: values[i*2], to: values[i*2+1]}
		}
		// Fisher-Yates with crypto/rand: range magnitude order must not leak
		// which header is which.
		for i := 3; i > 0; i-- {
			j, err := awgCryptoRandInt(0, uint64(i))
			if err != nil {
				return result, err
			}
			result[i], result[j] = result[j], result[i]
		}
		return result, nil
	}
	return result, fmt.Errorf("failed to generate disjoint Amnezia header ranges")
}

// awgCryptoRandInt returns a uniform random value in [min, max] inclusive
// using crypto/rand.
func awgCryptoRandInt(min, max uint64) (uint64, error) {
	if max < min {
		return 0, fmt.Errorf("invalid random range")
	}
	span := new(big.Int).SetUint64(max - min + 1)
	n, err := rand.Int(rand.Reader, span)
	if err != nil {
		return 0, err
	}
	return min + n.Uint64(), nil
}
