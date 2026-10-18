package scpclient_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

const jsonContentType = "application/json"

func TestDoSetsContentTypeByMethod(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		method string
		body   []byte
		want   string
	}{
		"POST without body":     {method: http.MethodPost, body: nil, want: jsonContentType},
		"POST with empty body":  {method: http.MethodPost, body: []byte{}, want: jsonContentType},
		"POST with body":        {method: http.MethodPost, body: []byte(`{"a":1}`), want: jsonContentType},
		"PUT without body":      {method: http.MethodPut, body: nil, want: jsonContentType},
		"PUT with empty body":   {method: http.MethodPut, body: []byte{}, want: jsonContentType},
		"PUT with body":         {method: http.MethodPut, body: []byte(`{"a":1}`), want: jsonContentType},
		"DELETE without body":   {method: http.MethodDelete, body: nil, want: jsonContentType},
		"PATCH without body":    {method: http.MethodPatch, body: nil, want: "application/merge-patch+json"},
		"PATCH with empty body": {method: http.MethodPatch, body: []byte{}, want: "application/merge-patch+json"},
		"PATCH with body":       {method: http.MethodPatch, body: []byte(`{"a":1}`), want: "application/merge-patch+json"},
		"GET has no body":       {method: http.MethodGet, body: nil, want: ""},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			checkRequest(t, tc.method, tc.body, tc.want)
		})
	}
}

func checkRequest(t *testing.T, method string, body []byte, want string) {
	t.Helper()
	var got string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Content-Type")
		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		w.Header().Set("Content-Type", jsonContentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := scpclient.New("token", "", srv.URL, srv.Client())
	if err := sendRequest(t, c, method, body); err != nil {
		t.Fatalf("%s request failed: %v", method, err)
	}
	if got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
	if !bytes.Equal(gotBody, body) {
		t.Fatalf("request body = %q, want %q", gotBody, body)
	}
}

func sendRequest(t *testing.T, c *scpclient.Client, method string, body []byte) error {
	t.Helper()

	ctx := context.Background()
	const path = "/api/v1/test"

	var err error
	switch method {
	case http.MethodGet:
		_, err = c.Get(ctx, path, nil)
	case http.MethodPost:
		_, err = c.Post(ctx, path, body)
	case http.MethodPut:
		_, err = c.Put(ctx, path, body)
	case http.MethodPatch:
		_, err = c.Patch(ctx, path, body)
	case http.MethodDelete:
		_, err = c.Delete(ctx, path)
	default:
		t.Fatalf("unhandled method %q", method)
	}
	return err
}
