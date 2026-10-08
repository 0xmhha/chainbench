package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/0xmhha/chainbench/internal/chains/external"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// ManifestInput carries declarations rather than filesystem paths or executable code.
type ManifestInput struct {
	Manifest json.RawMessage `json:"manifest"`
	Template string          `json:"template"`
}

type ManagedManifest struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	ManifestInput
}

// ManifestStore keeps immutable, content-addressed external declarations. Embedded
// plugins retain their registered implementation and cannot be replaced by imports.
type ManifestStore struct {
	files *session.ManifestFiles
	mu    sync.Mutex
}

func OpenManifestStore(root string) (*ManifestStore, error) {
	files, err := session.OpenManifestFiles(root)
	if err != nil {
		return nil, err
	}
	return &ManifestStore{files: files}, nil
}

// ValidateManifest delegates parsing, family/protocol/dialect resolution to the engine.
// Web imports may borrow existing mappings but cannot install executable plugins.
func ValidateManifest(in ManifestInput) (registry.ChainPlugin, error) {
	var decoded registry.Manifest
	dec := json.NewDecoder(bytes.NewReader(in.Manifest))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&decoded); err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return nil, errors.New("manifest: expected one JSON object")
	}
	p, err := external.FromBytes(in.Manifest, []byte(in.Template))
	if err != nil {
		return nil, err
	}
	base, err := registry.Get(p.Protocol().Name)
	if err != nil {
		return nil, err
	}
	m, b := p.Manifest(), base.Manifest()
	if m.ConsensusFamily != b.ConsensusFamily || m.Dialect != b.Dialect || m.MinerRecommit != b.MinerRecommit || m.Bootstrap != b.Bootstrap || m.Genesis.EngineField != b.Genesis.EngineField || m.Consensus != b.Consensus || m.Probe.Method != b.Probe.Method {
		return nil, errors.New("manifest requires an unsupported engine mapping for its borrowed protocol")
	}
	if _, err = nodeconfig.DialectFor(m.Dialect); err != nil {
		return nil, err
	}
	return p, nil
}

func manifestHash(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }

func (s *ManifestStore) Save(actor DeploymentActor, in ManifestInput) (ManagedManifest, error) {
	if actor.ID == "" || (actor.Role != "operator" && actor.Role != "admin") {
		return ManagedManifest{}, ErrDeploymentForbidden
	}
	p, err := ValidateManifest(in)
	if err != nil {
		return ManagedManifest{}, err
	}
	if _, err = registry.Get(p.Manifest().ID); err == nil {
		return ManagedManifest{}, errors.New("external manifest cannot replace an embedded chain ID")
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return ManagedManifest{}, err
	}
	item := ManagedManifest{ID: manifestHash(raw), Source: "external", ManifestInput: in}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err = s.files.Save(item.ID, raw); err != nil {
		return ManagedManifest{}, err
	}
	return item, nil
}

func (s *ManifestStore) Get(id string) (ManagedManifest, error) {
	if p, err := registry.Get(id); err == nil {
		raw, _ := json.Marshal(p.Manifest())
		return ManagedManifest{ID: id, Source: "builtin", ManifestInput: ManifestInput{Manifest: raw, Template: string(p.GenesisTemplate())}}, nil
	}
	if len(id) != 64 {
		return ManagedManifest{}, ErrDeploymentNotFound
	}
	if _, err := hex.DecodeString(id); err != nil {
		return ManagedManifest{}, ErrDeploymentNotFound
	}
	raw, err := s.files.Read(id)
	if err != nil {
		return ManagedManifest{}, ErrDeploymentNotFound
	}
	if manifestHash(raw) != id {
		return ManagedManifest{}, errors.New("manifest integrity check failed")
	}
	var in ManifestInput
	if err = json.Unmarshal(raw, &in); err != nil {
		return ManagedManifest{}, err
	}
	if _, err = ValidateManifest(in); err != nil {
		return ManagedManifest{}, err
	}
	return ManagedManifest{ID: id, Source: "external", ManifestInput: in}, nil
}

func (s *ManifestStore) List() ([]ManagedManifest, error) {
	items := []ManagedManifest{}
	for _, id := range registry.Names() {
		item, err := s.Get(id)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	ids, err := s.files.IDs()
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		item, err := s.Get(id)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// ApplyManifest records a validated selection through the shared ChainNew use case.
// Later composition steps read the pinned immutable manifest and template paths.
func (s *ManifestStore) ApplyManifest(ctx context.Context, d Deps, actor DeploymentActor, id, dir, keys, binary string) (ChainNewOut, error) {
	if actor.ID == "" || (actor.Role != "operator" && actor.Role != "admin") {
		return ChainNewOut{}, ErrDeploymentForbidden
	}
	item, err := s.Get(id)
	if err != nil {
		return ChainNewOut{}, err
	}
	p, err := ValidateManifest(item.ManifestInput)
	if err != nil {
		return ChainNewOut{}, err
	}
	in := ChainNewIn{DataDir: dir, Chain: p.Manifest().ID, KeysDir: keys, Binary: binary}
	if item.Source == "external" {
		// These files belong to this composition, never a client-supplied path.
		in.ManifestPath, in.TemplatePath, err = s.files.Pin(dir, item.Manifest, []byte(item.Template))
		if err != nil {
			return ChainNewOut{}, err
		}
	}
	return ChainNew(ctx, d, in)
}
