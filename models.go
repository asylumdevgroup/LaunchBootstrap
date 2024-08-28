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

type BootstrapSettings struct {
	ManifestURL string `json:"launcher_manifest"`
	Brand       string `json:"launcher_brand"`
	FolderName  string `json:"launcher_foldername"`

	LauncherPath string `json:"-"`
}

type LauncherVersion struct {
	Version string `json:"version"`
	Hash    string `json:"hash"`
}

type ManifestFile struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Hash string `json:"hash"`
	Url  string `json:"url"`
	Size int    `json:"size"`
}

type LauncherJavaManifest struct {
	ManifestURL string `json:"manifest"`
	Component   string `json:"component"`
}

type LauncherManifest struct {
	Version   string               `json:"version"`
	Files     []ManifestFile       `json:"files"`
	MainClass string               `json:"main_class"`
	Args      []string             `json:"args"`
	Java      LauncherJavaManifest `json:"jre"`
}

type JavaManifestFileDownload struct {
	Hash string `json:"sha1"`
	Size int    `json:"size"`
	Url  string `json:"url"`
}

type JavaManifestFile struct {
	Type       string `json:"type"`
	Executable bool   `json:"executable"`
	Downloads  struct {
		LZMA JavaManifestFileDownload `json:"lzma"`
		Raw  JavaManifestFileDownload `json:"raw"`
	} `json:"downloads"`
}

type JavaManifest struct {
	Files map[string]JavaManifestFile `json:"files"`
}

type MainJavaManifestVersion struct {
	Availability struct {
		Group    int `json:"group"`
		Progress int `json:"progress"`
	} `json:"availability"`
	Manifest JavaManifestFileDownload `json:"manifest"`
	Version  struct {
		Name     string `json:"name"`
		Released string `json:"released"`
	} `json:"version"`
}

// mjm["linux"]["java-runtime-gamma"]
type MainJavaManifest map[string]map[string][]MainJavaManifestVersion

type Downloadable struct {
	Url        string
	Path       string
	Sha1       string
	Sha256     string
	Size       int
	Executable bool
}
