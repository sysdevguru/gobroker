package relationship

import (
	"fmt"
	"strings"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/external/plaid"
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/entities"
	"github.com/alpacahq/gobroker/rest/api/controller/parameter"
	"github.com/alpacahq/gobroker/service/relationship"
	"github.com/alpacahq/gopaca/encryption"
	"github.com/kataras/iris"
	"github.com/shopspring/decimal"
)

func List(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	srv := relationship.Service().WithTx(ctx.Tx())

	rels, err := srv.List(accountID, nil)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	if len(rels) == 0 {
		ctx.Respond([]map[string]interface{}{})
		return
	}

	relationships := []map[string]interface{}{}
	for _, rel := range rels {
		if rel.PlaidToken != nil {
			relationships = append(relationships, map[string]interface{}{
				"id":              rel.ID,
				"created_at":      rel.CreatedAt,
				"updated_at":      rel.UpdatedAt,
				"deleted_at":      rel.DeletedAt,
				"account_id":      rel.AccountID,
				"status":          rel.Status,
				"bank_account":    rel.Mask,
				"account_name":    rel.Nickname,
				"institution":     rel.PlaidInstitution,
				"failed_attempts": rel.FailedAttempts,
				"reason":          rel.Reason,
			})
		}
	}

	ctx.Respond(relationships)
}

func Create(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	cReq := entities.CreateRelationshipRequest{}
	if err := ctx.Read(&cReq); err != nil {
		ctx.RespondError(gberrors.RequestBodyLoadFailure.WithError(err))
		return
	}

	srv := relationship.Service().WithTx(ctx.Tx())

	var (
		bInfo       relationship.BankAcctInfo
		bankAcct    string
		accountName string
	)

	if cReq.Method == "micro" {
		bankAcct = cReq.AccountNumber
		accountName = cReq.NickName

		bInfo = relationship.BankAcctInfo{
			Account:     accountID.String(),
			Institution: "micro_deposit",
			BankAccount: cReq.AccountNumber,
			Routing:     cReq.RoutingNumber,
			AccountType: strings.ToUpper(cReq.AccountType),
			Nickname:    cReq.NickName,
			Mask:        encryption.Mask(bankAcct, 0, len(bankAcct)-5),
			RelType:     apex.MicroDeposit,
		}
	} else {
		plaidExchange, err := srv.ExchangePlaidToken(cReq.PublicToken)
		if err != nil {
			if apiError, ok := err.(*plaid.APIError); ok && apiError.CanDisplay() {
				ctx.RespondError(gberrors.InternalServerError.WithMsg(*apiError.DisplayMessage).WithError(err))
			} else {
				ctx.RespondError(err)
			}
			return
		}

		item, err := srv.GetPlaidItem(plaidExchange.Token)
		if err != nil {
			if apiError, ok := err.(*plaid.APIError); ok && apiError.CanDisplay() {
				ctx.RespondError(gberrors.InternalServerError.WithMsg(*apiError.DisplayMessage).WithError(err))
			} else {
				ctx.RespondError(err)
			}
			return
		}

		plaidAuth, err := srv.AuthPlaid(plaidExchange.Token)
		if err != nil {
			if apiError, ok := err.(*plaid.APIError); ok && apiError.CanDisplay() {
				ctx.RespondError(gberrors.InternalServerError.WithMsg(*apiError.DisplayMessage).WithError(err))
			} else {
				ctx.RespondError(err)
			}
			return
		}

		var account map[string]interface{}
		for _, entry := range plaidAuth["accounts"].([]interface{}) {
			if acc, ok := entry.(map[string]interface{}); ok {
				if acc["account_id"] == cReq.AccountID {
					account = acc
				}
			}
		}
		var numbers map[string]interface{}
		for _, entry := range plaidAuth["numbers"].([]interface{}) {
			if num, ok := entry.(map[string]interface{}); ok {
				if num["account_id"] == cReq.AccountID {
					numbers = num
				}
			}
		}
		if numbers == nil || account == nil {
			ctx.RespondError(fmt.Errorf("account_id not associated with public_token"))
			return
		}

		bInfo = relationship.BankAcctInfo{
			Token:       plaidExchange.Token,
			Item:        plaidExchange.Item,
			Account:     account["account_id"].(string),
			Institution: item["institution_id"].(string),
			BankAccount: numbers["account"].(string),
			Routing:     numbers["routing"].(string),
			AccountType: strings.ToUpper(account["subtype"].(string)),
			Nickname:    account["name"].(string),
			Mask:        account["mask"].(string),
			RelType:     apex.Plaid,
		}

		bankAcct = numbers["account"].(string)

		// We saw account which does not have official_name.
		accountName = ""
		if account["official_name"] != nil {
			accountName = account["official_name"].(string)
		} else {
			accountName = account["name"].(string)
		}

	}

	rel, err := srv.Create(accountID, bInfo)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	relationship := map[string]interface{}{
		"id":              rel.ID,
		"created_at":      rel.CreatedAt,
		"updated_at":      rel.UpdatedAt,
		"deleted_at":      rel.DeletedAt,
		"account_id":      rel.AccountID,
		"status":          rel.Status,
		"bank_account":    encryption.Mask(bankAcct, 0, len(bankAcct)-5),
		"account_name":    accountName,
		"institution":     rel.PlaidInstitution, // Will be "micro_deposit" if not Plaid
		"failed_attempts": rel.FailedAttempts,
		"reason":          rel.Reason,
	}

	ctx.Respond(relationship)
}

func Delete(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	relID := ctx.Params().Get("relationship_id")
	if relID == "" {
		ctx.RespondError(gberrors.InternalServerError)
		return
	}

	srv := relationship.Service().WithTx(ctx.Tx())

	if err = srv.Cancel(accountID, relID); err != nil {
		ctx.RespondError(err)
	} else {
		ctx.RespondWithStatus(nil, iris.StatusNoContent)
	}
}

type ConfirmRelationshipRequest struct {
	Method    string          `json:"method"`
	AmountOne decimal.Decimal `json:"amount_one"`
	AmountTwo decimal.Decimal `json:"amount_two"`
}

func Approve(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	cReq := ConfirmRelationshipRequest{}
	if err := ctx.Read(&cReq); err != nil {
		ctx.RespondError(gberrors.RequestBodyLoadFailure.WithError(err))
		return
	}

	cReq.Method = "MICRO_DEPOSIT"

	// Grab Relationship ID for the account
	relID := ctx.Params().Get("relationship_id")
	if relID == "" {
		ctx.RespondError(gberrors.NotFound)
		return
	}

	srv := relationship.Service().WithTx(ctx.Tx())

	if rel, err := srv.Approve(accountID, relID, cReq.AmountOne, cReq.AmountTwo); err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(rel)
	}
}

func Reissue(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	// Grab Relationship ID for the account
	relID := ctx.Params().Get("relationship_id")
	if relID == "" {
		ctx.RespondError(gberrors.NotFound)
		return
	}

	srv := relationship.Service().WithTx(ctx.Tx())

	if rel, err := srv.Reissue(accountID, relID); err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(rel)
	}
}
