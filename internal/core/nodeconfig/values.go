package nodeconfig

import (
	"fmt"
	"net/netip"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// valueRule is what a valued flag accepts, read from the binaries' help in
// docs/chain-analysis. A key without a rule is free text the binary parses
// itself (lists, paths, URLs).
type valueRule struct {
	kind     string // integer, decimal, duration, enum, nat
	min, max int64
	enum     []string
	// byDialect overrides enum for one generation.
	byDialect map[string][]string
}

const unbounded = int64(1<<62 - 1)

var (
	portRule    = valueRule{kind: "integer", min: 0, max: 65535}
	countRule   = valueRule{kind: "integer", min: 0, max: unbounded}
	percentRule = valueRule{kind: "integer", min: 0, max: 100}
)

var valueRules = map[OptionKey]valueRule{
	KeyPort:          portRule,
	KeyHTTPPort:      portRule,
	KeyWSPort:        portRule,
	KeyAuthPort:      portRule,
	KeyMetricsPort:   portRule,
	KeyDiscoveryPort: portRule,
	// All three binaries' help: "0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail".
	KeyVerbosity: {kind: "integer", min: 0, max: 5},
	// geth114 help: "snap" or "full"; go-wemix adds "light".
	KeySyncMode: {kind: "enum", enum: []string{"full", "snap"}, byDialect: map[string][]string{"geth110-wemix": {"full", "snap", "light"}}},
	KeyGCMode:   {kind: "enum", enum: []string{"full", "archive"}},
	KeyNAT:      {kind: "nat"},

	KeyMaxPeers:            countRule,
	KeyMaxPendPeers:        countRule,
	KeyNetworkID:           {kind: "integer", min: 1, max: unbounded},
	KeyRPCGasCap:           countRule,
	KeyRPCTxFeeCap:         {kind: "decimal"},
	KeyMinerGasLimit:       countRule,
	KeyMinerGasPrice:       countRule,
	KeyMinerRecommit:       {kind: "duration"},
	KeyTxPoolRejournal:     {kind: "duration"},
	KeyTxPoolLifetime:      {kind: "duration"},
	KeyTxPoolPriceLimit:    countRule,
	KeyTxPoolPriceBump:     countRule,
	KeyTxPoolAccountSlots:  countRule,
	KeyTxPoolGlobalSlots:   countRule,
	KeyTxPoolAccountQueue:  countRule,
	KeyTxPoolGlobalQueue:   countRule,
	KeyCache:               countRule,
	KeyCacheDatabase:       percentRule,
	KeyCacheTrie:           percentRule,
	KeyCacheGC:             percentRule,
	KeyCacheSnapshot:       percentRule,
	KeyGPOBlocks:           {kind: "integer", min: 1, max: unbounded},
	KeyGPOPercentile:       percentRule,
	KeyGPOMaxPrice:         countRule,
	KeyGPOIgnorePrice:      countRule,
	KeyTxLookupLimit:       countRule,
	KeyDevPeriod:           countRule,
	KeyDevGasLimit:         countRule,
	KeyHistoryState:        countRule,
	KeyHistoryTransactions: countRule,
	KeyCacheBlockLogs:      countRule,
	// go-wemix help: "integer, 1=PoW, 2=PoA, 3=ETCD, 4=PBFT".
	KeyConsensusMethod:   {kind: "integer", min: 1, max: 4},
	KeyBlocksPerTurn:     {kind: "integer", min: 1, max: unbounded},
	KeyNonceLimit:        countRule,
	KeyMaxTxsPerBlock:    countRule,
	KeyBlockInterval:     {kind: "integer", min: 1, max: unbounded},
	KeyBlockTimeAdj:      countRule,
	KeyBlockMinBuildTime: countRule,
	KeyBlockMinBuildTxs:  countRule,
	KeyBlockTrailTime:    countRule,
	KeyBootnodeCount:     countRule,
	KeyMaxIdleBlock:      countRule,
	KeyFixedDifficulty:   countRule,
	KeyFixedGasLimit:     countRule,
	KeyMinerGasTarget:    countRule,
}

var integerText = regexp.MustCompile(`^-?[0-9]+$`)

func (r valueRule) values(d Dialect) []string {
	if e, ok := r.byDialect[d.ID]; ok {
		return e
	}
	return r.enum
}

// CheckValue reports whether v is a value the dialect's binary accepts for k.
func CheckValue(d Dialect, k OptionKey, v string) error {
	r, ok := valueRules[k]
	if !ok {
		return nil
	}
	switch r.kind {
	case "integer":
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || !integerText.MatchString(v) || n < r.min || n > r.max {
			if r.max == unbounded {
				return fmt.Errorf("%s must be an integer of at least %d, got %q", k, r.min, v)
			}
			return fmt.Errorf("%s must be an integer from %d to %d, got %q", k, r.min, r.max, v)
		}
	case "decimal":
		if f, err := strconv.ParseFloat(v, 64); err != nil || f < 0 {
			return fmt.Errorf("%s must be a non-negative number, got %q", k, v)
		}
	case "duration":
		if _, err := time.ParseDuration(v); err != nil {
			if n, err := strconv.ParseInt(v, 10, 64); err != nil || n < 0 {
				return fmt.Errorf("%s must be a duration such as 3s or a number of nanoseconds, got %q", k, v)
			}
		}
	case "enum":
		if allowed := r.values(d); !slices.Contains(allowed, v) {
			return fmt.Errorf("%s must be one of %s, got %q", k, strings.Join(allowed, ", "), v)
		}
	case "nat":
		mode, ip, hasIP := strings.Cut(v, ":")
		switch {
		case !hasIP && slices.Contains([]string{"any", "none", "upnp", "pmp"}, mode):
		case hasIP && (mode == "extip" || mode == "pmp" && d.ID == "geth114"):
			if _, err := netip.ParseAddr(ip); err != nil {
				return fmt.Errorf("%s %s needs an IP address, got %q", k, mode, v)
			}
		default:
			return fmt.Errorf("%s must be any, none, upnp, pmp or extip:<IP>, got %q", k, v)
		}
	}
	return nil
}

// valueSchema adds a rule's constraints to an option's JSON schema. Launch
// values stay strings, so numeric limits travel as x-minimum and x-maximum
// beside a pattern the form can check.
func valueSchema(d Dialect, k OptionKey, schema map[string]any) {
	r, ok := valueRules[k]
	if !ok {
		return
	}
	schema["x-valueKind"] = r.kind
	switch r.kind {
	case "integer":
		schema["pattern"] = `^-?[0-9]+$`
		schema["x-minimum"] = r.min
		if r.max != unbounded {
			schema["x-maximum"] = r.max
		}
	case "decimal":
		schema["pattern"] = `^[0-9]+(\.[0-9]+)?$`
	case "enum":
		schema["enum"] = r.values(d)
	case "nat":
		schema["pattern"] = `^(any|none|upnp|pmp|(extip|pmp):.+)$`
	}
}
