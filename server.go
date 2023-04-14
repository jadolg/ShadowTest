package main

import (
	"ShadowTest/ssproxy"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
)

type proxyJson struct {
	Address string `json:"address"`
	Timeout int    `json:"timeout,omitempty"`
}

//go:embed index.html
var indexFile embed.FS

func getRouter(ipv4Only bool) (*http.ServeMux, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/test", func(w http.ResponseWriter, r *http.Request) {
		defer closeBody(r)

		if r.Method != "POST" {
			http.Error(w, "Method is not supported.", http.StatusNotFound)
			return
		}
		address := ""

		timeout, err := getDefaultTimeout()
		if err != nil {
			log.Errorf("impossible to get default timeout %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		p := proxyJson{}

		var done bool
		if r.Header.Get("Content-Type") == "application/json" {
			address, timeout, done = getAddressAndTimeoutFromJSON(w, r, p, address, timeout)
		} else {
			address, timeout, err, done = getAddressAndTimeoutFromForm(w, r, address, timeout, err)
		}
		if done {
			return
		}

		if address == "" {
			http.Error(w, "Missing address in the request", http.StatusBadRequest)
			return
		}

		details, err := ssproxy.GetShadowsocksProxyDetails(address, ipv4Only, timeout)
		testsTotal.Inc()
		if err != nil {
			failuresTotal.Inc()
			fillCheckError(w, err, address)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(details)

		if err != nil {
			log.Errorf("error occurred when sending the data back to the client %v", err)
		}
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	var staticFS = http.FS(indexFile)
	mux.Handle("/", http.FileServer(staticFS))

	mux.Handle("/metrics", promhttp.Handler())

	return mux, nil
}

func fillCheckError(w http.ResponseWriter, err error, address string) {
	if err, ok := err.(net.Error); ok && err.Timeout() {
		http.Error(w, fmt.Sprintf("Timeout getting information for address %s", address), http.StatusGatewayTimeout)
	} else {
		http.Error(w, fmt.Sprintf("Unable to get information for address %s", address), http.StatusBadGateway)
	}
}

func closeBody(r *http.Request) {
	func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Errorf("impossible to close request body %v", err)
		}
	}(r.Body)
}

func getAddressAndTimeoutFromForm(w http.ResponseWriter, r *http.Request, address string, timeout int, err error) (string, int, error, bool) {
	if err := r.ParseForm(); err != nil {
		_, err := fmt.Fprintf(w, "ParseForm() err: %v", err)
		if err != nil {
			log.Errorf("impossible to write response %v", err)
			return "", 0, nil, true
		}
		http.Error(w, "Unable to parse request data", http.StatusBadRequest)
		return "", 0, nil, true
	}
	address = r.FormValue("address")
	if r.FormValue("timeout") != "" {
		timeout, err = strconv.Atoi(r.FormValue("timeout"))
		if err != nil {
			http.Error(w, "Unable to parse timeout", http.StatusBadRequest)
			return "", 0, nil, true
		}
	}
	return address, timeout, err, false
}

func getAddressAndTimeoutFromJSON(w http.ResponseWriter, r *http.Request, p proxyJson, address string, timeout int) (string, int, bool) {
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return "", 0, true
	}
	address = p.Address
	if p.Timeout > 0 {
		timeout = p.Timeout
	}
	return address, timeout, false
}

func getDefaultTimeout() (int, error) {
	timeout := 30
	timeoutFromEnv := os.Getenv("TIMEOUT")
	if timeoutFromEnv != "" {
		timeoutInt, err := strconv.Atoi(timeoutFromEnv)
		if err != nil {
			return 0, err
		}
		timeout = timeoutInt
	}
	return timeout, nil
}