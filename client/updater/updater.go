package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/minio/selfupdate"
)

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func CheckForUpdates(owner, repo, currentVersion string) (bool, string, string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)

	resp, err := http.Get(url)
	if err != nil {
		return false, "", "", fmt.Errorf("failed to fetch releases: %v", err)
	}
	defer resp.Body.Close()

	var releases []GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return false, "", "", fmt.Errorf("failed to decode releases: %v", err)
	}

	for _, release := range releases {
		if strings.HasPrefix(release.TagName, "client-") {
			if release.TagName != currentVersion {
				targetAsset := fmt.Sprintf("dms-%s-%s", runtime.GOOS, runtime.GOARCH) // Remember to compile to this naming convention
				for _, asset := range release.Assets {
					if strings.Contains(asset.Name, targetAsset) {
						return true, release.TagName, asset.BrowserDownloadURL, nil
					}
				}
			}
			return false, "", "", nil
		}
	}
	return false, "", "", nil
}

func DoUpdate(downloadURL string) error {
	resp, err := http.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	err = selfupdate.Apply(resp.Body, selfupdate.Options{})
	if err != nil {
		return fmt.Errorf("failed to apply update: %v", err)
	}
	return nil
}
