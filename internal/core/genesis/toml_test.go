package genesis

import (
	"strings"
	"testing"
)

// The genesis every case below renders. It is small, but it holds one of each
// thing the two decoders disagree about: an initialism, a uint64 written as
// hex, a []byte written as hex, a balance too large for an int64, a null, a key
// core.Genesis has no field for, and two tables whose keys are data.
const forkGenesis = `{
  "nonce": "0x0000000000000042",
  "timestamp": "0x00",
  "gasLimit": "105000000",
  "difficulty": "0x1",
  "extraData": "0x54686520",
  "rewards": "0x",
  "alloc": {
    "0xA8Ff92e6E162Ca365120AEC54999B5Ab4A8B2A94": {"balance": "200000000000000000000000"}
  },
  "config": {
    "chainId": 1111,
    "eip155Block": 0,
    "croissantBlock": 100,
    "croissant": {
      "wBFT": {"epochLength": 10, "maxRequestTimeoutSeconds": null},
      "init": {"blsPublicKeys": ["0xa00e"]},
      "govContracts": {"govConfig": {"params": {"minimumStaking": "10000"}}}
    }
  }
}`

func renderFork(t *testing.T, omit ...string) string {
	t.Helper()
	out, err := ConfigTOML([]byte(forkGenesis), "Eth.Genesis", omit)
	if err != nil {
		t.Fatalf("TOML: %v", err)
	}
	return string(out)
}

// TestTOML_SpellsEachFieldTheWayItsGoTypeReadsIt.
//
// The TOML decoder has none of the genesis document's own conversions: it takes
// a uint64 as an integer, a []byte as a list of integers, a *big.Int as text,
// and it matches the Go field name exactly. Each line below is a node that
// booted or a node that died at config load.
func TestTOML_SpellsEachFieldTheWayItsGoTypeReadsIt(t *testing.T) {
	got := renderFork(t, "rewards")
	for _, want := range []string{
		"Nonce = 66",                          // uint64, not "0x0000000000000042"
		"Timestamp = 0",                       // uint64 from "0x00"
		"GasLimit = 105000000",                // uint64 already decimal
		`Difficulty = "1"`,                    // *big.Int reads text
		"ExtraData = [84, 104, 101, 32]",      // []byte has no hex literal
		"ChainID = 1111",                      // not "ChainId"
		"EIP155Block = 0",                     // not "Eip155Block"
		"BLSPublicKeys",                       // not "BlsPublicKeys"
		"[Eth.Genesis.Config.Croissant.WBFT]", // not "WBFT" -> "WBFT" via first-letter rule
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered config is missing %q\n%s", want, got)
		}
	}
}

// TestTOML_KeysThatAreDataKeepTheirSpelling: an allocation is keyed by address
// and a contract's params by the name the contract looks them up under. Renaming
// either produces a config the binary loads and a chain it then refuses to
// extend ("invalid gov config params"), which is far worse than a boot failure.
func TestTOML_KeysThatAreDataKeepTheirSpelling(t *testing.T) {
	got := renderFork(t, "rewards")
	if !strings.Contains(got, "minimumStaking =") {
		t.Errorf("a params key was renamed; it is data, not a field name\n%s", got)
	}
	if !strings.Contains(got, "[Eth.Genesis.Alloc.0xA8Ff92e6E162Ca365120AEC54999B5Ab4A8B2A94]") {
		t.Errorf("an allocation address was renamed\n%s", got)
	}
	// One level only: inside the account, the keys are field names again.
	if !strings.Contains(got, `Balance = "200000000000000000000000"`) {
		t.Errorf("an account's Balance field lost its Go spelling\n%s", got)
	}
}

// TestTOML_LeavesOutWhatTheDecoderWouldRefuse: a null is the document saying
// "unset", and TOML has no null; a key no Go field matches is ignored by the
// JSON decoder and fatal to the TOML one.
func TestTOML_LeavesOutWhatTheDecoderWouldRefuse(t *testing.T) {
	got := renderFork(t, "rewards")
	if strings.Contains(got, "MaxRequestTimeoutSeconds") {
		t.Errorf("a null was rendered; the decoder has no null to read\n%s", got)
	}
	if strings.Contains(got, "Rewards") {
		t.Errorf("an omitted key was rendered\n%s", got)
	}
}

// TestTOML_AnUnomittedExtraIsStillRendered: omission is the chain's declaration,
// not a guess made here. A chain that writes a key core.Genesis has no field for
// and does not declare it gets the config its genesis describes, and the binary
// names the field it could not place.
func TestTOML_AnUnomittedExtraIsStillRendered(t *testing.T) {
	if got := renderFork(t); !strings.Contains(got, "Rewards") {
		t.Errorf("an undeclared extra was dropped anyway\n%s", got)
	}
}

// TestTOML_RendersTheSameBytesEveryTime: reuse compares configs by hash, so a
// map iterated in Go's order would report a changed config on every run.
func TestTOML_RendersTheSameBytesEveryTime(t *testing.T) {
	first := renderFork(t, "rewards")
	for range 8 {
		if again := renderFork(t, "rewards"); again != first {
			t.Fatal("the same genesis rendered two different configs")
		}
	}
}

// TestTOML_RefusesWhatItCannotSpell: a value with no TOML spelling is an error
// here, at the step that would have written the config, rather than a node that
// dies at boot.
func TestTOML_RefusesWhatItCannotSpell(t *testing.T) {
	if _, err := ConfigTOML([]byte(`{"gasLimit": "not a number"}`), "Eth.Genesis", nil); err == nil {
		t.Fatal("a uint64 field that holds no number was accepted")
	}
	if _, err := ConfigTOML([]byte(`{"extraData": "0xzz"}`), "Eth.Genesis", nil); err == nil {
		t.Fatal("a []byte field that holds no hex was accepted")
	}
	if _, err := ConfigTOML([]byte(`not json`), "Eth.Genesis", nil); err == nil {
		t.Fatal("a document that is not a genesis was accepted")
	}
	if _, err := ConfigTOML([]byte(`{}`), "", nil); err == nil {
		t.Fatal("a render with no table path was accepted")
	}
}

// TestConfigTOML_TheFieldsWhoseGoNameIsNotTheKey.
//
// Most genesis keys become their Go field by upper-casing the first letter, and
// the initialisms are the obvious exceptions. These two are not obvious at all:
// core.Genesis calls the base fee BaseFee and writes it as baseFeePerGas, and
// its mix hash is spelled one way in the struct and another in most documents.
//
// Measured: a stablenet genesis carrying baseFeePerGas rendered as
// "BaseFeePerGas" and every node died at config load naming the field.
func TestConfigTOML_TheFieldsWhoseGoNameIsNotTheKey(t *testing.T) {
	out, err := ConfigTOML([]byte(`{
	  "baseFeePerGas": "0x3b9aca00",
	  "mixHash": "0x00",
	  "gasLimit": "0x64"
	}`), "Eth.Genesis", nil)
	if err != nil {
		t.Fatalf("ConfigTOML: %v", err)
	}
	got := string(out)
	for _, want := range []string{`BaseFee = "1000000000"`, "Mixhash ="} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q\n%s", want, got)
		}
	}
	if strings.Contains(got, "BaseFeePerGas") || strings.Contains(got, "MixHash") {
		t.Errorf("a key was rendered under its document spelling\n%s", got)
	}
}
