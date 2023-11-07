//go:build linux

package main

import (
	"os"
	"path/filepath"
)

func GetLauncherDirectory(s *BootstrapSettings) (string, error) {
	lp := s.LauncherPath
	if len(lp) == 0 {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		lp = filepath.Join(homedir, ".local", "share", s.FolderName)
	}

	_, err := os.Stat(lp)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	} else if err != nil {
		err = os.MkdirAll(lp, os.ModePerm)
		if err != nil {
			return "", err
		}
	}

	return lp, nil
}
