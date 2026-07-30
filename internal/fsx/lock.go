package fsx

import (
	"math/rand"
	"os"
	"strings"
	"time"
)

func staleTimeout() time.Time {
	d := 5 * time.Minute
	if s := os.Getenv("HQ_LOCK_STALE_TIMEOUT"); s != "" {
		if v, err := time.ParseDuration(s); err == nil && v > 0 {
			d = v
		}
	}
	return time.Now().Add(-d)
}

func retryDelay(attempt int) time.Duration {
	base := 50 * time.Millisecond
	maxCap := 5 * time.Second
	d := base << min(attempt, 7)
	if d > maxCap {
		d = maxCap
	}
	jitter := time.Duration(rand.Int63n(int64(d) / 2))
	return d/2 + jitter
}

func resolveHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown-host"
	}
	return hostname
}

func parseCapOverride(override, rootPath string) Capabilities {
	parts := strings.SplitN(override, ",", 3)
	fsType := parts[0]
	if fsType == "" {
		fsType = "override"
	}
	supportReplace := len(parts) < 2 || parts[1] != "no-atomic"
	return Capabilities{
		SupportAtomicReplace:        supportReplace,
		SupportFileLocking:          true,
		FilesystemType:              fsType,
		RootPath:                    rootPath,
		SupportOwnerOnlyPermissions: true,
	}
}
