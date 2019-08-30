package auth

import (
	"sync"
	"time"

	"github.com/alpacahq/gopaca/redis"
	"github.com/go-redis/cache"
	"github.com/gofrs/uuid"
	"github.com/vmihailenco/msgpack"
)

var (
	once sync.Once
	c    *cache.Codec
)

type AuthInfo struct {
	ID         string
	Payload    Payload
	Expiration time.Duration
}

func (info *AuthInfo) Item() *cache.Item {
	return &cache.Item{
		Key:        info.ID,
		Object:     info.Payload,
		Expiration: info.Expiration,
	}
}

type Payload struct {
	AccountID    uuid.UUID
	Status       string
	DataSources  []string
	Salt         []byte
	HashedSecret []byte
}

func codec() *cache.Codec {
	once.Do(func() {
		c = &cache.Codec{
			Redis: redis.Client(),
			Marshal: func(v interface{}) ([]byte, error) {
				return msgpack.Marshal(v)
			},
			Unmarshal: func(b []byte, v interface{}) error {
				return msgpack.Unmarshal(b, v)
			},
		}
	})

	return c
}

func Get(id string) (*Payload, error) {
	pl := &Payload{}
	return pl, codec().Get(id, pl)
}

func Set(info *AuthInfo) error {
	return codec().Set(info.Item())
}

func Delete(id string) error {
	return codec().Delete(id)
}
