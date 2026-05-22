package protocol_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-service/pkg/http/protocol"
)

// unmarshalable forces json.Marshal to fail so we can exercise the
// marshal-error branch in SendSuccessResponse / SendErrorResponse.
type unmarshalable struct{}

func (unmarshalable) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal boom")
}

func TestSendErrorResponse_StatusAndBody(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	protocol.SendErrorResponse(rec, 418, "wrong", errors.New("kettle"))

	assert.Equal(t, 418, rec.Code, "WriteHeader must be called before Write so the status is preserved")
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body protocol.ExceptionResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, 418, body.Status)
	assert.Contains(t, body.ErrorMessage, "wrong")
	assert.Contains(t, body.ErrorMessage, "kettle")
}

func TestSendSuccessResponse_StatusAndBody(t *testing.T) {
	t.Parallel()

	type payload struct {
		Foo string `json:"foo"`
	}

	rec := httptest.NewRecorder()
	protocol.SendSuccessResponse(rec, 201, payload{Foo: "bar"})

	assert.Equal(t, 201, rec.Code, "WriteHeader must be called before Write so the status is preserved")
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"foo":"bar"}`, rec.Body.String())
}

func TestSendSuccessResponse_MarshalError(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	protocol.SendSuccessResponse(rec, http.StatusOK, unmarshalable{})

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "text/plain; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), "marshal boom")
}
