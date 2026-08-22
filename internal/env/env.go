package env

import (
	"os"
	"strconv"
)

func GetString(key string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return ""
	}

	return val
}

func GetInt(key string) int {
	val, ok := os.LookupEnv(key)
	if !ok {
		return 0
	}

	valAsInt, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}

	return valAsInt
}