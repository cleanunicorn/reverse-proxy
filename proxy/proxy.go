package proxy

import (
	"fmt"
	"net/http"
	"net/url"
)

type ProxyHandler struct {
	// Can be used to stop the proxy
	// stop chan bool

	// The proxy server
	destination    string
	destinationUrl *url.URL
	listenPort     int
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

func (ph *ProxyHandler) handle(w http.ResponseWriter, r *http.Request) {
	// TODO
	fmt.Println("Handling request", r.URL.String())

	r.Host = ph.destinationUrl.Host
	r.URL.Host = ph.destinationUrl.Host
	r.URL.Scheme = ph.destinationUrl.Scheme
	r.RequestURI = ""

	_, err := http.DefaultClient.Do(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, err)
	}
}

func (ph *ProxyHandler) Start() error {
	// TODO
	fmt.Println("Starting proxy on port", ph.listenPort, "to", ph.destination)

	http.Handle("/", http.HandlerFunc(ph.handle))
	http.ListenAndServe(fmt.Sprintf(":%d", ph.listenPort), nil)

	// Return error if the proxy could not be started
	return nil
}
