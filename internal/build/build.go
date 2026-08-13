package build

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/mod/semver"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type Info struct {
	Version string
	Commit  string
	Date    string
}

func Current() Info {
	return Info{Version: version, Commit: commit, Date: date}
}

func IsDev() bool {
	if version == "dev" {
		return true
	}
	return !semver.IsValid(semverPrefix(version))
}

func (i Info) String() string {
	return fmt.Sprintf("version=%s commit=%s date=%s", i.Version, i.Commit, i.Date)
}

func CheckLatest(ctx context.Context, client *http.Client, owner, repo string, current Info) (string, bool, error) {
	if client == nil {
		client = http.DefaultClient
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "yaps/"+current.Version)

	resp, err := client.Do(req)
	if err != nil {
		return "", false, err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			slog.Error("error closing response body", "error", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("github api status %d", resp.StatusCode)
	}

	var body struct {
		TagName string `json:"tag_name"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", false, err
	}

	latest := body.TagName
	cur := semverPrefix(current.Version)
	canon := semver.Canonical(latest)
	if !semver.IsValid(canon) {
		return latest, false, nil
	}

	outdated := semver.Compare(cur, canon) < 0
	return latest, outdated, nil
}

func semverPrefix(v string) string {
	if v == "" {
		return "v0.0.0"
	}
	if v[0] == 'v' {
		return v
	}
	return "v" + v
}
