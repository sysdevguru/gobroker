package api

import (
	"bytes"
	"encoding/json"
	"sync/atomic"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/service/registry"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/kataras/iris"
	irisCtx "github.com/kataras/iris/context"
	"github.com/vmihailenco/msgpack"
)

// MIME types
const (
	charsetUTF8 = "charset=utf-8"
)
const (
	MIMEApplicationJSON               = "application/json"
	MIMEApplicationJSONCharsetUTF8    = MIMEApplicationJSON + "; " + charsetUTF8
	MIMEApplicationXML                = "application/xml"
	MIMEApplicationXMLCharsetUTF8     = MIMEApplicationXML + "; " + charsetUTF8
	MIMETextXML                       = "text/xml"
	MIMETextXMLCharsetUTF8            = MIMETextXML + "; " + charsetUTF8
	MIMEApplicationProtobuf           = "application/protobuf"
	MIMEApplicationMsgpack            = "application/msgpack"
	MIMEApplicationMsgpackCharsetUTF8 = "application/msgpack" + "; " + charsetUTF8
	MIMETextPlain                     = "text/plain"
	MIMETextPlainCharsetUTF8          = MIMETextPlain + "; " + charsetUTF8
	MIMEApplicationPDF                = "application/pdf"
)

type Permission string

var (
	PermissionAll     Permission = "All"
	PermissionTrading Permission = "Trading"
	PermissionAdmin   Permission = "Admin"
)

type Session struct {
	ID         uuid.UUID
	Permission Permission
}

func (s *Session) Authorized(id uuid.UUID) bool {
	return bytes.Compare(s.ID.Bytes(), id.Bytes()) == 0
}

type Context interface {
	iris.Context
	Authorize(id uuid.UUID, perm Permission)
	Session() *Session
	Services() registry.Registry
	Commit() error
	Rollback()
	Tx() *gorm.DB
	RepeatableTx() *gorm.DB
	Respond(interface{})
	RespondWithStatus(interface{}, int)
	RespondWithContent(string, interface{})
	RespondError(error)
	Read(interface{}) error
}

type context struct {
	iris.Context
	session  *Session
	services registry.Registry
	tx       *gorm.DB
	txClosed atomic.Value
}

func (ctx *context) Authorize(id uuid.UUID, perm Permission) {
	ctx.session = &Session{
		ID:         id,
		Permission: perm,
	}
}

func (ctx *context) Services() registry.Registry {
	return ctx.services
}

func (ctx *context) Session() *Session {
	return ctx.session
}

func (ctx *context) Commit() error {
	if !ctx.TxClosed() && ctx.tx != nil {
		ctx.txClosed.Store(true)
		log.Debug("api tx committed", "path", ctx.RequestPath(false))
		err := ctx.tx.Commit().Error
		ctx.tx = nil
		return err
	}
	return nil
}

func (ctx *context) Rollback() {
	if !ctx.TxClosed() && ctx.tx != nil {
		ctx.txClosed.Store(true)
		log.Debug("api tx rolled back", "path", ctx.RequestPath(false))
		// if !db.IsConnectionError(ctx.tx.Error) && !db.InsufficientResources(ctx.tx.Error) {
		if !db.IsConnectionError(ctx.tx.Error) {
			ctx.tx.Rollback()
		}
		ctx.tx = nil
	}
}

func (ctx *context) TxClosed() bool {
	if v := ctx.txClosed.Load(); v != nil && v.(bool) {
		return true
	}
	return false
}

func (ctx *context) Tx() *gorm.DB {
	if ctx.tx == nil || ctx.TxClosed() {
		log.Debug("api tx opened", "path", ctx.RequestPath(false))
		ctx.tx = db.Begin()

		if ctx.tx.Error != nil && db.IsConnectionError(ctx.tx.Error) {
			// This is mainly for the case when long idle connection got
			// killed at tcp level by in-between router/switch. Worth retrying.
			// (although not sure if lib/pq sets this code ever)
			//
			// if we can't reconnect, let's not hold up the show - panic!
			if err := db.Reconnect(); err != nil {
				log.Panic("unable to connect to database", "error", err)
			}

			// we reconnected, and begin still fails - panic!
			if ctx.tx = db.Begin(); ctx.tx.Error != nil {
				log.Panic("unable to begin database transaction", "error", ctx.tx.Error)
			}
		} else if ctx.tx.Error != nil {
			// If otherwise BEGIN fails, that sounds like either programming
			// error (e.g. transaction is not finished -- how?) or other
			// connection errors (e.g. "too many clients already"). We should
			// probably still try error out cleanly, but the current code structure
			// doesn't expect error from Tx() unfortunately. We will have to
			// revisit this if this becomes the issue again.
			err := ctx.tx.Error
			ctx.tx = nil
			log.Panic("unrecoverable BEGIN failure", "error", err)
		}
		ctx.txClosed.Store(false)
	}

	return ctx.tx
}

func (ctx *context) RepeatableTx() *gorm.DB {
	return ctx.Tx().Exec("SET TRANSACTION ISOLATION LEVEL REPEATABLE READ")
}

func (ctx *context) Respond(body interface{}) {
	ctx.RespondWithStatus(body, iris.StatusOK)
}

func (ctx *context) RespondWithStatus(body interface{}, statusCode int) {
	ctx.StatusCode(statusCode)
	ctx.RespondWithContent(MIMEApplicationJSON, body)
}

func (ctx *context) RespondWithContent(contentType string, body interface{}) {
	if err := ctx.Commit(); err != nil {
		ctx.RespondError(err)
		return
	}

	ctx.ContentType(contentType)

	if body != nil {
		switch b := body.(type) {
		case []byte:
			ctx.Write(b)
		default:
			ctx.FormatResponse(body)
		}
	}
}

var masks = []string{
	"password",
	"ssn",
	"token",
}

func (ctx *context) RespondError(err error) {
	ctx.Rollback()

	// Need error logging here
	if gberr, ok := err.(gberrors.IException); ok {
		ctx.StatusCode(gberr.ExceptionStatusCode())
		body := gberr.ExceptionBody()
		if !utils.Prod() {
			if gberr.RawException() != nil {
				body["debug"] = gberr.RawException().Error()
			}
		}
		ctx.FormatResponse(body)
	} else {
		ctx.StatusCode(gberrors.InternalServerError.ExceptionStatusCode())
		ctx.FormatResponse(gberrors.InternalServerError.ExceptionBody())
	}

	// We'll track only status_code = 500 errors in detail for further investigation.
	if ctx.GetStatusCode() != 500 {
		return
	}

	var reqBody string
	parsing := map[string]interface{}{}
	if err := ctx.Read(&parsing); err == nil {
		// We need to mask credential fields not to be logged for security purpose.
		for i := range masks {
			if _, ok := parsing[masks[i]]; ok {
				parsing[masks[i]] = "xxx"
			}
		}
		// fluent logger couldn't take nested map, so marshal as json to put it in.
		reqBin, _ := json.Marshal(parsing)
		reqBody = string(reqBin)
	}

	log.Error(
		"http exception",
		"method", ctx.Request().Method,
		"url", ctx.Request().URL.String(),
		"error", gberrors.Format(err),
		"body", reqBody,
		"key_id", ctx.Request().Header.Get("APCA-API-KEY-ID"),
	)
}

func (ctx *context) Read(v interface{}) error {
	contentType := ctx.Request().Header.Get("Content-Type")
	var err error

	if v != nil {
		switch contentType {
		case MIMEApplicationMsgpack:
			err = ctx.UnmarshalBody(v, irisCtx.UnmarshalerFunc(func(data []byte, outPtr interface{}) error {
				dec := msgpack.NewDecoder(bytes.NewReader(data))
				// Using json tags on structs
				dec.UseJSONTag(true)
				return dec.Decode(&outPtr)
			}))

		default:
			err = ctx.ReadJSON(v)
		}
	}

	return err
}

// FormatResponse will format a reponse based on request Content-Type header
func (ctx *context) FormatResponse(body interface{}) {
	contentType := ctx.Request().Header.Get("Content-Type")
	ctx.ContentType(contentType)

	if body != nil {
		switch contentType {
		case MIMEApplicationMsgpack:
			var b bytes.Buffer
			enc := msgpack.NewEncoder(&b)
			// Using json tags on structs
			enc.UseJSONTag(true)
			err := enc.Encode(body)
			if err != nil {
				log.Panic("Failed to marshal response body (msgpack)", "error", err)
			}

			_, writeErr := ctx.Write(b.Bytes())
			if writeErr != nil {
				log.Panic("Failed to write response body (msgpack)", "error", writeErr)
			}
		case MIMEApplicationJSON, MIMEApplicationJSONCharsetUTF8:
			ctx.JSON(body)
		default:
			ctx.ContentType(MIMEApplicationJSON)
			ctx.JSON(body)
		}
	}
}
