package app

import "slices"

// WebOperation is one job operation the chain engine runs. Scope "test" marks
// the operations the test screen offers; the chain screen offers all of them.
type WebOperation struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Scope       string `json:"scope"`
}

// WebJobsContract lists the operations a plan may name, so screens never keep
// their own copy of the list.
type WebJobsContract struct {
	Operations []WebOperation `json:"operations"`
}

var webOperations = []WebOperation{
	{"chain.setup", "설정 생성 · 노드 초기화", "설정과 genesis를 만들고 노드 데이터베이스를 초기화합니다. 노드는 멈춘 상태로 남습니다.", "chain"},
	{"chain.deploy", "노드 구축 · 실행", "설정을 만들고 노드를 초기화한 뒤 실행합니다.", "chain"},
	{"node.start", "노드 시작", "기록된 노드 하나를 시작합니다.", "chain"},
	{"node.stop", "노드 정지", "기록된 노드 하나를 멈춥니다.", "chain"},
	{"node.restart", "노드 재실행 · 자료 보존", "노드 자료를 그대로 두고 다시 실행합니다.", "chain"},
	{"node.swap", "설정·바이너리 교체 · 재실행", "노드의 설정이나 바이너리를 바꾸고 다시 실행합니다.", "chain"},
	{"node.reset", "비생산자 초기화 · 정지 상태로 유지", "블록 생산자가 아닌 노드의 자료를 지우고 멈춘 상태로 둡니다.", "chain"},
	{webNetworkMonitorOperation, "SSH 노드 로그 수집 · 취소할 때까지", "기록된 네트워크의 노드 로그를 취소할 때까지 모읍니다.", "chain"},
	{"test.run", "저장한 DSL 케이스 실행", "저장한 케이스의 구성 선언으로 네트워크를 만들고 실행합니다.", "test"},
	{webTestAttachOperation, "기록된 네트워크에 연결해 케이스 실행", "Web이 배포하고 기록한 네트워크에 연결해 케이스를 실행합니다.", "test"},
}

// JobsContract returns a detached copy of the operation list.
func JobsContract() WebJobsContract { return WebJobsContract{Operations: slices.Clone(webOperations)} }

func webKnownOperation(operation string) bool {
	return slices.ContainsFunc(webOperations, func(o WebOperation) bool { return o.ID == operation })
}
