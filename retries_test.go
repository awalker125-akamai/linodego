package linodego

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestLinodeBusyRetryCondition(t *testing.T) {
	var retry bool

	// Initialize response body
	rawResponse := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(bytes.NewBuffer(nil)),
	}

	retry = LinodeBusyRetryCondition(rawResponse, nil)

	if retry {
		t.Errorf("Should not have retried")
	}

	apiError := APIError{
		Errors: []APIErrorReason{
			{Reason: "Linode busy."},
		},
	}
	rawResponse.Body = createResponseBody(apiError)

	retry = LinodeBusyRetryCondition(rawResponse, nil)

	if !retry {
		t.Errorf("Should have retried")
	}
}

func TestServiceUnavailableRetryCondition(t *testing.T) {
	rawResponse := &http.Response{
		StatusCode: http.StatusServiceUnavailable,
		Header:     http.Header{RetryAfterHeaderName: []string{"20"}},
		Body:       io.NopCloser(bytes.NewBuffer(nil)), // Initialize response body
	}

	if retry := ServiceUnavailableRetryCondition(rawResponse, nil); !retry {
		t.Error("expected request to be retried")
	}

	if retryAfter, err := RespectRetryAfter(rawResponse); err != nil {
		t.Errorf("expected error to be nil but got %s", err)
	} else if retryAfter != time.Second*20 {
		t.Errorf("expected retryAfter to be 20 but got %d", retryAfter)
	}
}

func TestServiceMaintenanceModeRetryCondition(t *testing.T) {
	rawResponse := &http.Response{
		StatusCode: http.StatusServiceUnavailable,
		Header: http.Header{
			RetryAfterHeaderName:      []string{"20"},
			MaintenanceModeHeaderName: []string{"Currently in maintenance mode."},
		},
		Body: io.NopCloser(bytes.NewBuffer(nil)), // Initialize response body
	}

	if retry := ServiceUnavailableRetryCondition(rawResponse, nil); retry {
		t.Error("expected retry to be skipped due to maintenance mode header")
	}
}

func TestRequestTransportTimeoutRetryCondition(t *testing.T) {
	if retry := RequestTransportTimeoutRetryCondition(nil, context.DeadlineExceeded); !retry {
		t.Error("expected context deadline exceeded to be retried")
	}

	if retry := RequestTransportTimeoutRetryCondition(nil, os.ErrDeadlineExceeded); !retry {
		t.Error("expected os deadline exceeded to be retried")
	}

	timeoutErr := &net.DNSError{IsTimeout: true}
	if retry := RequestTransportTimeoutRetryCondition(nil, timeoutErr); !retry {
		t.Error("expected net timeout error to be retried")
	}
}

func TestRequestEOFRetryCondition(t *testing.T) {
	if retry := RequestEOFRetryCondition(nil, io.EOF); !retry {
		t.Error("expected EOF to be retried")
	}

	if retry := RequestEOFRetryCondition(nil, io.ErrUnexpectedEOF); !retry {
		t.Error("expected unexpected EOF to be retried")
	}
}

// Helper function to create a response body from an object
func createResponseBody(obj interface{}) io.ReadCloser {
	body, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}
	return io.NopCloser(bytes.NewBuffer(body))
}
