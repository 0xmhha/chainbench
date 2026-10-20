package app_test

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
)

func TestPreparedDependenciesCannotPartiallyExecuteAsSeparateDefinitions(t *testing.T) {
	producer := []byte(`{"schemaVersion":"2","kind":"case","id":"prepare","chainPreset":{"chain":"wbft"},"prepared":{"exports":{"head":"$head"}},"steps":[{"do":"read","source":"blockNumber","save":"head"},{"expect":"blockNumber","is":0}]}`)
	consumer := []byte(`{"schemaVersion":"2","kind":"case","id":"consume","chainPreset":{"chain":"wbft"},"prepared":{"imports":{"head":{"case":"prepare","name":"head"}}},"steps":[{"expect":"blockNumber","is":"$head"}]}`)
	out, err := app.RunSuites(t.Context(), app.Deps{}, app.RunSuiteIn{SpecContent: [][]byte{producer, consumer}})
	if err == nil || !strings.Contains(err.Error(), "--suite") || len(out.Runs) != 0 {
		t.Fatal("a prepared batch was partially dispatched to separate executions", err, out)
	}
}
