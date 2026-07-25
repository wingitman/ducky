// Package update checks the published ducky repository without blocking startup.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/wingitman/ducky/internal/config"
)

// Result describes an update check.
type Result struct {
	Available bool
	Current   string
	Latest    string
	Error     error
}

// Check queries the repository's default branch for its latest commit.
func Check(ctx context.Context, cfg config.Config, current string) Result {
	result := Result{Current: current}
	if cfg.Updates.DisableChecks || current == "dev" {
		return result
	}
	repository := strings.TrimSuffix(strings.TrimSpace(cfg.Updates.Repository), "/")
	parts := strings.Split(strings.TrimPrefix(repository, "https://github.com/"), "/")
	if len(parts) < 2 {
		result.Error = fmt.Errorf("invalid update repository %q", repository)
		return result
	}
	url := "https://api.github.com/repos/" + parts[0] + "/" + parts[1] + "/commits?per_page=1"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		result.Error = err
		return result
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		result.Error = err
		return result
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		result.Error = fmt.Errorf("update check returned %s", response.Status)
		return result
	}
	var commits []struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(response.Body).Decode(&commits); err != nil {
		result.Error = err
		return result
	}
	if len(commits) == 0 {
		return result
	}
	result.Latest = commits[0].SHA
	result.Available = result.Latest != current && !strings.HasPrefix(result.Latest, current)
	return result
}
