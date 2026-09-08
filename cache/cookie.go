package cache

import (
	"time"

	"github.com/gasoid/merge-bot/v3/logger"
)

const (
	cookiesTTL  = time.Hour * 24 * 4
	cookies     = "mergebot:cookies"
	cookieIndex = "mergebot:cookies:index"
)

func GetCookies() ([]int64, error) {
	val, err := cookie.JsonGet(cookies)
	if err != nil {
		return nil, err
	}

	return val, cookie.ExtendTTL(cookies, cookiesTTL)
}

func SetCookies(fortunes []int64) error {
	logger.Debug("save fortunes", "size", len(fortunes))
	if err := cookie.JsonSet(cookies, fortunes); err != nil {
		return err
	}

	return cookie.ExtendTTL(cookies, cookiesTTL)
}

func GetCookie() (int64, error) {
	return cookie.Incr(cookieIndex)
}

func ResetCookie() error {
	return cookie.Set(cookieIndex, int64(0), cookiesTTL)
}
