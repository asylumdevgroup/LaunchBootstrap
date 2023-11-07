package main

import (
	"path/filepath"
	"runtime"
)

type BootstrapSettings struct {
	ManifestURL string `json:"launcher_manifest"`
	Brand       string `json:"launcher_brand"`
	FolderName  string `json:"launcher_foldername"`

	LauncherPath string `json:"-"`
}

func (bs *BootstrapSettings) GetLauncherExecutable() (string, error) {
	return filepath.Join(bs.LauncherPath, "launcher-"+runtime.GOOS+"-"+runtime.GOARCH), nil
}

type LauncherManifest struct {
	Version string                       `json:"latest_version"`
	Urls    map[string]map[string]string `json:"latest_version_urls"`
}

type LauncherVersion struct {
	Version string `json:"version"`
	Hash    string `json:"sha256"`
}
