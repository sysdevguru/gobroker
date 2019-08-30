package structures

import "time"

// PolyQuote is the reference structure sent
// by polygon for quote data
type PolyQuote struct {
	Symbol      string  `json:"sym"`
	Condition   int     `json:"-"`
	BidExchange int     `json:"-"`
	AskExchange int     `json:"-"`
	BidPrice    float64 `json:"bp"`
	AskPrice    float64 `json:"ap"`
	BidSize     int64   `json:"bs"`
	AskSize     int64   `json:"as"`
	Timestamp   int64   `json:"t"`
}

// Quote is the structure that the PolyQuote is
// coerced to before being stored in the cache
type Quote struct {
	Timestamp time.Time `json:"t" msgpack:"t"`
	AskPrice  float64   `json:"ap" msgpack:"ap"`
	AskSize   int64     `json:"as" msgpack:"as"`
	BidPrice  float64   `json:"bp" msgpack:"bp"`
	BidSize   int64     `json:"bs" msgpack:"bs"`
}
