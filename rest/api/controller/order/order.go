package order

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/parameter"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/price"
	"github.com/gofrs/uuid"
	"github.com/kataras/iris"
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

func List(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	oReq := OrdersRequest{}
	if err := oReq.Parse(ctx.Request()); err != nil {
		ctx.RespondError(err)
		return
	}

	srv := ctx.Services().Order().WithTx(ctx.Tx())

	orders, err := srv.List(
		accountID,
		enum.OrderStatusFromJSON(oReq.Status),
		oReq.Until,
		oReq.Limit,
		oReq.After,
		oReq.Direction == "asc",
	)

	if err != nil {
		ctx.RespondError(err)
		return
	}

	entities := make([]*OrderEntity, len(orders))
	for i, o := range orders {
		entities[i] = OrderToEntity(&o, ctx.Services().AssetCache().Get(o.AssetID))
	}
	ctx.Respond(entities)
}

func Get(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}
	orderID, err := uuid.FromString(ctx.Params().Get("order_id"))
	if err != nil {
		ctx.RespondError(gberrors.InvalidRequestParam.WithMsg("order_id is missing"))
		return
	}

	srv := ctx.Services().Order().WithTx(ctx.Tx())

	order, err := srv.GetByID(accountID, orderID)

	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(OrderToEntity(order, ctx.Services().AssetCache().Get(order.AssetID)))
	}
}

func GetByClientOrderID(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}
	clientOrderID := ctx.URLParam("client_order_id")
	if clientOrderID == "" {
		ctx.RespondError(gberrors.InvalidRequestParam.WithMsg("client_order_id is missing"))
		return
	}

	srv := ctx.Services().Order().WithTx(ctx.Tx())

	order, err := srv.GetByClientOrderID(accountID, clientOrderID)

	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(OrderToEntity(order, ctx.Services().AssetCache().Get(order.AssetID)))
	}
}

func Delete(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	if !queueOrders() &&
		!accountQueueable(accountID.String()) &&
		!calendar.IsMarketOpen(clock.Now()) {

		ctx.RespondError(
			gberrors.Forbidden.WithMsg("market is closed"))
		return
	}

	srv := ctx.Services().Order().WithTx(ctx.Tx())

	orderID, err := uuid.FromString(ctx.Params().Get("order_id"))
	if err != nil {
		ctx.RespondError(gberrors.InvalidRequestParam.WithMsg("order_id is missing"))
		return
	}

	if err := srv.Cancel(accountID, orderID); err != nil {
		ctx.RespondError(err)
	} else {
		ctx.RespondWithStatus(nil, iris.StatusNoContent)
	}
}

type CreateOrderRequest struct {
	AccountID     string           `json:"-"`
	AssetKey      *string          `json:"symbol"`
	Qty           decimal.Decimal  `json:"qty"`
	Side          enum.Side        `json:"side"`
	Type          enum.OrderType   `json:"type"`
	TimeInForce   enum.TimeInForce `json:"time_in_force"`
	LimitPrice    *decimal.Decimal `json:"limit_price"`
	StopPrice     *decimal.Decimal `json:"stop_price"`
	ClientOrderID string           `json:"client_order_id"`
}

func (req *CreateOrderRequest) ToOrder(asset *models.Asset) *models.Order {
	o := &models.Order{
		AssetID:       asset.ID,
		Qty:           req.Qty,
		Side:          req.Side,
		Type:          req.Type,
		TimeInForce:   req.TimeInForce,
		ClientOrderID: req.ClientOrderID,
		OrderCapacity: enum.Agency,
	}

	if req.LimitPrice != nil {
		px, _ := price.FormatForOrder(*req.LimitPrice)
		o.LimitPrice = &px
	}

	if req.StopPrice != nil {
		px, _ := price.FormatForOrder(*req.StopPrice)
		o.StopPrice = &px
	}

	o.SetSymbol(asset.Symbol)

	return o
}

func (req *CreateOrderRequest) verify() error {
	if req.AssetKey == nil || *req.AssetKey == "" {
		return gberrors.InvalidRequestParam.WithMsg("symbol is required.")
	}

	if req.LimitPrice != nil && req.LimitPrice.LessThanOrEqual(decimal.Zero) {
		return gberrors.InvalidRequestParam.WithMsg("limit price must be > 0")
	}

	if req.StopPrice != nil && req.StopPrice.LessThanOrEqual(decimal.Zero) {
		return gberrors.InvalidRequestParam.WithMsg("stop price must be > 0")
	}

	if req.Qty.LessThanOrEqual(decimal.Zero) {
		return gberrors.InvalidRequestParam.WithMsg("qty must be > 0")
	}

	if !req.Qty.Sub(req.Qty.Floor()).Equals(decimal.Zero) {
		return gberrors.InvalidRequestParam.WithMsg("qty must be integer")
	}

	if !enum.ValidOrderType(req.Type) {
		return gberrors.InvalidRequestParam.WithMsg("invalid order type")
	}

	if !enum.ValidSide(req.Side) {
		return gberrors.InvalidRequestParam.WithMsg("invalid side")
	}

	if !enum.ValidTimeInForce(req.TimeInForce) {
		return gberrors.InvalidRequestParam.WithMsg("invalid time_in_force")
	}

	return nil
}

func Create(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	orderRequest := &CreateOrderRequest{}
	if err = ctx.Read(orderRequest); err != nil {
		ctx.RespondError(gberrors.RequestBodyLoadFailure)
		return
	}

	// Is a new order prohibited by user request?
	// For now, do it strightforward... optmize it using cache for later.
	acct, err := ctx.Services().Account().WithTx(db.DB()).GetByID(accountID)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	if acct.TradeSuspendedByUser {
		ctx.RespondError(
			gberrors.Forbidden.WithMsg("new orders are rejected by user request"))
		return
	}

	if err = orderRequest.verify(); err != nil {
		ctx.RespondError(err)
		return
	}

	asset := ctx.Services().AssetCache().Get(*orderRequest.AssetKey)
	if asset == nil {
		ctx.RespondError(gberrors.InvalidRequestParam.WithMsg(
			fmt.Sprintf("could not find asset \"%s\"", *orderRequest.AssetKey)))
		return
	}

	// only process the order if this is an active, tradable asset
	if !asset.Tradable || asset.Status != enum.AssetActive {
		ctx.RespondError(
			gberrors.InvalidRequestParam.WithMsg(
				fmt.Sprintf("asset %v is not tradable", asset.Symbol)))
		return
	}

	if !queueOrders() &&
		!accountQueueable(accountID.String()) &&
		!calendar.IsMarketOpen(clock.Now()) {

		ctx.RespondError(
			gberrors.Forbidden.WithMsg("market is closed"))
		return
	}

	if len(orderRequest.ClientOrderID) > 50 {
		ctx.RespondError(gberrors.InvalidRequestParam.WithMsg(
			"client_order_id must be no more than 50 characters"))
		return
	}

	orderRequest.AccountID = accountID.String()

	// Auto fill client order id if it is not set up by the client
	if orderRequest.ClientOrderID == "" {
		clientOrderID, err := uuid.NewV4()
		if err != nil {
			ctx.RespondError(gberrors.InternalServerError.WithError(err))
			return
		}
		orderRequest.ClientOrderID = clientOrderID.String()
	}

	// need to use db.DB instead of ctx.Tx(), because it requires more than 2 tx.
	srv := ctx.Services().Order().WithTx(db.DB())

	order, err := srv.Create(accountID, orderRequest.ToOrder(asset))
	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(OrderToEntity(order, ctx.Services().AssetCache().Get(order.AssetID)))
	}
}

var (
	once  sync.Once
	queue bool
)

func queueOrders() bool {
	once.Do(func() {
		queue, _ = strconv.ParseBool(env.GetVar("QUEUE_ORDERS"))
	})

	return queue
}

// Returns true if the account is allowed for
// order queueing, or if no list of queueable accounts,
// then always returns true
func accountQueueable(accountID string) bool {
	accts := strings.Split(env.GetVar("QUEUEABLE_ACCOUNTS"), ",")

	if len(accts) == 0 {
		return true
	}

	for _, id := range accts {
		if strings.EqualFold(id, accountID) {
			return true
		}
	}

	return false
}
