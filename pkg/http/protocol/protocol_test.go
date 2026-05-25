package protocol_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-service/pkg/http/protocol"
)

type unmarshalable struct{}

func (unmarshalable) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal boom")
}

type recordedLog struct {
	calls []string
}

func (r *recordedLog) Error(msg string, _ error) {
	r.calls = append(r.calls, msg)
}

func newHandler(t *testing.T, withValidator bool) (*protocol.ResponseHandler, *recordedLog) {
	t.Helper()
	log := &recordedLog{}
	opts := []protocol.Option{}
	if withValidator {
		opts = append(opts, protocol.WithValidation(validator.New(validator.WithRequiredStructEnabled())))
	}
	return protocol.NewResponseHandler(log, opts...), log
}

func TestSendErrorResponse_StatusAndBody(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	h, _ := newHandler(t, false)
	h.SendErrorResponse(rec, http.StatusTeapot, errors.New("kettle"))

	assert.Equal(t, http.StatusTeapot, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body protocol.ExceptionResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, http.StatusTeapot, body.Status)
	assert.Contains(t, body.ErrorMessage, "kettle")
}

func TestSendErrorResponse_PrependsBadRequestPrefix(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	h, _ := newHandler(t, false)
	h.SendErrorResponse(rec, http.StatusBadRequest, errors.New("name required"))

	var body protocol.ExceptionResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Contains(t, body.ErrorMessage, "incorrect request")
	assert.Contains(t, body.ErrorMessage, "name required")
}

func TestSendSuccessResponse_StatusAndBody(t *testing.T) {
	t.Parallel()

	type payload struct {
		Foo string `json:"foo"`
	}
	rec := httptest.NewRecorder()
	h, _ := newHandler(t, false)
	h.SendSuccessResponse(rec, http.StatusCreated, payload{Foo: "bar"})

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.JSONEq(t, `{"foo":"bar"}`, rec.Body.String())
}

func TestSendSuccessResponse_MarshalError(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	h, log := newHandler(t, false)
	h.SendSuccessResponse(rec, http.StatusOK, unmarshalable{})

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "marshal boom")
	assert.NotEmpty(t, log.calls, "logger must record the marshal failure")
}

type person struct {
	Name string `json:"name" validate:"required,min=2"`
}

func TestBindJSON_DecodesAndValidates(t *testing.T) {
	t.Parallel()

	h, _ := newHandler(t, true)
	body := io.NopCloser(strings.NewReader(`{"name":"Bob"}`))

	var p person
	require.NoError(t, h.BindJSON(body, &p))
	assert.Equal(t, "Bob", p.Name)
}

func TestBindJSON_ValidationError(t *testing.T) {
	t.Parallel()

	h, _ := newHandler(t, true)
	body := io.NopCloser(strings.NewReader(`{"name":""}`))

	var p person
	err := h.BindJSON(body, &p)
	require.Error(t, err)

	var verrs validator.ValidationErrors
	require.ErrorAs(t, err, &verrs)
}

func TestBindJSON_DecodeError(t *testing.T) {
	t.Parallel()

	h, _ := newHandler(t, true)
	body := io.NopCloser(strings.NewReader(`not-json`))

	var p person
	err := h.BindJSON(body, &p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode body")
}

func TestBindJSON_WithoutValidatorSkipsValidation(t *testing.T) {
	t.Parallel()

	h, _ := newHandler(t, false)
	body := io.NopCloser(strings.NewReader(`{"name":""}`)) // would fail validation if enabled

	var p person
	require.NoError(t, h.BindJSON(body, &p))
}

func TestValidate_ReturnsFlattenedMessage(t *testing.T) {
	t.Parallel()

	h, _ := newHandler(t, true)
	err := h.Validate(&person{Name: "x"}) // min=2 fails
	require.Error(t, err)

	// SendErrorResponse must produce a readable line, not the raw verbose message
	rec := httptest.NewRecorder()
	h.SendErrorResponse(rec, http.StatusBadRequest, err)

	var body protocol.ExceptionResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Contains(t, body.ErrorMessage, "Name")
	assert.Contains(t, body.ErrorMessage, "at least")
}
