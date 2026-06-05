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

func candidatesKey(id int) string {
	return fmt.Sprintf("%s:%d:candidates", keyPrefix, id)
}

func JsonSet(id int, counts map[string]int) error {
	key := fmt.Sprintf("%s:%d", keyPrefix, id)

	data, err := json.Marshal(counts)
	if err != nil {
		return fmt.Errorf("can't serialize candidates %w", err)
	}

	return contributors.JsonSet(key, string(data))
}

func JsonIncr(id int, item string) error {

	contributors.JsonASet(fmt.Sprintf("%s:%d", keyPrefix, id), item, 0, 0)
	return nil
}

func JsonGet(id int) (map[string]int, error) {
	key := fmt.Sprintf("%s:%d", keyPrefix, id)

	val, err := contributors.JsonGet(key)
	if err != nil {
		return nil, err
	}

	result := []any{}

	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, fmt.Errorf("json data is invalid %w", err)
	}

	if len(result) == 0 {
		return nil, nil
	}

	if candidates, ok := result[0].(map[string]int); ok {
		return candidates, nil
	}

	return nil, nil
}

func Get(id int) ([]string, error) {
	val, err := contributors.Get(candidatesKey(id))
	if err != nil {
		return nil, err
	}

	candidates := []string{}
	if err := json.Unmarshal([]byte(val), &candidates); err != nil {
		return nil, fmt.Errorf("json data is invalid %w", err)
	}

	return candidates, nil
}

func Set(id int, candidates []string) error {
	data, err := json.Marshal(candidates)
	if err != nil {
		return fmt.Errorf("can't serialize candidates %w", err)
	}

	return contributors.Set(candidatesKey(id), string(data))
}

func Connect() error {
	return contributors.Connect()
}
