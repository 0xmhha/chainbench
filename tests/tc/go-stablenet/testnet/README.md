# tests/tc/go-stablenet/testnet — 우리가 세우지 않은 망에 붙어 도는 케이스

나머지 케이스는 자기가 쓸 망을 직접 세우고 끝나면 지운다. 여기 있는 것은 **이미 떠 있는
go-stablenet 망에 붙는다.** 그래서 전제가 다르고, 단언도 다르게 써야 한다.

## 어떻게 돌리나

주소는 `GSTABLE_TESTNET_RPC` 로만 받는다. **기본값이 없다.**

```sh
export GSTABLE_TESTNET_RPC=http://<testnet 엔드포인트>:8545
bin/chainbench run tests/tc/go-stablenet/testnet/02-block-advances.json
```

변수를 안 걸면 실행이 멈추고 그 이유를 말한다. `stablenet-attached` 처럼 기본값을 두면
변수를 잊었을 때 조용히 로컬 망에 붙어, testnet 을 쟀다고 믿으면서 자기 기계를 재게 된다.
그래서 이 preset 에는 기본값을 두지 않았다.

엔진은 붙기 전에 엔드포인트가 정말 go-stablenet 인지 확인한다. `web3_clientVersion` 이
매니페스트가 적은 바이너리 이름(`gstable`)을 담지 않으면 거부한다. 체인 ID 가 우리 기본값과
다른 것은 남의 망에서 당연하므로 ID 로는 판단하지 않는다.

## 무엇을 쓸 수 있고 무엇을 쓰면 안 되나

**key preset 의 라벨을 쓰지 않는다.** 이 preset 은 `keysDir` 을 적지 않는다. `presets/keys` 의
`node1`·`faucet`·`dev1` 은 우리 genesis 가 만든 주소라 남의 망에는 없다. 라벨을 쓰면 아무
상관없는 주소의 잔액을 묻게 된다. 라벨이 안 되면 케이스가 거기서 막히는데, 조용히 틀린
주소를 묻는 것보다 낫다.

쓸 수 있는 라벨은 케이스가 직접 선언한 것뿐이다. 자금을 쥔 계정은 `stablenet-testnet-funded`
가 `accounts.payer.keyFile` 로 엮고, 그 키는 `GSTABLE_TESTNET_KEY_FILE` 이 가리키는 저장소
밖 파일에서 읽는다. 읽기만 하는 케이스는 그 변수가 필요 없으므로 `stablenet-testnet` 을 쓴다.

**절대값과 견주지 않는다.** 블록 높이도 잔액도 우리가 모르는 값에서 시작하고, 다른 트래픽이
섞인다. "정확히 1 ETH" 같은 단언은 우리 망에서만 성립한다.

**"0 이상" 으로 도망가지도 않는다.** 블록 번호는 음수가 될 수 없어 그런 단언은 노드가 무엇을
답해도 통과하고, 창세 블록에 멈춘 노드도 초록불이 된다. 이 저장소가 2026-09-29 에 그런
단언 여럿을 걷어냈다.

대신 쓸 수 있는 것은 이런 모양이다.

| 재는 것 | 왜 남의 망에서도 성립하나 |
|---|---|
| 지금 값보다 올라가는가(`blockAdvance`) | 절대값을 몰라도 변화는 안다 |
| 형식이 맞는가(16진수, 32바이트 해시) | 상태와 무관하다 |
| 자기 일관성(해시로 다시 조회해도 같은 블록) | 외부 트래픽이 끼어들 자리가 없다 |
| 한 실행 안에서 같은 값인가(체인 ID 두 번 조회) | 프록시 뒤에 다른 체인이 섞였는지 드러난다 |
| **내가 보낸** 트랜잭션의 영수증과 전후 차분 | 내 해시만 보므로 남의 트래픽과 무관하다 |

## 지금 있는 것

| 파일 | 무엇을 보나 |
|---|---|
| `01-chain-identity` | clientVersion 이 Gstable 인가, 체인 ID 가 16진수이고 두 번 물어도 같은가 |
| `02-block-advances` | 블록이 계속 나오는가 |
| `03-block-fields-well-formed` | 최신 블록의 number·hash·parentHash·transactions 형식 |
| `04-block-by-hash-consistency` | 머리에서 네 칸 물러난 블록을 해시로 다시 조회해도 같은가 |
| `05-value-transfer` | payer 에서 방금 만든 주소로 1 Gwei 를 보내고, 영수증·받는 쪽 잔액·보낸 쪽 전후 차분 |

머리 블록을 쓰지 않는 것은 재구성될 수 있기 때문이다.

05 만 `stablenet-testnet-funded` 를 쓴다. 보내는 금액을 1 Gwei 로 둔 것은 공용 망에서 쓰는
자금을 최소로 하기 위해서다. 받는 쪽은 그 실행이 방금 만든 주소라 시작 잔액이 0 이고 절대값을
쓸 수 있다. 보내는 쪽은 절대값을 모르므로 전후 차분이 송금액과 수수료의 합과 같은지만 본다.

## 키 파일의 형식

`keyFile` 은 **평문 hex** 를 담은 파일을 읽는다. keystore JSON 은 비밀번호가 필요한데 선언에
그것을 적을 자리가 없어 거절하고, 거절할 때 꺼내는 방법을 함께 말한다.

```sh
chainbench keyring new --keyring-dir ~/.chainbench/keys/gstable-testnet --count 1 --validators 0
chainbench keyring export --keyring-dir ~/.chainbench/keys/gstable-testnet --name node1 --yes
```

꺼낸 hex 를 저장소 **밖** 파일에 `chmod 600` 으로 두고 그 경로를 변수에 건다. 그 주소에 자금을
넣는 것은 사람이 한다 — 남의 망에는 faucet 을 부를 권한이 없다.

## 확인 범위

다섯 건 모두 **실제 go-stablenet testnet 에 대고** 돌려 통과를 확인했다(2026-10-01). 그 망은
`Gstable/v1.1.0-stable-71e3f820`, chain id 8283, 높이 약 2,114만이었다. 05 는 그 망에서 1 Gwei
를 실제로 보냈고, 받는 쪽 잔액과 보낸 쪽 전후 차분이 모두 맞았다.

**로컬 망에서 먼저 돌릴 때 주의할 것.** testnet 대신 로컬 망을 세워 05 를 돌린다면 payer 에
`node1` 의 키를 쓰면 안 된다. node1 은 블록을 만드는 노드라 같은 구간에 보상이 들어와 전후
차분이 맞지 않는다. 그 망에서 새 계정을 만들어 자금을 보내고 그 키를 걸어야 한다. 실제
testnet 의 지불 계정은 생산자가 아니므로 단언 자체는 고칠 것이 없다.

겪어 보지 않은 것은 남의 망의 응답 지연과 rate limit 이다. 이번 실행은 다섯 건이 연달아
통과했을 뿐이고, 그 망이 혼잡할 때 어떻게 되는지는 모른다.
