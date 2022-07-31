package proxy

import (
	"fmt"
	"net/http"
)

type ProxyHandler struct {
	// Can be used to stop the proxy
	// stop chan bool

	// The proxy server
	destination string
	listenPort  int
}

func New(destination string, listenPort int) *ProxyHandler {
	ph := &ProxyHandler{
		destination: destination,
		listenPort:  listenPort,
	}

	return ph
}

func (ph *ProxyHandler) handle(w http.ResponseWriter, r *http.Request) {
	// TODO
	fmt.Println("Handling request", r.URL.String())
}

func (ph *ProxyHandler) Start() error {
	// TODO
	fmt.Println("Starting proxy on port", ph.listenPort, "to", ph.destination)

	http.Handle("/", http.HandlerFunc(ph.handle))
	http.ListenAndServe(fmt.Sprintf(":%d", ph.listenPort), nil)

	// Return error if the proxy could not be started
	return nil
}
