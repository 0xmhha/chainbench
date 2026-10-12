package app

// AuditManifest records Web mutations without retaining request bodies, paths to
// private key material, executable argv, or untrusted error messages.
func (s *ManifestStore) AuditManifest(actor, operation, target string, status int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.files.Audit(actor, operation, target, status)
}
