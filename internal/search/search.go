package search

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"gls/internal/gitlab"
	"gls/internal/model"
)

type Service struct {
	Client   *gitlab.Client
	Workers  int
	Verbose  bool
	Logf     func(string, ...any)
	Progress func(Progress)
}

// Progress describes group-search completion. Completed counts one
// project-and-keyword search task.
type Progress struct {
	Completed int
	Total     int
	Projects  int
}

func (s Service) Project(ctx context.Context, projectID int, keywords []string, branch string) ([]model.SearchResult, error) {
	project, err := s.Client.Project(ctx, projectID)
	if err != nil {
		return nil, err
	}
	var results []model.SearchResult
	for _, keyword := range keywords {
		hits, err := s.search(ctx, project, keyword, branch)
		if err != nil {
			if s.Verbose {
				s.log("项目 %d 搜索 %q 失败: %v", projectID, keyword, err)
			}
			continue
		}
		results = append(results, hits...)
	}
	return results, nil
}

func (s Service) Group(ctx context.Context, groupID int, keywords []string, branch string) ([]model.SearchResult, error) {
	projects, err := s.Client.GroupProjects(ctx, groupID)
	if err != nil {
		return nil, err
	}
	workers := s.Workers
	if workers <= 0 {
		workers = 10
	}
	total := len(projects) * len(keywords)
	s.progress(0, total, len(projects))
	type job struct {
		project model.Project
		keyword string
	}
	jobs := make(chan job)
	out := make(chan []model.SearchResult)
	var wg sync.WaitGroup
	var completed atomic.Int64
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				hits, err := s.search(ctx, j.project, j.keyword, branch)
				if err != nil {
					if s.Verbose {
						s.log("项目 %s 搜索 %q 失败: %v", j.project.Name, j.keyword, err)
					}
					s.progress(int(completed.Add(1)), total, len(projects))
					continue
				}
				out <- hits
				s.progress(int(completed.Add(1)), total, len(projects))
			}
		}()
	}
	go func() {
		for _, p := range projects {
			for _, k := range keywords {
				jobs <- job{p, k}
			}
		}
		close(jobs)
		wg.Wait()
		close(out)
	}()
	var results []model.SearchResult
	for hits := range out {
		results = append(results, hits...)
	}
	return results, nil
}

func (s Service) progress(completed, total, projects int) {
	if s.Progress != nil {
		s.Progress(Progress{Completed: completed, Total: total, Projects: projects})
	}
}

func (s Service) search(ctx context.Context, project model.Project, keyword, branch string) ([]model.SearchResult, error) {
	if s.Verbose {
		s.log("搜索项目 %d (%s), 关键字: %s", project.ID, branch, keyword)
	}
	hits, err := s.Client.SearchBlobs(ctx, project.ID, keyword, branch)
	if err != nil {
		return nil, err
	}
	results := make([]model.SearchResult, 0, len(hits))
	escapedBranch := url.PathEscape(branch)
	for _, hit := range hits {
		path := hit.Path
		if path == "" {
			path = hit.Filename
		}
		line := matchLineNumber(hit.StartLine, hit.Data, keyword)
		results = append(results, model.SearchResult{Keyword: keyword, ProjectID: project.ID, ProjectName: project.Name, Branch: branch, ProjectPath: project.WebURL, FilePath: path, LineNumber: line, LineContent: strings.TrimSpace(hit.Data), URL: fmt.Sprintf("%s/-/blob/%s/%s#L%d", project.WebURL, escapedBranch, encodePath(path), line)})
	}
	return results, nil
}

// matchLineNumber converts GitLab's snippet start line into the exact line
// containing the keyword. A blob search result can include surrounding context,
// so StartLine alone often points to an earlier line than the actual match.
func matchLineNumber(startLine int, snippet, keyword string) int {
	if startLine < 1 {
		startLine = 1
	}
	for offset, line := range strings.Split(snippet, "\n") {
		if strings.Contains(line, keyword) {
			return startLine + offset
		}
	}
	return startLine
}

func encodePath(path string) string {
	parts := strings.Split(path, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}
func (s Service) log(format string, args ...any) {
	if s.Logf != nil {
		s.Logf(format, args...)
	}
}
