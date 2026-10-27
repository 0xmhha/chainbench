package app

import "slices"

// WebOperation is one job operation the chain engine runs. Scope "test" marks
// the operations the test screen offers; the chain screen offers all of them.
// Command is the chainbench command it runs, and Arguments are exactly the
// plan arguments it reads; a plan naming any other argument is refused.
type WebOperation struct {
	ID          string        `json:"id"`
	Label       string        `json:"label"`
	Description string        `json:"description,omitempty"`
	Scope       string        `json:"scope"`
	Command     string        `json:"command"`
	Arguments   []WebArgument `json:"arguments"`
}

// WebArgument is one plan argument an operation reads and what it becomes on
// the chainbench command line. Flag is the flag the value is passed as, or
// empty when the engine uses the value without passing it on; CLI says how.
type WebArgument struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Minimum     *int64 `json:"minimum,omitempty"`
	Maximum     *int64 `json:"maximum,omitempty"`
	Flag        string `json:"flag,omitempty"`
	CLI         string `json:"cli"`
}

// WebJobsContract lists the operations a plan may name, so screens never keep
// their own copy of the list.
type WebJobsContract struct {
	Operations []WebOperation `json:"operations"`
}

func bound(n int64) *int64 { return &n }

// webArgument names an argument once; each operation says what it becomes.
func webArgument(name, flag, cli string) WebArgument {
	a := webArgumentKinds[name]
	a.Name, a.Flag, a.CLI = name, flag, cli
	return a
}

var webArgumentKinds = map[string]WebArgument{
	"manifestId":                {Type: "string", Title: "체인 manifest"},
	"assetId":                   {Type: "string", Title: "바이너리", Description: "등록하고 검증한 노드 바이너리입니다."},
	"serverRef":                 {Type: "string", Title: "서버", Description: "workspace가 고정한 서버 풀의 서버 이름입니다."},
	"allServers":                {Type: "boolean", Title: "모든 서버", Description: "서버 하나 대신 서버 풀의 모든 서버를 씁니다. serverRef와 함께 쓸 수 없습니다."},
	"validators":                {Type: "integer", Title: "블록 생산자 수", Minimum: bound(1), Maximum: bound(128)},
	"chainPresetRef":            {Type: "documentRef", Title: "체인 구성", Description: "저장한 체인 프리셋의 revision입니다. 노드 배치를 프리셋이 정하므로 validators와 함께 쓸 수 없습니다."},
	"replacementAssetId":        {Type: "string", Title: "교체할 바이너리"},
	"configOverrides":           {Type: "stringMap", Title: "바꿀 설정", Description: "체인 프리셋 계약의 설정 키와 값입니다. configAssetId와 함께 쓸 수 없습니다."},
	"configAssetId":             {Type: "string", Title: "설정 파일", Description: "등록한 설정 파일입니다."},
	"caseRefs":                  {Type: "documentRefs", Title: "케이스", Description: "저장한 케이스의 revision을 실행 순서대로 적습니다."},
	"runDefinitionRef":          {Type: "documentRef", Title: "실행 목록", Description: "저장한 실행 목록의 revision입니다. caseRefs와 함께 쓸 수 없습니다."},
	"binaryAssets":              {Type: "stringMap", Title: "추가 바이너리", Description: "케이스가 default 밖에 선언한 바이너리 이름마다 등록 바이너리를 고릅니다."},
	"noSkips":                   {Type: "boolean", Title: "건너뛴 케이스를 실패로", Description: "건너뛴 케이스가 하나라도 있으면 작업이 실패합니다."},
	"waitBlocks":                {Type: "integer", Title: "시작 전 대기 높이", Description: "체인 머리가 이 높이에 닿은 뒤 케이스를 시작합니다.", Minimum: bound(1), Maximum: bound(1_000_000)},
	"chainId":                   {Type: "integer", Title: "체인 id", Description: "genesis의 체인 id를 이 값으로 바꿉니다. 비우면 manifest의 체인 id입니다. 체인 구성을 고르면 쓸 수 없습니다.", Minimum: bound(1), Maximum: bound(9007199254740991)},
	"nodeMonitorTimeoutSeconds": {Type: "integer", Title: "노드 준비 대기(초)", Description: "아직 올라오는 노드를 준비 관문이 기다리는 시간입니다. 비우면 엔진 기본값을 씁니다.", Minimum: bound(1), Maximum: bound(3600)},
}

func composeArguments() []WebArgument {
	return []WebArgument{
		webArgument("manifestId", "", "내장 manifest면 --chain <id>로, 등록한 외부 manifest면 고정한 파일을 --manifest와 --genesis-template으로 넘깁니다."),
		webArgument("assetId", "--binary", "검증한 바이너리를 대상에 둔 경로를 넘깁니다."),
		webArgument("serverRef", "--server", "서버 이름을 그대로 넘깁니다."),
		webArgument("allServers", "--all-servers", "켜면 붙입니다."),
		webArgument("validators", "--bp", "체인 구성을 고르지 않았을 때 넘깁니다. 비우면 4입니다."),
		webArgument("chainPresetRef", "--chain-preset", "고정한 revision의 내용을 파일로 써서 넘깁니다."),
		webArgument("chainId", "--chain-id", "체인 구성을 고르지 않았을 때 그대로 넘깁니다."),
	}
}

func controlArguments() []WebArgument {
	return []WebArgument{
		webArgument("manifestId", "", "명령에는 넘기지 않습니다. 계획에서 이 manifest로 바이너리를 검증합니다."),
		webArgument("assetId", "", "명령에는 넘기지 않습니다. 계획에서 검증합니다."),
		webArgument("serverRef", "", "명령에는 넘기지 않습니다. 기록에서 이 서버의 노드를 찾고, 이 서버에 연결한 내 로그인을 --server-logins 파일에 넣습니다."),
		webArgument("allServers", "", "명령에는 넘기지 않습니다. 여러 서버에 걸친 기록을 서버마다 확인합니다."),
		webArgument("validators", "", "명령에는 넘기지 않습니다. 적으면 기록된 생산자 수와 같아야 합니다."),
	}
}

var webOperations = []WebOperation{
	{"chain.setup", "설정 생성 · 노드 초기화", "설정과 genesis를 만들고 노드 데이터베이스를 초기화합니다. 노드는 멈춘 상태로 남습니다.", "chain", "chainbench chain up --stage init", composeArguments()},
	{"chain.deploy", "노드 구축 · 실행", "설정을 만들고 노드를 초기화한 뒤 실행합니다.", "chain", "chainbench chain up --stage start", composeArguments()},
	{"node.start", "노드 시작", "기록된 노드 하나를 시작합니다.", "chain", "chainbench node start --index N", controlArguments()},
	{"node.stop", "노드 정지", "기록된 노드 하나를 멈춥니다.", "chain", "chainbench node stop --index N", controlArguments()},
	{"node.restart", "노드 재실행 · 자료 보존", "노드 자료를 그대로 두고 다시 실행합니다.", "chain", "chainbench chain restart --node N", controlArguments()},
	{"node.swap", "설정·바이너리 교체 · 재실행", "노드의 설정이나 바이너리를 바꾸고 다시 실행합니다.", "chain", "chainbench node swap --index N --purpose web-reviewed-config", append(controlArguments(),
		webArgument("replacementAssetId", "--binary", "검증한 교체 바이너리를 대상에 둔 경로를 넘깁니다."),
		webArgument("configOverrides", "--config", "기록된 실행 옵션과 맞춘 key=value를 하나씩 넘깁니다."),
		webArgument("configAssetId", "--config", "등록 설정 파일의 값을 key=value로 바꿔 하나씩 넘깁니다."),
	)},
	{"node.reset", "비생산자 초기화 · 정지 상태로 유지", "블록 생산자가 아닌 노드의 자료를 지우고 멈춘 상태로 둡니다.", "chain", "chainbench node reset --index N", controlArguments()},
	{webNetworkMonitorOperation, "SSH 노드 로그 수집 · 취소할 때까지", "기록된 네트워크의 노드 로그를 취소할 때까지 모읍니다.", "chain", "chainbench resource inspect --request <노드마다>", []WebArgument{
		webArgument("serverRef", "", "이 서버의 기록된 노드 로그를 읽습니다. 이 서버에 연결한 내 로그인을 --server-logins 파일에 넣습니다."),
		webArgument("allServers", "", "서버 풀의 모든 서버에서 기록된 노드 로그를 읽습니다."),
	}},
	{"test.run", "저장한 DSL 케이스 실행", "저장한 케이스의 구성 선언으로 네트워크를 만들고 실행합니다.", "test", "chainbench run <케이스 파일...> --suite", []WebArgument{
		webArgument("manifestId", "", "명령에는 넘기지 않습니다. --chain은 바이너리를 검증한 체인입니다."),
		webArgument("assetId", "--binary", "검증한 바이너리 경로를 --binary와 --binary-for default=로 넘깁니다."),
		webArgument("serverRef", "--server", "서버 이름을 그대로 넘깁니다."),
		webArgument("caseRefs", "", "고정한 revision의 케이스를 파일로 써서 위치 인자로 넘깁니다."),
		webArgument("runDefinitionRef", "", "실행 목록이 고정한 케이스를 순서대로 파일로 써서 위치 인자로 넘깁니다."),
		webArgument("binaryAssets", "--binary-for", "이름마다 검증한 바이너리 경로를 name=path로 넘깁니다."),
		webArgument("noSkips", "--no-skips", "켜면 붙입니다."),
		webArgument("waitBlocks", "--wait-blocks", "높이를 그대로 넘깁니다."),
		webArgument("nodeMonitorTimeoutSeconds", "--node-monitor-timeout", "초를 붙여(300s) 넘깁니다."),
	}},
	{webTestAttachOperation, "기록된 네트워크에 연결해 케이스 실행", "Web이 배포하고 기록한 네트워크에 연결해 케이스를 실행합니다.", "test", "chainbench run <케이스 파일...> --attach --no-node-control --read-only-keys", []WebArgument{
		webArgument("manifestId", "", "명령에는 넘기지 않습니다. 계획에서 이 manifest로 바이너리를 검증합니다."),
		webArgument("assetId", "", "명령에는 넘기지 않습니다. 기록된 바이너리와 같아야 합니다."),
		webArgument("serverRef", "", "명령에는 넘기지 않습니다. 기록에서 이 서버의 노드를 찾고, 이 서버에 연결한 내 로그인을 --server-logins 파일에 넣습니다."),
		webArgument("allServers", "", "명령에는 넘기지 않습니다. 여러 서버에 걸친 기록을 서버마다 확인합니다."),
		webArgument("caseRefs", "", "고정한 revision의 케이스를 파일로 써서 위치 인자로 넘깁니다."),
	}},
}

// JobsContract returns a detached copy of the operation list.
func JobsContract() WebJobsContract {
	ops := slices.Clone(webOperations)
	for i := range ops {
		ops[i].Arguments = slices.Clone(ops[i].Arguments)
	}
	return WebJobsContract{Operations: ops}
}

func webKnownOperation(operation string) bool {
	return slices.ContainsFunc(webOperations, func(o WebOperation) bool { return o.ID == operation })
}

// webOperationReads reports whether operation reads the named argument.
func webOperationReads(operation, name string) bool {
	i := slices.IndexFunc(webOperations, func(o WebOperation) bool { return o.ID == operation })
	return i >= 0 && slices.ContainsFunc(webOperations[i].Arguments, func(a WebArgument) bool { return a.Name == name })
}
