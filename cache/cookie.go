package cache

import (
	"errors"
	"time"

	"github.com/gasoid/merge-bot/v3/logger"
)

const (
	cookiesTTL  = time.Hour * 24 * 4
	cookies     = "mergebot:cookies"
	cookieIndex = "mergebot:cookies:index"
)

func GetCookies() ([]int64, error) {
	if cookie == nil {
		return nil, errors.New("cache is not initialized")
	}

	val, err := cookie.JsonGet(cookies)
	if err != nil {
		return nil, err
	}

	return val, cookie.ExtendTTL(cookies, cookiesTTL)
}

func SetCookies(fortunes []int64) error {
	if cookie == nil {
		return errors.New("cache is not initialized")
	}

	logger.Debug("save fortunes", "size", len(fortunes))
	if err := cookie.JsonSet(cookies, fortunes); err != nil {
		return err
	}

	if err := cookie.ExtendTTL(cookies, cookiesTTL); err != nil {
		return err
	}

	return ResetCookie()
}

func GetCookie() (int64, error) {
	if cookie == nil {
		return -1, errors.New("cache is not initialized")
	}

	if err := cookie.ExtendTTL(cookieIndex, cookiesTTL); err != nil {
		return -1, err
	}

	return cookie.Incr(cookieIndex)
}

func ResetCookie() error {
	if cookie == nil {
		return errors.New("cache is not initialized")
	}

	return cookie.Set(cookieIndex, int64(-1), cookiesTTL)
}
