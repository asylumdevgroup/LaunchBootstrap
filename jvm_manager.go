/**
 * Spectrum-Bootstrap - A bootstrap for Minecraft launchers
 * Copyright (C) 2023-2024 - Oxodao
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 **/

 package main

 import (
	 "errors"
	 "fmt"
	 "io/fs"
	 "os"
	 "path"
	 "path/filepath"
	 "runtime"
	 "slices"
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
	 legacyPath       string
 }
 
 func GetJvmManager(bs *BootstrapSettings, launcherManifest LauncherJavaManifest) (*JvmManager, error) {
	 os, err := detectOS()
	 if err != nil {
		 return nil, err
	 }
 
	 jvmManager := &JvmManager{
		 launcherManifest: launcherManifest,
		 bSettings:        bs,
		 os:               os,
	 }
 
	 mainManifest, err := GetOrCached[MainJavaManifest](
		 bs,
		 filepath.Join(bs.LauncherPath, ".cache", "main_java_manifest2.json"),
		 launcherManifest.ManifestURL,
	 )
	 if err != nil {
		 return nil, err
	 }
 
	 jvmManager.cachedMainManifest = mainManifest
 
	 versions, ok := (*jvmManager.cachedMainManifest)[os]
	 if !ok {
		 return nil, ErrNoJavaForOs
	 }
 
	 componentKeys := []string{launcherManifest.Component, launcherManifest.ComponentLegacy}
	 var versionManifest *JavaManifest
 
	 for _, component := range componentKeys {
		 version, ok := versions[component]
		 if ok {
			 versionManifest, err = GetOrCached[JavaManifest](
				 bs,
				 filepath.Join(bs.LauncherPath, ".cache", "java_"+os+"_"+component+".json"),
				 version.ManifestURL, // Adjusted to use a hypothetical field name
			 )
			 if err == nil {
				 break
			 }
		 }
	 }
 
	 if versionManifest == nil {
		 return nil, ErrNoJavaVersionForOs
	 }
 
	 jvmManager.cachedVersionManifest = versionManifest
	 jvmManager.legacyPath = jvmManager.GetPathLegacy()
 
	 return jvmManager, nil
 }
 
 func detectOS() (string, error) {
	 os := runtime.GOOS
	 arch := runtime.GOARCH
	 switch os {
	 case "linux":
		 if arch == "386" {
			 return "linux-i386", nil
		 } else if arch == "amd64" || arch == "amd64p32" {
			 return "linux", nil
		 }
	 case "darwin":
		 if arch == "arm64" {
			 return "mac-os-arm64", nil
		 } else if arch == "amd64" {
			 return "mac-os", nil
		 }
	 case "windows":
		 switch arch {
		 case "386":
			 return "windows-x86", nil
		 case "amd64", "amd64p32":
			 return "windows-x64", nil
		 case "arm64":
			 return "windows-arm64", nil
		 }
	 }
	 return "", ErrFailedDetermineOs
 }
 
 func (m *JvmManager) GetPath() string {
	 return path.Join(m.bSettings.LauncherPath, "runtime", m.launcherManifest.Component, m.os)
 }
 
 func (m *JvmManager) GetPathLegacy() string {
	 return path.Join(m.bSettings.LauncherPath, "runtime", m.launcherManifest.ComponentLegacy, m.os)
 }
 
 func (m *JvmManager) ValidateInstallation() ([]Downloadable, error) {
	 filesToDownload := []Downloadable{}
 
	 if m.launcherManifest.Component != "" {
		 primaryFiles, err := m.validateComponent(m.GetPath(), m.launcherManifest.Component)
		 if err != nil {
			 return nil, err
		 }
		 filesToDownload = append(filesToDownload, primaryFiles...)
	 }
 
	 if m.launcherManifest.ComponentLegacy != "" {
		 legacyFiles, err := m.validateComponent(m.GetPathLegacy(), m.launcherManifest.ComponentLegacy)
		 if err != nil {
			 return nil, err
		 }
		 filesToDownload = append(filesToDownload, legacyFiles...)
	 }
 
	 return filesToDownload, nil
 }
 
 func (m *JvmManager) validateComponent(basePath, component string) ([]Downloadable, error) {
	 filesToDownload := []Downloadable{}
	 fileList := []string{}
 
	 versions, ok := (*m.cachedMainManifest)[m.os]
	 if !ok {
		 return nil, ErrNoJavaForOs
	 }
 
	 version, ok := versions[component]
	 if !ok {
		 return nil, ErrNoJavaVersionForOs
	 }
 
	 // Assuming the correct field is `Artifacts` instead of `Files`
	 for k, v := range version.FilesZ { // Adjust this line based on the actual structure
		 file := filepath.Join(basePath, k)
		 fileList = append(fileList, file)
 
		 if v.Type == "directory" {
			 err := os.MkdirAll(file, os.ModePerm)
			 if err != nil {
				 return nil, err
			 }
		 } else if v.Type == "file" {
			 _, err := os.Stat(file)
			 if os.IsNotExist(err) {
				 fmt.Printf("Preparing to download Java file: %s from URL: %s\n", file, v.Downloads.Raw.Url)
				 filesToDownload = append(filesToDownload, Downloadable{
					 Url:        v.Downloads.Raw.Url,
					 Path:       file,
					 Sha1:       v.Downloads.Raw.Hash,
					 Size:       v.Downloads.Raw.Size,
					 Executable: v.Executable,
				 })
			 } else {
				 sha1 := GetHashSha1(file)
				 if sha1 != v.Downloads.Raw.Hash {
					 fmt.Printf("Preparing to download Java file: %s from URL: %s (hash mismatch)\n", file, v.Downloads.Raw.Url)
					 filesToDownload = append(filesToDownload, Downloadable{
						 Url:        v.Downloads.Raw.Url,
						 Path:       file,
						 Sha1:       v.Downloads.Raw.Hash,
						 Size:       v.Downloads.Raw.Size,
						 Executable: v.Executable,
					 })
				 }
			 }
		 }
	 }
 
	 err := filepath.Walk(basePath, func(currPath string, fi fs.FileInfo, err error) error {
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