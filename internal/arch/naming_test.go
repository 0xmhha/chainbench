package arch

import (
	"sort"
	"testing"
)

// nameShared holds names that more than one package declares on purpose.
//
// The line drawn here is between a STRUCTURAL role and DOMAIN vocabulary. A
// structural name says what a declaration is to its own package ("this
// package's dependencies", "this call's options"), and it only means anything
// with the package in front of it, so `collector.Options` and `health.Options`
// are two packages each naming their own thing correctly. Domain vocabulary
// names something in the chain, the node, the key or the resource, and there
// [[layers]] §5b applies: one concept keeps one name, and a second concept has
// to find another. Those go in nameCollisionDebt below.
//
// Verbs sit here for a different reason. `dsl.Parse` and `resource.Parse` read
// as English at the call site the way `json.Marshal` and `xml.Marshal` do; the
// package is part of the sentence rather than a disambiguator bolted on.
var nameShared = map[string]string{
	// 동사다. 호출하는 자리에 패키지 이름이 늘 앞에 붙어 문장이 된다.
	"Build":            "genesis 를 만드는 일과 report 를 만드는 일이다",
	"BuildPlan":        "핸드오프 계획과 하드포크 계획을 각각 세운다",
	"Compose":          "genesis 를 조립하는 일과, preflight 가 '아직 아무것도 조립되지 않았다'고 말하는 단계다",
	"DefaultKeySetDir": "app 에서 operation 을 거쳐 store 까지 그대로 전달한다",
	"Generate":         "키셋을 만드는 일과 리포트를 만드는 일이다",
	"List":             "키셋이 무엇을 담았는지와, 아티팩트 루트에 어떤 세션이 있는지를 각각 센다",
	"Load":             "토폴로지·외부 플러그인·검증자 명부를 각각 읽는다",
	"Parse":            "테스트 스펙을 읽는 일과 서버 지정자를 읽는 일이다",
	"Register":         "체인 플러그인을 등록하는 일과 DSL 어휘를 등록하는 일이다",
	"Resolve":          "설정 계층을 겹치는 일과 네트워크 ID 를 확정하는 일이다",

	// 구조적 역할 이름이다. 패키지가 앞에 붙어야 뜻이 완성되므로 겹치는 것이 정상이다.
	"Config":  "각 패키지가 자기 설정 구조체를 갖는다",
	"Deps":    "각 패키지가 자기 경계에서 받는 의존을 선언한다. 패키지 전역 상태를 두지 않기로 한 결과다",
	"Inputs":  "각 패키지가 자기 호출의 입력을 선언한다",
	"Options": "각 패키지가 자기 호출의 선택 인자를 선언한다",
	"Request": "각 패키지가 자기 요청 구조체를 갖는다",
	"Result":  "각 패키지가 자기 결과 구조체를 갖는다",

	// 계층을 넘기며 같은 상수를 다시 내놓는다.
	"KeySetEnv":     "app 과 operation 이 store 의 환경변수 이름을 표면 도움말용으로 되비친다",
	"GenesisParams": "registry 가 wbft 의 파라미터를 되비쳐, core 가 패밀리를 import 하지 않게 한다. 주석에 그 이유가 적혀 있다",
}

// nameCollisionDebt holds domain words that two concepts are currently sharing.
// Each entry names what the two things actually are, and where one can, the
// item that renames one of them. It may only shrink: a name that stops
// colliding must leave this map, so the list tracks the code rather than
// drifting above it.
//
// This is the A7 measurement, taken 2026-09-07 with [Collisions].
var nameCollisionDebt = map[string]string{
	"Runner":      "poa 는 명령을 실행하는 것을, process 는 원격 셸을 실행하는 것을 가리킨다",
	"Handler":     "S 트랙 — registry 와 mcp 가 같은 시그니처를 따로 선언한다. MCP 스키마를 줄일 때 하나로 모은다",
	"NewServer":   "S 트랙 — dashboard 와 mcp 가 각자 서버를 만든다",
	"Server":      "S 트랙 — 위 둘에 resource 의 '노드가 실제로 도는 기계'까지 셋이다",
	"Fingerprint": "B2 — session 의 타입과 interp 의 생성 함수다. 타입이 string 으로 돌아가면 함수만 남는다",
	"Network":     "app 은 붙여 둔 네트워크를 읽고, keyring 은 프리셋의 체인 파라미터를 가리킨다",
	"Chain":       "app 은 플러그인을 찾아 주고, nodeconfig 는 설정 구조체다",
	"GenerateKey": "accounts 것은 저장하지 않는 테스트용이라 같은 이름을 쓰면 안 된다",
	"Identity":    "derive 는 드러내도 되는 파생 신원을, nodeconfig 는 노드가 누구인지를 가리킨다",
	"Node":        "app 은 core/node 의 별칭이지만 preflight 것은 '원하는 역할을 낼 수 있는가'의 판정 대상이다",
	"NodeSpec":    "upgrade 는 확정된 기동 배정을, process 는 프로비전·기동에 필요한 것을 가리킨다",
	"NodeSwap":    "chainsetup 은 노드를 바꾸는 동작이고 hardfork 는 그 교체를 적은 구조체다",
	"Step":        "chainsetup 은 session.Step 의 별칭인데 poa 것은 부트스트랩의 한 동작이다",
	"Entry":       "arch 는 등록된 기능을, keyring 은 키 항목을, node 는 부트 항목을 가리킨다",
	"Account":     "poa 는 제네시스 선충전 계정을, validatorset 은 역할이 붙은 계정을, testhelper 는 DSL 이 쓰는 계정을 가리킨다",
	"Label":       "core/node 의 주석이 이미 '가끔 철자가 겹치는 다른 개념'이라고 적어 두었다",
	"Spec":        "노드 하나의 설정, 테스트 정의, 서버 지정자. 셋이 서로 남이다",
	"Plan":        "핸드오프 계획, 하드포크 계획, 기동 계획, 그리고 resource 의 포트 배치 함수다",
	"Report":      "app 의 조회, health 의 검증 결과, report 의 세션 리포트다",
	"Kind":        "collector 는 이벤트 종류를, resource 는 서버가 도는 방식을 가리킨다",
	"Phase":       "collector 는 파이프라인 구간을, registry 는 먼저 끝나야 하는 동작 묶음을 가리킨다",
	"Source":      "genesis 는 extraData 를 내놓는 것을, keyring 은 키를 내놓는 것을 가리킨다",
	"Store":       "collector 는 이벤트 저장소를, filestore 는 파일 저장소를 가리킨다",
	"Host":        "inspector 는 문을 두드릴 대상을, resource 는 주소가 붙은 기계를 가리킨다",
	"Ports":       "inspector 는 포트를 확인하는 동작이고 resource 는 배정된 포트 묶음이다. 08-25 에도 진짜 신호로 지목된 자리다",
	"Auth":        "core/node 와 core/remote 가 같은 map[string]any 를 각자 선언한다. 하나로 모을 수 있다",
	"Opener":      "operation 은 필요한 것만 추린 인터페이스이고 resource 것은 구현체다",
	"Registry":    "interp 는 어휘 레지스트리 인터페이스이고 testhelper 는 그것을 만들어 주는 함수다",
	"Lookup":      "registry 는 기능을, assert 는 단언을 찾고, resource 것은 자격증명을 찾는 함수 타입이다",
	"Defaults":    "nodeconfig 는 기본값을 만드는 함수이고 resource 는 기본값 구조체다",
	"Inventory":   "chainsetup 은 재고를 모으는 함수이고 resource 는 그 재고다",
	"Verdict":     "preflight 는 얼마나 다시 지어야 하는지를, nodemonitor 는 게이트가 다음에 무엇을 할지를 가리킨다",
	"Flag":        "resource 는 기동 플래그를 만들고, 도구는 선언된 플래그 변수를 가리킨다",
}

// TestNamesDoNotCollide is A7: an exported name declared at package level in
// more than one package is reported unless something accounts for it.
//
// It counts declarations rather than every identifier because a method is
// namespaced by its receiver. Including methods put 577 more names in the tally
// and buried `RoleValidator` among `Stop` and `Save`, which is how a test stops
// being read.
func TestNamesDoNotCollide(t *testing.T) {
	seenShared, seenDebt := map[string]bool{}, map[string]bool{}
	var explained int

	for _, c := range Collisions(moduleRoot) {
		if why := c.Explained(); why != "" {
			explained++
			continue
		}
		switch {
		case nameShared[c.Name] != "":
			seenShared[c.Name] = true
		case nameCollisionDebt[c.Name] != "":
			seenDebt[c.Name] = true
		default:
			t.Errorf("%s is declared in %v — one concept keeps one name ([[layers]] §5b); rename one, or record why both keep it in nameShared", c.Name, c.Pkgs)
		}
	}

	// A ratchet has to hold in both directions. An entry whose collision is
	// gone is a claim about the code that is no longer true, and leaving it
	// means the next reader trusts a list that has drifted.
	for _, m := range []struct {
		name string
		list map[string]string
		seen map[string]bool
	}{{"nameShared", nameShared, seenShared}, {"nameCollisionDebt", nameCollisionDebt, seenDebt}} {
		for name := range m.list {
			if !m.seen[name] {
				t.Errorf("%s[%q] matches no collision — the name is now unique, so remove the entry", m.name, name)
			}
		}
	}

	t.Logf("%d shared names are accounted for by rule, %d are tolerated, %d are debt",
		explained, len(seenShared), len(seenDebt))
	if testing.Verbose() {
		var debt []string
		for n := range seenDebt {
			debt = append(debt, n)
		}
		sort.Strings(debt)
		for _, n := range debt {
			t.Logf("  %s: %s", n, nameCollisionDebt[n])
		}
	}
}
