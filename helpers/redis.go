package helpers

import (
	"time"
	"ruscraper/core"
)

func SetTimeCounter(key string) {

	timeStr := time.Now().Format("20060102")

	result, _ := GetTimeCounter(key)

	if result == "" {
		_ = core.Units.Redis.Set(timeStr + key, 0, 0).Err()
	}

	core.Units.Redis.Incr(timeStr + key)
}

func GetTimeCounter(key string) (string, error) {

	timeStr := time.Now().Format("20060102")
	return core.Units.Redis.Get(timeStr + key).Result()
}