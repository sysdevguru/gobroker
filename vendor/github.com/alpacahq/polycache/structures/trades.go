package structures

import "time"

// PolyTrade is the reference structure sent
// by polygon for quote data
type PolyTrade struct {
	Symbol     string  `json:"sym"`
	Exchange   int     `json:"-"`
	Price      float64 `json:"p"`
	Size       int64   `json:"s"`
	Timestamp  int64   `json:"t"`
	Conditions []int   `json:"c"`
}

// Trade is the structure that the PolyTrade is
// coerced to before being stored in the cache
type Trade struct {
	Timestamp time.Time `json:"t" msgpack:"t"`
	Price     float64   `json:"p" msgpack:"p"`
	Size      int64     `json:"s" msgpack:"s"`
}
