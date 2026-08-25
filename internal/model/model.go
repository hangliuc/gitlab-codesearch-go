package model

// Project is the subset of the GitLab project response used by the searcher.
type Project struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	WebURL string `json:"web_url"`
}

// Blob is a hit returned by GitLab's blobs search endpoint.
type Blob struct {
	Path      string `json:"path"`
	Filename  string `json:"filename"`
	StartLine int    `json:"startline"`
	Data      string `json:"data"`
}

// SearchResult is a normalized, exportable code-search result.
type SearchResult struct {
	Keyword     string `json:"keyword"`
	ProjectID   int    `json:"project_id"`
	ProjectName string `json:"project_name"`
	Branch      string `json:"branch"`
	ProjectPath string `json:"project_path"`
	FilePath    string `json:"file_path"`
	LineNumber  int    `json:"line_number"`
	LineContent string `json:"line_content"`
	URL         string `json:"url"`
}
