package registry

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
)

func TestErrBasicAuth(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("www-authenticate", `Basic realm="Registry Realm",service="Docker registry"`)
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	authConfig := types.AuthConfig{
		Username:      "j3ss",
		Password:      "ss3j",
		ServerAddress: ts.URL,
	}
	r, err := New(ctx, authConfig, Opt{Insecure: true, Debug: true})
	if err != nil {
		t.Fatalf("expected no error creating client, got %v", err)
	}
	token, err := r.Token(ctx, ts.URL)
	if err != ErrBasicAuth {
		t.Fatalf("expected ErrBasicAuth getting token, got %v", err)
	}
	if token != "" {
		t.Fatalf("expected empty token, got %v", err)
	}
}

var authURI string

func oauthFlow(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/oauth2/accesstoken") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"access_token":"abcdef1234"}`))
		return
	}
	if strings.HasPrefix(r.URL.Path, "/oauth2/token") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"token":"abcdef1234"}`))
		return
	}
	auth := r.Header.Get("authorization")
	if !strings.HasPrefix(auth, "Bearer") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if authURI != "" {
			w.Header().Set("www-authenticate", `Bearer realm="`+authURI+`/oauth2/token",service="my.endpoint.here"`)
		}
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"errors":[{"code":"UNAUTHORIZED","message":"authentication required","detail":null}]}`))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func TestBothTokenAndAccessTokenWork(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(oauthFlow))
	defer ts.Close()

	for _, which := range []string{"token", "accesstoken"} {
		ctx := context.Background()
		authURI = ts.URL + "/oauth2/" + which + "?service=my.endpoint.here"
		authConfig := types.AuthConfig{
			Username:      "abc",
			Password:      "123",
			ServerAddress: ts.URL,
		}
		authConfig.Email = "me@email.com"
		r, err := New(ctx, authConfig, Opt{Insecure: true, Debug: true})
		if err != nil {
			t.Fatalf("expected no error creating client, got %v", err)
		}
		token, err := r.Token(ctx, ts.URL)
		if err != nil {
			t.Fatalf("err getting token from url: %v err: %v", ts.URL, err)
		}
		if token == "" {
			t.Fatalf("error got empty token")
		}
	}
}

type testTransport func(*http.Request) (*http.Response, error)

func (tt testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return tt(req)
}

func TestTokenTransportErrorHandling(t *testing.T) {
	tokenTransport := &TokenTransport{
		Transport: testTransport(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("transport failed")
		}),
	}
	_, err := tokenTransport.RoundTrip(httptest.NewRequest(http.MethodGet, "/", nil))
	if err == nil {
		t.Fatalf("got no error from round trip: %s", err)
	}
}

type testBody struct {
	t      *testing.T
	closed bool
}

func (tb *testBody) Read(p []byte) (n int, err error) {
	tb.t.Helper()
	panic("unexpected read")
}

func (tb *testBody) Close() error {
	tb.closed = true
	return nil
}

func TestTokenTransportTokenDemandErr(t *testing.T) {
	body := &testBody{t: t}
	tokenTransport := &TokenTransport{
		Transport: testTransport(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				Body:       body,
				StatusCode: http.StatusUnauthorized,
			}, nil
		}),
	}
	resp, err := tokenTransport.RoundTrip(httptest.NewRequest(http.MethodGet, "/", nil))
	if err == nil {
		t.Fatal("Expected error due to missing auth challenge header, got none")
	}
	if resp != nil {
		t.Fatal("Expected no response")
	}
	if !body.closed {
		t.Fatal("Expected body to be closed")
	}
}

func TestTokenTransportAuthLeak(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(oauthFlow))
	authURI = ts.URL + "/oauth2/token?service=my.endpoint.here"
	callCounter := 0
	body := &testBody{t: t}
	tokenTransport := &TokenTransport{
		Transport: testTransport(func(req *http.Request) (*http.Response, error) {
			callCounter++
			switch callCounter {
			case 1: // failing authentication
				header := http.Header{}
				header.Set("www-authenticate", `Bearer realm="`+authURI+`/oauth2/token",service="my.endpoint.here"`)
				return &http.Response{
					Body:       body,
					StatusCode: http.StatusUnauthorized,
					Header:     header,
				}, nil
			case 2: // auth request
				return ts.Client().Transport.RoundTrip(req)
			default:
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       &testBody{t: t},
					Header:     http.Header{},
				}, nil

			}
		}),
	}
	resp, err := tokenTransport.RoundTrip(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if resp == nil {
		t.Fatal("Response is missing")
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}
	if !body.closed {
		t.Fatal("Expected body to be closed")
	}
}
