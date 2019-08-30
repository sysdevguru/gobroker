package env

import (
	"os"
	"sync"
)

var dVal sync.Map

func RegisterDefault(key, defaultValue string) {
	dVal.Store(key, defaultValue)
}

func GetVar(key string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		if v, _ := dVal.Load(key); v != nil {
			return v.(string)
		} else {
			return ""
		}
	}
	return value
}
