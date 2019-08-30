package entities

import (
	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/shopspring/decimal"
)

type TransferRequest struct {
	RelationshipID string                 `json:"relationship_id"`
	Direction      apex.TransferDirection `json:"direction"`
	Amount         decimal.Decimal        `json:"amount"`
}

func (r *TransferRequest) Verify() error {
	if r.RelationshipID == "" {
		return gberrors.InvalidRequestParam.WithMsg("relationship_id is required")
	}
	if r.Direction != apex.Outgoing && r.Direction != apex.Incoming {
		return gberrors.InvalidRequestParam.WithMsg("invalid transfer direction")
	}
	if r.Amount.LessThanOrEqual(decimal.Zero) {
		return gberrors.InvalidRequestParam.WithMsg("amount must be > 0")
	}

	return nil
}
