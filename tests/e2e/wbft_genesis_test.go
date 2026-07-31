//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// wemixGenesisTemplate is a minimal complete wemix/croissant genesis with a
// govNCP section whose `ncps` CSV is the %s parameter — the field NODE-006/007
// exercise. Everything else is fixed dummy config sufficient for `gwemix init`.
const wemixGenesisTemplate = `{
  "config": {
    "chainId": 999999,
    "homesteadBlock": 0, "eip150Block": 0, "eip155Block": 0, "eip158Block": 0,
    "byzantiumBlock": 0, "constantinopleBlock": 0, "petersburgBlock": 0,
    "istanbulBlock": 0, "muirGlacierBlock": 0, "berlinBlock": 0, "londonBlock": 0,
    "briocheBlock": 0, "croissantBlock": 0,
    "croissant": {
      "wBFT": {
        "requestTimeoutSeconds": 2, "blockPeriodSeconds": 1, "epochLength": 10,
        "targetValidators": 1, "proposerPolicy": 0, "maxRequestTimeoutSeconds": null,
        "stabilizingStakersThreshold": 1, "useNCP": true
      },
      "init": {
        "validators": ["0x0000000000000000000000000000000000000001"],
        "blsPublicKeys": ["0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"]
      },
      "govContracts": {
        "govConfig": { "address": "0x0000000000000000000000000000000000001000", "version": "v1", "params": {
          "changeFeeDelay": "5", "feePrecision": "10000", "govCouncil": "0x0000000000000000000000000000000000001003",
          "maximumStaking": "1000000000000000000000000000", "minimumStaking": "500000000000000000000000",
          "unbondingPeriodDelegator": "5", "unbondingPeriodStaker": "15" } },
        "govStaking": { "address": "0x0000000000000000000000000000000000001001", "version": "v1", "params": null },
        "govRewardeeImp": { "address": "0x0000000000000000000000000000000000001002", "version": "v1", "params": null },
        "govNCP": { "address": "0x0000000000000000000000000000000000001003", "version": "v1", "params": { "ncps": "%s" } }
      }
    }
  },
  "alloc": {},
  "difficulty": "0x1",
  "gasLimit": "0x2faf080"
}`

// gwemixInit writes a genesis with the given ncps CSV and runs `gwemix init`,
// returning whether it succeeded and the combined output.
func gwemixInit(t *testing.T, bin, ncps string) (bool, string) {
	t.Helper()
	dir := t.TempDir()
	genesis := filepath.Join(dir, "genesis.json")
	if err := os.WriteFile(genesis, []byte(fmt.Sprintf(wemixGenesisTemplate, ncps)), 0o644); err != nil {
		t.Fatalf("write genesis: %v", err)
	}
	out, err := exec.Command(bin, "init", "--datadir", filepath.Join(dir, "data"), genesis).CombinedOutput()
	return err == nil, string(out)
}

// TestE2E_WbftGenesisNCPWhitespace ports wemix4 NODE-006: `gwemix init` must
// accept a genesis whose govNCP `ncps` list has whitespace-padded addresses
// (they are trimmed), i.e. init succeeds.
//
//	WBFT_BIN=/path/to/go-wbft/build/bin/gwemix go test -tags e2e -run TestE2E_WbftGenesisNCPWhitespace -v ./tests/e2e/
func TestE2E_WbftGenesisNCPWhitespace(t *testing.T) {
	bin := requireBinary(t, "WBFT_BIN", "gwbft")
	// Each address carries a different leading/trailing whitespace pattern.
	const ncps = "0xee189e09af114017d540ef3c15cd025e6423d277, 0x3e9837ba4c40510ec5bf1db58368b8e38e47c891," +
		"0x0f6f5ed0dc34278bea20334195f184db09d8f8df ,  0xc6431440769af72662427dfa23b10a7b65904110  ," +
		"   0xb7a6b5850ab7a56f28a793067e3eb57603bcdbcd,0xe234fe723580b4c1e4a8de9d2e2545942a5c5f7b   ," +
		"    0x71504dae060cdcceae845f4b1fb14df2e7b9331c    "
	ok, out := gwemixInit(t, bin, ncps)
	if !ok {
		t.Fatalf("gwemix init failed on whitespace-padded NCP addresses (should be trimmed):\n%s", out)
	}
}

// TestE2E_WbftGenesisEmptyNCP ports wemix4 NODE-007: an empty govNCP `ncps` list
// is invalid, so `gwemix init` must fail (useNCP=true requires at least one NCP).
//
//	WBFT_BIN=/path/to/go-wbft/build/bin/gwemix go test -tags e2e -run TestE2E_WbftGenesisEmptyNCP -v ./tests/e2e/
func TestE2E_WbftGenesisEmptyNCP(t *testing.T) {
	bin := requireBinary(t, "WBFT_BIN", "gwbft")
	ok, out := gwemixInit(t, bin, "")
	if ok {
		t.Fatalf("gwemix init succeeded with an empty NCP list (should fail):\n%s", out)
	}
}
