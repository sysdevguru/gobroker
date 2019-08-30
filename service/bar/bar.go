package bar

import (
	"fmt"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/external/mkts"
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/assetcache"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/marketstore/frontend"
	"github.com/alpacahq/marketstore/utils"
	"github.com/alpacahq/marketstore/utils/io"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
)

type BarService interface {
	GetByID(assetID uuid.UUID, timeframe string, start, end *time.Time, limit *int) (*AssetBars, error)
	GetByIDs(assetIDs []uuid.UUID, timeframe string, start, end *time.Time, limit *int) ([]*AssetBars, error)
	WithTx(tx *gorm.DB) BarService
}

type barService struct {
	BarService
	tx         *gorm.DB
	mktsdb     func(name string, args interface{}) (io.ColumnSeriesMap, error)
	now        func() time.Time
	assetcache assetcache.AssetCache
}

func Service(assetcache assetcache.AssetCache) BarService {
	return &barService{
		mktsdb: func(name string, args interface{}) (io.ColumnSeriesMap, error) {
			resp, err := mkts.Client().DoRPC(name, args)
			if err != nil {
				return nil, err
			}

			if resp == nil {
				return io.NewColumnSeriesMap(), nil
			}

			return *resp.(*io.ColumnSeriesMap), nil
		},
		now:        clock.Now,
		assetcache: assetcache,
	}
}

func (s *barService) WithTx(tx *gorm.DB) BarService {
	s.tx = tx
	return s
}

type AssetBars struct {
	AssetID  uuid.UUID       `json:"asset_id"`
	Symbol   string          `json:"symbol"`
	Exchange string          `json:"exchange"`
	Class    enum.AssetClass `json:"asset_class"`
	Bars     Bars            `json:"bars"`
}

type Bar struct {
	Open   float32   `json:"open"`
	High   float32   `json:"high"`
	Low    float32   `json:"low"`
	Close  float32   `json:"close"`
	Volume int32     `json:"volume"`
	Time   time.Time `json:"time"`
}

type Bars []*Bar

// GetByID provide candles with common OHLCV format.
func (s *barService) GetByID(assetID uuid.UUID, timeframe string, start, end *time.Time, limit *int) (*AssetBars, error) {
	tf := utils.TimeframeFromString(timeframe)
	if tf == nil {
		return nil, gberrors.InvalidRequestParam.WithMsg("invalid timeframe")
	}

	asset := s.assetcache.GetByID(assetID)

	if asset == nil {
		return nil, gberrors.NotFound.WithMsg(fmt.Sprintf("asset not found for %v", assetID))
	}

	args := &frontend.MultiQueryRequest{
		Requests: buildQuery([]string{asset.Symbol}, tf, start, end, limit),
	}

	csm, err := s.mktsdb("Query", args)

	if err != nil {
		return nil, gberrors.InternalServerError.WithError(err)
	}

	if csm.IsEmpty() {
		return &AssetBars{
			AssetID:  asset.IDAsUUID(),
			Symbol:   asset.Symbol,
			Exchange: asset.Exchange,
			Class:    asset.Class,
			Bars:     Bars{},
		}, nil
	}
	return s.csmToBars(csm, tf, map[string]*models.Asset{asset.Symbol: asset})[0], nil
}

const barAggThreshold = 30 * time.Second

func (s *barService) GetByIDs(assetIDs []uuid.UUID, timeframe string, start, end *time.Time, limit *int) ([]*AssetBars, error) {
	assets := []*models.Asset{}

	tf := utils.TimeframeFromString(timeframe)
	if tf == nil {
		return nil, gberrors.InvalidRequestParam.WithMsg("invalid timeframe")
	}

	q := s.tx.Where("id IN (?)", assetIDs).Find(&assets)

	if len(assets) == 0 {
		return []*AssetBars{}, nil
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	symbols := make([]string, len(assets))
	// To load asset from the query result
	assetMap := map[string]*models.Asset{}
	for i, asset := range assets {
		symbols[i] = asset.Symbol
		assetMap[asset.Symbol] = asset
	}

	args := &frontend.MultiQueryRequest{
		Requests: buildQuery(symbols, tf, start, end, limit),
	}

	csm, err := s.mktsdb("Query", args)

	if err != nil {
		return nil, gberrors.InternalServerError.WithError(err)
	}

	return s.csmToBars(csm, tf, assetMap), nil
}

func buildQuery(symbols []string, timeframe *utils.Timeframe, start, end *time.Time, limit *int) []frontend.QueryRequest {
	builder := mkts.NewQueryRequestBuilder(fmt.Sprintf("%v/%v/OHLCV", strings.Join(symbols, ","), timeframe))

	if start != nil {
		builder.EpochStart(start.Unix())
	}

	if end != nil {
		builder.EpochEnd(end.Unix())
	}

	if limit != nil {
		builder.LimitRecordCount(*limit)
	}

	// if it's an aggregated timeframe, let's get the latest 1Min bar too
	if timeframe.Duration > time.Minute {
		oneLimit := 1
		return append(
			buildQuery(
				symbols,
				utils.TimeframeFromString("1Min"),
				nil, end, &oneLimit,
			),
			builder.End(),
		)
	}

	return []frontend.QueryRequest{builder.End()}
}

func (s *barService) csmToBars(csm io.ColumnSeriesMap, tf *utils.Timeframe, assetMap map[string]*models.Asset) []*AssetBars {
	assetBarsList := []*AssetBars{}

	for tbk, cs := range csm {
		// don't process the extra 1Min bars
		if tbkTf, _ := tbk.GetTimeFrame(); tbkTf.Duration != tf.Duration {
			continue
		}
		symbol := tbk.GetItems()[0]
		asset := assetMap[symbol]

		// when Epoch is nil, response from marketstore is blank so we'll return empty bars.
		var bars []*Bar
		if cs.GetByName("Epoch") == nil {
			bars = Bars{}
		} else {
			t := cs.GetTime()
			o := cs.GetByName("Open").([]float32)
			h := cs.GetByName("High").([]float32)
			l := cs.GetByName("Low").([]float32)
			c := cs.GetByName("Close").([]float32)
			v := cs.GetByName("Volume").([]int32)

			idx := trimAt(csm, tbk, tf, s.now())
			bars = make(Bars, idx)

			for i := range bars {
				bars[i] = &Bar{
					Open:   o[i],
					High:   h[i],
					Low:    l[i],
					Close:  c[i],
					Volume: v[i],
					Time:   t[i],
				}
			}
		}

		assetBars := &AssetBars{
			AssetID:  asset.IDAsUUID(),
			Symbol:   asset.Symbol,
			Exchange: asset.Exchange,
			Class:    asset.Class,
			Bars:     bars,
		}
		assetBarsList = append(assetBarsList, assetBars)
	}
	return assetBarsList
}

func trimAt(csm io.ColumnSeriesMap, tbk io.TimeBucketKey, tf *utils.Timeframe, now time.Time) (index int) {
	cs := csm[tbk]
	t := cs.GetTime()

	if len(t) == 0 {
		return
	}

	index = len(t)

	if tf.Duration <= time.Minute {
		return
	}

	symbol := tbk.GetItems()[0]
	endTime := t[len(t)-1].Add(tf.Duration - time.Minute).In(calendar.NY)
	minTbk := io.NewTimeBucketKey(fmt.Sprintf("%v/1Min/OHLCV", symbol), tbk.GetCatKey())

	if endTime.Unix() >= csm[*minTbk].GetEpoch()[0] &&
		now.Sub(endTime.Add(time.Minute)) < barAggThreshold &&
		calendar.IsMarketOpen(now) {
		index = len(t) - 1
	}
	return
}
