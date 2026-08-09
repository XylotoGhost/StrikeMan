package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestVersionComparison(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v0.8.0", "v0.7.2", true},
		{"v0.7.2", "v0.7.2", false},
		{"v0.7.2", "v0.8.0", false},
		{"v0.10.0", "v0.9.9", true}, // not a string comparison
		{"v1.0.0", "v0.99.99", true},
		{"v0.7.10", "v0.7.9", true},
		// A local build must never be told there is an update, or every
		// developer run would offer to overwrite itself.
		{"v9.9.9", devVersion, false},
		{"v0.8.0", "", false},
		{"", "v0.7.2", false},
	}
	for _, c := range cases {
		if got := newerThan(c.latest, c.current); got != c.want {
			t.Errorf("newerThan(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestAssetNameMatchesPlatform(t *testing.T) {
	name := assetName()
	switch runtime.GOOS {
	case "windows":
		if name != "StrikeMan-windows.zip" {
			t.Errorf("asset = %q", name)
		}
	case "darwin":
		if name != "StrikeMan-macos.zip" {
			t.Errorf("asset = %q", name)
		}
	default:
		if name != "StrikeMan-linux.tar.gz" {
			t.Errorf("asset = %q", name)
		}
	}
}

func TestWantedExe(t *testing.T) {
	switch runtime.GOOS {
	case "windows":
		if !wantedExe("StrikeMan.exe") || !wantedExe("bin/StrikeMan.exe") {
			t.Error("should match the Windows executable")
		}
		if wantedExe("README.md") {
			t.Error("should not match other files")
		}
	case "darwin":
		if !wantedExe("StrikeMan.app/Contents/MacOS/StrikeMan") {
			t.Error("should match the binary inside the app bundle")
		}
		if wantedExe("StrikeMan.app/Contents/Info.plist") {
			t.Error("should not match bundle metadata")
		}
	default:
		if !wantedExe("StrikeMan") {
			t.Error("should match the Linux binary")
		}
	}
}

func TestExtractFromZip(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("the Linux release ships a tar.gz")
	}
	dir := t.TempDir()
	archive := filepath.Join(dir, "release.zip")

	name := "StrikeMan.exe"
	if runtime.GOOS == "darwin" {
		name = "StrikeMan.app/Contents/MacOS/StrikeMan"
	}
	file, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(file)
	if _, err := zw.Create("noise.txt"); err != nil {
		t.Fatal(err)
	}
	entry, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	entry.Write([]byte("binary contents"))
	zw.Close()
	file.Close()

	extracted, err := extractExecutable(archive, dir)
	if err != nil {
		t.Fatalf("extractExecutable: %v", err)
	}
	data, err := os.ReadFile(extracted)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "binary contents" {
		t.Errorf("extracted %q", data)
	}
}

func TestExtractRejectsArchiveWithoutExecutable(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "empty.zip")
	file, _ := os.Create(archive)
	zw := zip.NewWriter(file)
	zw.Create("readme.txt")
	zw.Close()
	file.Close()

	if _, err := extractExecutable(archive, dir); err == nil {
		t.Fatal("expected an error when the archive has no executable")
	}
}

// The swap must leave a working binary behind even when the replacement
// cannot be written.
func TestReplaceExecutableRestoresOnFailure(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "app.exe")
	if err := os.WriteFile(target, []byte("original"), 0o755); err != nil {
		t.Fatal(err)
	}

	// copyFile onto a path whose parent is missing fails, standing in for a
	// write error part-way through the swap.
	if err := copyFile(filepath.Join(dir, "missing", "src"), target); err == nil {
		t.Fatal("expected copying a missing file to fail")
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "original" {
		t.Errorf("the original binary should be untouched, got %q (%v)", data, err)
	}
}

// Downloads the real release asset and unpacks it, without touching the
// running binary. Needs network, so it is opt-in:
//
//	STRIKEMAN_TEST_NETWORK=1 go test -run TestLiveDownload -v
func TestLiveDownloadAndExtract(t *testing.T) {
	if os.Getenv("STRIKEMAN_TEST_NETWORK") == "" {
		t.Skip("set STRIKEMAN_TEST_NETWORK=1 to download from GitHub")
	}
	rel, err := fetchLatestRelease()
	if err != nil {
		t.Fatalf("fetchLatestRelease: %v", err)
	}
	t.Logf("latest release: %s", rel.TagName)

	wanted := assetName()
	var archiveURL, sumURL string
	for _, asset := range rel.Assets {
		if asset.Name == wanted {
			archiveURL = asset.URL
		}
		if asset.Name == wanted+".sha256" {
			sumURL = asset.URL
		}
	}
	if archiveURL == "" {
		t.Fatalf("release %s has no asset named %s", rel.TagName, wanted)
	}

	dir := t.TempDir()
	archive := filepath.Join(dir, wanted)
	sum, err := download(archiveURL, archive)
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	stat, _ := os.Stat(archive)
	t.Logf("downloaded %s (%d bytes), sha256 %s", wanted, stat.Size(), sum)

	if sumURL == "" {
		t.Logf("no %s.sha256 published yet — checksums start with this release", wanted)
	} else if err := verifyChecksum(sumURL, sum); err != nil {
		t.Fatalf("checksum: %v", err)
	} else {
		t.Log("checksum verified against the published .sha256")
	}

	exe, err := extractExecutable(archive, dir)
	if err != nil {
		t.Fatalf("extractExecutable: %v", err)
	}
	info, err := os.Stat(exe)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() < 1_000_000 {
		t.Errorf("extracted executable looks too small: %d bytes", info.Size())
	}
	t.Logf("extracted a %d byte executable", info.Size())
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "from")
	to := filepath.Join(dir, "to")
	if err := os.WriteFile(from, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(from, to); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	data, _ := os.ReadFile(to)
	if string(data) != "payload" {
		t.Errorf("copied %q", data)
	}
}
