package redis

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/log"
	"github.com/go-redis/redis"
)

var (
	once sync.Once
	rc   *redis.Client
)

func Client() *redis.Client {
	once.Do(func() {
		var scheme string
		if env.GetVar("REDIS_USE_SSL") != "" {
			scheme = "rediss"
		} else {
			scheme = "redis"
		}

		db, _ := strconv.ParseInt(env.GetVar("REDIS_DB"), 10, 8)

		password := env.GetVar("REDIS_PASSWORD")
		var options *redis.Options
		if password != "" {
			options, _ = redis.ParseURL(fmt.Sprintf("%v://:%v@%v:%v/%v", scheme, password, env.GetVar("REDIS_HOST"), env.GetVar("REDIS_PORT"), db))
		} else {
			options, _ = redis.ParseURL(fmt.Sprintf("%v://%v:%v/%v", scheme, env.GetVar("REDIS_HOST"), env.GetVar("REDIS_PORT"), db))
		}

		rc = redis.NewClient(options)
		if err := rc.Ping().Err(); err != nil {
			log.Fatal(err.Error())
		}
	})
	return rc
}
