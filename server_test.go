package main

import (
	"ShadowTest/ssproxy"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthcheck(t *testing.T) {
	router, err := getRouter(true)
	assert.NoError(t, err)

	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestGetProxyDetailsFromServerJSON(t *testing.T) {
	address := "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpwYXNzd29yZA@localhost:6276/?outline=1"

	router, err := getRouter(true)
	assert.NoError(t, err)

	body := bytes.NewBuffer([]byte(fmt.Sprintf("{ \"address\":\"%s\" }", address)))
	req, _ := http.NewRequest("POST", "/v2/test", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	details := ssproxy.WTFIsMyIPData{}
	err = json.NewDecoder(rr.Body).Decode(&details)
	assert.NoError(t, err)

	assert.NotEmpty(t, details.YourFuckingIPAddress)
	assert.NotEmpty(t, details.YourFuckingLocation)
}

func TestGetProxyDetailsFromServerTimeout(t *testing.T) {
	address := "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpwYXNzd29yZA@shadowtest.akiel.dev:6276"

	router, err := getRouter(true)
	assert.NoError(t, err)

	body := bytes.NewBuffer([]byte(fmt.Sprintf("{ \"address\":\"%s\", \"timeout\": 1 }", address)))
	req, _ := http.NewRequest("POST", "/v2/test", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestDeprecatedV1(t *testing.T) {
	router, err := getRouter(true)
	assert.NoError(t, err)

	body := bytes.NewBuffer([]byte(fmt.Sprintf("{ \"address\":\"%s\" }", "")))
	req, _ := http.NewRequest("POST", "/v1/test", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
	content, err := io.ReadAll(rr.Body)
	assert.NoError(t, err)
	assert.Equal(t, []byte("Deprecated endpoint. Use v2 instead.\n"), content)
}
