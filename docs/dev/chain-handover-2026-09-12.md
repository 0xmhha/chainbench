# 체인팀 인계 — go-wemix / go-wbft 관측 3건

> **[측정] 2026-09-12.** 기준 커밋: chainbench `ea486e50` · go-wemix `902f9fce8` ·
> go-wbft `7af50e45d`. 코드 인용은 이 세 커밋 기준이고, 다시 뽑으면 갱신한다.

## 0. 이 문서가 무엇인가

chainbench 는 go-stablenet / go-wbft / go-wemix 를 실제 다중 노드로 세워 돌리는
테스트벤치다. 15대 docker 서버셋에서 wemix→wbft 핸드오프(15 producer + 15 successor)를
반복 실행하는 동안, **chainbench 를 고쳐서는 해소되지 않는 결함 3건**이 남았다. 세 건 모두
원인 위치가 체인 바이너리 쪽이고, 셋 다 chainbench 의 워크리스트 안에서는 관측 메모로만
존재했다. 이 문서는 그 메모를 **증상·근거·재현·제안**으로 옮겨 착수 가능한 형태로 만든 것이다.

무엇이 아닌지도 적는다. 이것은 수정 제안서가 아니다. 각 항목의 "제안" 절은 chainbench 가
소스를 읽어 도달한 지점까지이고, 세 건 모두 **체인 저장소에서 고치는 것이 맞다는 판단만
확정**이며 고치는 방법은 그쪽 설계 판단이다. 수정을 시도하지 않았고, 체인 저장소를 건드리지
않았다.

심각도와 확신도를 항목마다 표기한다. 확신도 라벨은 근거의 종류로 매긴다 — 직접 읽은
코드·로그에서 한 단계 추론이면 [High], 간접 근거나 일반 원칙에서의 추론이면 [Mid].

| # | 항목 | 심각도 | 원인 위치 | 상태 |
|---|---|---|---|---|
| **W1** | go-wemix `verifyBlockSig` 가 nil `*big.Int` 을 역참조해 노드가 죽는다 | **[치명]** | `wemix/admin.go:1002` | 원인 확정 [High] |
| **W2** | go-wemix 부트 노드의 etcd 가 형성 뒤 사라진다 (chainbench R6) | **[중요]** | `wemix/etcdutil.go` · `wemix/admin.go:731` | 원인 미확정, 범위 좁힘 [High] |
| **B1** | go-wbft `istanbul_getWbftExtraInfo` 가 `"latest"` 를 못 푼다 | **[권장]** | `consensus/wbft/backend/api.go:417` | 원인 확정 [High] |

---

## 1. W1 — go-wemix 가 `verifyBlockSig` 에서 nil 포인터로 죽는다 [치명]

### 증상

15+15 핸드오프 중 **15대 중 한 대**가 동기화하다 프로세스째 죽는다. 3회 연속 재현했고,
죽는 노드는 한 대로 고정되지 않았다(node4 → node5 → node5). chainbench 쪽에는 그 노드가
응답하지 않는 것으로만 보이므로, 최종 실패는 원인과 무관한 문장으로 난다:

```
nodes not ready for mesh: endpoint … not ready within 30s
```

죽은 노드의 로그 끝:

```
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x1 addr=0x10 pc=0x15c924c]

math/big.(*Int).Sign(...)
github.com/ethereum/go-ethereum/wemix.verifyBlockSig(...)        /src/wemix/admin.go:1002
github.com/ethereum/go-ethereum/wemix/miner.VerifyBlockSig(...)  /src/wemix/miner/miner.go:112
github.com/ethereum/go-ethereum/consensus/ethash.(*Ethash).verifyHeader(...)
                                                                 /src/consensus/ethash/consensus.go:327
```

세 프레임의 줄 번호는 **현재 소스(`902f9fce8`)와 그대로 일치한다** — `admin.go:1002`,
`miner/miner.go:112`, `ethash/consensus.go:327`. 즉 이 트레이스는 지금 코드를 가리킨다.

### 원인 — 단락 평가(short-circuit)의 좌우가 뒤집혀 있다 [High]

`wemix/admin.go:997-1003` 이다:

```go
num := new(big.Int).Sub(height, common.Big1)
contracts, err := admin.getRegGovEnvContracts(ctx, num)
if err != nil {
    return err == wemixminer.ErrNotInitialized || errors.Is(err, ethereum.NotFound)
} else if count, err := contracts.GovImp.GetMemberLength(&bind.CallOpts{Context: ctx, BlockNumber: num}); err != nil || count.Sign() == 0 {
    return err == wemixminer.ErrNotInitialized || count.Sign() == 0   // ← 1002
}
```

같은 두 값을 두 줄에서 읽는데, **한 줄은 nil 안전하고 다른 줄은 아니다.**

1001 줄의 조건은 `err != nil || count.Sign() == 0` 이다. `err != nil` 이면 `||` 가 단락하므로
`count.Sign()` 은 **평가되지 않는다.** 그래서 이 줄은 `count` 가 nil 이어도 안전하고, 분기에
들어간다.

1002 줄의 반환식은 `err == ErrNotInitialized || count.Sign() == 0` 이다. 피연산자 순서가
바뀌었다. `err` 이 nil 이 아니면서 `ErrNotInitialized` **도 아닌** 다른 에러이면 첫 피연산자가
거짓이 되고, 그 순간 `count.Sign()` 이 평가된다. 그리고 `count` 는 nil 이다 — 생성된 바인딩이
에러 경로에서 nil 을 돌려주기 때문이다(`wemix/bind/gen_gov_abi.go:2129`):

```go
func (_GovImp *GovImpCaller) GetMemberLength(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _GovImp.contract.Call(opts, &out, "getMemberLength")
	if err != nil {
		return *new(*big.Int), err   // ← nil
	}
	…
```

`(*big.Int)(nil).Sign()` 은 `len(x.abs)` 를 읽으며 죽는다. `big.Int` 는 `{neg bool; abs nat}`
이므로 `abs` 슬라이스의 길이 필드는 오프셋 16 — 관측된 `addr=0x10` 과 같다 [Mid]. 스택
프레임 모양은 직접 확인했다: 인자 없이 `math/big.(*Int).Sign(...)` 으로 나오는 것이 nil
수신자에서 나는 그 프레임이다 [High].

### 언제 `err` 이 "그 밖의 에러" 가 되는가 [Mid]

핸드오프 초반, 거버넌스 컨트랙트가 `num` 높이에서 아직 읽히지 않는 창이 있다. 그 높이에
코드가 없으면 call 은 빈 출력을 돌려주고, 언팩이 실패한다
(`accounts/abi/argument.go:82`: `abi: attempting to unmarshall an empty string while
arguments are expected`). 이것은 nil 도 `ErrNotInitialized` 도 아니다 — 1002 줄이 정확히
기다리고 있는 값이다.

관측과도 맞다. 죽는 노드는 **ethash 검증 경로**를 타면서 wemix 블록 서명을 검증하고 있었고
(`consensus/ethash/consensus.go:327`), 죽는 노드가 한 대로 고정되지 않았다. 창에 먼저 들어간
노드가 죽는 모양이다.

그 검증 함수의 바로 위 두 줄이 창의 위치를 더 좁혀 준다 — `consensus.go:323` 은
`IsCroissant(header.Number)` 면 먼저 거부한다. 즉 패닉은 **fork 이후 블록이 아니라 fork
이전 블록**을 검증하다 나고, 거버넌스가 아직 그 높이에서 읽히지 않는 초반 창과 겹친다.

### 영향

- 노드 프로세스가 죽는다. 합의 헤더 검증 경로라 **동기화 중 어느 노드든** 들어갈 수 있다.
- 입력이 네트워크에서 온 **블록 헤더**라는 점을 명시해 둔다. 거버넌스가 그 높이에서 읽히지
  않는 상태를 만들 수 있으면 헤더 하나로 노드를 죽일 수 있다는 뜻이다. 다만 트리거가
  공격자 입력만으로 성립하는지는 확인하지 않았다 — 로컬 상태(그 높이의 거버넌스 가독성)에
  달려 있다 [Mid].
- chainbench 쪽 2차 영향: 실패가 "endpoint not ready" 로 나와 원인을 가린다. 15대 중 1대가
  죽으면 mesh 준비 게이트에서 멈춘다.

### chainbench 가 배제한 것 [High]

**chainbench 결함이 아니다.** 같은 날 오전에 세 번 성공했던 **명시 경로 호출**
(`--from-binary /data/chainbench/bin/gwemix`)을 문자 그대로 다시 돌려도 같은 패닉으로
실패한다. 즉 호출 형태·경로 해석과 무관하다. genesis 해시는 전 노드 동일하고, 배포된 키
경로를 노드의 command 가 그대로 쓰는 것도 확인했다.

### 제안

1002 줄이 1001 줄과 같은 nil 안전성을 갖게 하는 것이 최소 변경이다. `err` 이 nil 이 아닌
경로에서 `count` 를 읽지 않도록 두 조건을 분리하거나, 반환식의 피연산자 순서를 nil 안전한
쪽으로 맞춘다. 어느 쪽이든 **`GetMemberLength` 가 실패했을 때 무엇을 반환할 것인가**를
결정해야 하고(현행 1001 줄은 "멤버 0명과 같게 본다"는 뜻으로 분기에 들어간다), 그 결정이
동기화 중 미확정 거버넌스를 어떻게 취급하느냐와 같은 질문이라 체인팀 몫이다.

같은 패턴이 다른 곳에도 있는지 함께 보기를 권장한다 — 가드에서는 단락으로 살고 반환식에서
죽는 모양은 한 곳에서 나면 보통 여러 곳에 있다.

---

## 2. W2 — 부트 노드의 etcd 가 형성 뒤 사라진다 (chainbench R6) [중요]

### 증상

14 bp + 1 en 규모(15대 docker)의 느린 브리지에서, 부트 노드가 형성한 etcd 클러스터를 **가끔
잃는다.** 재현 2회 중 1회는 완전 성공(14멤버 형성·sealer 로테이션), 1회는 붕괴였다.

chainbench 의 브링업은 부트 노드에서 `admin.etcdInit()` 을 실행한 뒤 **형성을 되읽어
확인**한다(`internal/consensus/poa/bootstrap_exec.go` 의 `VerifyEtcd` — `admin.wemixInfo.etcd.cluster`
가 멤버를 적어도 하나 적을 때까지 폴링). 붕괴한 실행에서도 **이 확인은 통과했다.** 그 뒤에
관측된 것이:

```
etcd join failed name=node14 error="not found"
etcd failed to start: cannot fetch cluster info from peer urls
```

이고, 부트 노드에서 `admin.wemixInfo.etcd.cluster` 는 `undefined` 였다. 부트는 거버넌스
블록에서 정지하고, 후속 join 이 전부 "not found" 로 실패한다.

### 소스를 읽어 좁힌 범위 [High]

**(1) `undefined` 는 "멤버 목록이 비었다" 가 아니라 "`ma.etcd == nil`" 이다.**
`wemix/etcdutil.go:988` 의 `etcdInfo()` 는 `ma.etcd == nil` 이면 **에러 값을 결과로 돌려준다**:

```go
func (ma *wemixAdmin) etcdInfo() interface{} {
	if ma.etcd == nil {
		return ErrNotRunning
	}
```

이 값은 RPC 를 지나면 `{}` 가 되고, 그래서 `.cluster` 가 `undefined` 로 읽힌다. 반대로
`MemberList` 가 실패한 경우라면 `cluster` 는 **빈 문자열**이 된다(`bb` 가 비므로) — 키 자체는
있다. 즉 관측된 `undefined` 는 멤버 조회 실패가 아니라 **etcd 핸들 자체가 없는 상태**를
가리킨다.

**(2) 생산 코드 어디에서도 `ma.etcd` 를 다시 nil 로 만들지 않는다.** `wemix/` 전체에서
`etcd = nil`·`etcdCli = nil`·`etcd.Close()` 는 테스트(`wemix/etcd_test.go:103`) 한 곳뿐이다.
`etcdEventHandler` 가 `admin.etcd.Err()` 를 받아도 `etcdReady` 만 내리고 핸들은 남긴다.

**(1)+(2) 를 합치면 결론이 바뀐다.** 형성된 클러스터를 "잃은" 것이 아니다. `VerifyEtcd` 가
통과한 프로세스와 `undefined` 를 돌려준 프로세스는 **같은 프로세스일 수 없다.** 가능한
경우는 둘이다:

- 부트 노드 프로세스가 그 사이에 **재시작**했다. 새 프로세스는 `ma.etcd == nil` 로 시작한다.
- 두 관측이 **다른 노드**를 본 것이다.

chainbench 의 브링업은 부트 노드를 그 사이에 재시작하지 않는다. 그래서 첫 번째 경우라면
**노드가 스스로 죽은 것**이고, 그렇다면 **W1 이 그 죽음일 가능성이 있다** [Mid] — 확인하지
않았다. W2 는 2026-09-02 기록이고 W1 패닉은 2026-09-11 에 관측했으므로 그 실행의 로그로
대조하지 못했다. 다만 재시작 뒤에 나는 에러 문장이 정확히 관측된 두 줄과 맞는다(아래).

**(3) 두 줄은 코드에서 그대로 나온다 — 그리고 첫 줄은 정상 브링업에서도 매번 나온다
(2026-09-12 실측).** admin 루프가 거버넌스 파트너인데 etcd 가
안 돌면 `EtcdStart()` 를 부른다(`wemix/admin.go:731`):

```go
if ma.contracts != nil && ma.nodeInfo != nil {
    ma.update()
    if ma.amPartner() && ma.self != nil && !ma.etcdIsRunning() {
        EtcdStart()
    }
}
```

`etcdStart()` 는 `etcdNewConfig(false)` 로 시작하는데, 그 설정은 `ClusterState = existing`
이면서 **`InitialCluster` 에 자기 자신만** 적는다(`etcdutil.go:118-148`). 데이터 디렉토리에
멤버 기록이 없는 상태로 이 설정을 쓰면 etcd 는 클러스터 정보를 가져올 곳이 없다 —
`cannot fetch cluster info from peer urls` 가 그 문장이다.

> **실측 정정 (2026-09-12). 이 줄은 붕괴의 신호가 아니다 — 건강한 브링업에서도 매번 나온다.**
> 15대 docker 에 13 bp + 2 en 웹믹스를 올려 완주시킨 실행에서, **13개 bp 전부가**
> `etcd failed to start: cannot fetch cluster info from peer urls` 를 **정확히 한 번** 찍고
> 곧바로 `etcd started server` · `etcd server ready` 로 이어졌다. en 2대는 둘 다 찍지 않는다
> (etcd 멤버가 아니다). 체인은 블록을 생산했고 스펙은 통과했다.
>
> | 노드 | `etcd failed to start` | `etcd server ready` |
> |---|---|---|
> | node1~node13 (bp) | 각 1회 | 각 1회 |
> | node14·node15 (en) | 0 | 0 |
>
> **왜 그런가**: admin 루프의 `EtcdStart()` 는 거버넌스 파트너이고 etcd 가 안 돌면 무조건
> 불리는데, 그 설정(`ClusterState = existing` + `InitialCluster` 에 자기 자신만)은 **한 번도
> 멤버였던 적 없는 노드에서는 반드시 실패한다.** 클러스터를 실제로 만드는 것은 그 뒤의
> `etcdInit` 이다. 즉 이 실패는 **정상 경로의 일부**다.
>
> **체인팀에게 중요한 점 둘.** (가) 이 줄로 붕괴를 감지하려 하면 **건강한 실행마다 걸린다** —
> 탐지 신호로 쓸 수 없다. 붕괴를 가리는 것은 이 줄이 아니라 그 뒤에 `etcd server ready` 가
> **오지 않는 것**이다. (나) 위의 "재시작 뒤에 나는 문장" 이라는 설명은 좁았다. 재시작에서도
> 나지만, **첫 기동에서도 난다.**

그 다음 `EtcdStart` 는 `go admin.etcdAutoJoin()` 으로 넘어가고, `etcdAutoJoin` 은 `up` 이면서
`MiningPeers` 에 `*` 가 있는 다른 miner 를 찾는다. 없으면 `ErrNotFound` 를 남긴다
(`etcdutil.go:503-512`):

```go
for _, s := range getMiners("", 0) {
    if s.NodeName != admin.self.Name && s.Status == "up" && strings.Contains(s.MiningPeers, "*") {
        state = s
        break
    }
}
if state == nil {
    err := ErrNotFound
    log.Info("etcd join failed", "name", admin.self.Name, "error", err)
```

**그래서 `etcd join failed … "not found"` 는 원인이 아니라 결과다.** "join 하려다 형성된
클러스터를 잃었다" 는 chainbench 워크리스트의 이전 판독은 틀렸다 — join 경로는
`etcdIsRunning()` 으로 가드되어 있어 돌고 있는 로컬 etcd 를 내릴 수 없다. 관측된 문장이
말하는 것은 **etcd 를 돌리고 있다고 광고하는 피어가 하나도 없다**는 사실이고, 부트가 그
광고를 잃은 이유는 이 문장 밖에 있다.

### 영향

poa(wemix) 원격 브링업의 **재현 신뢰성**이다. 성공하면 14멤버 형성과 sealer 로테이션까지
정상 완주하므로 기능 결함이 아니라 안정성 결함이고, 2회 중 1회라 이 위에 올린 테스트는
간헐적으로 무너진다.

### chainbench 가 배제한 것 [High]

키와 genesis 는 원인이 아니다 — genesis 해시가 전 노드 동일하고, 배포된 키 경로를 노드의
command 가 실제로 쓰는 것을 확인했다. 브링업 순서도 원인이 아니다: 직렬 브링업(부트를 최상위
producer 로 두고 나머지를 내림차순으로 한 대씩 시작+join)으로 late-join 노드의 fork 경합
("unauthorized block")은 사라졌고, 그 뒤에 남은 것이 이 현상이다.

### 제안

**먼저 판정해야 할 것은 프로세스가 죽었는지다.** 위 (1)(2) 가 그것을 단일 질문으로 줄여
준다 — 붕괴한 실행에서 부트 노드의 프로세스 시작 시각(또는 재시작 횟수)과 로그 앞부분의
패닉 유무를 보면 두 갈래 중 어느 쪽인지 바로 갈린다. W1 이 그 죽음이면 W2 는 별도 결함이
아니라 W1 의 증상이고, 아니면 아래가 남는다.

프로세스가 죽지 않았는데도 재현되면, 두 가지가 후보다.

- **`etcdStart()` 의 설정이 신규 노드에 대해 성립하지 않는다.** `ClusterState = existing` +
  `InitialCluster = self` 는 데이터 디렉토리에 멤버 기록이 있는 **재시작**만 성립한다. 한
  번도 멤버였던 적 없는 노드에는 구조적으로 실패하는 호출이라, admin 루프가 그 노드에
  대해 이것을 반복해서 부르는 것이 맞는지 판정이 필요하다. 트레이드오프: 여기서 거르면
  "아직 멤버가 아니다" 와 "멤버인데 못 붙는다" 를 구분해야 하고, 그 구분이 거버넌스 멤버
  목록과 etcd 멤버 목록 중 무엇을 진실로 볼지를 정하는 일이다.
- **붕괴 감지 후 `etcdInit` 재발행을 브링업에 넣는 것**은 권장하지 않는다. 형성 중의 재발행이
  오히려 형성을 방해한 선례가 있다(chainbench 실측). 재발행이 답이라면 "형성 완료" 를 먼저
  단정할 수 있어야 하고, 그 단정은 지금 `ma.etcd != nil` 보다 강한 것이어야 한다.

**진단 가능성 자체를 고치는 것도 권장한다.** `etcdInfo()` 가 `ma.etcd == nil` 일 때 에러 값을
결과로 돌려주는 바람에, RPC 표면에서 이 상태는 **응답이 망가진 것과 구분되지 않는다**(`{}`).
"etcd 핸들이 없다" 를 이름 있는 필드로 돌려주면 이 조사는 한 번의 호출로 끝났을 것이다.

---

## 3. B1 — `istanbul_getWbftExtraInfo` 가 블록 태그를 못 푼다 [권장]

### 증상

같은 체인에서, 16진수 블록 번호를 주면 정상 응답하고 `"latest"` 를 주면 실패한다.

```bash
# 정상 — 16진수 블록 번호
curl -s -X POST -H 'Content-Type: application/json' \
  --data '{"jsonrpc":"2.0","id":1,"method":"istanbul_getWbftExtraInfo","params":["0xaa"]}' \
  http://<node>:8545
# → {"result":{"stabilizing":false,"validators":[...],"stakers":[...]}}

# 실패 — 같은 노드, 같은 체인
curl -s -X POST -H 'Content-Type: application/json' \
  --data '{"jsonrpc":"2.0","id":1,"method":"istanbul_getWbftExtraInfo","params":["latest"]}' \
  http://<node>:8545
# → {"error":{"message":"block is not a wbft block"}}   (wbftcommon.ErrIsNotWBFTBlock)
```

### 원인 [High]

`consensus/wbft/backend/api.go:417-421` 이 태그를 풀지 않고 블록 번호로 쓴다:

```go
func (api *API) GetWbftExtraInfo(number rpc.BlockNumber) (map[string]interface{}, error) {
	bNumber := big.NewInt(int64(number))

	if !api.chain.Config().IsCroissant(bNumber) {
		return nil, wbftcommon.ErrIsNotWBFTBlock
	}
```

`rpc.BlockNumber` 는 태그를 **음수 센티널**로 표현한다(`latest` = -1 등). 그 음수가 그대로
블록 번호가 되어 `IsCroissant` 가 거짓이 되고, 함수는 "wbft 블록이 아니다" 로 나간다.
이어지는 `api.chain.GetHeaderByNumber(uint64(number))` 도 음수를 uint64 로 변환한다.

**오류 메시지가 원인을 가린다.** 태그를 못 푼 것인데 체인·블록에 대한 사실처럼 말한다.
같은 체인의 같은 블록이 번호로는 응답하므로, 읽는 사람은 체인을 의심하게 된다.

**고칠 모양이 같은 파일 안에 이미 있다.** 285줄 위 `GetValidators` 는 포인터로 받고 태그를
먼저 판정한다(`api.go:132-139`):

```go
func (api *API) GetValidators(number *rpc.BlockNumber) ([]common.Address, error) {
	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = api.chain.CurrentHeader()
	} else {
		header = api.chain.GetHeaderByNumber(uint64(number.Int64()))
	}
```

`GetCommitSignersFromBlock`(86), `calculateBlockRange`(228), `IsValidator`(347) 도 같은
형태다. `GetWbftExtraInfo` 만 값으로 받고 판정을 생략한다.

### 영향

기능 결함은 아니다 — 번호를 주면 동작한다. 영향은 **쓰는 사람 쪽**이다. 같은 파일의 다른
`istanbul_*` 는 `"latest"` 를 받으므로 같은 습관으로 이 메서드를 쓰면 실패하고, 실패 문장이
원인을 가린다.

chainbench 는 **클라이언트 쪽에서 태그를 풀어** 우회하고 있다. DSL 에
`"@latest"` 센티넬이 있고(`internal/testhelper/derived.go:235`), 호출 시점에
`eth_blockNumber` 의 결과(16진수)로 치환한다. `waitFor` 안에서는 폴링마다 다시 치환되므로
머리가 올라가는 것을 따라간다. 이 메서드를 쓰는 기존 스펙들
(`tests/tc/basic/07-basic-wbft-consensus.json` 등)이 모두 그 센티넬을 쓴다 — 그 센티넬이
존재하는 이유 자체가 이 결함이다. 코드 주석에도 그렇게 적혀 있다.

### 제안

`GetValidators` 와 같은 형태로 맞추는 것이 최소 변경이다 — 포인터로 받고, nil 또는
`rpc.LatestBlockNumber` 면 `CurrentHeader()` 를 쓴다. 태그를 계속 받지 않기로 한다면 최소한
**오류 문장이 태그를 가리켜야** 한다. 지금 문장은 호출자가 고칠 수 없는 곳을 가리킨다.

---

## 4. 재현 환경

세 건 모두 **15 컨테이너 docker 환경**에서 나왔다. 한 서버(컨테이너)당 wemix 노드 1대와
wbft 노드 1대가 서로 다른 포트 대역에 올라간다.

- W1 · B1: wemix→wbft 핸드오프, producer 15 + successor 15. 프로파일과 검증된 호출은
  [`profiles/wemix-upgrade-15.yaml`](../../profiles/wemix-upgrade-15.yaml) 머리말에 그대로
  적혀 있다(`chainbench upgrade run --profile … --all-servers --docker`).
- W2: poa(wemix) 단독 원격 브링업, 14 bp + 1 en. chainbench 워크리스트 R6.

환경을 세우는 법은 [`../guide/config-files.md`](../guide/config-files.md) 와
`env/docker/` 다. 신원은 `keys/preset`(5노드)이 아니라 생성한 30노드 키 세트를 쓴다
(`chainbench validator set --nodes 30 --validators 15 --out <dir>`).

> `keys/preset` 의 키는 **테스트 픽스처 전용**이다. 그 키나 거기서 파생된 주소를 공용
> 네트워크로 옮기지 않는다 — [`../SECURITY_KEY_HANDLING.md`](../SECURITY_KEY_HANDLING.md).

---

## 5. chainbench 쪽에 남는 일

없다. 세 건 모두 체인 저장소에서 고치는 것이 맞다는 판정이 끝났고, chainbench 쪽 대응은
이미 들어가 있다.

- B1 은 DSL 의 `"@latest"` 센티넬로 우회하고 있다 — 클라이언트가 태그를 풀어 16진수 번호로
  보낸다. 체인 쪽이 고쳐지면 센티넬은 남겨도 무해하지만(머리를 매 폴링 재해석하는 용도가
  따로 있다) **그것이 존재하는 이유**는 없어진다.
- W1 은 우회할 수 없다. 노드가 죽으면 mesh 준비 게이트가 막는 것이 의도된 동작이다.
- W2 는 브링업 순서로 줄일 수 있는 부분(fork 경합)까지 줄인 상태다.

세 건의 원래 관측 기록은 [`chainbench-worklist.md`](chainbench-worklist.md) 에 있고, 이
문서가 그 기록의 인계본이다. 워크리스트가 작업 상태의 정본이므로, 체인팀 쪽에서 진행되면
그쪽 이슈 번호를 워크리스트 항목에 붙인다.
