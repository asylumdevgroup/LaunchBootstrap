package main

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
)

type LauncherManager struct {
	launcherManifest *LauncherManifest
	bSettings        *BootstrapSettings
}

func GetLauncherManager(bs *BootstrapSettings) (*LauncherManager, error) {
	launcherManager := &LauncherManager{
		bSettings: bs,
	}

	// We load the main manifest
	mainManifest, err := GetOrCached[LauncherManifest](
		bs,
		filepath.Join(bs.LauncherPath, ".cache", "launcher_manifest.json"),
		bs.ManifestURL,
	)
	if err != nil {
		return nil, err
	}

	launcherManager.launcherManifest = mainManifest

	return launcherManager, nil
}

func (m *LauncherManager) GetPath() string {
	return path.Join(m.bSettings.LauncherPath, "launcher")
}

// Returns a list of files to re-download
func (m *LauncherManager) ValidateInstallation() ([]Downloadable, error) {
	bp := m.GetPath()

	filesToDownload := []Downloadable{}
	fileList := []string{}

	for _, v := range m.launcherManifest.Files {
		file := filepath.Join(bp, v.Path)
		fileList = append(fileList, file)

		if v.Type == "directory" {
			err := os.MkdirAll(file, os.ModePerm)
			if err != nil {
				return nil, err
			}
		} else if v.Type == "file" || v.Type == "classpath" {
			_, err := os.Stat(file)
			if !os.IsNotExist(err) {
				hash := GetHash(file)
				if hash == v.Hash {
					// The file exists and has the correct hash
					// No need to redownload
					continue
				}
			}

			filesToDownload = append(filesToDownload, Downloadable{
				Url:        v.Url,
				Path:       file,
				Sha256:     v.Hash,
				Size:       v.Size,
				Executable: false,
				// @TODO Maybe later, but there should no need to have an executable
				// Unless we want to support Java in other languages
				// Like go which produces direct executables or python
				// Maybe really later
				// This could lead this bootstrap to be more generic
				// instead of a Minecraft focused thing
			})
		}
	}

	// Removing the files that should not exist
	err := filepath.Walk(bp, func(currPath string, fi fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		if !slices.Contains(fileList, currPath) {
			fmt.Printf("File / dir %v should not exist. Removing it.\n", currPath)
			if err := os.RemoveAll(currPath); err != nil {
				return err
			}
		}

		return nil
	})

	return filesToDownload, err
}
