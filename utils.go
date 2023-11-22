package main

import (
	"crypto/sha1"
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

func GetOrCached[T interface{}](bs *BootstrapSettings, cachePath, url string) (*T, error) {
	cached, cachedErr := LoadFromCache[T](cachePath)
	// There is no error for file not found or file corrupted
	// So if we have an error here, there is a deeper issue and we need to raise
	if cachedErr != nil {
		return nil, cachedErr
	}

	live, liveErr := DoGetRequest[T](bs, url)
	// If we can't get it but the cache is loaded, no issue
	// If we can't get it and no cache: CRASH
	if liveErr != nil && cached != nil {
		return cached, nil
	} else if liveErr != nil {
		return nil, liveErr
	}

	// We got it, lets cache it while we're at it!
	err := os.MkdirAll(filepath.Dir(cachePath), os.ModePerm)
	if err != nil {
		return nil, err
	}

	f, err := os.Create(cachePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, _ := json.MarshalIndent(live, "", "  ")
	_, err = f.Write(data)

	return live, err
}

func DoGetRequest[T interface{}](bs *BootstrapSettings, url string) (*T, error) {
	client := &http.Client{}

	req, err := http.NewRequest(
		"GET",
		url,
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

	manifest := new(T)
	err = json.NewDecoder(resp.Body).Decode(manifest)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}

func LoadFromCache[T interface{}](filepath string) (*T, error) {
	_, err := os.Stat(filepath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	} else if err != nil {
		return nil, nil
	}

	out, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	manifest := new(T)
	err = json.Unmarshal(out, manifest)
	if err != nil {
		// If the file is corrupted
		// We want to download the new one directly
		fmt.Println(err)
		return nil, nil
	}

	return manifest, nil
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

func GetHashSha1(filepath string) string {
	f, err := os.Open(filepath)
	if err != nil {
		return ""
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}
