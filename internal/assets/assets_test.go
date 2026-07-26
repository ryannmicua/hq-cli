package assets_test

import (
	"os"
	"testing"

	"github.com/ryannmicua/hq-cli/internal/assets"
)

func TestResultSchemaV1_MatchesSourceFile(t *testing.T) {
	sourceBytes, err := os.ReadFile("schemas/result-v1.json")
	if err != nil {
		t.Fatalf("failed to read source schema file: %v", err)
	}

	if len(assets.ResultSchemaV1) == 0 {
		t.Fatal("embedded ResultSchemaV1 is empty")
	}

	if string(assets.ResultSchemaV1) != string(sourceBytes) {
		t.Fatal("embedded schema bytes do not match source file")
	}
}
