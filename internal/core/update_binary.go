package core

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gzjacktang/single-ui/internal/database"
	"github.com/gzjacktang/single-ui/internal/util"
	"golang.org/x/mod/semver"
)

const (
	coreReleasesURL  = "https://api.github.com/repos/gzjacktang/single-ui/releases?per_page=100"
	coreDownloadsURL = "https://github.com/gzjacktang/single-ui/releases/download"
	maxReleaseJSON   = 8 << 20
	maxChecksums     = 1 << 20
	maxCoreArchive   = 200 << 20
	maxCoreBinary    = 300 << 20
)

var (
	coreVersionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$`)
	ErrCoreUpdateBusy  = errors.New("内核更新正在进行中")
)

type CoreUpdateInfo struct {
	CurrentVersion string        `json:"current_version"`
	LatestVersion  string        `json:"latest_version"`
	HasUpdate      bool          `json:"has_update"`
	Versions       []CoreVersion `json:"versions"`
}

type CoreVersion struct {
	Version    string `json:"version"`
	Prerelease bool   `json:"prerelease"`
}

func validCoreVersion(version string) bool {
	return coreVersionPattern.MatchString(version) && semver.IsValid("v"+version)
}

func availableCoreVersions(releases []coreRelease) []CoreVersion {
	versions := []CoreVersion{}
	seen := map[string]bool{}
	for _, release := range releases {
		if release.Draft || !strings.HasPrefix(release.TagName, "sing-box-") {
			continue
		}
		version := strings.TrimPrefix(release.TagName, "sing-box-")
		if !validCoreVersion(version) || seen[version] {
			continue
		}
		seen[version] = true
		versions = append(versions, CoreVersion{Version: version, Prerelease: release.Prerelease || semver.Prerelease("v"+version) != ""})
	}
	sort.Slice(versions, func(i, j int) bool { return semver.Compare("v"+versions[i].Version, "v"+versions[j].Version) > 0 })
	return versions
}

type coreRelease struct {
	TagName    string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

type BinaryUpdater struct {
	client       *http.Client
	releasesURL  string
	downloadsURL string
	manager      *Manager
	platformOS   string
	arch         string
	restart      func() error
	mu           sync.Mutex
}

var DefaultBinaryUpdater = &BinaryUpdater{
	client:       &http.Client{Timeout: 3 * time.Minute},
	releasesURL:  coreReleasesURL,
	downloadsURL: coreDownloadsURL,
	manager:      Default,
	platformOS:   runtime.GOOS,
	arch:         runtime.GOARCH,
	restart:      restartUpdatedCore,
}

func restartUpdatedCore() error {
	if err := Default.Restart(); err != nil {
		return err
	}
	time.Sleep(time.Second)
	if Default.Status() != "running" {
		return errors.New("内核启动后立即退出")
	}
	return nil
}

func newestCoreRelease(releases []coreRelease) string {
	var latest string
	for _, release := range releases {
		if release.Draft || release.Prerelease || !strings.HasPrefix(release.TagName, "sing-box-") {
			continue
		}
		version := strings.TrimPrefix(release.TagName, "sing-box-")
		if !validCoreVersion(version) || semver.Prerelease("v"+version) != "" {
			continue
		}
		if latest == "" || semver.Compare("v"+version, "v"+latest) > 0 {
			latest = version
		}
	}
	return latest
}

func parseCoreChecksum(data []byte, asset string) ([sha256.Size]byte, error) {
	var result [sha256.Size]byte
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || fields[1] != asset {
			continue
		}
		decoded, err := hex.DecodeString(fields[0])
		if err != nil || len(decoded) != sha256.Size {
			return result, fmt.Errorf("内核校验文件无效: %s", asset)
		}
		copy(result[:], decoded)
		return result, nil
	}
	return result, fmt.Errorf("校验文件缺少 %s", asset)
}

func (u *BinaryUpdater) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("下载服务器返回 HTTP %d", resp.StatusCode)
	}
	return resp, nil
}

func readLimited(body io.Reader, max int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, errors.New("下载内容超过大小限制")
	}
	return data, nil
}

func (u *BinaryUpdater) releases(ctx context.Context) ([]coreRelease, error) {
	base, err := url.Parse(u.releasesURL)
	if err != nil {
		return nil, err
	}
	var all []coreRelease
	for page := 1; ; page++ {
		if page > 100 {
			return nil, errors.New("内核版本列表分页超过上限")
		}
		query := base.Query()
		query.Set("page", fmt.Sprint(page))
		base.RawQuery = query.Encode()
		resp, err := u.get(ctx, base.String())
		if err != nil {
			return nil, err
		}
		data, err := readLimited(resp.Body, maxReleaseJSON)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		var releases []coreRelease
		if err := json.Unmarshal(data, &releases); err != nil {
			return nil, fmt.Errorf("解析内核版本失败: %w", err)
		}
		all = append(all, releases...)
		if !strings.Contains(resp.Header.Get("Link"), `rel="next"`) {
			return all, nil
		}
	}
}

func binaryVersion(ctx context.Context, path string) (string, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(cmdCtx, path, "version").Output()
	if err != nil {
		return "", fmt.Errorf("读取内核版本失败: %w", err)
	}
	fields := strings.Fields(strings.SplitN(string(out), "\n", 2)[0])
	if len(fields) < 3 || fields[0] != "sing-box" || fields[1] != "version" {
		return "", errors.New("内核版本输出格式无效")
	}
	version := strings.TrimPrefix(fields[2], "v")
	if !validCoreVersion(version) {
		return "", fmt.Errorf("内核版本号无效: %s", version)
	}
	return version, nil
}

func installedCorePath(path string) (string, error) {
	installed, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	stat, err := os.Lstat(installed)
	if err != nil || !stat.Mode().IsRegular() || filepath.Base(installed) != "sing-box" {
		return "", errors.New("内核路径不是普通的 sing-box 文件")
	}
	return installed, nil
}

func (u *BinaryUpdater) Check(ctx context.Context) (CoreUpdateInfo, error) {
	var info CoreUpdateInfo
	installed, err := installedCorePath(u.manager.BinPath)
	if err != nil {
		return info, err
	}
	current, err := binaryVersion(ctx, installed)
	if err != nil {
		return info, err
	}
	releases, err := u.releases(ctx)
	if err != nil {
		return info, err
	}
	latest := newestCoreRelease(releases)
	versions := availableCoreVersions(releases)
	if len(versions) == 0 {
		return info, errors.New("没有可用的内核安装包")
	}
	info.CurrentVersion = current
	info.LatestVersion = latest
	info.HasUpdate = latest != "" && semver.Compare("v"+latest, "v"+current) > 0
	info.Versions = versions
	return info, nil
}

func (u *BinaryUpdater) downloadCandidate(ctx context.Context, workDir, version string) (string, error) {
	asset := "sing-box_linux_" + u.arch + ".gz"
	base := fmt.Sprintf("%s/sing-box-%s", strings.TrimRight(u.downloadsURL, "/"), version)
	checksumResp, err := u.get(ctx, base+"/SHA256SUMS")
	if err != nil {
		return "", fmt.Errorf("下载校验文件失败: %w", err)
	}
	checksumData, checksumErr := readLimited(checksumResp.Body, maxChecksums)
	checksumResp.Body.Close()
	if checksumErr != nil {
		return "", checksumErr
	}
	want, err := parseCoreChecksum(checksumData, asset)
	if err != nil {
		return "", err
	}

	archiveResp, err := u.get(ctx, base+"/"+asset)
	if err != nil {
		return "", fmt.Errorf("下载内核失败: %w", err)
	}
	defer archiveResp.Body.Close()
	archivePath := filepath.Join(workDir, asset)
	archive, err := os.OpenFile(archivePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(archive, io.TeeReader(io.LimitReader(archiveResp.Body, maxCoreArchive+1), hash))
	closeErr := archive.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if written > maxCoreArchive {
		return "", errors.New("内核安装包超过大小限制")
	}
	if !equalSHA256(hash.Sum(nil), want) {
		return "", errors.New("内核安装包 SHA256 校验失败")
	}

	archive, err = os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer archive.Close()
	compressed, err := gzip.NewReader(archive)
	if err != nil {
		return "", fmt.Errorf("内核压缩包无效: %w", err)
	}
	defer compressed.Close()
	candidate := filepath.Join(workDir, "sing-box-new")
	output, err := os.OpenFile(candidate, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	if err != nil {
		return "", err
	}
	written, copyErr = io.Copy(output, io.LimitReader(compressed, maxCoreBinary+1))
	closeErr = output.Close()
	if copyErr != nil {
		return "", fmt.Errorf("解压内核失败: %w", copyErr)
	}
	if closeErr != nil {
		return "", closeErr
	}
	if written > maxCoreBinary {
		return "", errors.New("内核程序超过大小限制")
	}
	return candidate, nil
}

func equalSHA256(got []byte, want [sha256.Size]byte) bool {
	return subtle.ConstantTimeCompare(got, want[:]) == 1
}

func replaceCoreBinary(installed, candidate string, wasRunning bool, restart func() error) error {
	stat, err := os.Lstat(installed)
	if err != nil {
		return err
	}
	if !stat.Mode().IsRegular() || filepath.Base(installed) != "sing-box" {
		return errors.New("只允许更新普通的 sing-box 内核文件")
	}
	backupDir, err := os.MkdirTemp(filepath.Dir(installed), ".sing-box-rollback-")
	if err != nil {
		return err
	}
	keepBackup := false
	defer func() {
		if !keepBackup {
			os.RemoveAll(backupDir)
		}
	}()
	backup := filepath.Join(backupDir, "previous")
	if err := os.Link(installed, backup); err != nil {
		return fmt.Errorf("备份原内核失败: %w", err)
	}
	if err := os.Rename(candidate, installed); err != nil {
		return fmt.Errorf("替换内核失败: %w", err)
	}
	if !wasRunning {
		return nil
	}
	if err := restart(); err != nil {
		restoreErr := os.Rename(backup, installed)
		if restoreErr != nil {
			keepBackup = true
			return errors.Join(fmt.Errorf("新内核启动失败: %w", err), fmt.Errorf("恢复原内核失败，原文件保留在 %s: %w", backup, restoreErr))
		}
		if rollbackErr := restart(); rollbackErr != nil {
			return errors.Join(fmt.Errorf("新内核启动失败，已恢复原文件: %w", err), fmt.Errorf("原内核重新启动失败: %w", rollbackErr))
		}
		return fmt.Errorf("新内核启动失败，已恢复原内核: %w", err)
	}
	return nil
}

func (u *BinaryUpdater) Update(ctx context.Context, target string) (CoreUpdateInfo, error) {
	var info CoreUpdateInfo
	if !u.mu.TryLock() {
		return info, ErrCoreUpdateBusy
	}
	defer u.mu.Unlock()
	if u.platformOS != "linux" || (u.arch != "amd64" && u.arch != "arm64") {
		return info, errors.New("仅支持 Linux amd64 和 arm64 更新内核")
	}
	var err error
	info, err = u.Check(ctx)
	if err != nil {
		return info, err
	}
	if target == "" {
		if !info.HasUpdate {
			return info, nil
		}
		target = info.LatestVersion
	}
	found := false
	for _, version := range info.Versions {
		if version.Version == target {
			found = true
			break
		}
	}
	if !found {
		return info, fmt.Errorf("版本 %q 没有可用的项目内核安装包，请刷新版本列表", target)
	}
	if target == info.CurrentVersion {
		return info, nil
	}
	installed, err := installedCorePath(u.manager.BinPath)
	if err != nil {
		return info, err
	}
	workDir, err := os.MkdirTemp(filepath.Dir(installed), ".sing-box-update-")
	if err != nil {
		return info, err
	}
	defer os.RemoveAll(workDir)
	candidate, err := u.downloadCandidate(ctx, workDir, target)
	if err != nil {
		return info, err
	}
	candidateVersion, err := binaryVersion(ctx, candidate)
	if err != nil {
		return info, fmt.Errorf("下载的内核无法执行: %w", err)
	}
	if candidateVersion != target {
		return info, fmt.Errorf("下载的内核版本与发布版本不一致: %s", candidateVersion)
	}
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if output, err := exec.CommandContext(checkCtx, candidate, "check", "-c", u.manager.ConfigPath).CombinedOutput(); err != nil {
		return info, fmt.Errorf("新内核无法读取当前配置: %s: %w", strings.TrimSpace(string(output)), err)
	}
	if err := replaceCoreBinary(installed, candidate, u.manager.Status() == "running", u.restart); err != nil {
		return info, err
	}
	info.CurrentVersion = target
	info.HasUpdate = semver.Compare("v"+info.LatestVersion, "v"+target) > 0
	if err := database.DB.Model(&database.Core{}).Where("id = 1").Update("version", info.CurrentVersion).Error; err != nil {
		util.Warn("[core] 内核已更新，但记录版本失败: %v", err)
	}
	util.Info("[core] 内核已更新至 %s", info.CurrentVersion)
	return info, nil
}
