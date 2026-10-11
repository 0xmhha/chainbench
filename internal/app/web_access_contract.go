package app

import "slices"

// WebChoice is one server-owned choice and the label screens show for it.
type WebChoice struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// WebAccessContract lists the account roles and personal credential kinds the
// server accepts, so screens never keep their own copy of either list.
type WebAccessContract struct {
	Roles           []WebChoice `json:"roles"`
	CredentialKinds []WebChoice `json:"credentialKinds"`
}

var webRoles = []WebChoice{
	{"viewer", "조회자", "구성과 결과를 읽기만 합니다."},
	{"operator", "운영자", "본인 자격증명으로 배포, 테스트, 소유 노드 제어를 합니다."},
	{"administrator", "관리자", "운영자 권한에 더해 사용자, 역할, 감사 기록을 관리합니다."},
}

var webCredentialKinds = []WebChoice{
	{"private-key", "SSH 개인키", "서버에 개인키로 로그인합니다."},
	{"password", "SSH 비밀번호", "서버에 비밀번호로 로그인합니다."},
	{accountKeyCredential, "테스트 계정 개인키", "테스트 케이스가 트랜잭션에 서명합니다. SSH 로그인에는 쓰지 않습니다."},
}

// AccessContract returns detached copies of the role and credential kind lists.
func AccessContract() WebAccessContract {
	return WebAccessContract{Roles: slices.Clone(webRoles), CredentialKinds: slices.Clone(webCredentialKinds)}
}

func knownCredentialKind(kind string) bool {
	return slices.ContainsFunc(webCredentialKinds, func(c WebChoice) bool { return c.ID == kind })
}
