package fsx

import (
	"os"
	"testing"
	"time"
)

func TestStaleTimeout_Default(t *testing.T) {
	os.Unsetenv("HQ_LOCK_STALE_TIMEOUT")
	cutoff := staleTimeout()
	if time.Since(cutoff) < 4*time.Minute || time.Since(cutoff) > 7*time.Minute {
		t.Fatalf("expected cutoff ~5min ago, got %v ago", time.Since(cutoff))
	}
}

func TestStaleTimeout_Custom(t *testing.T) {
	os.Setenv("HQ_LOCK_STALE_TIMEOUT", "30s")
	defer os.Unsetenv("HQ_LOCK_STALE_TIMEOUT")
	cutoff := staleTimeout()
	if time.Since(cutoff) < 25*time.Second || time.Since(cutoff) > 35*time.Second {
		t.Fatalf("expected cutoff ~30s ago, got %v ago", time.Since(cutoff))
	}
}

func TestStaleTimeout_Invalid(t *testing.T) {
	os.Setenv("HQ_LOCK_STALE_TIMEOUT", "not-a-duration")
	defer os.Unsetenv("HQ_LOCK_STALE_TIMEOUT")
	cutoff := staleTimeout()
	if time.Since(cutoff) < 4*time.Minute || time.Since(cutoff) > 7*time.Minute {
		t.Fatalf("expected fallback to ~5min, got %v ago", time.Since(cutoff))
	}
}

func TestStaleTimeout_Negative(t *testing.T) {
	os.Setenv("HQ_LOCK_STALE_TIMEOUT", "-30s")
	defer os.Unsetenv("HQ_LOCK_STALE_TIMEOUT")
	cutoff := staleTimeout()
	if time.Since(cutoff) < 4*time.Minute || time.Since(cutoff) > 7*time.Minute {
		t.Fatalf("expected fallback to ~5min for negative, got %v ago", time.Since(cutoff))
	}
}

func TestStaleTimeout_Zero(t *testing.T) {
	os.Setenv("HQ_LOCK_STALE_TIMEOUT", "0s")
	defer os.Unsetenv("HQ_LOCK_STALE_TIMEOUT")
	cutoff := staleTimeout()
	if time.Since(cutoff) < 4*time.Minute || time.Since(cutoff) > 7*time.Minute {
		t.Fatalf("expected fallback to ~5min for zero, got %v ago", time.Since(cutoff))
	}
}

func TestRetryDelay_Monotonic(t *testing.T) {
	for i := 0; i < 10; i++ {
		d := retryDelay(i)
		if d <= 0 {
			t.Fatalf("retryDelay(%d) = %v, expected > 0", i, d)
		}
	}
}

func TestRetryDelay_Capped(t *testing.T) {
	prev := time.Duration(0)
	for i := 0; i < 20; i++ {
		d := retryDelay(i)
		if d <= 0 {
			t.Fatalf("retryDelay(%d) = %v, expected > 0", i, d)
		}
		if d < prev/2 && i > 0 {
			t.Fatalf("retryDelay(%d) = %v, should not drop below half of previous %v", i, d, prev)
		}
		if d > 10*time.Second {
			t.Fatalf("retryDelay(%d) = %v, exceeds 10s cap", i, d)
		}
		prev = d
	}
}

func TestResolveHostname_NeverEmpty(t *testing.T) {
	h := resolveHostname()
	if h == "" {
		t.Fatal("expected non-empty hostname")
	}
}

func TestParseCapOverride_Basic(t *testing.T) {
	c := parseCapOverride("ext4", "/tmp")
	if c.FilesystemType != "ext4" {
		t.Fatalf("expected ext4, got %s", c.FilesystemType)
	}
	if !c.SupportAtomicReplace {
		t.Fatal("expected atomic replace support without no-atomic flag")
	}
}

func TestParseCapOverride_NoAtomic(t *testing.T) {
	c := parseCapOverride("nfs,no-atomic", "/tmp")
	if c.FilesystemType != "nfs" {
		t.Fatalf("expected nfs, got %s", c.FilesystemType)
	}
	if c.SupportAtomicReplace {
		t.Fatal("expected no atomic replace when no-atomic is set")
	}
}

func TestParseCapOverride_ExtraComma(t *testing.T) {
	c := parseCapOverride("nfs,no-atomic,ro", "/tmp")
	if c.SupportAtomicReplace {
		t.Fatal("expected no atomic replace when no-atomic with extra comma")
	}
}

func TestParseCapOverride_EmptyFsType(t *testing.T) {
	c := parseCapOverride(",no-atomic", "/tmp")
	if c.FilesystemType == "" {
		t.Fatal("expected non-empty fsType fallback")
	}
}

func TestParseCapOverride_SingleValue(t *testing.T) {
	c := parseCapOverride("ntfs", "/tmp")
	if c.FilesystemType != "ntfs" {
		t.Fatalf("expected ntfs, got %s", c.FilesystemType)
	}
	if !c.SupportAtomicReplace {
		t.Fatal("expected atomic replace support")
	}
}
