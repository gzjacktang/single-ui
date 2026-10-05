package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureSboxCommand(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "slinx")
	if err := os.WriteFile(legacy, []byte("legacy command"), 0755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := EnsureSboxCommand(dir); err != nil {
			t.Fatal(err)
		}
	}
	target, err := os.Readlink(filepath.Join(dir, "sbox"))
	if err != nil || target != legacy {
		t.Fatalf("sbox command target = %q, %v; want %q", target, err, legacy)
	}
}
