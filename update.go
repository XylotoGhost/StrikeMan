package main

// Self-update from the project's GitHub releases.
//
// The portable archives are used for updating regardless of how StrikeMan was
// installed: both forms are just a folder with the executable in it, so the
// update is always "replace that file". A per-user install lives under
// %LOCALAPPDATA% (or the user's home), so this never needs admin rights.

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	releasesAPI  = "https://api.github.com/repos/XylotoGhost/StrikeMan/releases/latest"
	devVersion   = "dev"
	oldExeSuffix = ".old" // left behind on Windows, cleaned up on next start
)

var updateClient = &http.Client{Timeout: 60 * time.Second}

type UpdateInfo struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Available bool   `json:"available"`
	AssetName string `json:"assetName"`
	Notes     string `json:"notes"`
	Error     string `json:"error"`
}

func (a *App) GetAppVersion() string { return version }

// assetName is the portable archive built for this platform by release.yml.
func assetName() string {
	switch runtime.GOOS {
	case "windows":
		return "StrikeMan-windows.zip"
	case "darwin":
		return "StrikeMan-macos.zip"
	default:
		return "StrikeMan-linux.tar.gz"
	}
}

type release struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func fetchLatestRelease() (*release, error) {
	resp, err := updateClient.Get(releasesAPI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub answered %s", resp.Status)
	}
	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// newerThan compares release tags like "v0.8.1". Anything unparsable, and any
// build that is not a release, simply reports "no update".
func newerThan(latest, current string) bool {
	if current == devVersion || current == "" || latest == "" {
		return false
	}
	return compareVersions(latest, current) > 0
}

func compareVersions(a, b string) int {
	partsA := versionParts(a)
	partsB := versionParts(b)
	for i := 0; i < 3; i++ {
		switch {
		case partsA[i] > partsB[i]:
			return 1
		case partsA[i] < partsB[i]:
			return -1
		}
	}
	return 0
}

func versionParts(v string) [3]int {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	for i, field := range strings.SplitN(v, ".", 3) {
		if i > 2 {
			break
		}
		n := 0
		for _, r := range field {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		out[i] = n
	}
	return out
}

// CheckForUpdate asks GitHub what the newest release is.
func (a *App) CheckForUpdate() UpdateInfo {
	info := UpdateInfo{Current: version, AssetName: assetName()}
	rel, err := fetchLatestRelease()
	if err != nil {
		info.Error = err.Error()
		return info
	}
	info.Latest = rel.TagName
	info.Available = newerThan(rel.TagName, version)
	info.Notes = strings.TrimSpace(rel.Body)
	return info
}

// InstallUpdate downloads the newest build, checks it against the checksum
// published beside it, swaps it in and restarts.
func (a *App) InstallUpdate() error {
	rel, err := fetchLatestRelease()
	if err != nil {
		return tErr("error.updateFailed", err.Error())
	}
	if !newerThan(rel.TagName, version) {
		return tErr("error.noUpdate")
	}

	wanted := assetName()
	var archiveURL, sumURL string
	for _, asset := range rel.Assets {
		switch asset.Name {
		case wanted:
			archiveURL = asset.URL
		case wanted + ".sha256":
			sumURL = asset.URL
		}
	}
	if archiveURL == "" {
		return tErr("error.updateNoAsset", wanted)
	}

	a.logKey("log.updateDownloading", rel.TagName)
	work, err := os.MkdirTemp("", "strikeman-update")
	if err != nil {
		return tErr("error.updateFailed", err.Error())
	}
	defer os.RemoveAll(work)

	archive := filepath.Join(work, wanted)
	sum, err := download(archiveURL, archive)
	if err != nil {
		return tErr("error.updateFailed", err.Error())
	}
	if sumURL != "" {
		if err := verifyChecksum(sumURL, sum); err != nil {
			return tErr("error.updateChecksum", err.Error())
		}
	} else {
		a.logKey("log.updateNoChecksum")
	}

	newExe, err := extractExecutable(archive, work)
	if err != nil {
		return tErr("error.updateFailed", err.Error())
	}
	if err := replaceExecutable(newExe); err != nil {
		return tErr("error.updateFailed", err.Error())
	}

	a.logKey("log.updateRestarting", rel.TagName)
	return restart()
}

// download writes url to path and returns the SHA-256 of what arrived.
func download(url, path string) (string, error) {
	resp, err := updateClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download answered %s", resp.Status)
	}

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(file, hash), resp.Body); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// verifyChecksum compares against the published "<sha256>  <filename>" file.
func verifyChecksum(url, got string) error {
	resp, err := updateClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return err
	}
	want := strings.ToLower(strings.TrimSpace(string(body)))
	if fields := strings.Fields(want); len(fields) > 0 {
		want = fields[0]
	}
	if want == "" || want != got {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", want, got)
	}
	return nil
}

// extractExecutable pulls the StrikeMan binary out of a release archive and
// returns the path it was written to.
func extractExecutable(archive, dir string) (string, error) {
	if strings.HasSuffix(archive, ".zip") {
		return extractFromZip(archive, dir)
	}
	return extractFromTarGz(archive, dir)
}

// wantedExe reports whether a path inside an archive is the executable. The
// macOS archive holds StrikeMan.app/Contents/MacOS/StrikeMan.
func wantedExe(name string) bool {
	base := filepath.Base(name)
	switch runtime.GOOS {
	case "windows":
		return strings.EqualFold(base, "StrikeMan.exe")
	case "darwin":
		return base == "StrikeMan" && strings.Contains(filepath.ToSlash(name), "Contents/MacOS/")
	default:
		return base == "StrikeMan"
	}
}

func extractFromZip(archive, dir string) (string, error) {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() || !wantedExe(entry.Name) {
			continue
		}
		source, err := entry.Open()
		if err != nil {
			return "", err
		}
		defer source.Close()
		return writeExecutable(source, dir)
	}
	return "", fmt.Errorf("no StrikeMan executable inside %s", filepath.Base(archive))
}

func extractFromTarGz(archive, dir string) (string, error) {
	file, err := os.Open(archive)
	if err != nil {
		return "", err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if header.Typeflag != tar.TypeReg || !wantedExe(header.Name) {
			continue
		}
		return writeExecutable(reader, dir)
	}
	return "", fmt.Errorf("no StrikeMan executable inside %s", filepath.Base(archive))
}

func writeExecutable(source io.Reader, dir string) (string, error) {
	path := filepath.Join(dir, "StrikeMan.new")
	out, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, source); err != nil {
		return "", err
	}
	return path, nil
}

// replaceExecutable puts the downloaded build where the running one lives.
// A running executable cannot be overwritten on Windows, but it can be
// renamed out of the way, which is what makes this work without a helper
// process. The leftover is removed on the next start.
func replaceExecutable(newExe string) error {
	current, err := os.Executable()
	if err != nil {
		return err
	}
	current, err = filepath.EvalSymlinks(current)
	if err != nil {
		return err
	}

	old := current + oldExeSuffix
	os.Remove(old)
	if err := os.Rename(current, old); err != nil {
		return err
	}
	if err := copyFile(newExe, current); err != nil {
		// Put the working build back rather than leaving nothing behind.
		os.Rename(old, current)
		return err
	}
	return nil
}

func copyFile(from, to string) error {
	source, err := os.Open(from)
	if err != nil {
		return err
	}
	defer source.Close()
	dest, err := os.OpenFile(to, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer dest.Close()
	_, err = io.Copy(dest, source)
	return err
}

// cleanupOldExecutable removes the previous build left behind by an update.
func cleanupOldExecutable() {
	if current, err := os.Executable(); err == nil {
		os.Remove(current + oldExeSuffix)
	}
}

func restart() error {
	current, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(current)
	cmd.Dir = filepath.Dir(current)
	if err := cmd.Start(); err != nil {
		return err
	}
	// Give the new process a moment to take over the window.
	go func() {
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()
	return nil
}
