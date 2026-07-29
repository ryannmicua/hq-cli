package write

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

type PolicyClass string

const (
	PolicyAllowed          PolicyClass = "allowed"
	PolicyApprovalRequired PolicyClass = "approval-required"
	PolicySubmitOnly       PolicyClass = "submit-only"
	PolicyDenied           PolicyClass = "denied"
)

type PolicyRule struct {
	Operation     string      `json:"operation"`
	TargetPattern string      `json:"targetPattern"`
	Class         PolicyClass `json:"class"`
}

type Policy struct {
	SchemaVersion string       `json:"schemaVersion"`
	Rules         []PolicyRule `json:"rules"`
	DefaultClass  PolicyClass  `json:"defaultClass"`
}

func NewPolicy(rules []PolicyRule, defaultClass PolicyClass) (*Policy, error) {
	seen := make(map[string]bool)
	for _, r := range rules {
		key := r.Operation + ":" + r.TargetPattern
		if seen[key] {
			return nil, fmt.Errorf("duplicate rule for operation=%q targetPattern=%q", r.Operation, r.TargetPattern)
		}
		seen[key] = true
	}
	return &Policy{Rules: rules, DefaultClass: defaultClass}, nil
}

func (p *Policy) Classify(operation, target string) (PolicyClass, error) {
	for _, rule := range p.Rules {
		if rule.Operation == "*" || rule.Operation == operation {
			if matchPattern(rule.TargetPattern, target) {
				return rule.Class, nil
			}
		}
	}
	return p.DefaultClass, nil
}

func matchPattern(pattern, target string) bool {
	patternParts := strings.Split(pattern, "/")
	targetParts := strings.Split(target, "/")

	pi := 0
	ti := 0
	for pi < len(patternParts) && ti < len(targetParts) {
		if patternParts[pi] == "**" {
			if pi == len(patternParts)-1 {
				return true
			}
			remaining := strings.Join(patternParts[pi+1:], "/")
			for ti < len(targetParts) {
				if matchPattern(remaining, strings.Join(targetParts[ti:], "/")) {
					return true
				}
				ti++
			}
			return false
		}
		if patternParts[pi] == "*" || matchComponent(patternParts[pi], targetParts[ti]) {
			pi++
			ti++
			continue
		}
		return false
	}
	return pi == len(patternParts) && ti == len(targetParts)
}

func matchComponent(pattern, target string) bool {
	if pattern == target {
		return true
	}
	matched, _ := filepath.Match(pattern, target)
	return matched
}

func ParsePolicyJSON(data []byte) (*Policy, error) {
	var p Policy
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("unmarshal policy: %w", err)
	}
	if p.DefaultClass == "" {
		p.DefaultClass = PolicyDenied
	}
	return &p, nil
}

func ValidatePolicy(p *Policy) error {
	if p.DefaultClass != PolicyDenied {
		return fmt.Errorf("defaultClass must be %q, got %q", PolicyDenied, p.DefaultClass)
	}

	deniedPatterns := []string{
		"identity/",
		"templates/",
		"decisions/",
		"people/",
		"references/",
		"work-types/",
		"AGENTS.md",
		"safety-boundaries.md",
	}

	for _, rule := range p.Rules {
		for _, denied := range deniedPatterns {
			if strings.Contains(rule.TargetPattern, denied) && rule.Class != PolicyDenied {
				return fmt.Errorf("rule for %q pattern %q attempts to allow a denied path", rule.Operation, rule.TargetPattern)
			}
		}
	}

	seen := make(map[string]bool)
	for _, r := range p.Rules {
		key := r.Operation + ":" + r.TargetPattern
		if seen[key] {
			return fmt.Errorf("duplicate rule for operation=%q targetPattern=%q", r.Operation, r.TargetPattern)
		}
		seen[key] = true
	}

	return nil
}
