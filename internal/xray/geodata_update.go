package xray

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	geoIPDownloadURL   = "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat"
	geoSiteDownloadURL = "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat"
	ghProxyPrefix      = "https://gh-proxy.com/"
)

var geoDataUpdateMu sync.Mutex

type downloadedGeoFile struct {
	name      string
	temporary string
	target    string
	backup    string
	hadTarget bool
	installed bool
}

// UpdateGeoData downloads both Xray GeoData assets before replacing either
// existing file. Temporary and backup files live beside the targets so all
// renames stay on the same filesystem on macOS, Linux, and Windows.
func UpdateGeoData(assetDir string, useGHProxy bool) error {
	geoDataUpdateMu.Lock()
	defer geoDataUpdateMu.Unlock()

	if strings.TrimSpace(assetDir) == "" {
		assetDir = "~/.krab"
	}
	dir, err := resolveGeoAssetDir(assetDir)
	if err != nil {
		return err
	}
	if dir == "" {
		return fmt.Errorf("GeoData resource directory is required")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create GeoData resource directory: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	files := []downloadedGeoFile{
		{name: "geoip.dat", target: filepath.Join(dir, "geoip.dat")},
		{name: "geosite.dat", target: filepath.Join(dir, "geosite.dat")},
	}
	urls := []string{geoIPDownloadURL, geoSiteDownloadURL}
	for index := range files {
		downloadURL := geoDataDownloadURL(urls[index], useGHProxy)
		temporary := files[index].target + ".new"
		err := downloadGeoFile(client, files[index].name, downloadURL, temporary)
		if err != nil {
			cleanupGeoDownloads(files)
			return err
		}
		files[index].temporary = temporary
	}

	if err := installGeoFiles(files); err != nil {
		return err
	}
	return nil
}

func geoDataDownloadURL(source string, useGHProxy bool) string {
	if useGHProxy {
		return ghProxyPrefix + source
	}
	return source
}

func downloadGeoFile(client *http.Client, name, downloadURL, temporaryPath string) (resultErr error) {
	request, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("prepare %s download: %w", name, err)
	}
	request.Header.Set("User-Agent", "Krab/1.0")
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download %s: %w", name, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("download %s: server returned %s", name, response.Status)
	}

	temporary, err := os.OpenFile(temporaryPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create %s.new: %w", name, err)
	}
	defer func() {
		if resultErr != nil {
			_ = os.Remove(temporaryPath)
		}
	}()
	written, copyErr := io.Copy(temporary, response.Body)
	syncErr := temporary.Sync()
	closeErr := temporary.Close()
	if copyErr != nil {
		return fmt.Errorf("write %s.new: %w", name, copyErr)
	}
	if syncErr != nil {
		return fmt.Errorf("sync %s.new: %w", name, syncErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close %s.new: %w", name, closeErr)
	}
	if written == 0 {
		return fmt.Errorf("download %s: empty response", name)
	}
	if err := os.Chmod(temporaryPath, 0o600); err != nil {
		return fmt.Errorf("set %s.new permissions: %w", name, err)
	}
	return nil
}

func installGeoFiles(files []downloadedGeoFile) error {
	defer cleanupGeoDownloads(files)
	for index := range files {
		backup, err := os.CreateTemp(filepath.Dir(files[index].target), "."+files[index].name+".backup-*")
		if err != nil {
			rollbackGeoFiles(files)
			return fmt.Errorf("prepare %s backup: %w", files[index].name, err)
		}
		files[index].backup = backup.Name()
		if err := backup.Close(); err != nil {
			rollbackGeoFiles(files)
			return fmt.Errorf("close %s backup: %w", files[index].name, err)
		}
		_ = os.Remove(files[index].backup)
		if err := os.Rename(files[index].target, files[index].backup); err == nil {
			files[index].hadTarget = true
		} else if !os.IsNotExist(err) {
			rollbackGeoFiles(files)
			return fmt.Errorf("back up %s: %w", files[index].name, err)
		}
		if err := os.Rename(files[index].temporary, files[index].target); err != nil {
			rollbackGeoFiles(files)
			return fmt.Errorf("install %s: %w", files[index].name, err)
		}
		files[index].temporary = ""
		files[index].installed = true
	}
	return nil
}

func rollbackGeoFiles(files []downloadedGeoFile) {
	for index := len(files) - 1; index >= 0; index-- {
		if files[index].installed {
			_ = os.Remove(files[index].target)
		}
		if files[index].hadTarget {
			_ = os.Rename(files[index].backup, files[index].target)
			files[index].backup = ""
		}
	}
}

func cleanupGeoDownloads(files []downloadedGeoFile) {
	for _, file := range files {
		if file.temporary != "" {
			_ = os.Remove(file.temporary)
		}
		if file.backup != "" {
			_ = os.Remove(file.backup)
		}
	}
}
