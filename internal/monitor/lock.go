package monitor

import (
	"fmt"
	"os"
	"time"
)

// lockStaleAfter bounds how long a lock file is honored. It guards against a
// crashed run leaving a stale lock behind forever, while still being long
// enough that a legitimate `monitor check` run never gets pre-empted by
// itself under normal scheduler intervals.
const lockStaleAfter = 10 * time.Minute

// acquireLock creates an exclusive lock file at path, refusing to proceed if
// another run already holds a fresh one. It returns a release function that
// must be called (typically via defer) once the caller is done.
func acquireLock(path string) (func(), error) {
	if info, err := os.Stat(path); err == nil {
		if time.Since(info.ModTime()) < lockStaleAfter {
			return nil, fmt.Errorf("já existe uma verificação em andamento (lock: %s)", path)
		}
		_ = os.Remove(path)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("não foi possível criar lock em %s: %w", path, err)
	}
	_ = f.Close()

	return func() { _ = os.Remove(path) }, nil
}
