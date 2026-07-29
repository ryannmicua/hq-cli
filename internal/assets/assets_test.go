package assets_test

import (
	"os"
	"testing"

	"github.com/ryannmicua/hq-cli/internal/assets"
	"github.com/ryannmicua/hq-cli/internal/write"
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

func TestPolicyV1_NonEmpty(t *testing.T) {
	if len(assets.PolicyV1) == 0 {
		t.Fatal("PolicyV1 is empty")
	}
}

func TestPolicyV1_ParsesAsValidPolicy(t *testing.T) {
	p, err := write.ParsePolicyJSON(assets.PolicyV1)
	if err != nil {
		t.Fatalf("ParsePolicyJSON failed: %v", err)
	}
	if err := write.ValidatePolicy(p); err != nil {
		t.Fatalf("ValidatePolicy failed: %v", err)
	}
}

func TestPolicyV1_MatchesConfigSource(t *testing.T) {
	sourceBytes, err := os.ReadFile("../../config/policy-v1.json")
	if err != nil {
		t.Fatalf("failed to read config/policy-v1.json: %v", err)
	}
	if string(assets.PolicyV1) != string(sourceBytes) {
		t.Fatal("embedded policy bytes do not match config/policy-v1.json source")
	}
}

func TestPolicyV1_MatchesSourceFile(t *testing.T) {
	sourceBytes, err := os.ReadFile("policy-v1.json")
	if err != nil {
		t.Fatalf("failed to read source policy file: %v", err)
	}
	if string(assets.PolicyV1) != string(sourceBytes) {
		t.Fatal("embedded policy bytes do not match source file")
	}
}

func TestPolicyV1_DefaultDenied(t *testing.T) {
	p, _ := write.ParsePolicyJSON(assets.PolicyV1)
	if p.DefaultClass != write.PolicyDenied {
		t.Fatalf("defaultClass = %q, want %q", p.DefaultClass, write.PolicyDenied)
	}
}

func TestReceiptSchemaV1_MatchesSourceFile(t *testing.T) {
	sourceBytes, err := os.ReadFile("schemas/receipt-v1.json")
	if err != nil {
		t.Fatalf("failed to read source receipt schema: %v", err)
	}

	if len(assets.ReceiptSchemaV1) == 0 {
		t.Fatal("embedded ReceiptSchemaV1 is empty")
	}

	if string(assets.ReceiptSchemaV1) != string(sourceBytes) {
		t.Fatal("embedded receipt schema bytes do not match source file")
	}
}

func TestReceiptSchemaV1_MatchesSchemasDir(t *testing.T) {
	sourceBytes, err := os.ReadFile("../../schemas/receipt-v1.json")
	if err != nil {
		t.Fatalf("failed to read schemas/receipt-v1.json: %v", err)
	}

	if len(assets.ReceiptSchemaV1) == 0 {
		t.Fatal("embedded ReceiptSchemaV1 is empty")
	}

	if string(assets.ReceiptSchemaV1) != string(sourceBytes) {
		t.Fatal("embedded receipt schema bytes do not match schemas/ source")
	}
}

func TestPolicyV1_NoDuplicateRules(t *testing.T) {
	p, _ := write.ParsePolicyJSON(assets.PolicyV1)
	seen := make(map[string]bool)
	for _, r := range p.Rules {
		key := r.Operation + ":" + r.TargetPattern
		if seen[key] {
			t.Fatalf("duplicate rule: %s", key)
		}
		seen[key] = true
	}
}
