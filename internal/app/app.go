package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"gls/internal/gitlab"
	"gls/internal/model"
	"gls/internal/output"
	"gls/internal/search"
)

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("gls", flag.ContinueOnError)
	fs.SetOutput(stderr)
	url, token, group, outputFile := fs.String("url", "", "GitLab URL"), fs.String("token", "", "GitLab API Private Token"), fs.Int("group", 0, "Group ID"), fs.String("output", "", "输出文件（.xlsx/.csv/.json）")
	fs.StringVar(url, "u", "", "--url 的简写")
	fs.StringVar(token, "t", "", "--token 的简写")
	fs.IntVar(group, "g", 0, "--group 的简写")
	fs.StringVar(outputFile, "o", "", "--output 的简写")
	project, workers := fs.Int("project", 0, "Project ID"), fs.Int("workers", 10, "并发数")
	fs.IntVar(project, "p", 0, "--project 的简写")
	var keywords, branches stringList
	fs.Var(&keywords, "keywords", "关键字，可重复或用逗号分隔（必填）")
	fs.Var(&keywords, "k", "--keywords 的简写")
	fs.Var(&branches, "branch", "分支，可重复或用逗号分隔（默认 master,main）")
	fs.Var(&branches, "b", "--branch 的简写")
	verbose := fs.Bool("verbose", false, "显示详细日志")
	fs.BoolVar(verbose, "v", false, "--verbose 的简写")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "用法: gls -u URL -t TOKEN (-g GROUP | -p PROJECT) -k KEYWORD [选项]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *url == "" || *token == "" || len(keywords) == 0 {
		return fmt.Errorf("--url、--token 和 --keywords 是必填参数")
	}
	if (*group == 0) == (*project == 0) {
		return fmt.Errorf("必须且只能指定 --group 或 --project")
	}
	if len(branches) == 0 {
		branches = []string{"master", "main"}
	}
	client, err := gitlab.NewClient(*url, *token)
	if err != nil {
		return err
	}
	svc := search.Service{Client: client, Workers: *workers, Verbose: *verbose, Logf: func(f string, a ...any) { fmt.Fprintf(stderr, f+"\n", a...) }}
	var allResults []model.SearchResult
	for _, branch := range branches {
		if *verbose {
			fmt.Fprintf(stderr, "正在搜索分支: %s\n", branch)
		}
		var found []model.SearchResult
		if *project != 0 {
			found, err = svc.Project(ctx, *project, keywords, branch)
		} else {
			found, err = svc.Group(ctx, *group, keywords, branch)
		}
		if err != nil {
			return err
		}
		allResults = append(allResults, found...)
	}
	if err := output.Write(stdout, allResults, *outputFile); err != nil {
		return err
	}
	if *outputFile != "" && len(allResults) > 0 {
		fmt.Fprintln(stdout, output.SuccessMessage(*outputFile))
	}
	return nil
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(value string) error {
	for _, v := range strings.Split(value, ",") {
		if v = strings.TrimSpace(v); v != "" {
			*s = append(*s, v)
		}
	}
	return nil
}
