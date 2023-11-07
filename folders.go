//go:build !linux

package main

import (
	"os"

	"github.com/kirsle/configdir"
)

func GetLauncherDirectory(s *BootstrapSettings) (string, error) {
	lp := s.LauncherPath
	if len(lp) == 0 {
		lp = configdir.LocalConfig(s.FolderName)
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
