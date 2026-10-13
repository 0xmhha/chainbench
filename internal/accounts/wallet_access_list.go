package accounts

import (
	"encoding/hex"
	"fmt"
	"strings"

	sdktx "github.com/0xmhha/accounts/tx"
	sdktypes "github.com/0xmhha/accounts/types"
)

// sdkAccessList uses one exact address/storage-key contract for both fee models.
func sdkAccessList(tuples []AccessTuple) (sdktx.AccessList, error) {
	list := make(sdktx.AccessList, len(tuples))
	for i, tuple := range tuples {
		address, err := sdktypes.HexToAddress(tuple.Address)
		if err != nil {
			return nil, fmt.Errorf("accounts: accessList address: %w", err)
		}
		list[i].Address = address
		for _, key := range tuple.StorageKeys {
			value, err := hex.DecodeString(strings.TrimPrefix(key, "0x"))
			if err != nil || len(value) != 32 || !strings.HasPrefix(key, "0x") {
				return nil, fmt.Errorf("accounts: accessList storage key must be 32-byte hex")
			}
			list[i].StorageKeys = append(list[i].StorageKeys, sdktypes.BytesToHash(value))
		}
	}
	return list, nil
}
