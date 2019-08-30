package entities

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/rest/api/controller/parameter"
	"github.com/shopspring/decimal"
)

type OrdersRequest struct {
	Status    string     `url:"status"`
	Until     *time.Time `url:"until"`
	Limit     *int       `url:"limit"`
	After     *time.Time `url:"after"`
	Direction string     `url:"direction"`
}

// OrderEntity is the schema for orders in the API responses.
// It is a basically subset of models.Order and this is only
// for the API output.
type OrderEntity struct {
	ID             string           `json:"id"`
	ClientOrderID  string           `json:"client_order_id"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
	SubmittedAt    time.Time        `json:"submitted_at"`
	FilledAt       *time.Time       `json:"filled_at"`
	ExpiredAt      *time.Time       `json:"expired_at"`
	CanceledAt     *time.Time       `json:"canceled_at"`
	FailedAt       *time.Time       `json:"failed_at"`
	AssetID        string           `json:"asset_id"`
	Symbol         string           `json:"symbol"`
	Class          enum.AssetClass  `json:"asset_class"`
	Qty            decimal.Decimal  `json:"qty"`
	FilledQty      decimal.Decimal  `json:"filled_qty"`
	FilledAvgPrice *decimal.Decimal `json:"filled_avg_price"`
	// TODO: remove this field, only for compatibility
	OrderType   enum.OrderType   `json:"order_type"`
	Type        enum.OrderType   `json:"type"`
	Side        enum.Side        `json:"side"`
	TimeInForce enum.TimeInForce `json:"time_in_force"`
	LimitPrice  *decimal.Decimal `json:"limit_price"`
	StopPrice   *decimal.Decimal `json:"stop_price"`
	Status      string           `json:"status"`
}

func OrderToEntity(o *models.Order, asset *models.Asset) *OrderEntity {

	filledQty := decimal.Zero
	if o.FilledQty != nil {
		filledQty = *o.FilledQty
	}

	return &OrderEntity{
		ID:             o.ID,
		ClientOrderID:  o.ClientOrderID,
		CreatedAt:      o.CreatedAt,
		UpdatedAt:      o.UpdatedAt,
		SubmittedAt:    o.SubmittedAt,
		FilledAt:       o.FilledAt,
		ExpiredAt:      o.ExpiredAt,
		CanceledAt:     o.CanceledAt,
		FailedAt:       o.FailedAt,
		AssetID:        o.AssetID,
		Symbol:         o.GetSymbol(),
		Class:          asset.Class,
		Qty:            o.Qty,
		FilledQty:      filledQty,
		FilledAvgPrice: o.FilledAvgPrice,
		OrderType:      o.ClientOrderType,
		Type:           o.ClientOrderType,
		Side:           o.Side,
		TimeInForce:    o.TimeInForce,
		LimitPrice:     o.LimitForJSON(),
		StopPrice:      o.StopPrice,
		Status:         string(o.Status),
	}
}

func (o *OrdersRequest) Parse(r *http.Request) error {
	params := r.URL.Query()
	status := params.Get("status")
	if status == "" {
		o.Status = "open"
	} else {
		o.Status = status
	}

	if params.Get("until") != "" {
		until, err := parameter.ParseTimestamp(params.Get("until"), "until")
		if err != nil {
			return err
		}
		o.Until = until
	}

	if params.Get("after") != "" {
		after, err := parameter.ParseTimestamp(params.Get("after"), "after")
		if err != nil {
			return err
		}
		o.After = after
	}

	var limit int
	if params.Get("limit") == "" {
		limit = 50
	} else {
		l, _ := strconv.ParseInt(params.Get("limit"), 10, 32)
		limit = int(l)
		if limit > 500 {
			limit = 500
		} else if limit < 0 {
			limit = 50
		}
	}

	o.Direction = "desc"
	dir := strings.ToLower(params.Get("direction"))
	if dir != "" {
		if dir != "asc" && dir != "desc" {
			return gberrors.InvalidRequestParam.WithMsg("direction can only be \"asc\" or \"desc\"")
		}
		o.Direction = dir
	}

	o.Limit = &limit
	return nil
}
