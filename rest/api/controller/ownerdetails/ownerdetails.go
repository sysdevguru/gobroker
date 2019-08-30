package ownerdetails

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/parameter"
	"github.com/alpacahq/gobroker/service/ownerdetails"
	"github.com/alpacahq/gobroker/utils/address"
	"github.com/kataras/iris"
)

func Get(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	srv := ownerdetails.Service().WithTx(ctx.Tx())

	od, err := srv.GetPrimaryByAccountID(accountID)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	if od.DateOfBirth != nil {
		od.DateOfBirth = od.DateOfBirthString()
	}

	ctx.Respond(od)
}

func Patch(ctx api.Context) {

	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	patches, err := getPatches(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	srv := ownerdetails.Service().WithTx(ctx.Tx())

	_, err = srv.Patch(accountID, patches)
	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.RespondWithStatus(nil, iris.StatusNoContent)
	}
}

func getPatches(ctx api.Context) (map[string]interface{}, error) {
	updates := map[string]interface{}{}

	body, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		return updates, gberrors.RequestBodyLoadFailure.WithError(err)
	}

	if err = verifyAccountDetails(body); err != nil {
		return updates, gberrors.InvalidRequestParam.WithMsg(err.Error())
	}

	if err = json.Unmarshal(body, &updates); err != nil {
		return updates, gberrors.RequestBodyLoadFailure.WithError(err)
	}

	// trim trailing whitespace
	for key, update := range updates {
		switch update.(type) {
		case string:
			updates[key] = strings.TrimSpace(update.(string))
		}
	}

	addr, addrExists := updates["street_address"]
	if addrExists {
		apexAddr, err := address.HandleApiAddress(addr)
		if err != nil {
			return nil, gberrors.RequestBodyLoadFailure.WithError(err)
		}
		updates["street_address"] = apexAddr
	}

	return updates, nil
}

func verifyAccountDetails(body []byte) error {
	od := models.OwnerDetails{}
	if err := json.Unmarshal(body, &od); err != nil {
		return err
	}
	return od.Validate()
}
