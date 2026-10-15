package testengine_test

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"testing"
)

func TestAuthorizationRecorderReadsGzipWithoutChangingWireBytes(t *testing.T) {
	request := []byte(`{"jsonrpc":"2.0","id":1,"method":"eth_estimateGas","params":[{}]}`)
	response := []byte(`{"jsonrpc":"2.0","id":1,"result":"0x5208"}`)
	for _, encoding := range []string{"", "gzip"} {
		t.Run(encoding, func(t *testing.T) {
			wire := response
			if encoding == "gzip" {
				var buf bytes.Buffer
				writer := gzip.NewWriter(&buf)
				if _, err := writer.Write(response); err != nil {
					t.Fatal(err)
				}
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
				wire = buf.Bytes()
			}
			before := bytes.Clone(wire)
			recorded, ok := authorizationRPCObservation(request, wire, encoding)
			if !ok {
				t.Fatal("recorder discarded a valid encoded node response")
			}
			raw, err := base64.StdEncoding.DecodeString(recorded["responseBytesBase64"].(string))
			if err != nil || !bytes.Equal(raw, before) || !bytes.Equal(wire, before) || recorded["contentEncoding"] != encoding || recorded["requestBody"] != string(request) || recorded["responseBody"] != string(response) {
				t.Fatal("recording changed the original wire body", err)
			}
		})
	}
}
