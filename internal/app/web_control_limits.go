package app

import "slices"

// webNodeControls are the node operations a node page offers, in that order.
var webNodeControls = []string{"node.start", "node.stop", "node.restart", "node.swap", "node.reset"}

// webControlLimits names, for each node control the node does not offer, why
// it is refused. The most specific reason wins: who owns the network, then
// the node's role and config, then what observation established.
func webControlLimits(n WebNode, ownership string, configFile bool) map[string]string {
	limits := map[string]string{}
	for _, op := range webNodeControls {
		if slices.Contains(n.SupportedControls, op) {
			continue
		}
		switch {
		case ownership != "owned":
			limits[op] = "attach 네트워크의 노드는 Web이 시작, 정지, 교체, 초기화하지 않습니다."
		case op == "node.reset" && n.Role == "bp":
			limits[op] = "블록 생산자는 초기화할 수 없습니다. 합의에 참여하는 노드의 자료를 지우면 네트워크가 멈출 수 있습니다."
		case op == "node.swap" && configFile:
			limits[op] = "별도 설정 파일이 지정된 노드는 설정을 교체할 수 없습니다."
		case n.State == "ownership_mismatch":
			limits[op] = "기록과 실제 프로세스가 맞지 않아 제어하지 않습니다."
		case n.ObservationReason == "binary_launch_unavailable":
			limits[op] = "기록된 바이너리를 대상에서 실행할 수 없습니다. 실행 중인 노드는 정지만 할 수 있습니다."
		case n.ObservationReason == "probe_unavailable":
			limits[op] = "대상 서버에서 프로세스를 확인하지 못했습니다. 내 자격증명과 서버 연결을 확인한 뒤 다시 보세요."
		case n.State == "unknown" || n.State == "":
			limits[op] = "실제 프로세스를 아직 확인하지 않았습니다. 노드 상태를 먼저 확인하세요."
		default:
			limits[op] = "현재 상태(" + n.State + ")에서는 이 제어를 쓸 수 없습니다."
		}
	}
	return limits
}
