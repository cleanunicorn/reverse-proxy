package proxy

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/http2"
)

type ProxyHandler struct {
	// Can be used to stop the proxy
	// stop chan bool

	// Forward requests to this URL
	destination    string
	destinationUrl *url.URL

	// Start a server on this port
	listenPort int

	// Use TLS / HTTPS
	useTls bool
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

func (ph *ProxyHandler) handle(w http.ResponseWriter, r *http.Request) {
	fmt.Println("=== Handling request", r.URL.String())

	// Replace received request properties with destination properties
	r.Host = ph.destinationUrl.Host
	r.URL.Host = ph.destinationUrl.Host
	// TODO: check this for correctness
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

}

func (ph *ProxyHandler) Start() error {
	var err error = nil

	http.Handle("/", http.HandlerFunc(ph.handle))

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
