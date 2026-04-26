package cli

import (
	"os"
	"runtime"
	"testing"
	"time"
)

func TestStoreNameCompletions_WritesPrivateFile(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("SONOSCLI_COMPLETION_CACHE_DIR", cacheDir)

	if err := storeNameCompletions(time.Now(), []string{"Kitchen"}); err != nil {
		t.Fatalf("store cache: %v", err)
	}

	path, err := nameCompletionCachePath()
	if err != nil {
		t.Fatalf("cache path: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat cache file: %v", err)
	}

	// Best-effort permission assertion: on Unix-like systems the cache should not be readable by group/other.
	if runtime.GOOS != "windows" {
		if info.Mode().Perm()&0o077 != 0 {
			t.Fatalf("cache file perms = %v, want no group/other bits", info.Mode().Perm())
		}
	}
}
