package accounts

import "fmt"

// TamperFeeDelegatedRaw changes one signature component in an existing signed
// 0x16 envelope without rebuilding its transaction fields. This supports raw
// transactions signed by a node's personal API. It validates the outer and
// sender wire shapes, not the validity of the original signatures or access
// list. The caller still has to require the node's specific signature refusal.
func TamperFeeDelegatedRaw(raw []byte, which string) ([]byte, error) {
	if len(raw) < 2 || len(raw) > 1<<20 || raw[0] != 0x16 {
		return nil, fmt.Errorf("accounts: raw must be a signed 0x16 envelope (at most 1 MiB)")
	}
	outer, err := signatureRLPAt(raw, 1)
	if err != nil || !outer.list || outer.end != len(raw) {
		return nil, fmt.Errorf("accounts: invalid raw outer RLP")
	}
	fields, err := signatureRLPFields(raw, outer, 5)
	if err != nil || !fields[0].list || fields[1].list || fields[1].end-fields[1].payload != 20 {
		return nil, fmt.Errorf("accounts: raw requires sender list and 20-byte fee payer")
	}
	inner, err := signatureRLPFields(raw, fields[0], 12)
	if err != nil || inner[5].list || (inner[5].end-inner[5].payload != 0 && inner[5].end-inner[5].payload != 20) || inner[7].list || !inner[8].list {
		return nil, fmt.Errorf("accounts: invalid raw sender transaction")
	}
	for _, field := range append(append([]signatureRLPItem{}, inner[:5]...), inner[6], inner[9], fields[2]) {
		if field.list || field.end-field.payload > 32 || (field.end > field.payload && raw[field.payload] == 0) {
			return nil, fmt.Errorf("accounts: invalid raw transaction integer")
		}
	}
	for _, v := range []signatureRLPItem{inner[9], fields[2]} {
		if v.end-v.payload > 1 || (v.end > v.payload && raw[v.payload] != 1) {
			return nil, fmt.Errorf("accounts: invalid raw signature parity")
		}
	}
	for _, component := range []signatureRLPItem{inner[10], inner[11], fields[3], fields[4]} {
		if component.list || component.end-component.payload < 1 || component.end-component.payload > 32 || raw[component.payload] == 0 {
			return nil, fmt.Errorf("accounts: raw signature component missing or noncanonical")
		}
	}
	var selected signatureRLPItem
	switch which {
	case "sender":
		selected = inner[11] // Sender.S; the payer signature covers this field.
	case "feepayer":
		selected = fields[3] // FeePayer.R.
	default:
		return nil, fmt.Errorf("accounts: raw signature must be sender or feepayer")
	}
	out := append([]byte(nil), raw...)
	mask := byte(1)
	if selected.end-selected.payload == 1 && out[selected.payload] == 1 {
		mask = 2 // Keep a one-byte integer nonzero and its RLP encoding canonical.
	}
	out[selected.end-1] ^= mask
	return out, nil
}

type signatureRLPItem struct {
	payload, end int
	list         bool
}

func signatureRLPAt(raw []byte, at int) (signatureRLPItem, error) {
	if at < 0 || at >= len(raw) {
		return signatureRLPItem{}, fmt.Errorf("truncated RLP")
	}
	tag := raw[at]
	if tag < 0x80 {
		return signatureRLPItem{payload: at, end: at + 1}, nil
	}
	base := byte(0x80)
	list := tag >= 0xc0
	if list {
		base = 0xc0
	}
	size, payload := uint64(tag-base), at+1
	if size > 55 {
		width := int(size - 55)
		if width > 8 || width > len(raw)-payload || raw[payload] == 0 {
			return signatureRLPItem{}, fmt.Errorf("invalid RLP length")
		}
		size = 0
		for _, b := range raw[payload : payload+width] {
			size = size<<8 | uint64(b)
		}
		payload += width
		if size < 56 {
			return signatureRLPItem{}, fmt.Errorf("noncanonical long RLP")
		}
	}
	if size > uint64(len(raw)-payload) || (!list && size == 1 && raw[payload] < 0x80) {
		return signatureRLPItem{}, fmt.Errorf("truncated or noncanonical RLP")
	}
	return signatureRLPItem{payload: payload, end: payload + int(size), list: list}, nil
}

func signatureRLPFields(raw []byte, list signatureRLPItem, count int) ([]signatureRLPItem, error) {
	fields := make([]signatureRLPItem, 0, count)
	for at := list.payload; at < list.end; {
		field, err := signatureRLPAt(raw, at)
		if err != nil || field.end > list.end || len(fields) == count {
			return nil, fmt.Errorf("invalid RLP fields")
		}
		fields = append(fields, field)
		at = field.end
	}
	if len(fields) != count {
		return nil, fmt.Errorf("wrong RLP field count")
	}
	return fields, nil
}
