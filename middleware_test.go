package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/casbin/casbin"
)

func TestAuthorizationMiddleware(t *testing.T) {
	var (
		e    = casbin.NewEnforcer("./model.conf")
		body = strings.NewReader("{\"query\": \"query { foo }\"}")
	)

	req, err := http.NewRequest("POST", "/graphql", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Subject", "alice")

	e.AddPolicy("alice", "foo", ActionQuery)

	r1 := handleRequest(e, req)

	// Check the status code is what we expect.
	if status := r1.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	e.RemovePolicy("alice", "foo", ActionQuery)

	r2 := handleRequest(e, req)

	// Check the status code is what we expect.
	if status := r2.Code; status != http.StatusForbidden {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusForbidden)
	}
}

func handleRequest(e *casbin.Enforcer, req *http.Request) *httptest.ResponseRecorder {
	r := httptest.NewRecorder()
	handler := WithAuthorization(e)(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(""))
	})
	handler.ServeHTTP(r, req)
	return r
}
