package handlers

import (
	"embed"
	"errors"
	"math/rand"
	"sync"

	"github.com/gasoid/merge-bot/v3/cache"
	"github.com/gasoid/merge-bot/v3/logger"
	"gopkg.in/yaml.v3"
)

const (
	fortuneFile = "fortunes/fortune.yaml"
)

var (
	//go:embed fortunes/fortune.yaml
	fortuneFS embed.FS
	once      = sync.OnceValue(extractEmbeddedFortunes)
)

type fortune struct {
	Fortunes []string `yaml:"fortunes"`
}

func (f *fortune) get() string {
	if len(f.Fortunes) == 0 {
		logger.Info("no fortunes")
		return ""
	}

	item, err := cache.GetCookie()
	if err != nil {
		logger.Info("can't GetCookie", "err", err)
		return ""
	}

	shuffleBag, err := cache.GetCookies()
	if err != nil {
		logger.Info("can't GetCookies", "err", err)
		return ""
	}

	if item < int64(len(shuffleBag)) && shuffleBag[item] < int64(len(f.Fortunes)) {
		return f.Fortunes[shuffleBag[item]]
	}

	if err := cache.ResetCookie(); err != nil {
		return ""
	}

	return f.Fortunes[shuffleBag[0]]
}

func extractEmbeddedFortunes() *fortune {
	phrases := new(fortune{Fortunes: []string{}})

	data, err := fortuneFS.ReadFile(fortuneFile)
	if err != nil {
		logger.Info("can't ReadFile", "err", err)
		return phrases
	}

	if err := yaml.Unmarshal(data, phrases); err != nil {
		logger.Info("can't Unmarshal fortune.yaml", "err", err)
		return nil
	}

	if len(phrases.Fortunes) == 0 {
		logger.Info("no fortunes")
		return phrases
	}

	shuffleBag, err := cache.GetCookies()
	if err != nil {
		logger.Info("can't GetCookies", "err", err)
		return phrases
	}

	if len(shuffleBag) != len(phrases.Fortunes) {
		cookies := make([]int64, 0, len(phrases.Fortunes))
		for i := int64(0); i < int64(len(phrases.Fortunes)); i++ {
			cookies = append(cookies, i)
		}

		rand.Shuffle(len(cookies), func(i, j int) {
			cookies[i], cookies[j] = cookies[j], cookies[i]
		})

		if err := cache.SetCookies(cookies); err != nil {
			logger.Info("can't SetCookies", "err", err)
			return phrases
		}
	}

	return phrases
}

func getCookie() (string, error) {
	phrases := once()
	if len(phrases.Fortunes) == 0 {
		return "", errors.New("extractEmbeddedFortunes didn't extract phrases")
	}

	sentence := phrases.get()
	if sentence == "" {
		return "", errors.New("can't get any phrase, check logs")
	}

	return sentence, nil
}
