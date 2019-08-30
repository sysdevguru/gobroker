package asset

import (
	"encoding/json"
	"time"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/entities"
)

type AssetMarshaller struct {
	models.Asset
}

func (m *AssetMarshaller) MarshalJSON() ([]byte, error) {
	a := map[string]interface{}{
		"id":         m.ID,
		"created_at": m.CreatedAt.Format(time.RFC3339),
		"updated_at": m.UpdatedAt.Format(time.RFC3339),
		"class":      m.Class,
		"exchange":   m.Exchange,
		"symbol":     m.Symbol,
		"cusip":      m.CUSIP,
		"status":     m.Status,
		"tradable":   m.Tradable,
	}
	return json.Marshal(a)
}

func (m *AssetMarshaller) UnmarshalJSON(data []byte) error {
	a := map[string]interface{}{}
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	m.ID = a["id"].(string)
	createdAt, err := time.Parse(time.RFC3339, a["created_at"].(string))
	if err != nil {
		return err
	}
	m.CreatedAt = createdAt

	updatedAt, err := time.Parse(time.RFC3339, a["updated_at"].(string))
	if err != nil {
		return err
	}
	m.UpdatedAt = updatedAt
	m.Class = enum.AssetClass(a["class"].(string))
	m.Exchange = a["exchange"].(string)
	m.Symbol = a["symbol"].(string)
	m.CUSIP = a["cusip"].(string)
	m.Status = enum.AssetStatus(a["status"].(string))
	m.Tradable = a["tradable"].(bool)
	return nil
}

func List(ctx api.Context) {
	var (
		status *enum.AssetStatus
		class  *enum.AssetClass
	)

	if q := ctx.URLParam("status"); q != "" {
		s := enum.AssetStatus(q)
		status = &s
	}

	if q := ctx.URLParam("asset_class"); q != "" {
		s := enum.AssetClass(q)
		class = &s
	}

	srv := ctx.Services().Asset().WithTx(ctx.Tx())

	assets, err := srv.List(class, status)

	if err != nil {
		ctx.RespondError(err)
		return
	}

	marshallers := make([]entities.AssetMarshaller, len(assets))
	for i := range assets {
		marshallers[i] = entities.AssetMarshaller{Asset: *assets[i]}
	}

	ctx.Respond(marshallers)
}
