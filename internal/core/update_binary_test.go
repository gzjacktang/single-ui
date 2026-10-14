package core

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gzjacktang/single-ui/internal/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBinaryUpdaterInstallsVerifiedCore(t *testing.T) {
	dir := t.TempDir()
	installed := filepath.Join(dir, "sing-box")
	config := filepath.Join(dir, "config.json")
	if err := os.WriteFile(installed, []byte("#!/bin/sh\necho 'sing-box version 1.13.13'\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	newBinary := []byte("#!/bin/sh\nif [ \"$1\" = version ]; then echo 'sing-box version 1.14.3'; fi\n")
	var archive bytes.Buffer
	compressed := gzip.NewWriter(&archive)
	if _, err := compressed.Write(newBinary); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archive.Bytes())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases":
			fmt.Fprint(w, `[{"tag_name":"sing-box-1.14.3"}]`)
		case "/sing-box-1.14.3/SHA256SUMS":
			fmt.Fprintf(w, "%x  sing-box_linux_amd64.gz\n", sum)
		case "/sing-box-1.14.3/sing-box_linux_amd64.gz":
			w.Write(archive.Bytes())
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	oldDB := database.DB
	t.Cleanup(func() { database.DB = oldDB })
	var err error
	database.DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.DB.AutoMigrate(&database.Core{}); err != nil {
		t.Fatal(err)
	}
	if err := database.DB.Create(&database.Core{ID: 1, Name: "sing-box", Version: "1.13.13"}).Error; err != nil {
		t.Fatal(err)
	}
	u := &BinaryUpdater{
		client: server.Client(), releasesURL: server.URL + "/releases", downloadsURL: server.URL,
		manager: &Manager{BinPath: installed, ConfigPath: config}, platformOS: "linux", arch: "amd64",
	}
	info, err := u.Update(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if info.HasUpdate || info.CurrentVersion != "1.14.3" {
		t.Fatalf("unexpected update info: %+v", info)
	}
	data, err := os.ReadFile(installed)
	if err != nil || !bytes.Equal(data, newBinary) {
		t.Fatalf("installed binary = %q, %v", data, err)
	}
	var record database.Core
	if err := database.DB.First(&record, 1).Error; err != nil || record.Version != "1.14.3" {
		t.Fatalf("stored core version = %q, %v", record.Version, err)
	}
}

func TestDownloadCandidateVerifiesArchiveChecksum(t *testing.T) {
	var archive bytes.Buffer
	compressed := gzip.NewWriter(&archive)
	if _, err := compressed.Write([]byte("new sing-box binary")); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256(archive.Bytes())
	checksum := fmt.Sprintf("%x  sing-box_linux_amd64.gz\n", want)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sing-box-1.14.3/SHA256SUMS":
			fmt.Fprint(w, checksum)
		case "/sing-box-1.14.3/sing-box_linux_amd64.gz":
			w.Write(archive.Bytes())
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	u := &BinaryUpdater{client: server.Client(), downloadsURL: server.URL, arch: "amd64"}
	path, err := u.downloadCandidate(context.Background(), t.TempDir(), "1.14.3")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "new sing-box binary" {
		t.Fatalf("downloaded binary = %q, %v", data, err)
	}

	checksum = fmt.Sprintf("%064x  sing-box_linux_amd64.gz\n", 1)
	if _, err := u.downloadCandidate(context.Background(), t.TempDir(), "1.14.3"); err == nil {
		t.Fatal("archive with wrong SHA256 must be rejected")
	}
}

func TestNewestCoreReleaseIgnoresDraftsAndPrereleases(t *testing.T) {
	releases := []coreRelease{
		{TagName: "sing-box-1.13.13"},
		{TagName: "sing-box-1.14.2"},
		{TagName: "sing-box-1.15.0-alpha.1"},
		{TagName: "sing-box-1.14.4", Draft: true},
		{TagName: "sing-box-1.14.3"},
		{TagName: "sing-box-1.15.0", Prerelease: true},
	}
	if got := newestCoreRelease(releases); got != "1.14.3" {
		t.Fatalf("newestCoreRelease = %q, want 1.14.3", got)
	}
}

func TestParseCoreChecksumRequiresExactAssetName(t *testing.T) {
	asset := "sing-box_linux_amd64.gz"
	sum := sha256.Sum256([]byte("core archive"))
	text := hex.EncodeToString(sum[:]) + "  " + asset + "\n" + hex.EncodeToString(sum[:]) + "  other.gz\n"
	got, err := parseCoreChecksum([]byte(text), asset)
	if err != nil || got != sum {
		t.Fatalf("parseCoreChecksum = %x, %v; want %x", got, err, sum)
	}
	if _, err := parseCoreChecksum([]byte(text), "sing-box_linux_arm64.gz"); err == nil {
		t.Fatal("missing architecture checksum must be rejected")
	}
}

func TestReplaceCoreBinaryRollsBackAfterRestartFailure(t *testing.T) {
	dir := t.TempDir()
	installed := filepath.Join(dir, "sing-box")
	candidate := filepath.Join(dir, "candidate")
	if err := os.WriteFile(installed, []byte("old core"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidate, []byte("new core"), 0755); err != nil {
		t.Fatal(err)
	}
	restarts := 0
	err := replaceCoreBinary(installed, candidate, true, func() error {
		restarts++
		if restarts == 1 {
			return errors.New("new core failed to start")
		}
		return nil
	})
	if err == nil || restarts != 2 {
		t.Fatalf("replaceCoreBinary = %v, restarts = %d; want rollback restart", err, restarts)
	}
	data, err := os.ReadFile(installed)
	if err != nil || string(data) != "old core" {
		t.Fatalf("installed core after rollback = %q, %v", data, err)
	}
}

func TestReplaceCoreBinaryKeepsNewCoreWhenStopped(t *testing.T) {
	dir := t.TempDir()
	installed := filepath.Join(dir, "sing-box")
	candidate := filepath.Join(dir, "candidate")
	if err := os.WriteFile(installed, []byte("old core"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidate, []byte("new core"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := replaceCoreBinary(installed, candidate, false, func() error {
		t.Fatal("stopped core should not be restarted")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(installed)
	if err != nil || string(data) != "new core" {
		t.Fatalf("installed core = %q, %v", data, err)
	}
}

func TestBinaryUpdaterRejectsConcurrentUpdates(t *testing.T) {
	u := &BinaryUpdater{}
	u.mu.Lock()
	defer u.mu.Unlock()
	if _, err := u.Update(context.Background()); !errors.Is(err, ErrCoreUpdateBusy) {
		t.Fatalf("concurrent update error = %v, want ErrCoreUpdateBusy", err)
	}
}

func TestInstalledCorePathRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "other-core")
	installed := filepath.Join(dir, "sing-box")
	if err := os.WriteFile(target, []byte("not sing-box"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, installed); err != nil {
		t.Fatal(err)
	}
	if _, err := installedCorePath(installed); err == nil {
		t.Fatal("symlink must not be replaced by core updater")
	}
}
