package app

import (
	"github.com/0xmhha/chainbench/internal/core/remote"
	"github.com/0xmhha/chainbench/internal/resource"
)

// deploymentText is what the editor shows for one contract field: a label,
// and when the name alone does not say it, what the engine does with the value.
type deploymentText struct{ title, description string }

// deploymentFieldText names every field of the deployment contract by its path
// (kind, then keys; [] is an array item and {} a map entry). The words are the
// screen's language; the field names stay the engine's.
var deploymentFieldText = map[string]deploymentText{
	"server-set":                          {"서버 풀", "노드를 놓을 수 있는 주소와, 주소마다 몇 개의 포트 칸을 쓸지 정합니다."},
	"server-set.version":                  {"형식 버전", ""},
	"server-set.pool":                     {"풀", ""},
	"server-set.pool.hosts":               {"서버", "노드를 이 순서대로 놓습니다."},
	"server-set.pool.hosts[]":             {"서버", ""},
	"server-set.pool.hosts[].name":        {"이름", "작업과 경로가 이 서버를 부르는 이름입니다. 비우면 주소가 이름이 됩니다."},
	"server-set.pool.hosts[].addr":        {"주소", "IP 주소나 DNS 이름입니다. 루프백 주소면 이 컴퓨터에서 실행합니다."},
	"server-set.pool.slots":               {"서버당 노드 수", "한 서버가 포트를 바꿔 가며 띄울 수 있는 노드 수입니다."},
	"server-set.pool.ports":               {"포트 대역", "노드 칸마다 시작 포트에서 간격만큼 올라갑니다."},
	"server-set.pool.ports.p2p":           {"P2P", ""},
	"server-set.pool.ports.p2p.base":      {"P2P 시작 포트", ""},
	"server-set.pool.ports.p2p.step":      {"P2P 간격", ""},
	"server-set.pool.ports.rpc":           {"RPC", "ws와 auth 대역을 따로 두지 않으면 RPC 간격 안에서 함께 씁니다."},
	"server-set.pool.ports.rpc.base":      {"RPC 시작 포트", ""},
	"server-set.pool.ports.rpc.step":      {"RPC 간격", ""},
	"server-set.pool.ports.ws":            {"WebSocket", "따로 두지 않으면 RPC 대역에서 이어 씁니다."},
	"server-set.pool.ports.ws.base":       {"WebSocket 시작 포트", ""},
	"server-set.pool.ports.ws.step":       {"WebSocket 간격", ""},
	"server-set.pool.ports.auth":          {"인증 RPC", "따로 두지 않으면 RPC 대역에서 이어 씁니다."},
	"server-set.pool.ports.auth.base":     {"인증 RPC 시작 포트", ""},
	"server-set.pool.ports.auth.step":     {"인증 RPC 간격", ""},
	"server-set.pool.ports.metrics":       {"지표", "간격이 0이면 서버마다 첫 노드만 지표 포트를 엽니다."},
	"server-set.pool.ports.metrics.base":  {"지표 시작 포트", ""},
	"server-set.pool.ports.metrics.step":  {"지표 간격", ""},
	"server-set.ssh":                      {"SSH 연결", "루프백이 아닌 모든 서버에 쓰는 연결 설정입니다. 로그인 비밀은 개인 자격증명으로 따로 연결합니다."},
	"server-set.ssh.user":                 {"SSH 사용자", "개인 자격증명의 사용자가 이 값보다 먼저 쓰입니다."},
	"server-set.ssh.port":                 {"SSH 포트", ""},
	"server-set.ssh.known_hosts_file":     {"known_hosts 파일", "서버의 호스트 키를 이 파일로 확인합니다. 비우면 ~/.ssh/known_hosts를 씁니다."},
	"server-set.ssh.insecure_host_key":    {"호스트 키 확인 건너뛰기", "닫힌 망의 일회용 서버에만 씁니다. known_hosts 파일과 함께 쓸 수 없습니다."},
	"server-set.localmap":                 {"docker 접속 주소 변환", "서버가 이 컴퓨터의 docker 컨테이너일 때 씁니다. 서버 주소는 컨테이너의 실제 주소로 두고, 이 컴퓨터가 접속할 때만 여기 적은 주소와 게시 포트로 바꿉니다. 이 값이 있으면 이 풀의 모든 작업이 docker 모드로 돕니다."},
	"server-set.localmap.hosts":           {"컨테이너", "풀의 서버 주소마다 이 컴퓨터에서 닿는 주소를 적습니다."},
	"server-set.localmap.hosts{}":         {"컨테이너 접속", ""},
	"server-set.localmap.hosts{}.host":    {"이 컴퓨터에서 닿는 주소", "보통 127.0.0.1입니다."},
	"server-set.localmap.hosts{}.ports":   {"게시 포트", "컨테이너 포트마다 이 컴퓨터에 게시된 포트입니다. 적지 않은 포트는 그대로 씁니다."},
	"server-set.localmap.hosts{}.ports{}": {"게시된 포트", ""},
	"server-set.ssh.sudo":                 {"sudo 허용", "실행 단계가 필요할 때 권한을 올릴 수 있는지 알립니다."},

	"workspace-config":                            {"대상 경로", "대상 서버에서 파일을 둘 위치와 실행 방식을 정합니다."},
	"workspace-config.version":                    {"형식 버전", ""},
	"workspace-config.dataRoot":                   {"데이터 루트", "대상 서버의 절대 경로입니다. ~나 환경 변수를 쓸 수 없습니다."},
	"workspace-config.paths":                      {"용도별 디렉터리", "데이터 루트 아래의 상대 경로입니다. ..를 쓸 수 없습니다."},
	"workspace-config.paths.binaries":             {"바이너리", ""},
	"workspace-config.paths.configs":              {"설정", ""},
	"workspace-config.paths.genesis":              {"genesis", ""},
	"workspace-config.paths.keystore":             {"keystore", ""},
	"workspace-config.paths.keyrings":             {"키링", ""},
	"workspace-config.paths.nodes":                {"노드 데이터", ""},
	"workspace-config.paths.runtime":              {"실행 상태", ""},
	"workspace-config.paths.logs":                 {"로그", ""},
	"workspace-config.binaryAliases":              {"바이너리 별칭", "바이너리 이름을 바이너리 디렉터리 아래의 다른 상대 경로로 바꿉니다."},
	"workspace-config.binaryAliases{}":            {"별칭 경로", ""},
	"workspace-config.control":                    {"제어 쪽", "실행을 지휘하는 컴퓨터의 경로입니다."},
	"workspace-config.control.artifactRoot":       {"결과 루트", "실행 결과와 보고서를 모으는 이 컴퓨터의 디렉터리입니다. ~는 홈 디렉터리이고, 상대 경로는 설정 파일 기준입니다."},
	"workspace-config.control.binaries":           {"로컬 바이너리", "원격 대상에 보낼 노드 바이너리를 둔 이 컴퓨터의 디렉터리입니다. 대상의 사본이 없거나 sha256이 다르면 여기서 보냅니다. 비우면 대상이 바이너리를 준비합니다."},
	"workspace-config.inputs":                     {"입력", ""},
	"workspace-config.inputs.mode":                {"입력 방식", "generated는 새로 만들고, existing은 기존 입력 묶음을 씁니다."},
	"workspace-config.inputs.name":                {"기존 입력 이름", "입력 방식이 existing일 때 기존 입력 목록의 이름입니다."},
	"workspace-config.existingInputs":             {"기존 입력 목록", "대상에 이미 있는 genesis, 키링, 설정을 한 묶음으로 이름 붙입니다."},
	"workspace-config.existingInputs{}":           {"기존 입력", ""},
	"workspace-config.existingInputs{}.genesis":   {"genesis", ""},
	"workspace-config.existingInputs{}.keyring":   {"키링", ""},
	"workspace-config.existingInputs{}.configs":   {"설정 파일", ""},
	"workspace-config.existingInputs{}.configs{}": {"설정 파일 경로", ""},
	"workspace-config.execution":                  {"실행", ""},
	"workspace-config.execution.chain":            {"체인 준비", "fresh는 새로 띄우고, reuse-if-matching은 같은 구성이면 다시 쓰고, attach는 떠 있는 체인에 붙습니다."},
	"workspace-config.limits":                     {"제한", ""},
	"workspace-config.limits.minFreeDisk":         {"최소 여유 디스크", "2GiB, 500MiB 또는 바이트 수입니다. 노드를 놓는 모든 서버의 데이터 루트에 이만큼 여유가 없으면 아무것도 쓰지 않습니다. 0이면 검사하지 않고, 비우면 엔진 기본값을 씁니다."},
}

// deploymentDefaults are the values the engine fills itself when a field is
// left out or zero, so the editor can offer exactly those.
func deploymentDefaults() map[string]any {
	builtin := resource.BuiltinPorts()
	return map[string]any{
		"server-set.version":             resource.SupportedVersion,
		"server-set.pool.slots":          1,
		"server-set.pool.ports.p2p.base": builtin.P2PBase,
		"server-set.pool.ports.p2p.step": builtin.P2PStep,
		"server-set.pool.ports.rpc.base": builtin.RPCBase,
		"server-set.pool.ports.rpc.step": builtin.RPCStep,
		"server-set.ssh.port":            remote.DefaultSSHPort,
		"workspace-config.version":       resource.SupportedWorkspaceVersion,
	}
}

// deploymentNonEmpty are the strings the engine refuses when empty.
var deploymentNonEmpty = []string{
	"server-set.pool.hosts[].addr",
	"workspace-config.dataRoot",
	"workspace-config.control.artifactRoot",
	"workspace-config.paths.binaries", "workspace-config.paths.configs", "workspace-config.paths.genesis",
	"workspace-config.paths.keystore", "workspace-config.paths.keyrings", "workspace-config.paths.nodes",
	"workspace-config.paths.runtime", "workspace-config.paths.logs",
}

// describeDeploymentField fills a field and everything below it with its
// title, description, engine default and non-empty rule.
func describeDeploymentField(path string, f DeploymentField, defaults map[string]any, nonEmpty map[string]bool) DeploymentField {
	text := deploymentFieldText[path]
	f.Title, f.Description = text.title, text.description
	f.Default = defaults[path]
	if nonEmpty[path] {
		one := 1
		f.MinLength = &one
	}
	if f.Properties != nil {
		properties := make(map[string]DeploymentField, len(f.Properties))
		for key, child := range f.Properties {
			properties[key] = describeDeploymentField(path+"."+key, child, defaults, nonEmpty)
		}
		f.Properties = properties
	}
	if f.Items != nil {
		item := describeDeploymentField(path+"[]", *f.Items, defaults, nonEmpty)
		f.Items = &item
	}
	if f.Additional != nil {
		entry := describeDeploymentField(path+"{}", *f.Additional, defaults, nonEmpty)
		f.Additional = &entry
	}
	return f
}
