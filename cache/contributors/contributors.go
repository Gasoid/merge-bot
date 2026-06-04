package contributors

import (
	"encoding/json"
	"fmt"

	"github.com/gasoid/merge-bot/cache"
)

var (
	contributors cache.Cache = &cache.RedisCache{}
)

const (
	keyPrefix = "mergebot:contributors"
)

func candidatesKey(repo string) string {
	return fmt.Sprintf("%s:%s:candidates", keyPrefix, repo)
}

func Add(repo, candidate string, count int) error {
	key := fmt.Sprintf("%s:%s", keyPrefix, repo)
	return contributors.Add(key, candidate, float64(count))
}

// func Pop(repo string) (string, error) {
// 	key := fmt.Sprintf("%s:%s", keyPrefix, repo)
// 	return contributors.Pop(key)
// }

func Get(repo string) ([]string, error) {
	val, err := contributors.Get(candidatesKey(repo))
	if err != nil {
		return nil, err
	}

	candidates := []string{}
	if err := json.Unmarshal([]byte(val), &candidates); err != nil {
		return nil, fmt.Errorf("json data is invalid %w", err)
	}

	return candidates, nil
}

func Set(repo string, candidates []string) error {
	data, err := json.Marshal(candidates)
	if err != nil {
		return fmt.Errorf("can't serialize candidates %w", err)
	}

	return contributors.Set(candidatesKey(repo), string(data))
}

func Connect() error {
	return contributors.Connect()
}
