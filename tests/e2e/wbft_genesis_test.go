//go:build e2e

package e2e

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
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

type ncpInitOutcome struct {
	ExitCode                                                int
	Binary, BinarySHA256, Genesis, OutputPath, OutputSHA256 string
	Args                                                    []string
	Output                                                  string `json:"-"`
}

// gwemixInit preserves its input, exact command, exit status and output before
// reclaiming only the completed process's disposable node databases.
func gwemixInit(t *testing.T, bin, ncps string) ncpInitOutcome {
	t.Helper()
	dir := ncpInitRuntime(t)
	genesis := filepath.Join(dir, "genesis.json")
	if err := os.WriteFile(genesis, []byte(fmt.Sprintf(wemixGenesisTemplate, ncps)), 0o644); err != nil {
		t.Fatalf("write genesis: %v", err)
	}
	args := []string{"init", "--datadir", filepath.Join(dir, "node1"), genesis}
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, args...).CombinedOutput()
	result := ncpInitOutcome{Binary: bin, Genesis: genesis, Args: args, Output: string(out), OutputPath: filepath.Join(dir, "reports", "init-output.log"), OutputSHA256: fmt.Sprintf("%x", sha256.Sum256(out))}
	var nodeExit *exec.ExitError
	if err != nil {
		result.ExitCode = -1
		if errors.As(err, &nodeExit) {
			result.ExitCode = nodeExit.ExitCode()
		}
	}
	binary, readErr := os.ReadFile(bin)
	if readErr != nil {
		t.Fatal(readErr)
	}
	result.BinarySHA256 = fmt.Sprintf("%x", sha256.Sum256(binary))
	if err := os.WriteFile(result.OutputPath, out, 0o600); err != nil {
		t.Fatal(err)
	}
	body, marshalErr := json.MarshalIndent(result, "", "  ")
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if err := os.WriteFile(filepath.Join(dir, "reports", "init-result.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("ncp-init-proof: %s", body)
	if ctx.Err() != nil {
		t.Fatalf("init command deadline: %v", ctx.Err())
	}
	if err != nil && nodeExit == nil {
		t.Fatalf("node process did not run: %v", err)
	}
	return result
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
	result := gwemixInit(t, bin, ncps)
	if result.ExitCode != 0 {
		t.Fatalf("gwemix init failed on whitespace-padded NCP addresses (should be trimmed):\n%s", result.Output)
	}
}

// TestE2E_WbftGenesisEmptyNCP ports wemix4 NODE-007: an empty govNCP `ncps` list
// is invalid, so `gwemix init` must fail (useNCP=true requires at least one NCP).
//
//	WBFT_BIN=/path/to/go-wbft/build/bin/gwemix go test -tags e2e -run TestE2E_WbftGenesisEmptyNCP -v ./tests/e2e/
func TestE2E_WbftGenesisEmptyNCP(t *testing.T) {
	bin := requireBinary(t, "WBFT_BIN", "gwbft")
	result := gwemixInit(t, bin, "")
	if err := verifyEmptyNCPRefusal(result.ExitCode, result.Output); err != nil {
		t.Fatalf("%v\n%s", err, result.Output)
	}
}
