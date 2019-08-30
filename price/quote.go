package price

import (
	"fmt"
	"time"

	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/polycache/rest/client"
)

type Quote struct {
	BidTimestamp  time.Time `json:"bid_timestamp"`
	Bid           float32   `json:"bid"`
	AskTimestamp  time.Time `json:"ask_timestamp"`
	Ask           float32   `json:"ask"`
	LastTimestamp time.Time `json:"last_timestamp"`
	Last          float32   `json:"last"`
}

// Quotes returns a list of quotes corresponding to the
// provided symbol list by using the polycache client.
func Quotes(symbols []string) ([]Quote, error) {
	tradesResp, err := client.GetTrades(symbols)
	if err != nil {
		return nil, err
	}

	quotesResp, err := client.GetQuotes(symbols)
	if err != nil {
		return nil, err
	}

	if len(quotesResp) != len(tradesResp) {
		return nil, fmt.Errorf("quotes unavailable for this set of symbols")
	}

	quotes := make([]Quote, len(symbols))

	for i, symbol := range symbols {
		trade, quote := tradesResp[symbol], quotesResp[symbol]

		quotes[i] = Quote{
			BidTimestamp:  quote.Timestamp.In(calendar.NY),
			Bid:           float32(quote.BidPrice),
			AskTimestamp:  quote.Timestamp.In(calendar.NY),
			Ask:           float32(quote.AskPrice),
			LastTimestamp: trade.Timestamp.In(calendar.NY),
			Last:          float32(trade.Price),
		}
	}

	return quotes, nil
}
