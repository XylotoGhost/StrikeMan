package app

// The Wails-facing side of self-updating. The mechanics live in
// internal/update; this decides what to tell the user about them.

import (
	"errors"
	"strings"

	"strikeman/internal/update"
)

// UpdateInfo is what the Server card renders.
type UpdateInfo struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Available bool   `json:"available"`
	AssetName string `json:"assetName"`
	Notes     string `json:"notes"`
	Error     string `json:"error"`
}

func (a *App) GetAppVersion() string { return a.version }

// CheckForUpdate asks GitHub what the newest release is.
func (a *App) CheckForUpdate() UpdateInfo {
	info := UpdateInfo{Current: a.version, AssetName: update.AssetName()}
	rel, err := update.Latest()
	if err != nil {
		info.Error = err.Error()
		return info
	}
	info.Latest = rel.TagName
	info.Available = update.Newer(rel.TagName, a.version)
	info.Notes = strings.TrimSpace(rel.Body)
	return info
}

// InstallUpdate downloads the newest build, checks it against the checksum
// published beside it, swaps it in and restarts.
func (a *App) InstallUpdate() error {
	rel, err := update.Latest()
	if err != nil {
		return tErr("error.updateFailed", err.Error())
	}
	if !update.Newer(rel.TagName, a.version) {
		return tErr("error.noUpdate")
	}

	a.logKey("log.updateDownloading", rel.TagName)
	result, err := update.Apply(rel)
	switch {
	case err == nil:
		// installed
	case errors.As(err, &update.ErrNoAsset{}):
		return tErr("error.updateNoAsset", update.AssetName())
	case errors.As(err, &update.ErrChecksum{}):
		return tErr("error.updateChecksum", err.Error())
	default:
		return tErr("error.updateFailed", err.Error())
	}
	if !result.ChecksumVerified {
		a.logKey("log.updateNoChecksum")
	}

	a.logKey("log.updateRestarting", rel.TagName)
	return update.Restart()
}
