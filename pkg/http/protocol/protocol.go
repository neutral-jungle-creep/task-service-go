// Package protocol provides a ResponseHandler that owns JSON encoding,
// decoding+validation and error responses for HTTP handlers.
package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

const (
	internalServerErrorMsg = "internal server error"
	incorrectRequestErrMsg = "incorrect request"
)

// Logger is the minimal logging contract the handler needs to report
// JSON-encoding failures.
type Logger interface {
	Error(msg string, err error)
}

// Validator is satisfied by *validator.Validate from go-playground/validator
// (and by any other type with the same Struct method). Keeping it tiny lets
// callers swap in a no-op or a mock in tests.
type Validator interface {
	Struct(s any) error
}

// ResponseHandler owns JSON encoding, body binding+validation and error
// responses. Construct it once at startup and pass into every API surface.
type ResponseHandler struct {
	logger    Logger
	validator Validator
}

func NewResponseHandler(logger Logger, opts ...Option) *ResponseHandler {
	h := &ResponseHandler{logger: logger}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// ExceptionResponse is the JSON shape of a 4xx/5xx response body.
type ExceptionResponse struct {
	ErrorMessage string    `json:"errorMessage"`
	Status       int       `json:"status"`
	Timestamp    time.Time `json:"timestamp"`
}

// SendErrorResponse writes a 4xx/5xx JSON response. The status drives the
// default human message (validation errors are flattened into a readable line).
func (h *ResponseHandler) SendErrorResponse(w http.ResponseWriter, status int, err error) {
	defaultMsg := ""
	switch status {
	case http.StatusInternalServerError:
		defaultMsg = internalServerErrorMsg
	case http.StatusBadRequest:
		defaultMsg = incorrectRequestErrMsg
	}

	msg := strings.TrimSpace(fmt.Sprintf("%s %s", defaultMsg, flattenError(err)))

	body, marshalErr := json.Marshal(&ExceptionResponse{
		ErrorMessage: msg,
		Status:       status,
		Timestamp:    time.Now().Truncate(time.Second),
	})
	if marshalErr != nil {
		h.logger.Error("SendErrorResponse: marshal", marshalErr)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(marshalErr.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, writeErr := w.Write(body)
	if writeErr != nil {
		h.logger.Error("SendErrorResponse: write", writeErr)
	}
}

// SendSuccessResponse writes a 2xx JSON response with the given body.
func (h *ResponseHandler) SendSuccessResponse(w http.ResponseWriter, status int, body any) {
	payload, err := json.Marshal(body)
	if err != nil {
		h.logger.Error("SendSuccessResponse: marshal", err)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, writeErr := w.Write(payload)
	if writeErr != nil {
		h.logger.Error("SendSuccessResponse: write", writeErr)
	}
}

// BindJSON decodes JSON from body into params and, if a Validator was
// configured, runs validate rules. Always closes body.
func (h *ResponseHandler) BindJSON(body io.ReadCloser, params any) error {
	defer func() { _ = body.Close() }()

	if err := json.NewDecoder(body).Decode(params); err != nil {
		return fmt.Errorf("decode body: %w", err)
	}
	if h.validator == nil {
		return nil
	}
	err := h.validator.Struct(params)
	if err != nil {
		return err
	}
	return nil
}

// Validate runs validator rules without decoding — useful for URL query
// structs that are populated manually before validation.
func (h *ResponseHandler) Validate(params any) error {
	if h.validator == nil {
		return nil
	}
	return h.validator.Struct(params)
}

// flattenError converts validator.ValidationErrors into a human-readable
// "Field rule; Field2 rule2" string; other errors are returned as-is.
func flattenError(err error) string {
	if err == nil {
		return ""
	}
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return err.Error()
	}
	parts := make([]string, 0, len(verrs))
	for _, fe := range verrs {
		parts = append(parts, formatFieldError(fe))
	}
	return strings.Join(parts, "; ")
}

func formatFieldError(fe validator.FieldError) string {
	field := fe.Field()
	switch fe.Tag() {
	case "required":
		return field + " is required"
	case "min":
		return fmt.Sprintf("%s must be at least %s", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", field, fe.Param())
	case "gte":
		return fmt.Sprintf("%s must be >= %s", field, fe.Param())
	case "lte":
		return fmt.Sprintf("%s must be <= %s", field, fe.Param())
	default:
		return fmt.Sprintf("%s fails %q", field, fe.Tag())
	}
}
