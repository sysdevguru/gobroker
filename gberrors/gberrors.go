package gberrors

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kataras/iris"
)

// IException provides interface for
//   - user facing error message with status code
//   - raw error for tracking them
type IException interface {
	ExceptionBody() map[string]interface{}
	ExceptionStatusCode() int
	RawException() error
}

type Error struct {
	IException
	Code       int
	Message    string
	StatusCode int
	RawError   error
}

func (e *Error) Error() string {
	return fmt.Sprintf("%v (Code = %v)", e.Message, e.Code)
}

func (e *Error) ExceptionBody() map[string]interface{} {
	return map[string]interface{}{"code": e.Code, "message": e.Message}
}

func (e *Error) ExceptionStatusCode() int {
	return e.StatusCode
}

func (e *Error) RawException() error {
	return e.RawError
}

// WithMsg modify user visible message
func (e Error) WithMsg(msg string) *Error {
	e.Message = msg
	return &e
}

// WithError returns raw error struct which is not exposed to user.
// It is used for internal error tracking.
func (e Error) WithError(err error) *Error {
	e.RawError = err
	return &e
}

func New(code int, message string, statusCode int) *Error {
	return &Error{Code: code, Message: message, StatusCode: statusCode}
}

func NewInternalServerError(code int, message string) *Error {
	return New(code, message, iris.StatusInternalServerError)
}

func NewUnprocessableEntity(code int, message string) *Error {
	return New(code, message, iris.StatusUnprocessableEntity)
}

func NewNotFound(code int, message string) *Error {
	return New(code, message, iris.StatusNotFound)
}

func NewConflict(code int, message string) *Error {
	return New(code, message, iris.StatusConflict)
}

func NewUnauthorized(code int, message string) *Error {
	return New(code, message, iris.StatusUnauthorized)
}

func NewBadRequest(code int, message string) *Error {
	return New(code, message, iris.StatusBadRequest)
}

func NewForbidden(code int, message string) *Error {
	return New(code, message, iris.StatusForbidden)
}

func Format(err error) string {
	var errmsg string
	if gberr, ok := err.(IException); ok {
		if gberr.RawException() != nil {
			errmsg = fmt.Sprintf("%v : %v", err.Error(), gberr.RawException().Error())
		} else {
			errmsg = fmt.Sprintf("%v", err.Error())
		}
	} else {
		errmsg = fmt.Sprintf("%v", err.Error())
	}
	return errmsg
}

func IsNotFound(err error) bool {
	return strings.Contains(err.Error(), strconv.FormatInt(int64(NotFound.Code), 10))
}

// code convention is http_status_code:custom_code where custom code starts from 10000
var (
	// 400
	RequestBodyLoadFailure = NewBadRequest(40010000, "request body format is invalid")
	InvalidRequestParam    = NewUnprocessableEntity(40010001, "request parameters are invalid")

	// 401
	Unauthorized = NewUnauthorized(40110000, "request is unauthorized (generate APCA-API-KEY-ID and APCA-API-ACCESS-SECRET-KEY from dashboard and include in HTTP header)")

	// 403
	Forbidden = NewForbidden(40310000, "request is forbidden")

	// 404
	NotFound = NewNotFound(40410000, "resource not found")

	// 409
	Conflict = NewConflict(40910000, "resource conflict")

	// 500
	InternalServerError = NewInternalServerError(50010000, "internal server error occurred")
)
