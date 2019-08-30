package pfhistory

import (
	"fmt"
	"time"

	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/price"
	"github.com/alpacahq/marketstore/utils/io"
	"github.com/alpacahq/polycache/structures"
	"github.com/shopspring/decimal"
)

// Position for PL calculation
type Position interface {
	GetQty() decimal.Decimal
	GetSymbol() string
}

// Order for PL calculation
type Order interface {
	GetQty() decimal.Decimal
	GetPrice() decimal.Decimal
	GetSymbol() string
	GetSide() enum.Side
	GetFilledAt() *time.Time
}

// P struct to hold position to calc PL
type P struct {
	qty       decimal.Decimal
	costBasis decimal.Decimal
	pl        decimal.Decimal
}

func (p *P) computePL(livePrice decimal.Decimal) decimal.Decimal {
	unrealizedPL := livePrice.Sub(p.costBasis).Mul(p.qty)
	return p.pl.Add(unrealizedPL)
}

type PFHistoryResponse struct {
	Close []*decimal.Decimal
	Time  []time.Time
}

type PLReplayer struct {
	pMap           map[string]P
	executions     []Order
	executionIndex int
	pricing        io.ColumnSeriesMap
	pricingIndex   map[string]int
	livePriceMap   map[string]decimal.Decimal
	latestPrices   map[string]decimal.Decimal
	steps          []time.Time
	stepIndex      int
	step           time.Time
	now            time.Time
	isLastStep     bool
}

func NewPLReplayer(
	positions []Position,
	executions []Order,
	csm io.ColumnSeriesMap,
	beginningPrices map[string]structures.Trade,
	latestPrices map[string]structures.Trade,
	begin time.Time,
	end time.Time,
	now time.Time) *PLReplayer {

	pmap := map[string]P{}
	for _, v := range positions {
		pmap[v.GetSymbol()] = P{
			qty:       v.GetQty(),
			costBasis: price.FormatFloat64ForCalc(beginningPrices[v.GetSymbol()].Price),
			pl:        decimal.Zero,
		}
	}

	livePriceMap := map[string]decimal.Decimal{}
	for k, v := range beginningPrices {
		livePriceMap[k] = price.FormatFloat64ForCalc(v.Price)
	}

	latestPriceMap := map[string]decimal.Decimal{}
	for k, v := range latestPrices {
		latestPriceMap[k] = price.FormatFloat64ForCalc(v.Price)
	}

	steps, _ := calendar.NewRange(begin, end, calendar.Min5)

	return &PLReplayer{
		pMap:           pmap,
		executions:     executions,
		executionIndex: -1,
		pricing:        csm,
		pricingIndex:   map[string]int{},
		livePriceMap:   livePriceMap,
		latestPrices:   latestPriceMap,
		steps:          steps,
		stepIndex:      -1,
		now:            now,
		isLastStep:     false,
	}
}

func (r *PLReplayer) next() bool {
	nextIndex := r.stepIndex + 1
	if nextIndex <= len(r.steps)-1 && !r.steps[nextIndex].After(r.now) {
		r.stepIndex = nextIndex
		r.step = r.steps[nextIndex]
		// check the next one and mark it if it is the last step
		nextIndex = r.stepIndex + 1
		r.isLastStep = !(nextIndex <= len(r.steps)-1 && !r.steps[nextIndex].After(r.now))
		return true
	}
	return false
}

func (r *PLReplayer) nextExecution() bool {
	if r.executionIndex >= len(r.executions)-1 {
		return false
	}

	nt := r.step.Add(5 * time.Minute)

	next := r.executions[r.executionIndex+1]

	if next.GetFilledAt().Before(r.step) {
		return false
	}

	if next.GetFilledAt().After(nt) {
		return false
	}

	if next.GetFilledAt().Equal(nt) {
		return false
	}

	r.executionIndex++

	return true
}

func (r *PLReplayer) getExecution() Order {
	return r.executions[r.executionIndex]
}

func (r *PLReplayer) updateLivePrices() {

	for k := range r.livePriceMap {
		// If last step and latestPrice is available, then use the value instead. Used to override the last price with
		// quote.last price, not marketstore price data.
		if r.isLastStep {
			latestPrice, ok := r.latestPrices[k]
			if ok {
				r.livePriceMap[k] = latestPrice
				continue
			}
		}

		tbk := io.NewTimeBucketKey(
			fmt.Sprintf("%v/5Min/OHLCV", k),
			io.DefaultTimeBucketSchema,
		)

		// If ColumnSeries not found, it means there are no trades happend at that period.
		// So can skip update live price for that asset.
		cs, ok := r.pricing[*tbk]
		if !ok || cs.IsEmpty() {
			continue
		}

		cst := cs.GetTime()

		var _pi int
		if v, ok := r.pricingIndex[k]; ok {
			_pi = v
		} else {
			_pi = 0
		}

		// seek to latest price
		for _pi <= len(cst)-1 {
			priceAt := cst[_pi]
			if priceAt.After(r.step) {
				break
			}
			if priceAt.Equal(r.step) {
				csc := cs.GetByName("Close").([]float32)
				currentPrice := decimal.NewFromFloat(float64(csc[_pi]))
				r.livePriceMap[k] = currentPrice
				r.pricingIndex[k] = _pi
				break
			}
			_pi++
		}

	}

}

func (r *PLReplayer) Replay() PFHistoryResponse {

	values := make([]*decimal.Decimal, len(r.steps))

	for r.next() {

		for r.nextExecution() {
			o := r.getExecution()
			switch o.GetSide() {
			case enum.Buy:
				if p, ok := r.pMap[o.GetSymbol()]; ok {
					newquant := p.qty.Add(o.GetQty())
					newap := p.costBasis.Mul(p.qty).Add(o.GetPrice().Mul(o.GetQty())).Div(p.qty.Add(o.GetQty()))
					r.pMap[o.GetSymbol()] = P{
						qty:       newquant,
						costBasis: newap,
						pl:        p.pl,
					}
				} else {
					r.pMap[o.GetSymbol()] = P{
						qty:       o.GetQty(),
						costBasis: o.GetPrice(),
						pl:        decimal.Zero,
					}
				}
			case enum.Sell:
				p := r.pMap[o.GetSymbol()]
				newquant := p.qty.Sub(o.GetQty())
				profit := o.GetPrice().Sub(p.costBasis).Mul(o.GetQty())
				newpl := p.pl.Add(profit)
				var newap decimal.Decimal
				if newquant.Equal(decimal.Zero) {
					newap = decimal.Zero
				} else {
					newap = p.costBasis
				}
				r.pMap[o.GetSymbol()] = P{
					qty:       newquant,
					costBasis: newap,
					pl:        newpl,
				}
			}
		}

		r.updateLivePrices()

		stepPL := decimal.Zero
		for k, p := range r.pMap {
			stepPL = stepPL.Add(p.computePL(r.livePriceMap[k]))
		}

		values[r.stepIndex] = &stepPL
	}

	return PFHistoryResponse{
		Close: values,
		Time:  r.steps,
	}
}

func ComputePL(
	positions []Position,
	orders []Order,
	csm io.ColumnSeriesMap,
	beginningPrices map[string]structures.Trade,
	begin time.Time,
	end time.Time,
	now time.Time) PFHistoryResponse {

	return NewPLReplayer(positions, orders, csm, beginningPrices, map[string]structures.Trade{}, begin, end, now).Replay()
}

func ComputePLWithQuoteOverride(
	positions []Position,
	orders []Order,
	csm io.ColumnSeriesMap,
	beginningPrices map[string]structures.Trade,
	latestPrices map[string]structures.Trade,
	begin time.Time,
	end time.Time,
	now time.Time) PFHistoryResponse {
	return NewPLReplayer(positions, orders, csm, beginningPrices, latestPrices, begin, end, now).Replay()
}
