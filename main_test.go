package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	healthHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", response.Code, http.StatusOK)
	}
	if body := response.Body.String(); body != "MC Bot is Alive!\n" {
		t.Fatalf("GET / body = %q, want %q", body, "MC Bot is Alive!\n")
	}
}

func TestHealthHandlerRejectsNonGet(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	response := httptest.NewRecorder()

	healthHandler().ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST / status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}
