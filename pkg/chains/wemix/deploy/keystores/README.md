# keystores/

Local landing directory for keystore files pulled from the remote servers.
**Everything here except this README is gitignored** (see `../.gitignore`).

## How keys are obtained (closed-network policy)

Keys already exist on each remote server at fixed paths. The remote key-read
(a later phase — `chainbench remote keys read`, see
`docs/REMOTE_WEMIX_DEPLOY_DESIGN.md`) automates what wemix4 did by hand:

1. Derive the address + BLS material on the server:
   ```
   bootnode -nodekey /data/go-wbft/conf/nodekey -writeaddress
   #   address: 0x...                        -> validator coinbase addr
   #   derived bls public key: 0x...         -> bls (48 bytes)
   #   bls PoP (Proof of Possession): 0x...  -> bls_pop (96 bytes)
   ```
2. Pull the keystore files here (SCP), named `keystore_<index>` / `operator_<index>`:
   ```
   keystore_3     # coinbase keystore for server 3, from /data/go-wbft/conf/keystore/coinbase
   operator_3     # operator keystore for server 3, from /data/go-wbft/conf/keystore/operator
   .password      # the keystore password
   ```
3. The derived values populate `../accounts`.

> This read-from-remote flow is a **test convenience** for the current closed
> network. The normal flow generates keys locally and ships them with the binary
> (the existing `FileProvisioner` on the RemoteDriver).
