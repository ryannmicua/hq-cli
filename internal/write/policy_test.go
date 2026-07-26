package write_test

import (
	"testing"

	"github.com/ryannmicua/hq-cli/internal/write"
)

func TestClassifyProjectCheckInState(t *testing.T) {
	p := mustPolicy(t)
	class, err := p.Classify("project-check-in", "projects/hq-cli/STATE.md")
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}
	if class != write.PolicyAllowed {
		t.Fatalf("class = %q, want %q", class, write.PolicyAllowed)
	}
}

func TestClassifySessionEntry(t *testing.T) {
	p := mustPolicy(t)
	class, err := p.Classify("session-entry", "SESSION-LOG.md")
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}
	if class != write.PolicyAllowed {
		t.Fatalf("class = %q, want %q", class, write.PolicyAllowed)
	}
}

func TestClassifyDeniedIdentityPath(t *testing.T) {
	p := mustPolicy(t)
	class, err := p.Classify("project-check-in", "identity/me.md")
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}
	if class != write.PolicyDenied {
		t.Fatalf("class = %q, want %q", class, write.PolicyDenied)
	}
}

func TestClassifyDeniedAgentsFile(t *testing.T) {
	p := mustPolicy(t)
	class, err := p.Classify("project-check-in", "AGENTS.md")
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}
	if class != write.PolicyDenied {
		t.Fatalf("class = %q, want %q", class, write.PolicyDenied)
	}
}

func TestClassifyDeniedTemplatesPath(t *testing.T) {
	p := mustPolicy(t)
	class, err := p.Classify("project-check-in", "templates/example.md")
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}
	if class != write.PolicyDenied {
		t.Fatalf("class = %q, want %q", class, write.PolicyDenied)
	}
}

func TestClassifyUnknownOperationDefaultDenied(t *testing.T) {
	p := mustPolicy(t)
	class, err := p.Classify("unknown-op", "projects/example/STATE.md")
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}
	if class != write.PolicyDenied {
		t.Fatalf("class = %q, want %q", class, write.PolicyDenied)
	}
}

func TestClassifyNonexistentPathDefaultDenied(t *testing.T) {
	p := mustPolicy(t)
	class, err := p.Classify("project-check-in", "nonexistent/path.md")
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}
	if class != write.PolicyDenied {
		t.Fatalf("class = %q, want %q", class, write.PolicyDenied)
	}
}

func TestNewPolicyDuplicateRule(t *testing.T) {
	_, err := write.NewPolicy([]write.PolicyRule{
		{Operation: "project-check-in", TargetPattern: "projects/*/STATE.md", Class: write.PolicyAllowed},
		{Operation: "project-check-in", TargetPattern: "projects/*/STATE.md", Class: write.PolicyAllowed},
	}, write.PolicyDenied)
	if err == nil {
		t.Fatal("expected error for duplicate rule")
	}
	if err.Error() != `duplicate rule for operation="project-check-in" targetPattern="projects/*/STATE.md"` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParsePolicyJSONValid(t *testing.T) {
	data := []byte(`{"schemaVersion":"1.0","rules":[{"operation":"project-check-in","targetPattern":"projects/*/STATE.md","class":"allowed"}],"defaultClass":"denied"}`)
	p, err := write.ParsePolicyJSON(data)
	if err != nil {
		t.Fatalf("ParsePolicyJSON failed: %v", err)
	}
	if p.DefaultClass != write.PolicyDenied {
		t.Fatalf("DefaultClass = %q, want %q", p.DefaultClass, write.PolicyDenied)
	}
	if len(p.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(p.Rules))
	}
}

func TestParsePolicyJSONInvalid(t *testing.T) {
	_, err := write.ParsePolicyJSON([]byte(`invalid json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestGlobStarMatchesMultiDepth(t *testing.T) {
	p := mustPolicySingleRule("project-check-in", "identity/**", write.PolicyDenied)

	class, _ := p.Classify("project-check-in", "identity/me.md")
	if class != write.PolicyDenied {
		t.Fatalf("expected denied for identity/me.md (1 level), got %q", class)
	}

	class, _ = p.Classify("project-check-in", "identity/subdir/file.md")
	if class != write.PolicyDenied {
		t.Fatalf("expected denied for identity/subdir/file.md (3 levels), got %q", class)
	}

	class, _ = p.Classify("project-check-in", "identity/a/b/c/d.md")
	if class != write.PolicyDenied {
		t.Fatalf("expected denied for identity/a/b/c/d.md (5 levels), got %q", class)
	}

	class, _ = p.Classify("project-check-in", "projects/foo/STATE.md")
	if class != write.PolicyDenied {
		t.Fatalf("expected denied (default) for non-matching path, got %q", class)
	}
}

func TestWildcardMatchesOneComponent(t *testing.T) {
	p := mustPolicySingleRule("project-check-in", "projects/*/STATE.md", write.PolicyAllowed)

	class, _ := p.Classify("project-check-in", "projects/foo/STATE.md")
	if class != write.PolicyAllowed {
		t.Fatalf("expected allowed for projects/foo/STATE.md, got %q", class)
	}

	class, _ = p.Classify("project-check-in", "projects/foo/bar/STATE.md")
	if class != write.PolicyDenied {
		t.Fatalf("expected denied for projects/foo/bar/STATE.md (two levels deep), got %q", class)
	}
}

func TestValidatePolicyDefaultDenied(t *testing.T) {
	p := mustPolicy(t)
	if err := write.ValidatePolicy(p); err != nil {
		t.Fatalf("ValidatePolicy failed: %v", err)
	}
}

func TestValidatePolicyFailsNonDeniedDefault(t *testing.T) {
	rules := []write.PolicyRule{
		{Operation: "project-check-in", TargetPattern: "projects/*/STATE.md", Class: write.PolicyAllowed},
	}
	p, _ := write.NewPolicy(rules, write.PolicyAllowed)
	err := write.ValidatePolicy(p)
	if err == nil {
		t.Fatal("expected error for non-denied default")
	}
}

func TestValidatePolicyFailsAllowedDeniedPath(t *testing.T) {
	rules := []write.PolicyRule{
		{Operation: "project-check-in", TargetPattern: "identity/*", Class: write.PolicyAllowed},
	}
	p, _ := write.NewPolicy(rules, write.PolicyDenied)
	err := write.ValidatePolicy(p)
	if err == nil {
		t.Fatal("expected error for allowing denied path")
	}
}

func TestValidatePolicyEmptyRulesValid(t *testing.T) {
	p, _ := write.NewPolicy(nil, write.PolicyDenied)
	if err := write.ValidatePolicy(p); err != nil {
		t.Fatalf("expected no error for empty rules with denied default, got: %v", err)
	}
}

func mustPolicy(t *testing.T) *write.Policy {
	t.Helper()
	data := []byte(`{
		"schemaVersion": "1.0",
		"rules": [
			{"operation":"project-check-in","targetPattern":"projects/*/STATE.md","class":"allowed"},
			{"operation":"session-entry","targetPattern":"SESSION-LOG.md","class":"allowed"},
			{"operation":"draft-record","targetPattern":"projects/*/*.md","class":"allowed"},
			{"operation":"current-work-update","targetPattern":"CURRENT-WORK.md","class":"allowed"},
			{"operation":"*","targetPattern":"identity/**","class":"denied"},
			{"operation":"*","targetPattern":"templates/**","class":"denied"},
			{"operation":"*","targetPattern":"decisions/**","class":"denied"},
			{"operation":"*","targetPattern":"people/**","class":"denied"},
			{"operation":"*","targetPattern":"references/**","class":"denied"},
			{"operation":"*","targetPattern":"work-types/**","class":"denied"},
			{"operation":"*","targetPattern":"AGENTS.md","class":"denied"},
			{"operation":"*","targetPattern":"safety-boundaries.md","class":"denied"}
		],
		"defaultClass": "denied"
	}`)
	p, err := write.ParsePolicyJSON(data)
	if err != nil {
		t.Fatalf("mustPolicy: %v", err)
	}
	return p
}

func mustPolicySingleRule(op, pattern string, class write.PolicyClass) *write.Policy {
	p, _ := write.NewPolicy([]write.PolicyRule{
		{Operation: op, TargetPattern: pattern, Class: class},
	}, write.PolicyDenied)
	return p
}
