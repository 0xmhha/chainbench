package genesis

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"math/big"
	"slices"
	"strconv"
	"strings"
)

// Carrying a genesis in the node's config file rather than in genesis.json.
//
// A geth-family binary reads its genesis from two places, and they are not the
// same reader. `init` reads genesis.json through the document's own JSON
// decoder, which is generated: it knows that GasLimit arrives as "0x64", that
// ExtraData arrives as a hex string, and it ignores a key no field matches.
// A config file is read through the TOML decoder, which knows none of that: it
// matches the Go field name exactly, it wants a uint64 as an integer and a
// []byte as a list of integers, and it refuses a key no field matches.
//
// So the same genesis is written twice, in two spellings. ConfigTOML renders
// the one a config file takes. Every rule below was measured against go-wbft and
// go-wemix rather than derived from the schema, because the schema is
// go-ethereum's Go types and the document is JSON — the mismatch between them
// is the whole problem.

// goFieldNames are the genesis keys whose Go field name is not the JSON key
// with its first letter upper-cased. They are the initialisms: Go writes an
// initialism in one case, and the JSON document writes it in camel case.
//
// The TOML decoder geth configures matches field names exactly (its
// NormFieldName is the identity function), so "chainId" reaches no field and
// the node dies at config load naming the field it could not place.
var goFieldNames = map[string]string{
	"chainId":        "ChainID",
	"eip150Block":    "EIP150Block",
	"eip150Hash":     "EIP150Hash",
	"eip155Block":    "EIP155Block",
	"eip158Block":    "EIP158Block",
	"daoForkBlock":   "DAOForkBlock",
	"daoForkSupport": "DAOForkSupport",
	"blsPublicKeys":  "BLSPublicKeys",
	"wBFT":           "WBFT",
	"mixhash":        "Mixhash",
}

// The Go kind of a genesis field, by its Go field name. JSON carries all three
// of these as hex strings; TOML has to carry each one differently.
var (
	// uint64Fields arrive as "0x42" and have to be written as 66.
	uint64Fields = map[string]bool{
		"Nonce": true, "Timestamp": true, "GasLimit": true,
		"Number": true, "GasUsed": true,
	}
	// bytesFields are []byte, which TOML can only express as a list of
	// integers — there is no hex literal and no UnmarshalText on a byte slice.
	bytesFields = map[string]bool{"ExtraData": true, "Code": true}
	// bigIntFields are *big.Int, whose UnmarshalText reads a decimal or
	// 0x-prefixed string. Written in decimal so a reader is not asked to
	// recognise a base.
	bigIntFields = map[string]bool{"Difficulty": true, "BaseFee": true, "Balance": true}
)

// mapTables are the genesis tables whose keys are data rather than Go field
// names: an allocation is keyed by address, a contract's params by the name the
// contract looks them up under. Their keys are written exactly as the document
// spells them.
//
// Only their immediate keys. One level below an allocation is a types.Account,
// whose keys are field names again — lower-casing Balance there is a config the
// binary loads and a chain it then refuses to extend.
var mapTables = map[string]bool{"Alloc": true, "Params": true, "Storage": true}

// ConfigTOML renders a genesis document as the TOML tables a geth-family binary reads
// from its config file, rooted at the given table path (normally "Eth.Genesis").
//
// omit names genesis keys to leave out: keys this chain's template writes that
// core.Genesis has no field for. The JSON decoder ignores them and the TOML
// decoder refuses them, so a chain that writes any must say so (see
// registry.GenesisSpec.ConfigOmit) or its own genesis cannot be carried here.
//
// The whole document is rendered, never a fragment. A config that carries only
// the part that differs is refused by the node: it compares what the config
// declares against the genesis its database was initialized from, and a
// fragment is a different genesis.
//
// Keys are sorted, so the same genesis renders byte-identical every time —
// reuse compares configs by hash.
func ConfigTOML(genesisJSON []byte, table string, omit []string) ([]byte, error) {
	if table == "" {
		return nil, fmt.Errorf("genesis: toml: no table path to render under")
	}
	doc, err := decodeObject(genesisJSON)
	if err != nil {
		return nil, err
	}
	for _, k := range omit {
		delete(doc, k)
	}
	var b strings.Builder
	if err := writeTable(&b, strings.Split(table, "."), doc, false); err != nil {
		return nil, err
	}
	return []byte(b.String()), nil
}

// decodeObject reads the document with numbers left as text, so a chain id or a
// balance too large for a float64 survives the round trip.
func decodeObject(raw []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var doc map[string]any
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("genesis: toml: read genesis: %w", err)
	}
	return doc, nil
}

// writeTable renders one table: its scalars, then its sub-tables. verbatim says
// this table's own keys are data and are not renamed.
func writeTable(b *strings.Builder, path []string, obj map[string]any, verbatim bool) error {
	keys := slices.Sorted(maps.Keys(obj))
	var scalars, tables []string
	for _, k := range keys {
		if _, ok := obj[k].(map[string]any); ok {
			tables = append(tables, k)
		} else if obj[k] != nil {
			// A null is the document saying "unset". TOML has no null, and the
			// decoder cannot be told to skip one.
			scalars = append(scalars, k)
		}
	}
	fmt.Fprintf(b, "[%s]\n", strings.Join(path, "."))
	for _, k := range scalars {
		name := k
		if !verbatim {
			name = goFieldName(k)
		}
		v, err := scalarTOML(goFieldName(k), obj[k])
		if err != nil {
			return fmt.Errorf("genesis: toml: %s.%s: %w", strings.Join(path, "."), name, err)
		}
		fmt.Fprintf(b, "%s = %s\n", tomlKey(name), v)
	}
	b.WriteString("\n")
	for _, k := range tables {
		name := k
		if !verbatim {
			name = goFieldName(k)
		}
		sub := obj[k].(map[string]any)
		if err := writeTable(b, append(path, tomlKey(name)), sub, mapTables[name]); err != nil {
			return err
		}
	}
	return nil
}

// goFieldName is the Go field a genesis key names.
func goFieldName(key string) string {
	if n, ok := goFieldNames[key]; ok {
		return n
	}
	if key == "" {
		return key
	}
	return strings.ToUpper(key[:1]) + key[1:]
}

// tomlKey quotes a key TOML cannot spell bare. An allocation's address is bare-
// legal (digits and letters); a key with a dot or a dash in it is not.
func tomlKey(k string) string {
	for _, r := range k {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
		default:
			return strconv.Quote(k)
		}
	}
	if k == "" {
		return `""`
	}
	return k
}

// scalarTOML writes one value in the spelling the field's Go type takes.
func scalarTOML(field string, v any) (string, error) {
	switch {
	case uint64Fields[field]:
		n, err := integer(v)
		if err != nil {
			return "", err
		}
		return n.String(), nil
	case bytesFields[field]:
		raw, err := hexBytes(v)
		if err != nil {
			return "", err
		}
		parts := make([]string, len(raw))
		for i, c := range raw {
			parts[i] = strconv.Itoa(int(c))
		}
		return "[" + strings.Join(parts, ", ") + "]", nil
	case bigIntFields[field]:
		n, err := integer(v)
		if err != nil {
			return "", err
		}
		// A quoted decimal: big.Int reads it through UnmarshalText, and an
		// unquoted one overflows the decoder's int64 for any real balance.
		return strconv.Quote(n.String()), nil
	}
	return plainTOML(v)
}

// plainTOML writes a value whose TOML spelling is its JSON one.
func plainTOML(v any) (string, error) {
	switch t := v.(type) {
	case bool:
		return strconv.FormatBool(t), nil
	case json.Number:
		return t.String(), nil
	case string:
		return strconv.Quote(t), nil
	case []any:
		parts := make([]string, 0, len(t))
		for _, e := range t {
			s, err := plainTOML(e)
			if err != nil {
				return "", err
			}
			parts = append(parts, s)
		}
		return "[" + strings.Join(parts, ", ") + "]", nil
	}
	return "", fmt.Errorf("no TOML spelling for %T", v)
}

// integer reads a genesis number, which the document writes either as hex text
// or as a bare number.
func integer(v any) (*big.Int, error) {
	switch t := v.(type) {
	case json.Number:
		n, ok := new(big.Int).SetString(t.String(), 10)
		if !ok {
			return nil, fmt.Errorf("%q is not a number", t.String())
		}
		return n, nil
	case string:
		s := strings.TrimPrefix(strings.TrimPrefix(t, "0x"), "0X")
		base := 10
		if len(s) != len(t) {
			base = 16
		}
		if s == "" {
			return big.NewInt(0), nil
		}
		n, ok := new(big.Int).SetString(s, base)
		if !ok {
			return nil, fmt.Errorf("%q is not a number", t)
		}
		return n, nil
	}
	return nil, fmt.Errorf("%T is not a number", v)
}

// hexBytes reads a 0x-prefixed byte string. An odd digit count is left-padded,
// which is what the document means by "0x0".
func hexBytes(v any) ([]byte, error) {
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("%T is not a hex string", v)
	}
	s = strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	if len(s)%2 == 1 {
		s = "0" + s
	}
	raw, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("not hex: %w", err)
	}
	return raw, nil
}
