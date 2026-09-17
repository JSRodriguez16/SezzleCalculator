// Package exceptions defines domain exceptions and typed errors shared by the business and presentation layers.
package exceptions

// Code is a stable identifier that clients can use independently of the message.
type Code string

const (
	InvalidInput         Code = "INVALID_INPUT"
	InvalidJSON          Code = "INVALID_JSON"
	InvalidOperation     Code = "INVALID_OPERATION"
	DivisionByZero       Code = "DIVISION_BY_ZERO"
	InvalidSquareRoot    Code = "INVALID_SQUARE_ROOT"
	InvalidExponent      Code = "INVALID_EXPONENT"
	NonFiniteResult      Code = "NON_FINITE_RESULT"
	UnsupportedMediaType Code = "UNSUPPORTED_MEDIA_TYPE"
	PayloadTooLarge      Code = "PAYLOAD_TOO_LARGE"
	NotFound             Code = "NOT_FOUND"
	MethodNotAllowed     Code = "METHOD_NOT_ALLOWED"
	InternalError        Code = "INTERNAL_ERROR"
)

// Error carries a machine-readable code and a safe, user-facing explanation.
type Error struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Message }

// New constructs an application exception. The presentation layer maps it to an HTTP status.
func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}
