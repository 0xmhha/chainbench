package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestWebAuthConcurrentBootstrapAndRevocation(t *testing.T) {
	root := t.TempDir()
	a, err := OpenWebAuth(root, "personal")
	if err != nil {
		t.Fatal(err)
	}
	setup, _ := os.ReadFile(filepath.Join(root, "setup.token"))
	var wg sync.WaitGroup
	wins := make(chan WebSession, 4)
	for range 4 {
		wg.Go(func() {
			s, _, err := a.Bootstrap("admin", "long-admin-password", string(setup))
			if err == nil {
				wins <- s
			}
		})
	}
	wg.Wait()
	close(wins)
	if len(wins) != 1 {
		t.Fatal("bootstrap not single use")
	}
	s := <-wins
	actor := DeploymentActor{s.User.ID, "admin"}
	u, err := a.CreateUser(actor, "operator", "long-user-password", "operator")
	if err != nil {
		t.Fatal(err)
	}
	_, token, _ := a.Login(u.Username, "long-user-password")
	active := false
	if _, err = a.UpdateUser(actor, u.ID, "", "", &active); err != nil {
		t.Fatal(err)
	}
	if _, err = a.Session(token); err == nil {
		t.Fatal("deactivated account session accepted")
	}
	if _, _, err = a.Login(u.Username, "long-user-password"); err == nil {
		t.Fatal("disabled account logged in")
	}
	if _, err = a.UpdateUser(actor, actor.ID, "viewer", "", nil); err == nil {
		t.Fatal("last administrator demoted")
	}
	if _, err = a.CreateUser(DeploymentActor{u.ID, "operator"}, "bad", "long-user-password", "administrator"); err == nil {
		t.Fatal("operator registered admin")
	}
}

func TestWebRedactionPreservesJSONAndSSE(t *testing.T) {
	s, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{`"`, "unique-multiline\nsecret"} {
		if _, err = s.SaveCredential(DeploymentActor{"owner", "operator"}, DeploymentCredentialInput{Label: "key", Kind: "password", SSHUser: "ssh", Password: secret}); err != nil {
			t.Fatal(err)
		}
	}
	raw := `{"message":"unique-multiline\nsecret","fields":{"password":"not-registered","privateKey":"material"}}`
	clean := s.RedactWeb(raw)
	var value map[string]any
	if err = json.Unmarshal([]byte(clean), &value); err != nil {
		t.Fatal(clean, err)
	}
	if value["message"] != "[REDACTED]" {
		t.Fatal(clean)
	}
	stream := s.RedactWeb("data: " + raw + "\n\n")
	if !strings.HasPrefix(stream, "data: ") || !strings.HasSuffix(stream, "\n\n") {
		t.Fatal(stream)
	}
}
