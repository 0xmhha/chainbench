#!/usr/bin/env python3
"""
fee_delegate.py — go-stablenet FeeDelegateDynamicFeeTx (type 0x16) encoder & signer

go-stablenet 저장소 `core/types/tx_fee_delegation.go`의 RLP 구조 기반:

    sigHash_sender = keccak256(0x02 || rlp([chainId, nonce, tipCap, feeCap, gas, to, value, data, accessList]))
        (= 표준 DynamicFeeTx sigHash)

    sigHash_feePayer = keccak256(0x16 || rlp([
        [chainId, nonce, tipCap, feeCap, gas, to, value, data, accessList, senderV, senderR, senderS],
        feePayer
    ]))

    raw tx bytes = 0x16 || rlp([
        [chainId, nonce, tipCap, feeCap, gas, to, value, data, accessList, senderV, senderR, senderS],
        feePayer, fpV, fpR, fpS
    ])

사용법 (CLI):

  python3 fee_delegate.py sign \
    --rpc http://127.0.0.1:8501 \
    --sender-pk 0x... --fee-payer-pk 0x... \
    --to 0x... --value 1000000000000000000 \
    --gas 21000 \
    [--data 0x...] [--tip 27600000000000] [--tamper sender|feepayer]

출력: JSON {"rawTx": "0x16...", "senderHash": "0x...", "feePayerHash": "0x...", "senderAddr": "0x...", "feePayerAddr": "0x..."}

종속성: pip install eth-account requests rlp eth-utils
"""

import argparse
import json
import sys

import requests
import rlp
from eth_account import Account
from eth_account._utils.signing import to_bytes32  # noqa: F401 (unused but keeps import check)
from eth_keys import keys
from eth_utils import keccak, to_bytes, to_canonical_address

FEE_DELEGATE_TX_TYPE = 0x16


def rpc_call(url: str, method: str, params):
    r = requests.post(url, json={"jsonrpc": "2.0", "method": method, "params": params, "id": 1})
    return r.json()


def build_and_sign(
    rpc_url: str,
    sender_pk: str,
    fee_payer_pk: str,
    to: str,
    value: int,
    gas: int,
    data: bytes = b"",
    tip_cap: int = 27_600_000_000_000,
    tamper: str = None,
):
    sender_acct = Account.from_key(sender_pk)
    fee_payer_acct = Account.from_key(fee_payer_pk)

    # 1) Query chain state
    chain_id = int(rpc_call(rpc_url, "eth_chainId", [])["result"], 16)
    nonce = int(rpc_call(rpc_url, "eth_getTransactionCount", [sender_acct.address, "pending"])["result"], 16)
    base_fee = int(
        rpc_call(rpc_url, "eth_getBlockByNumber", ["latest", False])["result"]["baseFeePerGas"], 16
    )

    fee_cap = base_fee * 2 + tip_cap
    to_bytes_addr = to_canonical_address(to)
    fee_payer_addr = to_canonical_address(fee_payer_acct.address)

    # 2) Sender signs a standard DynamicFeeTx (type 0x02)
    #    The sender signature hash is identical between DynamicFeeTx and FeeDelegateDynamicFeeTx.SenderTx.
    std_tx = {
        "type": 2,
        "chainId": chain_id,
        "nonce": nonce,
        "maxPriorityFeePerGas": tip_cap,
        "maxFeePerGas": fee_cap,
        "gas": gas,
        "to": to,
        "value": value,
        "data": "0x" + data.hex() if data else "0x",
        "accessList": [],
    }
    sender_signed = sender_acct.sign_transaction(std_tx)
    sender_v = sender_signed.v
    sender_r = sender_signed.r
    sender_s = sender_signed.s

    # Optional tampering of sender signature
    if tamper == "sender":
        sender_s = (sender_s ^ 0xFF) & ((1 << 256) - 1)

    # 3) Build the inner RLP list used for fee payer sigHash
    #    [[chainId, nonce, tipCap, feeCap, gas, to, value, data, accessList, sV, sR, sS], feePayer]
    inner_sender_tx = [
        chain_id,
        nonce,
        tip_cap,
        fee_cap,
        gas,
        to_bytes_addr,
        value,
        data,
        [],  # access list
        sender_v,
        sender_r,
        sender_s,
    ]
    fee_payer_payload = [inner_sender_tx, fee_payer_addr]
    fee_payer_rlp = rlp.encode(fee_payer_payload)
    fee_payer_sig_hash = keccak(bytes([FEE_DELEGATE_TX_TYPE]) + fee_payer_rlp)

    # 4) Fee payer signs the hash
    fp_privkey = keys.PrivateKey(to_bytes(hexstr=fee_payer_pk))
    fp_sig = fp_privkey.sign_msg_hash(fee_payer_sig_hash)
    fp_v = fp_sig.v
    fp_r = fp_sig.r
    fp_s = fp_sig.s

    if tamper == "feepayer":
        fp_s = (fp_s ^ 0xFF) & ((1 << 256) - 1)

    # 5) Full RLP: [[sender tx with sig], feePayer, fpV, fpR, fpS]
    full_payload = [inner_sender_tx, fee_payer_addr, fp_v, fp_r, fp_s]
    raw_tx_bytes = bytes([FEE_DELEGATE_TX_TYPE]) + rlp.encode(full_payload)

    return {
        "rawTx": "0x" + raw_tx_bytes.hex(),
        "senderAddr": sender_acct.address,
        "feePayerAddr": fee_payer_acct.address,
        "chainId": chain_id,
        "nonce": nonce,
        "baseFee": base_fee,
        "tipCap": tip_cap,
        "feeCap": fee_cap,
        "gas": gas,
        "to": to,
        "value": value,
        "senderSigHash": sender_signed.hash.hex(),
        "feePayerSigHash": "0x" + fee_payer_sig_hash.hex(),
    }


def main():
    p = argparse.ArgumentParser()
    sub = p.add_subparsers(dest="cmd", required=True)

    sign_p = sub.add_parser("sign", help="Build and sign a FeeDelegateDynamicFeeTx")
    sign_p.add_argument("--rpc", default="http://127.0.0.1:8501")
    sign_p.add_argument("--sender-pk", required=True)
    sign_p.add_argument("--fee-payer-pk", required=True)
    sign_p.add_argument("--to", required=True)
    sign_p.add_argument("--value", type=int, default=1)
    sign_p.add_argument("--gas", type=int, default=21000)
    sign_p.add_argument("--data", default="")
    sign_p.add_argument("--tip", type=int, default=27_600_000_000_000)
    sign_p.add_argument(
        "--tamper",
        choices=[None, "sender", "feepayer"],
        default=None,
        help="Tamper with sender or feepayer signature for negative tests",
    )

    send_p = sub.add_parser("send", help="Sign + submit via eth_sendRawTransaction")
    send_p.add_argument("--rpc", default="http://127.0.0.1:8501")
    send_p.add_argument("--sender-pk", required=True)
    send_p.add_argument("--fee-payer-pk", required=True)
    send_p.add_argument("--to", required=True)
    send_p.add_argument("--value", type=int, default=1)
    send_p.add_argument("--gas", type=int, default=21000)
    send_p.add_argument("--data", default="")
    send_p.add_argument("--tip", type=int, default=27_600_000_000_000)
    send_p.add_argument("--tamper", choices=[None, "sender", "feepayer"], default=None)

    args = p.parse_args()

    data_bytes = b""
    if args.data:
        h = args.data[2:] if args.data.startswith("0x") else args.data
        data_bytes = bytes.fromhex(h)

    result = build_and_sign(
        rpc_url=args.rpc,
        sender_pk=args.sender_pk,
        fee_payer_pk=args.fee_payer_pk,
        to=args.to,
        value=args.value,
        gas=args.gas,
        data=data_bytes,
        tip_cap=args.tip,
        tamper=args.tamper,
    )

    if args.cmd == "send":
        resp = rpc_call(args.rpc, "eth_sendRawTransaction", [result["rawTx"]])
        result["rpcResponse"] = resp
        if "result" in resp:
            result["txHash"] = resp["result"]

    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
