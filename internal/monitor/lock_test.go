package monitor

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAcquireLockBlocksConcurrentRun(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "test.lock")

	release, err := acquireLock(lockPath)
	if err != nil {
		t.Fatalf("expected first lock to succeed: %v", err)
	}
	defer release()

	if _, err := acquireLock(lockPath); err == nil {
		t.Fatal("expected second concurrent lock to fail")
	}
}

func TestAcquireLockReleaseAllowsNextRun(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "test.lock")

	release, err := acquireLock(lockPath)
	if err != nil {
		t.Fatalf("expected lock to succeed: %v", err)
	}
	release()

	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatal("expected lock file to be removed after release")
	}

	release2, err := acquireLock(lockPath)
	if err != nil {
		t.Fatalf("expected lock to succeed after release: %v", err)
	}
	release2()
}

func TestAcquireLockStealsStaleLock(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "test.lock")

	if err := os.WriteFile(lockPath, nil, 0644); err != nil {
		t.Fatalf("failed to create fake stale lock: %v", err)
	}
	staleTime := time.Now().Add(-2 * lockStaleAfter)
	if err := os.Chtimes(lockPath, staleTime, staleTime); err != nil {
		t.Fatalf("failed to backdate lock file: %v", err)
	}

	release, err := acquireLock(lockPath)
	if err != nil {
		t.Fatalf("expected stale lock to be stolen: %v", err)
	}
	release()
}
