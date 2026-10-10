package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateDownloadsBeforeReplacingRunningBinary(t *testing.T) {
	tempDir := t.TempDir()
	oldDir, oldVersion := dir, Version
	dir, Version = tempDir, "v0.0.7"
	t.Cleanup(func() { dir, Version = oldDir, oldVersion })
	t.Setenv("SLINX_TEST_DIR", tempDir)
	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	oldBinary := filepath.Join(tempDir, "slinx")
	if err := os.WriteFile(oldBinary, []byte("old-binary"), 0755); err != nil {
		t.Fatal(err)
	}
	commands := map[string]string{
		"curl":      "#!/bin/sh\nprintf '{\\n  \"tag_name\": \"v0.0.8\"\\n}\\n'\n",
		"uname":     "#!/bin/sh\nprintf 'x86_64\\n'\n",
		"wget":      "#!/bin/sh\nif [ \"$3\" = \"$SLINX_TEST_DIR/slinx\" ]; then exit 26; fi\nprintf 'new-binary' > \"$3\"\n",
		"systemctl": "#!/bin/sh\nexit 0\n",
	}
	for name, script := range commands {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte(script), 0755); err != nil {
			t.Fatal(err)
		}
	}

	message := string(update()()().(updateResultMsg))
	if !strings.Contains(message, "更新成功") {
		t.Fatalf("update result = %q, want success", message)
	}
	data, err := os.ReadFile(oldBinary)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new-binary" {
		t.Fatalf("binary = %q, want downloaded binary", data)
	}
}

func TestUpdateFailedDownloadKeepsExistingBinary(t *testing.T) {
	tempDir := t.TempDir()
	oldDir, oldVersion := dir, Version
	dir, Version = tempDir, "v0.0.7"
	t.Cleanup(func() { dir, Version = oldDir, oldVersion })
	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	oldBinary := filepath.Join(tempDir, "slinx")
	if err := os.WriteFile(oldBinary, []byte("old-binary"), 0755); err != nil {
		t.Fatal(err)
	}
	commands := map[string]string{
		"curl":  "#!/bin/sh\nprintf '{\\n  \"tag_name\": \"v0.0.8\"\\n}\\n'\n",
		"uname": "#!/bin/sh\nprintf 'x86_64\\n'\n",
		"wget":  "#!/bin/sh\nexit 8\n",
	}
	for name, script := range commands {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte(script), 0755); err != nil {
			t.Fatal(err)
		}
	}

	message := string(update()()().(updateResultMsg))
	if !strings.Contains(message, "下载失败") {
		t.Fatalf("update result = %q, want download failure", message)
	}
	data, err := os.ReadFile(oldBinary)
	if err != nil || string(data) != "old-binary" {
		t.Fatalf("existing binary changed after failed download: %q, %v", data, err)
	}
	leftovers, err := filepath.Glob(filepath.Join(tempDir, ".slinx-update-*"))
	if err != nil || len(leftovers) != 0 {
		t.Fatalf("temporary files left behind: %v, %v", leftovers, err)
	}
}

func TestRemoveManagementCommands(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"slinx", "sbox"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("command"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := removeManagementCommands(dir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"slinx", "sbox"} {
		if _, err := os.Lstat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("%s still exists: %v", name, err)
		}
	}
}

func TestUninstallStopsBeforeDeletingConfig(t *testing.T) {
	installDir := t.TempDir()
	dbPath := filepath.Join(installDir, "data", "slinx.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dbPath, []byte("saved settings"), 0600); err != nil {
		t.Fatal(err)
	}
	removed := false
	err := uninstallAt(installDir, filepath.Join(t.TempDir(), "slinx.service"), t.TempDir(),
		func(_ string, args ...string) error {
			if args[0] == "stop" {
				return errors.New("service still running")
			}
			return nil
		},
		func(string) error { removed = true; return nil },
	)
	if err == nil || !strings.Contains(err.Error(), "停止") {
		t.Fatalf("stop failure was hidden: %v", err)
	}
	if removed {
		t.Fatal("configuration removed while service was still running")
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("configuration should remain after failed stop: %v", err)
	}
}

func TestUninstallReportsDeleteFailure(t *testing.T) {
	installDir := t.TempDir()
	dbPath := filepath.Join(installDir, "data", "slinx.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dbPath, []byte("saved settings"), 0600); err != nil {
		t.Fatal(err)
	}
	err := uninstallAt(installDir, filepath.Join(t.TempDir(), "slinx.service"), t.TempDir(),
		func(string, ...string) error { return nil },
		func(string) error { return errors.New("read-only filesystem") },
	)
	if err == nil || !strings.Contains(err.Error(), "删除") {
		t.Fatalf("delete failure was hidden: %v", err)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("configuration should remain after failed delete: %v", err)
	}
}

func TestUninstallRejectsUnremovedDirectory(t *testing.T) {
	installDir := t.TempDir()
	err := uninstallAt(installDir, filepath.Join(t.TempDir(), "slinx.service"), t.TempDir(),
		func(string, ...string) error { return nil },
		func(string) error { return nil },
	)
	if err == nil || !strings.Contains(err.Error(), "仍存在") {
		t.Fatalf("existing directory was reported as removed: %v", err)
	}
}

func TestUninstallRejectsBroadTarget(t *testing.T) {
	run := false
	err := uninstallAt("/", "/tmp/unused-slinx.service", "/tmp/unused-commands",
		func(string, ...string) error { run = true; return nil },
		func(string) error { run = true; return nil },
	)
	if err == nil || run {
		t.Fatalf("unsafe uninstall target was accepted: %v", err)
	}
}

func TestUninstallRemovesDatabaseAndCertificates(t *testing.T) {
	root := t.TempDir()
	installDir := filepath.Join(root, "slinx")
	commandDir := filepath.Join(root, "bin")
	unitFile := filepath.Join(root, "slinx.service")
	for _, path := range []string{
		filepath.Join(installDir, "data", "slinx.db"),
		filepath.Join(installDir, "cert", "panel.pem"),
		filepath.Join(commandDir, "sbox"),
		filepath.Join(commandDir, "slinx"),
		unitFile,
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var commands []string
	err := uninstallAt(installDir, unitFile, commandDir,
		func(_ string, args ...string) error {
			commands = append(commands, strings.Join(args, " "))
			return nil
		},
		os.RemoveAll,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(installDir); !os.IsNotExist(err) {
		t.Fatalf("install directory remains: %v", err)
	}
	if _, err := os.Stat(unitFile); !os.IsNotExist(err) {
		t.Fatalf("service file remains: %v", err)
	}
	for _, name := range []string{"sbox", "slinx"} {
		if _, err := os.Stat(filepath.Join(commandDir, name)); !os.IsNotExist(err) {
			t.Fatalf("command %s remains: %v", name, err)
		}
	}
	if got := strings.Join(commands, ","); got != "stop slinx.service,disable slinx.service,daemon-reload" {
		t.Fatalf("systemctl calls = %q", got)
	}
}
