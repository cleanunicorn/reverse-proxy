/*
Copyright © 2022 Daniel Luca

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package proxy

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
)

type CachedResponse struct {
	dump []byte
}

type ProxyHandler struct {
	// Logger
	logger logrus.Logger

	// Forward requests to this URL
	destination    string
	destinationUrl *url.URL

	// Start a server on this port
	listenPort int

	// Use TLS / HTTPS
	useTls bool

	// Cache map
	cache map[string]CachedResponse
}

func New(destination string, listenPort int) (*ProxyHandler, error) {
	ph := &ProxyHandler{
		destination: destination,
		listenPort:  listenPort,
	}

	// Parse the destination URL
	var err error
	ph.destinationUrl, err = url.Parse(destination)
	if err != nil {
		return nil, err
	}

	return ph, nil
}

func (ph *ProxyHandler) EnableTls() {
	ph.useTls = true
}

func (ph *ProxyHandler) DisableTls() {
	ph.useTls = false
}

func (ph *ProxyHandler) SetLogger(logger *logrus.Logger) {
	ph.logger = *logger
}

func (ph *ProxyHandler) makeRequestKey(r *http.Request) string {
	return fmt.Sprintf("%s %s %s", r.Method, r.URL.String(), r.RemoteAddr)
}

func (ph *ProxyHandler) handle(w http.ResponseWriter, r *http.Request) {
	ph.logger.Info(`Handling request `, r.URL.String())

	// Check if the request is already cached
	requestKey := ph.makeRequestKey(r)
	ph.logger.Debug(`Cache key: `, requestKey)
	if cachedResponse, ok := ph.cache[requestKey]; ok {
		ph.logger.Debug(`Cache hit`)

		// // Send headers
		// for headerName, headerValue := range cachedResponse.headers {
		// 	w.Header().Set(headerName, headerValue)
		// }

		// Write the cached response
		w.Write(cachedResponse.dump)

		return
	}

	// Replace received request properties with destination properties
	r.Host = ph.destinationUrl.Host
	r.URL.Host = ph.destinationUrl.Host
	r.URL.Scheme = ph.destinationUrl.Scheme
	r.RequestURI = ""

	// Add X-Forwarded-For header
	client, _, _ := net.SplitHostPort(r.RemoteAddr)
	r.Header.Set("X-Forwarded-For", client)

	// Enable http2
	http2.ConfigureTransport(http.DefaultTransport.(*http.Transport))

	// Skip tls verification
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	destinationResponse, err := http.DefaultClient.Do(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err)
		return
	}

	// Flush periodically to avoid buffering
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-time.Tick(10 * time.Millisecond):
				w.(http.Flusher).Flush()
			case <-done:
				return
			}
		}
	}()
	defer close(done)

	// Forward headers from destination to client
	for key, values := range destinationResponse.Header {
		w.Header().Set(key, strings.Join(values, ", "))
	}

	// Send status code
	w.WriteHeader(destinationResponse.StatusCode)

	// Send body
	io.Copy(w, destinationResponse.Body)

	// Cache response
	cachedResponse, err := httputil.DumpResponse(destinationResponse, false)
	if err != nil {
		ph.logger.Error(`Could not dump response: `, err)
	} else {
		ph.cache[requestKey] = CachedResponse{dump: cachedResponse}
	}
}

func (ph *ProxyHandler) Start() error {
	var err error = nil

	http.Handle("/", http.HandlerFunc(ph.handle))

	// Initialize cache
	ph.cache = make(map[string]CachedResponse)

	// Start server
	if ph.useTls {
		fmt.Println("Starting https proxy on port", ph.listenPort)
		fmt.Println("Forwarding requests to", ph.destination)

		err = http.ListenAndServeTLS(fmt.Sprintf(":%d", ph.listenPort), "cert/server.pem", "cert/server.key", nil)
	} else {
		fmt.Println("Starting http proxy on port", ph.listenPort)
		fmt.Println("Forwarding requests to", ph.destination)

		err = http.ListenAndServe(fmt.Sprintf(":%d", ph.listenPort), nil)
	}

	// Return error if the proxy could not be started
	return err
}
