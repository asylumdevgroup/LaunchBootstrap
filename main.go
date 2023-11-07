package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/Masterminds/semver/v3"
)

//go:embed bs_settings.json
var BOOTSTRAP_SETTINGS_STR []byte

var basepath *string

var BOOTSTRAP_VERSION = "1.0.0"

func init() {
	basepath = flag.String("path", "", "The path to store launcher data (i.e. portable-mode)")
}

func main() {
	flag.Parse()

	app := app.New()
	window := app.NewWindow("SpectrumBootstrap")
	window.SetFixedSize(true)

	go func() {
		window.SetContent(
			container.NewVBox(
				widget.NewLabel(Localize("fetching_launcher_updates", nil)),
			),
		)
		window.CenterOnScreen()

		settings := BootstrapSettings{}
		err := json.Unmarshal(BOOTSTRAP_SETTINGS_STR, &settings)
		if err != nil {
			window.SetContent(
				container.NewVBox(
					widget.NewLabel(Localize("failed_load_bs_settings", map[string]string{"Err": err.Error()})),
				),
			)
			window.CenterOnScreen()

			return
		}

		if len(*basepath) > 0 {
			settings.LauncherPath = *basepath
		}

		settings.LauncherPath, err = GetLauncherDirectory(&settings)
		if err != nil {
			window.SetContent(
				container.NewVBox(
					widget.NewLabel(Localize("failed_init", map[string]string{"Err": err.Error()})),
				),
			)
			window.CenterOnScreen()

			return
		}

		window.SetTitle(settings.Brand + " - Bootstrap")
		launcherExec, _ := settings.GetLauncherExecutable()

		latestManifest, err := DownloadManifest(&settings)
		if err != nil {
			window.SetContent(
				container.NewVBox(
					widget.NewLabel(Localize("failed_init", map[string]string{"Err": err.Error()})),
				),
			)
			window.CenterOnScreen()

			return
		}

		installedVersion, err := GetInstalledLauncherVersion(&settings, launcherExec)
		if err != nil {
			window.SetContent(
				container.NewVBox(
					widget.NewLabel(Localize("failed_init", map[string]string{"Err": err.Error()})),
				),
			)
			window.CenterOnScreen()

			return
		}

		if installedVersion.Version == NOT_DOWNLOADED {
			fmt.Println("Not downloaded")
			DownloadLatestLauncher(&settings, latestManifest, window, launcherExec)
		} else {
			currentVersionSemver, err1 := semver.NewVersion(installedVersion.Version)
			latestVersionSemver, err2 := semver.NewVersion(latestManifest.Version)

			if err1 != nil || err2 != nil {
				fmt.Println("Failed to parse semver")
				fmt.Println("Current: ", err1)
				fmt.Println("Latest: ", err2)

				DownloadLatestLauncher(&settings, latestManifest, window, launcherExec)
				return
			}

			if currentVersionSemver.LessThan(latestVersionSemver) {
				fmt.Println("Old launcher: " + currentVersionSemver.String() + " / " + latestVersionSemver.String())
				buttonBox := container.NewHBox(
					layout.NewSpacer(),
					widget.NewButton(Localize("skip_button", nil), func() {
						RunLauncher(&settings, latestManifest, installedVersion, window, launcherExec, false)
					}),
					widget.NewButton(Localize("update_button", nil), func() {
						DownloadLatestLauncher(&settings, latestManifest, window, launcherExec)
					}),
				)

				window.SetContent(
					container.NewVBox(
						widget.NewLabel(Localize("installed_version", map[string]string{
							"Version": installedVersion.Version,
						})),
						widget.NewLabel(Localize("latest_version", map[string]string{
							"Version": latestManifest.Version,
						})),
						buttonBox,
					),
				)
				window.CenterOnScreen()
			} else {
				fmt.Println("Launcher up to date")
				RunLauncher(&settings, latestManifest, installedVersion, window, launcherExec, false)
			}

		}
	}()

	window.ShowAndRun()
}

func ShowError(w fyne.Window, translation string, err error) bool {
	if err != nil {
		w.SetContent(
			container.NewVBox(
				widget.NewLabel(Localize(translation, nil)),
				widget.NewLabel(err.Error()),
			),
		)
		w.CenterOnScreen()

		return true
	}

	return false
}

func DownloadLatestLauncher(bs *BootstrapSettings, manifest *LauncherManifest, w fyne.Window, launcherExec string) {
	versionUrl, ok := manifest.Urls[runtime.GOOS+"-"+runtime.GOARCH]
	if !ok {
		w.SetContent(
			container.NewVBox(
				widget.NewLabel(Localize("not_available_os", nil)),
				widget.NewLabel(runtime.GOOS+" / "+runtime.GOARCH),
			),
		)
		w.CenterOnScreen()

		return
	}

	_, err := os.Stat(launcherExec)
	if err == nil {
		_, err2 := os.Stat(launcherExec + ".old")
		if err2 == nil {
			err2 = os.Remove(launcherExec + ".old")
			if ShowError(w, "fail_delete_old", err2) {
				return
			}
		}

		err = os.Rename(launcherExec, launcherExec+".old")
		if ShowError(w, "fail_rename_old", err) {
			return
		}
	}

	timeLabel := widget.NewLabel("00:00:00")
	progressBar := widget.NewProgressBar()

	w.SetContent(container.NewVBox(
		widget.NewLabel(Localize("downloading", nil)),
		container.NewHBox(
			widget.NewLabel(Localize("elapsed_time", nil)),
			timeLabel,
		),
		progressBar,
	))

	start := time.Now()
	out, err := os.Create(launcherExec)
	if ShowError(w, "fail_download", err) {
		return
	}
	defer out.Close()

	headReq, err := http.NewRequest("HEAD", versionUrl["url"], nil)
	if ShowError(w, "fail_download", err) {
		return
	}

	headResp, err := http.DefaultClient.Do(headReq)
	if ShowError(w, "fail_download", err) {
		return
	}
	defer headResp.Body.Close()

	size, err := strconv.Atoi(headResp.Header.Get("Content-Length"))
	if ShowError(w, "fail_download", err) {
		return
	}

	done := make(chan int64)
	go func() {
		var stop bool = false

		for {
			select {
			case <-done:
				stop = true
			default:
				fi, err := os.Stat(launcherExec)
				if err != nil {
					log.Fatal(err)
				}

				currSize := fi.Size()
				if currSize == 0 {
					currSize = 1
				}

				progressBar.SetValue(float64(currSize) / float64(size))

				duration := time.Since(start).Round(time.Second)
				hours := duration / time.Hour
				duration -= hours * time.Hour
				minutes := duration / time.Minute
				duration -= minutes * time.Minute
				seconds := duration / time.Second

				timeLabel.SetText(fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds))
			}

			if stop {
				break
			}

			time.Sleep(time.Second)
		}
	}()

	req, err := http.NewRequest("GET", versionUrl["url"], nil)
	if ShowError(w, "fail_download", err) {
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if ShowError(w, "fail_download", err) {
		return
	}
	defer resp.Body.Close()

	n, err := io.Copy(out, resp.Body)
	if ShowError(w, "fail_download", err) {
		return
	}

	done <- n

	lv := LauncherVersion{
		Version: manifest.Version,
		Hash:    versionUrl["sha256"],
	}

	lvjson, _ := json.Marshal(lv)
	err = os.WriteFile(filepath.Join(bs.LauncherPath, "launcher_version.json"), lvjson, os.ModePerm)
	if ShowError(w, "fail_download", err) {
		return
	}

	RunLauncher(bs, manifest, &lv, w, launcherExec, true)
}

func RunLauncher(bs *BootstrapSettings, manifest *LauncherManifest, installedVersion *LauncherVersion, w fyne.Window, launcherExec string, justDownloaded bool) {
	if installedVersion.Hash != GetHash(launcherExec) {
		if manifest == nil {
			w.SetContent(
				container.NewVBox(
					widget.NewLabel(Localize("hash_not_match", nil)),
				),
			)
			w.CenterOnScreen()

			return
		} else {
			if justDownloaded {
				w.SetContent(
					container.NewVBox(
						widget.NewLabel(Localize("newly_corrupted", nil)),
					),
				)
				w.CenterOnScreen()

				return
			}

			DownloadLatestLauncher(bs, manifest, w, launcherExec)
			return
		}
	}

	w.Hide()

	cmd := exec.Command(launcherExec, "--path="+bs.LauncherPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	shouldExit := true

	err := cmd.Run()
	if err != nil {
		shouldExit = false

		w.Show()
		w.SetContent(
			container.NewVBox(
				widget.NewLabel("Execution stopped: "),
				widget.NewLabel(err.Error()),
				widget.NewButton("Exit", func() {
					w.Close()
				}),
			),
		)

		w.CenterOnScreen()

		return
	}

	if shouldExit {
		w.Close()
	}
}
