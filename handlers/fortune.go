package handlers

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/gasoid/merge-bot/v3/logger"
	"github.com/vromero/gofortune/pkg/fortune"
)

const (
	fortunesDir = "fortunes"
)

var (
	//go:embed fortunes/*
	fortuneFS embed.FS
)

func extractEmbeddedFortunes() string {
	tmpDir, err := os.MkdirTemp("", fortunesDir+"-*")
	if err != nil {
		logger.Info("MkdirTemp failed", "err", err)
		return ""
	}

	entries, _ := fortuneFS.ReadDir(fortunesDir)
	for _, e := range entries {
		data, _ := fortuneFS.ReadFile(filepath.Join(fortunesDir, e.Name()))
		os.WriteFile(filepath.Join(tmpDir, e.Name()), data, 0644)
	}
	return tmpDir
}

func getCookie() (string, error) {
	once := sync.OnceValue(extractEmbeddedFortunes)
	tmpDir := once()
	if tmpDir == "" {
		return "", errors.New("extractEmbeddedFortunes didn't extract files")
	}

	paths := []fortune.ProbabilityPath{{Path: tmpDir}}

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
