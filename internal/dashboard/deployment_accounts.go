package dashboard

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/0xmhha/chainbench/internal/app"
	"golang.org/x/crypto/bcrypt"
)

// DeploymentAccount is a provisioned account record; passwords are bcrypt hashes.
// This adapter can be replaced by the application account/session provider.
type DeploymentAccount struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	Role         string `json:"role"`
	PasswordHash string `json:"passwordHash"`
}

// DeploymentAccounts loads a private account file and provides authenticated identities.
func DeploymentAccounts(path string) (DeploymentAuthenticator, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("deployment accounts require private permissions")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var records []DeploymentAccount
	if err = json.Unmarshal(b, &records); err != nil {
		return nil, err
	}
	accounts := map[string]DeploymentAccount{}
	ids := map[string]bool{}
	for _, a := range records {
		if a.ID == "" || a.Username == "" || ids[a.ID] || accounts[a.Username].ID != "" || (a.Role != "admin" && a.Role != "operator" && a.Role != "viewer") {
			return nil, errors.New("invalid deployment account")
		}
		if _, err = bcrypt.Cost([]byte(a.PasswordHash)); err != nil {
			return nil, errors.New("invalid account password hash")
		}
		accounts[a.Username] = a
		ids[a.ID] = true
	}
	return func(r *http.Request) (app.DeploymentActor, error) {
		user, password, ok := r.BasicAuth()
		a, found := accounts[user]
		if !ok || !found || bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(password)) != nil {
			return app.DeploymentActor{}, errors.New("login required")
		}
		return app.DeploymentActor{ID: a.ID, Role: a.Role}, nil
	}, nil
}
