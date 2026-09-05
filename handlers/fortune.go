package handlers

import (
	"embed"
	"fmt"
	"path"

	"github.com/vromero/gofortune/pkg/fortune"
)

const (
	fortunesDir = "fortunes"
)

var (
	//go:embed fortunes/*
	fortuneFS embed.FS
)

func getCookie() (string, error) {
	dir, err := fortuneFS.ReadDir(fortunesDir)
	if err != nil {
		return "", fmt.Errorf("can't read dir with fortunes: %w", err)
	}

	paths := make([]fortune.ProbabilityPath, 0, len(dir))

	for _, entry := range dir {
		if entry.Type().IsDir() {
			continue
		}

		paths = append(paths, fortune.ProbabilityPath{Path: path.Join(fortunesDir, entry.Name())})
	}

	tree, err := fortune.LoadPaths(paths, 0, ^uint32(0))
	if err != nil {
		return "", fmt.Errorf("can't LoadPaths: %w", err)
	}

	fortune.SetProbabilities(&tree, false)

	cookie, err := fortune.GetRandomFortune(tree)
	if err != nil {
		return "", fmt.Errorf("GetRandomFortune failed: %w", err)
	}

	return cookie.Data, nil
}
