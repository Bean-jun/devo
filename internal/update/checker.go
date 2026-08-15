package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"devo/internal/config"
)

const (
	githubRepoOwner = "Bean-jun"
	githubRepoName  = "devo"
	githubAPIURL    = "https://api.github.com/repos/" + githubRepoOwner + "/" + githubRepoName + "/releases/latest"
	cacheFileName   = "update_cache.json"
	checkInterval   = 24 * time.Hour
)

type Result struct {
	HasUpdate      bool   `json:"has_update"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	ReleaseURL     string `json:"release_url"`
	ReleaseName    string `json:"release_name"`
	ReleaseBody    string `json:"release_body"`
	PublishedAt    string `json:"published_at"`
	CheckedAt      string `json:"checked_at"`
}

type githubRelease struct {
	TagName     string `json:"tag_name"`
	HTMLURL     string `json:"html_url"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
}

type cacheEntry struct {
	CheckedAt     time.Time `json:"checked_at"`
	LatestVersion string    `json:"latest_version"`
	ReleaseURL    string    `json:"release_url"`
	ReleaseName   string    `json:"release_name"`
	ReleaseBody   string    `json:"release_body"`
	PublishedAt   string    `json:"published_at"`
}

func cachePath() string {
	return filepath.Join(config.DevoDir(), cacheFileName)
}

func loadCache() *cacheEntry {
	data, err := os.ReadFile(cachePath())
	if err != nil {
		return nil
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil
	}
	return &entry
}

func saveCache(entry *cacheEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	os.MkdirAll(config.DevoDir(), 0755)
	os.WriteFile(cachePath(), data, 0644)
}

func extractBaseVersion(v string) string {
	idx := strings.Index(v, "-")
	if idx >= 0 {
		return v[:idx]
	}
	return v
}

func stripVPrefix(v string) string {
	return strings.TrimPrefix(v, "v")
}

func compareVersions(a, b string) int {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")
	maxLen := len(partsA)
	if len(partsB) > maxLen {
		maxLen = len(partsB)
	}
	for i := 0; i < maxLen; i++ {
		numA := 0
		numB := 0
		if i < len(partsA) {
			numA, _ = strconv.Atoi(partsA[i])
		}
		if i < len(partsB) {
			numB, _ = strconv.Atoi(partsB[i])
		}
		if numA < numB {
			return -1
		}
		if numA > numB {
			return 1
		}
	}
	return 0
}

func fetchLatestRelease() (*githubRelease, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", githubAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	req.Header.Set("User-Agent", "devo-update-checker")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no release found")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &release, nil
}

func CheckForUpdate(currentVersion string) (*Result, error) {
	baseVersion := extractBaseVersion(currentVersion)
	now := time.Now()

	if cached := loadCache(); cached != nil {
		if now.Sub(cached.CheckedAt) < checkInterval {
			latestClean := stripVPrefix(cached.LatestVersion)
			hasUpdate := compareVersions(latestClean, baseVersion) > 0
			return &Result{
				HasUpdate:      hasUpdate,
				CurrentVersion: currentVersion,
				LatestVersion:  cached.LatestVersion,
				ReleaseURL:     cached.ReleaseURL,
				ReleaseName:    cached.ReleaseName,
				ReleaseBody:    cached.ReleaseBody,
				PublishedAt:    cached.PublishedAt,
				CheckedAt:      cached.CheckedAt.Format(time.RFC3339),
			}, nil
		}
	}

	release, err := fetchLatestRelease()
	if err != nil {
		if cached := loadCache(); cached != nil {
			latestClean := stripVPrefix(cached.LatestVersion)
			hasUpdate := compareVersions(latestClean, baseVersion) > 0
			return &Result{
				HasUpdate:      hasUpdate,
				CurrentVersion: currentVersion,
				LatestVersion:  cached.LatestVersion,
				ReleaseURL:     cached.ReleaseURL,
				ReleaseName:    cached.ReleaseName,
				ReleaseBody:    cached.ReleaseBody,
				PublishedAt:    cached.PublishedAt,
				CheckedAt:      cached.CheckedAt.Format(time.RFC3339),
			}, nil
		}
		return nil, fmt.Errorf("check update failed: %w", err)
	}

	entry := &cacheEntry{
		CheckedAt:     now,
		LatestVersion: release.TagName,
		ReleaseURL:    release.HTMLURL,
		ReleaseName:   release.Name,
		ReleaseBody:   release.Body,
		PublishedAt:   release.PublishedAt,
	}
	saveCache(entry)

	latestClean := stripVPrefix(release.TagName)
	hasUpdate := compareVersions(latestClean, baseVersion) > 0

	return &Result{
		HasUpdate:      hasUpdate,
		CurrentVersion: currentVersion,
		LatestVersion:  release.TagName,
		ReleaseURL:     release.HTMLURL,
		ReleaseName:    release.Name,
		ReleaseBody:    release.Body,
		PublishedAt:    release.PublishedAt,
		CheckedAt:      now.Format(time.RFC3339),
	}, nil
}
