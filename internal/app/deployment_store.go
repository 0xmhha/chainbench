package app

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/0xmhha/chainbench/internal/core/session"
	"os"
	"sync"
	"time"
)

var (
	ErrDeploymentNotFound  = errors.New("deployment object not found")
	ErrDeploymentConflict  = errors.New("revision changed; reload before editing")
	ErrDeploymentForbidden = errors.New("deployment operation forbidden")
)

// DeploymentActor is supplied by the authenticated account layer, never by request JSON.
type DeploymentActor struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

func (a DeploymentActor) canEdit() bool {
	return a.ID != "" && (a.Role == "admin" || a.Role == "operator")
}

// DeploymentDocument is a team-shared immutable document revision.
type DeploymentDocument struct {
	DeploymentDocumentInput
	ID        string    `json:"id"`
	Revision  int       `json:"revision"`
	UpdatedAt time.Time `json:"updatedAt"`
	UpdatedBy string    `json:"updatedBy"`
}

// DeploymentDocumentRef pins the document a workspace uses.
type DeploymentDocumentRef struct {
	ID       string `json:"id"`
	Revision int    `json:"revision"`
}

// DeploymentWorkspaceInput contains shared references; personal bindings are separate.
type DeploymentWorkspaceInput struct {
	Name      string                  `json:"name"`
	Documents []DeploymentDocumentRef `json:"documents"`
}

// DeploymentWorkspace is a versioned shared deployment environment.
type DeploymentWorkspace struct {
	DeploymentWorkspaceInput
	ID       string `json:"id"`
	Revision int    `json:"revision"`
}

// DeploymentCredential contains metadata only, including for its owner.
type DeploymentCredential struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"ownerId"`
	Label     string    `json:"label"`
	Kind      string    `json:"kind"`
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"createdAt"`
}

// DeploymentCredentialInput is encrypted before it is persisted.
type DeploymentCredentialInput struct {
	Label      string `json:"label"`
	Kind       string `json:"kind"`
	SSHUser    string `json:"sshUser"`
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"privateKey,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
}
type encryptedDeploymentCredential struct {
	Metadata   DeploymentCredential `json:"metadata"`
	Ciphertext []byte               `json:"ciphertext"`
}
type deploymentAudit struct {
	Actor     string    `json:"actorId"`
	Operation string    `json:"operation"`
	Target    string    `json:"target"`
	Time      time.Time `json:"time"`
}
type deploymentState struct {
	Documents   map[string][]DeploymentDocument          `json:"documents"`
	Workspaces  map[string]DeploymentWorkspace           `json:"workspaces"`
	Credentials map[string]encryptedDeploymentCredential `json:"credentials"`
	Bindings    map[string]map[string]map[string]string  `json:"bindings"`
	Audit       []deploymentAudit                        `json:"audit"`
	Imports     map[string]deploymentImport              `json:"imports,omitempty"`
}

// DeploymentStore owns team configuration and encrypted, user-owned SSH overlays.
// Open one store per data directory; all changes are committed atomically.
type DeploymentStore struct {
	mu        sync.Mutex
	storage   *session.WebStore
	aead      cipher.AEAD
	state     deploymentState
	authorize func(DeploymentActor) error
}

// SetAuthorizer connects private access checks to current server-owned account
// permissions. Configure at service startup; nil supports provisioned accounts.
func (s *DeploymentStore) SetAuthorizer(fn func(DeploymentActor) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authorize = fn
}

// OpenDeploymentStore opens persistent Web deployment data, failing closed on corruption.
func OpenDeploymentStore(root string) (*DeploymentStore, error) {
	storage, err := session.OpenWebStore(root)
	if err != nil {
		return nil, err
	}
	key, err := storage.CredentialKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	s := &DeploymentStore{storage: storage, aead: aead, state: deploymentState{Documents: map[string][]DeploymentDocument{}, Workspaces: map[string]DeploymentWorkspace{}, Credentials: map[string]encryptedDeploymentCredential{}, Bindings: map[string]map[string]map[string]string{}}}
	b, err := storage.ReadDeployment()
	if err == nil {
		if err = json.Unmarshal(b, &s.state); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if s.state.Documents == nil || s.state.Workspaces == nil || s.state.Credentials == nil || s.state.Bindings == nil {
		return nil, errors.New("invalid deployment state")
	}
	return s, nil
}

func deploymentID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// commit uses a detached state so a disk failure cannot publish an uncommitted revision.
func (s *DeploymentStore) commit(actor DeploymentActor, operation, target string, change func(*deploymentState)) error {
	b, err := json.Marshal(s.state)
	if err != nil {
		return err
	}
	var next deploymentState
	if err = json.Unmarshal(b, &next); err != nil {
		return err
	}
	change(&next)
	next.Audit = append(next.Audit, deploymentAudit{actor.ID, operation, target, time.Now().UTC()})
	b, err = json.Marshal(next)
	if err != nil {
		return err
	}
	var committed deploymentState
	if err = json.Unmarshal(b, &committed); err != nil {
		return err
	}
	if err = s.storage.WriteDeployment(b); err != nil {
		return err
	}
	s.state = committed
	return nil
}

// SaveDocument creates or appends a revision guarded by the caller's If-Match value.
func (s *DeploymentStore) SaveDocument(a DeploymentActor, id string, revision int, in DeploymentDocumentInput) (DeploymentDocument, error) {
	if !a.canEdit() {
		return DeploymentDocument{}, ErrDeploymentForbidden
	}
	if err := ValidateDeploymentDocument(in); err != nil {
		return DeploymentDocument{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		id = deploymentID()
		revision = 0
	} else {
		history := s.state.Documents[id]
		if len(history) == 0 {
			return DeploymentDocument{}, ErrDeploymentNotFound
		}
		if history[len(history)-1].Revision != revision {
			return DeploymentDocument{}, ErrDeploymentConflict
		}
		if history[0].Kind != in.Kind {
			return DeploymentDocument{}, errors.New("document kind cannot change")
		}
	}
	// Detach caller-owned JSON so subsequent edits cannot modify stored revisions.
	in.Content = append(json.RawMessage(nil), in.Content...)
	in.AssetRefs = append([]string{}, in.AssetRefs...)
	d := DeploymentDocument{in, id, revision + 1, time.Now().UTC(), a.ID}
	err := s.commit(a, "document.save", id, func(next *deploymentState) { next.Documents[id] = append(next.Documents[id], d) })
	return d, err
}

// RevokeCredential disables only the owner's private credential. Ciphertext is
// retained for audit redaction, but no later access may decrypt it for a dial.
func (s *DeploymentStore) RevokeCredential(a DeploymentActor, id string) error {
	if !a.canEdit() {
		return ErrDeploymentForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.state.Credentials[id]
	if !ok || c.Metadata.OwnerID != a.ID {
		return ErrDeploymentNotFound
	}
	if c.Metadata.Revoked {
		return nil
	}
	return s.commit(a, "credential.revoked", id, func(next *deploymentState) {
		v := next.Credentials[id]
		v.Metadata.Revoked = true
		next.Credentials[id] = v
	})
}

// Documents returns shared latest revisions, optionally filtered by kind.
func (s *DeploymentStore) Documents(kind string) []DeploymentDocument {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []DeploymentDocument{}
	for _, h := range s.state.Documents {
		d := h[len(h)-1]
		if kind == "" || kind == d.Kind {
			d.Content = append(json.RawMessage(nil), d.Content...)
			d.AssetRefs = append([]string{}, d.AssetRefs...)
			out = append(out, d)
		}
	}
	return out
}

// Document reads a shared latest revision.
func (s *DeploymentStore) Document(id string) (DeploymentDocument, error) {
	return s.DocumentRevision(id, 0)
}

// DocumentRevision reads a pinned revision; zero selects the latest.
func (s *DeploymentStore) DocumentRevision(id string, revision int) (DeploymentDocument, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h := s.state.Documents[id]
	if len(h) == 0 {
		return DeploymentDocument{}, ErrDeploymentNotFound
	}
	if revision < 0 || revision > len(h) {
		return DeploymentDocument{}, ErrDeploymentNotFound
	}
	if revision == 0 {
		revision = len(h)
	}
	d := h[revision-1]
	d.Content = append(json.RawMessage(nil), d.Content...)
	d.AssetRefs = append([]string{}, d.AssetRefs...)
	return d, nil
}

func (s *DeploymentStore) workspaceDocuments(w DeploymentWorkspaceInput) (DeploymentDocument, DeploymentDocument, error) {
	var set, config DeploymentDocument
	if w.Name == "" || len(w.Documents) != 2 {
		return set, config, errors.New("workspace requires a name and one server-set and workspace-config revision")
	}
	for _, ref := range w.Documents {
		history := s.state.Documents[ref.ID]
		if ref.Revision < 1 || ref.Revision > len(history) {
			return set, config, ErrDeploymentConflict
		}
		d := history[ref.Revision-1]
		switch d.Kind {
		case "server-set":
			if set.ID != "" {
				return set, config, errors.New("duplicate server-set")
			}
			set = d
		case "workspace-config":
			if config.ID != "" {
				return set, config, errors.New("duplicate workspace-config")
			}
			config = d
		}
	}
	if set.ID == "" || config.ID == "" {
		return set, config, errors.New("missing deployment document")
	}
	return set, config, nil
}

// SaveWorkspace pins validated shared revisions, guarded against concurrent edits.
func (s *DeploymentStore) SaveWorkspace(a DeploymentActor, id string, revision int, in DeploymentWorkspaceInput) (DeploymentWorkspace, error) {
	if !a.canEdit() {
		return DeploymentWorkspace{}, ErrDeploymentForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, _, err := s.workspaceDocuments(in); err != nil {
		return DeploymentWorkspace{}, err
	}
	if id == "" {
		id = deploymentID()
		revision = 0
	} else {
		old, ok := s.state.Workspaces[id]
		if !ok {
			return DeploymentWorkspace{}, ErrDeploymentNotFound
		}
		if old.Revision != revision {
			return DeploymentWorkspace{}, ErrDeploymentConflict
		}
	}
	in.Documents = append([]DeploymentDocumentRef(nil), in.Documents...)
	w := DeploymentWorkspace{in, id, revision + 1}
	err := s.commit(a, "workspace.save", id, func(next *deploymentState) { next.Workspaces[id] = w })
	return w, err
}

// Workspaces returns shared workspace metadata.
func (s *DeploymentStore) Workspaces() []DeploymentWorkspace {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []DeploymentWorkspace{}
	for _, w := range s.state.Workspaces {
		w.Documents = append([]DeploymentDocumentRef(nil), w.Documents...)
		out = append(out, w)
	}
	return out
}

// Workspace reads a shared environment.
func (s *DeploymentStore) Workspace(id string) (DeploymentWorkspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.state.Workspaces[id]
	if !ok {
		return w, ErrDeploymentNotFound
	}
	w.Documents = append([]DeploymentDocumentRef(nil), w.Documents...)
	return w, nil
}

// SaveCredential encrypts SSH material with the credential identity as authenticated data.
func (s *DeploymentStore) SaveCredential(a DeploymentActor, in DeploymentCredentialInput) (DeploymentCredential, error) {
	if !a.canEdit() {
		return DeploymentCredential{}, ErrDeploymentForbidden
	}
	if in.Label == "" || in.SSHUser == "" || (in.Kind != "password" && in.Kind != "private-key") || (in.Kind == "password" && (in.Password == "" || in.PrivateKey != "" || in.Passphrase != "")) || (in.Kind == "private-key" && (in.PrivateKey == "" || in.Password != "")) {
		return DeploymentCredential{}, errors.New("invalid private credential")
	}
	if err := validateDeploymentKey(in); err != nil {
		return DeploymentCredential{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	m := DeploymentCredential{deploymentID(), a.ID, in.Label, in.Kind, false, time.Now().UTC()}
	b, err := json.Marshal(in)
	if err != nil {
		return m, err
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return m, err
	}
	encrypted := s.aead.Seal(nonce, nonce, b, []byte(m.ID+":"+m.OwnerID))
	err = s.commit(a, "credential.create", m.ID, func(next *deploymentState) { next.Credentials[m.ID] = encryptedDeploymentCredential{m, encrypted} })
	return m, err
}

// Credentials lists only the caller's metadata, even for administrators.
func (s *DeploymentStore) Credentials(a DeploymentActor) []DeploymentCredential {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []DeploymentCredential{}
	for _, c := range s.state.Credentials {
		if c.Metadata.OwnerID == a.ID {
			out = append(out, c.Metadata)
		}
	}
	return out
}

// Credential retrieves metadata without exposing stored material.
func (s *DeploymentStore) Credential(a DeploymentActor, id string) (DeploymentCredential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.state.Credentials[id]
	if !ok || c.Metadata.OwnerID != a.ID {
		return DeploymentCredential{}, ErrDeploymentNotFound
	}
	return c.Metadata, nil
}

// BindCredential stores a personal overlay without changing the shared workspace revision.
func (s *DeploymentStore) BindCredential(a DeploymentActor, workspace, server, credential string) error {
	if !a.canEdit() {
		return ErrDeploymentForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, _, _, err := s.resolveAccess(a, workspace, server, credential); err != nil {
		return err
	}
	return s.commit(a, "credential.bind", workspace, func(next *deploymentState) {
		if next.Bindings[a.ID] == nil {
			next.Bindings[a.ID] = map[string]map[string]string{}
		}
		if next.Bindings[a.ID][workspace] == nil {
			next.Bindings[a.ID][workspace] = map[string]string{}
		}
		next.Bindings[a.ID][workspace][server] = credential
	})
}

// Bindings exposes only the current user's workspace overlay.
func (s *DeploymentStore) Bindings(a DeploymentActor, workspace string) map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]string{}
	for server, credential := range s.state.Bindings[a.ID][workspace] {
		out[server] = credential
	}
	return out
}
