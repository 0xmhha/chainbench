package app

import "testing"

// Screens build role and credential kind choices from this contract, so it
// must list exactly what the server accepts.
func TestAccessContractListsWhatTheServerAccepts(t *testing.T) {
	c := AccessContract()
	if len(c.Roles) != 3 || len(c.CredentialKinds) != 3 {
		t.Fatalf("contract = %+v", c)
	}
	for _, r := range c.Roles {
		if !validWebRole(r.ID) || r.Label == "" {
			t.Errorf("role %+v", r)
		}
	}
	for _, k := range c.CredentialKinds {
		if !knownCredentialKind(k.ID) || k.Label == "" {
			t.Errorf("kind %+v", k)
		}
	}
	for _, unknown := range []string{"admin", "root", ""} {
		if validWebRole(unknown) {
			t.Errorf("role %q accepted", unknown)
		}
	}
	if knownCredentialKind("token") || validCredentialShape(DeploymentCredentialInput{Label: "x", Kind: "token", SSHUser: "u", Password: "p"}) {
		t.Error("an unlisted credential kind was accepted")
	}
}
