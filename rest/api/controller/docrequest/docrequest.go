package docrequest

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/parameter"
	"github.com/alpacahq/gobroker/service/docrequest"
	"github.com/alpacahq/gopaca/bytes"
	"github.com/kataras/iris"
	"github.com/pkg/errors"
)

func List(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	srv := docrequest.Service().WithTx(ctx.Tx())

	docReqs, err := srv.ListLast(accountID, apex.Client())
	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(docReqs)
	}
}

func Post(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	if err := ctx.Request().ParseMultipartForm(int64(5 * bytes.MB.Bytes())); err != nil {
		ctx.RespondError(gberrors.InvalidRequestParam.WithMsg("upload file size need to be less than 5MB").WithError(err))
		return
	}
	file, _, err := ctx.Request().FormFile("file")

	if err != nil {
		ctx.RespondError(errors.Wrap(err, "failed to get form file"))
		return
	}
	buf, err := ioutil.ReadAll(file)
	if err != nil {
		ctx.RespondError(errors.Wrap(err, "failed to read form file"))
		return
	}

	mimeType := http.DetectContentType(buf)

	if !strings.HasPrefix(mimeType, "image/") {
		ctx.RespondError(gberrors.InvalidRequestParam.WithMsg("upload file need to be image file."))
		return
	}

	docType, err := models.NewDocumentType(ctx.Request().FormValue("document_type"))
	if err != nil {
		ctx.RespondError(err)
		return
	}

	docSubType, err := models.NewDocumentSubType(ctx.Request().FormValue("document_sub_type"))
	if err != nil {
		ctx.RespondError(err)
		return
	}

	// Explicit read lock for document_requests table row with read-committed.
	srv := docrequest.Service().WithTx(ctx.Tx())

	if err = srv.Upload(
		accountID,
		buf, *docType,
		docSubType, mimeType,
		apex.Client()); err != nil {
		ctx.RespondError(err)
	} else {
		ctx.RespondWithStatus(nil, iris.StatusNoContent)
	}
}
