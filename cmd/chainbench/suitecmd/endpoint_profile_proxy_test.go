package suitecmd_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func inspectNativeEstimate(response *http.Response) (string, error) {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	if err := response.Body.Close(); err != nil {
		return "", err
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	decoded := body
	if strings.EqualFold(response.Header.Get("Content-Encoding"), "gzip") {
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		decoded, err = io.ReadAll(reader)
		closeErr := reader.Close()
		if err != nil {
			return "", err
		}
		if closeErr != nil {
			return "", closeErr
		}
	}
	var reply struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(decoded, &reply); err != nil {
		return "", err
	}
	return reply.Result, nil
}

func TestNativeEstimateObservationPreservesCompressedWireBytes(t *testing.T) {
	for _, encoding := range []string{"", "gzip"} {
		t.Run(encoding, func(t *testing.T) {
			body := []byte(`{"jsonrpc":"2.0","id":1,"result":"0x10000"}`)
			if encoding == "gzip" {
				var buffer bytes.Buffer
				writer := gzip.NewWriter(&buffer)
				if _, err := writer.Write(body); err != nil {
					t.Fatal(err)
				}
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
				body = buffer.Bytes()
			}
			response := &http.Response{Body: io.NopCloser(bytes.NewReader(body)), Header: http.Header{"Content-Encoding": []string{encoding}}, ContentLength: int64(len(body))}
			gas, err := inspectNativeEstimate(response)
			forwarded, readErr := io.ReadAll(response.Body)
			if readErr != nil || !bytes.Equal(forwarded, body) || response.Header.Get("Content-Encoding") != encoding || response.ContentLength != int64(len(body)) {
				t.Fatal("observation changed forwarded bytes or headers")
			}
			if err != nil || gas != "0x10000" {
				t.Fatalf("observation must decode %q without changing its wire bytes: %q %v", encoding, gas, err)
			}
		})
	}
}
