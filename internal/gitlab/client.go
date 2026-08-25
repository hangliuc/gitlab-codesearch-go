package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"gls/internal/model"
)

const defaultPerPage = 100

// Client wraps the small GitLab REST API surface used by this program.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	mu         sync.Mutex
	projects   map[int]model.Project
}

func NewClient(gitLabURL, token string) (*Client, error) {
	base, err := normalizeURL(gitLabURL)
	if err != nil {
		return nil, err
	}
	return &Client{baseURL: base + "/api/v4", token: token, httpClient: &http.Client{Timeout: 15 * time.Second}, projects: make(map[int]model.Project)}, nil
}

func normalizeURL(value string) (string, error) {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" {
		return "", fmt.Errorf("GitLab URL 不能为空")
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("GitLab URL 必须是有效地址，例如 https://gitlab.example.com")
	}
	return strings.TrimRight(u.String(), "/"), nil
}

func (c *Client) request(ctx context.Context, path string, query url.Values, dst any) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return err
		}
		req.Header.Set("PRIVATE-TOKEN", c.token)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
		} else {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				lastErr = readErr
			} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return json.Unmarshal(body, dst)
			} else if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
				lastErr = fmt.Errorf("GitLab API 返回 %s", resp.Status)
				if seconds, e := strconv.Atoi(resp.Header.Get("Retry-After")); e == nil && seconds > 0 {
					if err := wait(ctx, time.Duration(seconds)*time.Second); err != nil {
						return err
					}
					continue
				}
			} else {
				return fmt.Errorf("GitLab API 返回 %s: %s", resp.Status, strings.TrimSpace(string(body)))
			}
		}
		if attempt < 2 {
			if err := wait(ctx, time.Duration(1<<attempt)*time.Second); err != nil {
				return err
			}
		}
	}
	return lastErr
}

func wait(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

func (c *Client) Project(ctx context.Context, id int) (model.Project, error) {
	c.mu.Lock()
	project, ok := c.projects[id]
	c.mu.Unlock()
	if ok {
		return project, nil
	}
	if err := c.request(ctx, "/projects/"+strconv.Itoa(id), nil, &project); err != nil {
		return model.Project{}, err
	}
	c.mu.Lock()
	c.projects[id] = project
	c.mu.Unlock()
	return project, nil
}

func (c *Client) GroupProjects(ctx context.Context, groupID int) ([]model.Project, error) {
	var all []model.Project
	for page := 1; ; page++ {
		var items []model.Project
		q := url.Values{"per_page": {strconv.Itoa(defaultPerPage)}, "page": {strconv.Itoa(page)}, "include_subgroups": {"true"}}
		if err := c.request(ctx, "/groups/"+strconv.Itoa(groupID)+"/projects", q, &items); err != nil {
			return nil, err
		}
		all = append(all, items...)
		if len(items) < defaultPerPage {
			return all, nil
		}
	}
}

func (c *Client) SearchBlobs(ctx context.Context, projectID int, keyword, branch string) ([]model.Blob, error) {
	var all []model.Blob
	for page := 1; ; page++ {
		var items []model.Blob
		q := url.Values{"scope": {"blobs"}, "search": {keyword}, "ref": {branch}, "per_page": {strconv.Itoa(defaultPerPage)}, "page": {strconv.Itoa(page)}}
		if err := c.request(ctx, "/projects/"+strconv.Itoa(projectID)+"/search", q, &items); err != nil {
			return nil, err
		}
		all = append(all, items...)
		if len(items) < defaultPerPage {
			return all, nil
		}
	}
}
