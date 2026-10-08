package app

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/0xmhha/chainbench/internal/core/session"
	"os"
	"regexp"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// WebUser exposes only public account fields. Password hashes never enter API data.
type WebUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Active   bool   `json:"active"`
}
type webAccount struct {
	WebUser
	Hash string `json:"passwordHash"`
}
type WebSession struct {
	User      WebUser   `json:"user"`
	CSRFToken string    `json:"csrfToken"`
	ExpiresAt time.Time `json:"expiresAt"`
	Mode      string    `json:"mode"`
}
type webLogin struct {
	UserID  string
	CSRF    string
	Expires time.Time
}

// WebAuth owns accounts and server-side sessions. Sessions intentionally expire on restart.
type WebAuth struct {
	mu          sync.Mutex
	mode, setup string
	storage     *session.AccountStore
	users       map[string]webAccount
	sessions    map[string]webLogin
}

var webUsername = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)

func webToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func tokenDigest(s string) string { b := sha256.Sum256([]byte(s)); return hex.EncodeToString(b[:]) }
func validWebRole(s string) bool  { return s == "administrator" || s == "operator" || s == "viewer" }
func OpenWebAuth(root, mode string) (*WebAuth, error) {
	if mode != "personal" && mode != "team" {
		return nil, errors.New("invalid web mode")
	}
	storage, err := session.OpenAccountStore(root)
	if err != nil {
		return nil, err
	}
	a := &WebAuth{storage: storage, mode: mode, users: map[string]webAccount{}, sessions: map[string]webLogin{}}
	b, err := storage.ReadUsers()
	if err == nil {
		if err = json.Unmarshal(b, &a.users); err != nil {
			return nil, err
		}
		if len(a.users) == 0 {
			return nil, errors.New("invalid empty account store")
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	for id, u := range a.users {
		if id != u.ID || !webUsername.MatchString(u.Username) || !validWebRole(u.Role) {
			return nil, errors.New("invalid account store")
		}
		if _, err = bcrypt.Cost([]byte(u.Hash)); err != nil {
			return nil, err
		}
	}
	if len(a.users) == 0 {
		token, err := storage.SetupToken(webToken())
		if err != nil {
			return nil, err
		}
		a.setup = token
	}

	return a, nil
}
func (a *WebAuth) save(next map[string]webAccount) error {
	b, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if err = a.storage.WriteUsers(b); err != nil {
		return err
	}
	a.users = next
	return nil
}
func (a *WebAuth) copyUsers() map[string]webAccount {
	n := map[string]webAccount{}
	for k, v := range a.users {
		n[k] = v
	}
	return n
}
func (a *WebAuth) create(username, password, role string) (webAccount, error) {
	if !webUsername.MatchString(username) || len(password) < 12 || len(password) > 72 || !validWebRole(role) {
		return webAccount{}, errors.New("invalid account; password must be 12 to 72 bytes")
	}
	for _, u := range a.users {
		if u.Username == username {
			return webAccount{}, ErrDeploymentConflict
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return webAccount{}, err
	}
	return webAccount{WebUser: WebUser{webToken(), username, role, true}, Hash: string(hash)}, nil
}
func (a *WebAuth) session(u WebUser) (WebSession, string) {
	token := webToken()
	s := WebSession{u, webToken(), time.Now().UTC().Add(12 * time.Hour), a.mode}
	a.sessions[tokenDigest(token)] = webLogin{u.ID, s.CSRFToken, s.ExpiresAt}
	return s, token
}
func (a *WebAuth) Bootstrap(username, password, token string) (WebSession, string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.users) != 0 {
		return WebSession{}, "", ErrDeploymentConflict
	}
	if a.setup == "" || tokenDigest(token) != tokenDigest(a.setup) {
		return WebSession{}, "", ErrDeploymentForbidden
	}
	u, err := a.create(username, password, "administrator")
	if err != nil {
		return WebSession{}, "", err
	}
	if err = a.save(map[string]webAccount{u.ID: u}); err != nil {
		return WebSession{}, "", err
	}
	a.setup = ""
	_ = a.storage.RemoveSetupToken()
	s, t := a.session(u.WebUser)
	return s, t, nil
}
func (a *WebAuth) Login(username, password string) (WebSession, string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, u := range a.users {
		if u.Username == username && u.Active && bcrypt.CompareHashAndPassword([]byte(u.Hash), []byte(password)) == nil {
			s, t := a.session(u.WebUser)
			return s, t, nil
		}
	}
	return WebSession{}, "", ErrDeploymentForbidden
}
func (a *WebAuth) Session(token string) (WebSession, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	s, ok := a.sessions[tokenDigest(token)]
	u := a.users[s.UserID]
	if !ok || !u.Active || !time.Now().Before(s.Expires) {
		delete(a.sessions, tokenDigest(token))
		return WebSession{}, ErrDeploymentForbidden
	}
	return WebSession{u.WebUser, s.CSRF, s.Expires, a.mode}, nil
}
func (a *WebAuth) Logout(token string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.sessions, tokenDigest(token))
}
func (a *WebAuth) Users() []WebUser {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := []WebUser{}
	for _, u := range a.users {
		out = append(out, u.WebUser)
	}
	return out
}

// AuthorizeJobActor checks current account permissions independently of browser
// sessions, so logout preserves work while revocation stops subsequent work.
func (a *WebAuth) AuthorizeJobActor(actor DeploymentActor) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	u, ok := a.users[actor.ID]
	if !ok || !u.Active || u.Role == "viewer" {
		return ErrDeploymentForbidden
	}
	return nil
}
func (a *WebAuth) CreateUser(actor DeploymentActor, username, password, role string) (WebUser, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if actor.Role != "admin" {
		return WebUser{}, ErrDeploymentForbidden
	}
	u, err := a.create(username, password, role)
	if err != nil {
		return WebUser{}, err
	}
	n := a.copyUsers()
	n[u.ID] = u
	return u.WebUser, a.save(n)
}
func (a *WebAuth) UpdateUser(actor DeploymentActor, id, role, password string, active *bool) (WebUser, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if actor.Role != "admin" {
		return WebUser{}, ErrDeploymentForbidden
	}
	u, ok := a.users[id]
	if !ok {
		return WebUser{}, ErrDeploymentNotFound
	}
	if role != "" {
		if !validWebRole(role) {
			return WebUser{}, errors.New("invalid role")
		}
		u.Role = role
	}
	if active != nil {
		u.Active = *active
	}
	if password != "" {
		if len(password) < 12 || len(password) > 72 {
			return WebUser{}, errors.New("invalid password length")
		}
		h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return WebUser{}, err
		}
		u.Hash = string(h)
	}
	n := a.copyUsers()
	n[id] = u
	admins := 0
	for _, v := range n {
		if v.Active && v.Role == "administrator" {
			admins++
		}
	}
	if admins == 0 {
		return WebUser{}, ErrDeploymentConflict
	}
	if err := a.save(n); err != nil {
		return WebUser{}, err
	}
	for token, s := range a.sessions {
		if s.UserID == id {
			delete(a.sessions, token)
		}
	}
	return u.WebUser, nil
}

// AuditWeb records bounded server-selected operations, never request bodies or secrets.
func (a *WebAuth) AuditWeb(actor, operation string, status int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	b, err := json.Marshal(map[string]any{"actorId": actor, "operation": operation, "status": status, "time": time.Now().UTC()})
	if err != nil {
		return err
	}
	return a.storage.AppendAudit(append(b, '\n'))
}

// BootstrapRequired exposes setup state without exposing the local setup capability.
func (a *WebAuth) BootstrapRequired() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.users) == 0
}
