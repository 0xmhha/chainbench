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
	"slices"
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
	User            WebUser   `json:"user"`
	CSRFToken       string    `json:"csrfToken"`
	ExpiresAt       time.Time `json:"expiresAt"`
	Mode            string    `json:"mode"`
	AccountsEnabled bool      `json:"accountsEnabled"`
}
type webLogin struct {
	UserID  string
	CSRF    string
	Expires time.Time
}

type webIdentityMode uint8

const (
	webAccountIdentity webIdentityMode = iota
	webLocalIdentity   webIdentityMode = iota
	localWebActorID                    = "local-web-operator"
	webSessionLifetime                 = 12 * time.Hour
)

// WebAuth owns account or local browser identities and server-side sessions.
// Sessions intentionally expire on restart; the local actor's ownership does not.
type WebAuth struct {
	mu          sync.Mutex
	mode, setup string
	storage     *session.AccountStore
	users       map[string]webAccount
	sessions    map[string]webLogin
	identity    webIdentityMode
	clock       func() time.Time
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
func validWebRole(s string) bool {
	return slices.ContainsFunc(webRoles, func(c WebChoice) bool { return c.ID == s })
}
func OpenWebAuth(root, mode string) (*WebAuth, error) {
	if mode != "personal" && mode != "team" {
		return nil, errors.New("invalid web mode")
	}
	storage, err := session.OpenAccountStore(root)
	if err != nil {
		return nil, err
	}
	a := &WebAuth{storage: storage, mode: mode, users: map[string]webAccount{}, sessions: map[string]webLogin{}, clock: time.Now}
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

// OpenLocalWebAuth opens account-free browser access without reading or creating
// registered users or a setup token. Existing accounts remain untouched.
func OpenLocalWebAuth(root string) (*WebAuth, error) {
	storage, err := session.OpenAccountStore(root)
	if err != nil {
		return nil, err
	}
	return &WebAuth{storage: storage, mode: "personal", identity: webLocalIdentity, clock: time.Now,
		users: map[string]webAccount{}, sessions: map[string]webLogin{}}, nil
}

// AccountsEnabled reports whether browser access uses registered accounts.
func (a *WebAuth) AccountsEnabled() bool { return a.identity == webAccountIdentity }

func localWebUser() WebUser {
	return WebUser{ID: localWebActorID, Username: "local", Role: "administrator", Active: true}
}

// LocalSession issues a browser session for the stable local operator. It never
// creates a registered user and is forbidden on account-enabled services.
func (a *WebAuth) LocalSession() (WebSession, string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.AccountsEnabled() {
		return WebSession{}, "", ErrDeploymentForbidden
	}
	// Automatic reconnects must not retain sessions that have already expired.
	now := a.clock()
	for token, login := range a.sessions {
		if !now.Before(login.Expires) {
			delete(a.sessions, token)
		}
	}
	s, token := a.session(localWebUser())
	return s, token, nil
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
	s := WebSession{User: u, CSRFToken: webToken(), ExpiresAt: a.clock().UTC().Add(webSessionLifetime), Mode: a.mode, AccountsEnabled: a.AccountsEnabled()}
	a.sessions[tokenDigest(token)] = webLogin{u.ID, s.CSRFToken, s.ExpiresAt}
	return s, token
}
func (a *WebAuth) Bootstrap(username, password, token string) (WebSession, string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.AccountsEnabled() {
		return WebSession{}, "", ErrDeploymentForbidden
	}
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
	if !a.AccountsEnabled() {
		return WebSession{}, "", ErrDeploymentForbidden
	}
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
	if !a.AccountsEnabled() && s.UserID == localWebActorID {
		u.WebUser = localWebUser()
	}
	if !ok || !u.Active || !a.clock().Before(s.Expires) {
		delete(a.sessions, tokenDigest(token))
		return WebSession{}, ErrDeploymentForbidden
	}
	return WebSession{User: u.WebUser, CSRFToken: s.CSRF, ExpiresAt: s.Expires, Mode: a.mode, AccountsEnabled: a.AccountsEnabled()}, nil
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
	if !a.AccountsEnabled() {
		if actor.ID == localWebActorID && actor.canEdit() {
			return nil
		}
		return ErrDeploymentForbidden
	}
	u, ok := a.users[actor.ID]
	if !ok || !u.Active || u.Role == "viewer" {
		return ErrDeploymentForbidden
	}
	return nil
}
func (a *WebAuth) CreateUser(actor DeploymentActor, username, password, role string) (WebUser, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.AccountsEnabled() || actor.Role != "admin" {
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

// PreviewUserUpdate validates a change and returns the account it would
// produce, without saving it, so its consequences can be applied first.
func (a *WebAuth) PreviewUserUpdate(actor DeploymentActor, id, role, password string, active *bool) (WebUser, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	u, _, err := a.updatedUser(actor, id, role, password, active)
	return u.WebUser, err
}

func (a *WebAuth) UpdateUser(actor DeploymentActor, id, role, password string, active *bool) (WebUser, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	u, n, err := a.updatedUser(actor, id, role, password, active)
	if err != nil {
		return WebUser{}, err
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

// updatedUser applies a change to a copy of the accounts; the caller holds a.mu.
func (a *WebAuth) updatedUser(actor DeploymentActor, id, role, password string, active *bool) (webAccount, map[string]webAccount, error) {
	if !a.AccountsEnabled() || actor.Role != "admin" {
		return webAccount{}, nil, ErrDeploymentForbidden
	}
	u, ok := a.users[id]
	if !ok {
		return webAccount{}, nil, ErrDeploymentNotFound
	}
	if role != "" {
		if !validWebRole(role) {
			return webAccount{}, nil, errors.New("invalid role")
		}
		u.Role = role
	}
	if active != nil {
		u.Active = *active
	}
	if password != "" {
		if len(password) < 12 || len(password) > 72 {
			return webAccount{}, nil, errors.New("invalid password length")
		}
		h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return webAccount{}, nil, err
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
		return webAccount{}, nil, ErrDeploymentConflict
	}
	return u, n, nil
}

// AuditWeb records bounded server-selected operations, never request bodies or secrets.
func (a *WebAuth) AuditWeb(actor, operation string, status int) error {
	return a.AuditWebRequest(actor, operation, "", "", status)
}

// AuditWebRequest also records the request's target, as the route's own
// identifiers, and the request id its response carried.
func (a *WebAuth) AuditWebRequest(actor, operation, target, requestID string, status int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	b, err := json.Marshal(map[string]any{"actorId": actor, "operation": operation, "target": target, "requestId": requestID, "status": status, "time": a.clock().UTC()})
	if err != nil {
		return err
	}
	return a.storage.AppendAudit(append(b, '\n'))
}

// BootstrapRequired exposes setup state without exposing the local setup capability.
func (a *WebAuth) BootstrapRequired() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.AccountsEnabled() && len(a.users) == 0
}
