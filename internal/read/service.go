// Package read provides the read-only services for the HQ CLI:
// version, health, context, get, list, and search.
package read

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/ryannmicua/hq-cli/internal/assets"
	"github.com/ryannmicua/hq-cli/internal/config"
	"github.com/ryannmicua/hq-cli/internal/hq"
)

// Service holds the dependencies for read-only HQ operations.
type Service struct {
	config   *config.Config
	resolver *hq.Resolver
}

// NewService creates a read Service from the given config. It initializes
// a path resolver from the config root.
func NewService(cfg *config.Config) (*Service, error) {
	resolver, err := hq.NewResolver(cfg.Root)
	if err != nil {
		return nil, fmt.Errorf("create resolver: %w", err)
	}
	return &Service{config: cfg, resolver: resolver}, nil
}

// Resolver returns the underlying path resolver.
func (s *Service) Resolver() *hq.Resolver {
	return s.resolver
}

// Config returns the service's configuration.
func (s *Service) Config() *config.Config {
	return s.config
}

// VersionInfo represents the version command data payload.
type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"buildTime"`
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// Version returns build and runtime version information.
// In development builds, ldflags values are empty strings.
func (s *Service) Version() VersionInfo {
	return VersionInfo{
		Version:   version,
		Commit:    commit,
		BuildTime: buildTime,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// Build-time ldflags variables (set at release; empty in dev).
var (
	version   = "0.1.0"
	commit    = ""
	buildTime = ""
)

// HealthCheck represents a single health check result.
type HealthCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "pass", "warn", "fail", "not-applicable"
	Message string `json:"message"`
}

// HealthResult represents the health command data payload.
type HealthResult struct {
	Overall string        `json:"overall"`
	Checks  []HealthCheck `json:"checks"`
}

// Health runs all Phase 0 capability checks and returns a composite result.
func (s *Service) Health() HealthResult {
	var checks []HealthCheck

	// Binary / executable presence (self-check)
	checks = append(checks, HealthCheck{
		Name:    "binary",
		Status:  "pass",
		Message: "executable is running",
	})

	// Embedded contract assets
	assetCheck := HealthCheck{Name: "contract-assets", Status: "pass", Message: "embedded result schema present"}
	if len(assets.ResultSchemaV1) == 0 {
		assetCheck = HealthCheck{Name: "contract-assets", Status: "fail", Message: "embedded result schema is empty"}
	}
	checks = append(checks, assetCheck)

	// HQ root readability
	rootCheck := HealthCheck{Name: "hq-root", Status: "pass", Message: fmt.Sprintf("root is readable: %s", s.config.Root)}
	if _, err := os.Stat(s.config.Root); err != nil {
		rootCheck = HealthCheck{Name: "hq-root", Status: "fail", Message: fmt.Sprintf("cannot access root: %v", err)}
	}
	checks = append(checks, rootCheck)

	// Git availability (read-only metadata only)
	gitCheck := HealthCheck{Name: "git", Status: "pass", Message: "Git is available"}
	if _, err := exec.LookPath("git"); err != nil {
		gitCheck = HealthCheck{Name: "git", Status: "warn", Message: "Git metadata unavailable"}
	}
	checks = append(checks, gitCheck)

	// Read-path containment (can resolve a path within root)
	pathCheck := HealthCheck{Name: "read-path", Status: "pass", Message: "path resolution working"}
	if _, err := s.resolver.Resolve("."); err != nil {
		pathCheck = HealthCheck{Name: "read-path", Status: "fail", Message: fmt.Sprintf("path resolution failed: %v", err)}
	}
	checks = append(checks, pathCheck)

	// Determine overall status.
	overall := "pass"
	for _, c := range checks {
		if c.Status == "fail" {
			overall = "fail"
			break
		}
	}

	return HealthResult{Overall: overall, Checks: checks}
}

// ContextOpts configures the context command's output.
type ContextOpts struct {
	Project      string // explicit project slug (optional)
	SessionCount int    // number of recent session entries (default 20)
}
