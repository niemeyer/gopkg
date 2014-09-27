package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/niemeyer/gopkg"
)

var httpFlag = flag.String("http", ":8080", "Serve HTTP at given address")
var httpsFlag = flag.String("https", "", "Serve HTTPS at given address")
var certFlag = flag.String("cert", "", "Use the provided TLS certificate")
var keyFlag = flag.String("key", "", "Use the provided TLS key")

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()

	http.HandleFunc("/", handler)

	if *httpFlag == "" && *httpsFlag == "" {
		return fmt.Errorf("must provide -http and/or -https")
	}
	if (*httpsFlag != "" || *certFlag != "" || *keyFlag != "") && (*httpsFlag == "" || *certFlag == "" || *keyFlag == "") {
		return fmt.Errorf("-https -cert and -key must be used together")
	}

	ch := make(chan error, 2)

	if *httpFlag != "" {
		go func() {
			ch <- http.ListenAndServe(*httpFlag, nil)
		}()
	}
	if *httpsFlag != "" {
		go func() {
			ch <- http.ListenAndServeTLS(*httpsFlag, *certFlag, *keyFlag, nil)
		}()
	}
	return <-ch
}

var pkgHandler = &gopkg.Handler{
	Host: "gopkg.in",
	Client: &http.Client{
		Timeout: 10 * time.Second,
	},
}

func handler(resp http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/health-check" {
		resp.Write([]byte("ok"))
		return
	}

	log.Printf("%s requested %s", req.RemoteAddr, req.URL)

	if req.URL.Path == "/" {
		resp.Header().Set("Location", "http://labix.org/gopkg.in")
		resp.WriteHeader(http.StatusTemporaryRedirect)
		return
	}

	// Ask the gopkg handler to handle the HTTP request, if it can.
	repo, handled := pkgHandler.Handle(resp, req)
	if handled {
		// The request was handled by gopkg.
		return
	}
	if repo != nil {
		// The request was not handled by gopkg, but it is a request for the
		// package page (e.g. when entering gopkg.in/pkg.v1 in a browser).
		renderPackagePage(resp, req, repo)
		return
	}
}
