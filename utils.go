package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

const NOT_DOWNLOADED = "NOT_DOWNLOADED"

func SetUserAgent(bs *BootstrapSettings, req *http.Request) {
	req.Header.Set(
		"User-Agent",
		bs.Brand+" (SpectrumBootstrap v"+BOOTSTRAP_VERSION+", "+runtime.GOOS+", "+runtime.GOARCH+")",
	)
}

func GetInstalledLauncherVersion(settings *BootstrapSettings, execPath string) (*LauncherVersion, error) {
	_, err := os.Stat(execPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	} else if err != nil {
		return &LauncherVersion{
			Version: NOT_DOWNLOADED,
		}, nil
	}

	versionFilePath := filepath.Join(settings.LauncherPath, "launcher_version.json")
	_, err = os.Stat(versionFilePath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	} else if err != nil {
		return &LauncherVersion{
			Version: NOT_DOWNLOADED,
		}, nil
	}

	out, err := os.ReadFile(versionFilePath)
	if err != nil {
		return nil, err
	}

	launcherVersion := LauncherVersion{}
	err = json.Unmarshal(out, &launcherVersion)
	return &launcherVersion, err
}

func DownloadManifest(bs *BootstrapSettings) (*LauncherManifest, error) {
	client := &http.Client{}

	req, err := http.NewRequest(
		"GET",
		bs.ManifestURL,
		nil,
	)

	if err != nil {
		return nil, err
	}

	SetUserAgent(bs, req)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	manifest := LauncherManifest{}
	err = json.NewDecoder(resp.Body).Decode(&manifest)
	if err != nil {
		return nil, err
	}

	return &manifest, nil
}

func GetHash(filepath string) string {
	f, err := os.Open(filepath)
	if err != nil {
		return ""
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}
