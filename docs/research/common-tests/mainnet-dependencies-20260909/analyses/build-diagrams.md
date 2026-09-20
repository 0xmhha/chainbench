# 테스트가 관측하는 주요 빌드 모듈

아래는 darwin/arm64 실제 패키지 import 그래프의 일부다. 화살표는 함수 호출이 아니라 패키지 import다. 그림에 없는 모듈과 간선은 생략했으며 전체 자료는 graph 아래 JSON에 있다.

## go-wemix

```mermaid
flowchart LR
  N0["cmd/geth"]
  N1["core"]
  N2["core/types"]
  N3["core/vm"]
  N4["eth"]
  N5["internal/ethapi"]
  N6["miner"]
  N7["wemix"]
  N0 --> N1
  N0 --> N2
  N0 --> N4
  N0 --> N5
  N0 --> N7
  N1 --> N2
  N1 --> N3
  N3 --> N2
  N4 --> N1
  N4 --> N2
  N4 --> N3
  N4 --> N5
  N4 --> N6
  N5 --> N1
  N5 --> N2
  N5 --> N3
  N6 --> N1
  N6 --> N2
  N6 --> N3
  N7 --> N2
```

이 그림은 공통 거래·RPC 관찰 경로가 core, types, vm에 모이고, 합의와 거버넌스 결합은 프로젝트별로 달라지는 부분을 보여 준다. 같은 패키지 이름이어도 구현 의미가 같다고 판정하지 않으며 세부 차이는 chain-differences.md에 있다.

## go-wbft

```mermaid
flowchart LR
  N0["cmd/gwemix"]
  N1["consensus/wbft/backend"]
  N2["core"]
  N3["core/types"]
  N4["core/vm"]
  N5["eth"]
  N6["internal/ethapi"]
  N7["miner"]
  N8["wemixgov"]
  N0 --> N2
  N0 --> N3
  N0 --> N5
  N0 --> N6
  N1 --> N3
  N2 --> N3
  N2 --> N4
  N4 --> N3
  N5 --> N2
  N5 --> N3
  N5 --> N4
  N5 --> N6
  N5 --> N7
  N6 --> N2
  N6 --> N3
  N6 --> N4
  N7 --> N1
  N7 --> N2
  N7 --> N3
  N7 --> N4
```

이 그림은 공통 거래·RPC 관찰 경로가 core, types, vm에 모이고, 합의와 거버넌스 결합은 프로젝트별로 달라지는 부분을 보여 준다. 같은 패키지 이름이어도 구현 의미가 같다고 판정하지 않으며 세부 차이는 chain-differences.md에 있다.

## go-stablenet

```mermaid
flowchart LR
  N0["cmd/gstable"]
  N1["consensus/wbft/backend"]
  N2["core"]
  N3["core/types"]
  N4["core/vm"]
  N5["eth"]
  N6["internal/ethapi"]
  N7["miner"]
  N8["systemcontracts"]
  N0 --> N2
  N0 --> N3
  N0 --> N5
  N0 --> N6
  N1 --> N3
  N2 --> N3
  N2 --> N4
  N2 --> N8
  N4 --> N3
  N5 --> N2
  N5 --> N3
  N5 --> N4
  N5 --> N6
  N5 --> N7
  N6 --> N2
  N6 --> N3
  N6 --> N4
  N7 --> N1
  N7 --> N2
  N7 --> N3
  N7 --> N4
  N7 --> N8
  N8 --> N3
```

이 그림은 공통 거래·RPC 관찰 경로가 core, types, vm에 모이고, 합의와 거버넌스 결합은 프로젝트별로 달라지는 부분을 보여 준다. 같은 패키지 이름이어도 구현 의미가 같다고 판정하지 않으며 세부 차이는 chain-differences.md에 있다.
