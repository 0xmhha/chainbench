package app

import (
	"encoding/json"
	"io"
	"regexp"
	"strings"
)

var webPEM = regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`)
var webSecretField = regexp.MustCompile(`(?i)("(?:password|passwordHash|privateKey|private_key|passphrase|setupToken|key_passphrase)"\s*:\s*)"(?:\\.|[^"\\])*"`)

// RedactWeb removes all users' registered material from logs and exports.
// Parse serialized JSON before replacing values so quotes and short passwords
// cannot corrupt the wire format or rewrite contract field names.
func (s *DeploymentStore) RedactWeb(text string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	secrets := []string{}
	for _, c := range s.state.Credentials {
		n := s.aead.NonceSize()
		if len(c.Ciphertext) < n {
			continue
		}
		b, err := s.aead.Open(nil, c.Ciphertext[:n], c.Ciphertext[n:], []byte(c.Metadata.ID+":"+c.Metadata.OwnerID))
		if err != nil {
			continue
		}
		var in DeploymentCredentialInput
		if json.Unmarshal(b, &in) != nil {
			continue
		}
		secrets = append(secrets, in.Password, in.PrivateKey, in.Passphrase)
	}
	prefix, suffix := "", ""
	raw := text
	if strings.HasPrefix(text, "data: ") {
		prefix = "data: "
		suffix = "\n\n"
		raw = strings.TrimSpace(strings.TrimPrefix(text, prefix))
	}
	var value any
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&value) == nil && decoder.Decode(new(any)) == io.EOF {
		value = redactWebObject(value, secrets)
		b, err := json.Marshal(value)
		if err == nil {
			return prefix + string(b) + suffix
		}
	}
	return redactWebString(text, secrets)
}
func redactWebObject(value any, secrets []string) any {
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			switch strings.ToLower(key) {
			case "password", "passwordhash", "privatekey", "private_key", "passphrase", "setuptoken", "key_passphrase":
				v[key] = "[REDACTED]"
			default:
				v[key] = redactWebObject(item, secrets)
			}
		}
	case []any:
		for i, item := range v {
			v[i] = redactWebObject(item, secrets)
		}
	case string:
		return redactWebString(v, secrets)
	}
	return value
}
func redactWebString(text string, secrets []string) string {
	for _, secret := range secrets {
		if secret == text && secret != "" {
			return "[REDACTED]"
		}
		if len(secret) >= 8 {
			text = strings.ReplaceAll(text, secret, "[REDACTED]")
		}
	}
	return RedactWebText(text)
}
func RedactWebText(text string) string {
	text = webPEM.ReplaceAllString(text, "[REDACTED]")
	return webSecretField.ReplaceAllString(text, `${1}"[REDACTED]"`)
}
