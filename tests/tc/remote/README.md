# remote — 외부 체인에 붙어서 읽기만 하는 검사

레거시 `tests/remote/` 의 4개(balance-check, chain-info, rpc-health, tx-send)에
해당하는 자리다. 이 갈래는 전용 케이스 파일 대신 attach 모드가 대신한다.

```
chainbench network attach --rpc <endpoint>
chainbench run --chain <name> --rpc <endpoint> tests/tc/go-stablenet/regression/api/*.json
```

노드를 띄우지 않고 이미 떠 있는 체인에 붙는 실행 방식이라, 같은 스펙을 그대로
읽기 전용으로 돌린다. 별도 케이스를 두면 같은 검증이 두 벌이 된다.
