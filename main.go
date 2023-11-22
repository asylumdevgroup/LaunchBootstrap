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
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
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

		launcherManager, err := GetLauncherManager(&settings)
		if err != nil {
			window.SetContent(
				container.NewVBox(
					widget.NewLabel(Localize("failed_init", map[string]string{"Err": err.Error()})),
				),
			)
			window.CenterOnScreen()

			return
		}

		jvmManager, err := GetJvmManager(&settings, launcherManager.launcherManifest.Java)
		if err != nil {
			window.SetContent(
				container.NewVBox(
					widget.NewLabel(Localize("failed_init", map[string]string{"Err": err.Error()})),
				),
			)
			window.CenterOnScreen()

			return
		}

		jvmFilesToDownload, err := jvmManager.ValidateInstallation()
		if err != nil {
			window.SetContent(
				container.NewVBox(
					widget.NewLabel(Localize("failed_init", map[string]string{"Err": err.Error()})),
				),
			)
			window.CenterOnScreen()

			return
		}

		launcherFilesToDownload, err := launcherManager.ValidateInstallation()
		if err != nil {
			window.SetContent(
				container.NewVBox(
					widget.NewLabel(Localize("failed_init", map[string]string{"Err": err.Error()})),
				),
			)
			window.CenterOnScreen()

			return
		}

		filesToDownload := append(jvmFilesToDownload, launcherFilesToDownload...)

		timeLabel := widget.NewLabel("00:00:00")
		mainProgressBar := widget.NewProgressBar()

		// @TODO Make this base on goroutine to download multiple file at once
		filenameLabel := widget.NewLabel("-")
		fileProgressBar := widget.NewProgressBar()

		window.SetContent(container.NewVBox(
			widget.NewLabel(Localize("downloading", nil)),
			container.NewHBox(
				widget.NewLabel(Localize("elapsed_time", nil)),
				timeLabel,
			),
			mainProgressBar,
			filenameLabel,
			fileProgressBar,
		))

		start := time.Now()
		amtFiles := len(filesToDownload)
		processedFiles := 0
		for _, f := range filesToDownload {
			err := os.MkdirAll(filepath.Dir(f.Path), os.ModePerm)
			if err != nil {
				window.SetContent(
					container.NewVBox(
						widget.NewLabel(Localize("fail_download", map[string]string{"Err": err.Error()})),
					),
				)
				window.CenterOnScreen()

				return
			}

			out, err := os.Create(f.Path)
			if ShowError(window, "fail_download", err) {
				return
			}

			done := make(chan int64)
			go func(f Downloadable) {
				var stop bool = false

				for {
					select {
					case <-done:
						stop = true
					default:
						fi, err := os.Stat(f.Path)
						if err != nil {
							log.Fatal(err)
						}

						currSize := fi.Size()
						if currSize == 0 {
							currSize = 1
						}

						fileProgressBar.SetValue(float64(currSize) / float64(f.Size))

						duration := time.Since(start).Round(time.Second)
						hours := duration / time.Hour
						duration -= hours * time.Hour
						minutes := duration / time.Minute
						duration -= minutes * time.Minute
						seconds := duration / time.Second

						timeLabel.SetText(fmt.Sprintf("%02d:%02d:%02d (%v/%v)", hours, minutes, seconds, processedFiles, amtFiles))
					}

					if stop {
						break
					}

					time.Sleep(time.Second)
				}
			}(f)

			dlFilePath := strings.TrimPrefix(
				f.Path,
				settings.LauncherPath,
			)
			if len(dlFilePath) > 20 {
				dlFilePath = "..." + dlFilePath[len(dlFilePath)-20:]
			}
			filenameLabel.SetText(dlFilePath)

			window.CenterOnScreen()

			// @TODO: 3 Retries per file
			req, err := http.NewRequest("GET", f.Url, nil)
			if ShowError(window, "fail_download", err) {
				return
			}

			resp, err := http.DefaultClient.Do(req)
			if ShowError(window, "fail_download", err) {
				return
			}
			defer resp.Body.Close()

			n, err := io.Copy(out, resp.Body)
			if ShowError(window, "fail_download", err) {
				return
			}

			out.Close()

			if f.Executable {
				err := os.Chmod(f.Path, os.ModePerm)
				if ShowError(window, "fail_download", err) {
					return
				}
			}

			done <- n

			processedFiles += 1
			mainProgressBar.SetValue(float64(processedFiles) / float64(len(filesToDownload)))
		}

		// Launching the launcher
		executablePath := ""
		classpathSeparator := ":"
		if runtime.GOOS == "darwin" {
			executablePath = "jre.bundle/Contents/Home/bin/java"
		} else if runtime.GOOS == "linux" {
			executablePath = "bin/java"
		} else if runtime.GOOS == "windows" {
			executablePath = "bin/javaw.exe"
			classpathSeparator = ";"
		} else {
			panic("How did we get here?")
		}

		classpath := []string{}
		for _, f := range launcherManager.launcherManifest.Files {
			if f.Type == "classpath" {
				classpath = append(classpath, filepath.Join(settings.LauncherPath, "launcher", f.Path))
			}
		}

		cmd := exec.Command(
			filepath.Join(jvmManager.GetPath(), executablePath),
			"-classpath",
			strings.Join(classpath, classpathSeparator),
			launcherManager.launcherManifest.MainClass,
		)

		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		cmd.Dir = settings.LauncherPath

		if err = cmd.Start(); err != nil {
			fmt.Println("Failed to run the launcher:")
			fmt.Println(err)
			os.Exit(1)
		}

		window.Hide()

		if err = cmd.Wait(); err != nil {
			fmt.Println("Failed to run the launcher:")
			fmt.Println(err)
			os.Exit(1)
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

/*
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
*/
