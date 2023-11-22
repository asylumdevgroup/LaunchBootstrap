package main

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

var (
	ErrFailedDetermineOs  = errors.New("failed to determine os/arch")
	ErrNoJavaForOs        = errors.New("no java found for this os")
	ErrNoJavaVersionForOs = errors.New("java version for this os doesn't include the required component")
)

type JvmManager struct {
	cachedMainManifest    *MainJavaManifest
	cachedVersionManifest *JavaManifest

	launcherManifest LauncherJavaManifest
	os               string
	bSettings        *BootstrapSettings
}

func GetJvmManager(bs *BootstrapSettings, launcherManifest LauncherJavaManifest) (*JvmManager, error) {
	//#region Detecting os
	// runtime.GOARCH = 386 amd64 amd64p32 arm arm64
	os := runtime.GOOS
	arch := runtime.GOARCH
	if os == "linux" {
		os = "linux"
		if arch == "386" {
			os += "-i386"
		} else if arch != "amd64" && arch != "amd64p32" {
			return nil, ErrFailedDetermineOs
		}
	} else if os == "darwin" {
		os = "mac-os"
		if arch == "arm64" {
			os += "-arm64"
		} else if arch != "amd64" {
			return nil, ErrFailedDetermineOs
		}
	} else if os == "windows" {
		os = "windows"
		if arch == "386" {
			os += "-x86"
		} else if arch == "amd64" || arch == "amd64p32" {
			os += "-x64"
		} else if arch == "arm64" {
			os += "-arm64"
		} else {
			return nil, ErrFailedDetermineOs
		}
	} else {
		return nil, ErrFailedDetermineOs
	}
	//#endregion

	jvmManager := &JvmManager{
		launcherManifest: launcherManifest,
		bSettings:        bs,
		os:               os,
	}

	// We load the main manifest
	mainManifest, err := GetOrCached[MainJavaManifest](
		bs,
		filepath.Join(bs.LauncherPath, "launcher", "main_java_manifest.json"),
		launcherManifest.ManifestURL,
	)
	if err != nil {
		return nil, err
	}

	jvmManager.cachedMainManifest = mainManifest

	// We load the manifest for the os/version
	versions, ok := (*jvmManager.cachedMainManifest)[os]
	if !ok {
		return nil, ErrNoJavaForOs
	}

	version, ok := versions[launcherManifest.Component]
	if !ok {
		return nil, ErrNoJavaVersionForOs
	}
	versionManifest, err := GetOrCached[JavaManifest](
		bs,
		filepath.Join(bs.LauncherPath, "launcher", "java_"+os+"_"+launcherManifest.Component+".json"),
		version[0].Manifest.Url, // @TODO: Check how versions are handled, should we DL the first or the last?
	)
	if err != nil {
		return nil, err
	}

	jvmManager.cachedVersionManifest = versionManifest

	return jvmManager, nil
}

func (m *JvmManager) GetPath() string {
	return path.Join(m.bSettings.LauncherPath, "runtime", m.launcherManifest.Component, m.os, m.launcherManifest.Component)
}

// Returns a list of files to re-download
func (m *JvmManager) ValidateInstallation() ([]Downloadable, error) {
	bp := m.GetPath()

	filesToDownload := []Downloadable{}

	for k, v := range m.cachedVersionManifest.Files {
		file := filepath.Join(bp, k)
		if v.Type == "directory" {
			err := os.MkdirAll(file, os.ModePerm)
			if err != nil {
				return nil, err
			}
		} else if v.Type == "file" {
			_, err := os.Stat(file)
			if !os.IsNotExist(err) {
				sha1 := GetHashSha1(file)
				if sha1 == v.Downloads.Raw.Hash {
					// The file exists and has the correct hash
					// No need to redownload

					// Just checking the executable flag
					if v.Executable {
						err := os.Chmod(file, os.ModePerm)
						if err != nil {
							return nil, err
						}
					}
					continue
				}
			}

			filesToDownload = append(filesToDownload, Downloadable{
				Url:        v.Downloads.Raw.Url,
				Path:       file,
				Sha1:       v.Downloads.Raw.Hash,
				Size:       v.Downloads.Raw.Size,
				Executable: v.Executable,
			})
		}
	}

	return filesToDownload, nil
}
