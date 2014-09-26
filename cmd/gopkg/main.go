package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"text/template"
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

var gogetTemplate = template.Must(template.New("").Parse(`
<html>
<head>
<meta name="go-import" content="{{.GopkgRoot}} git https://{{.GopkgRoot}}">
</head>
<body>
go get {{.GopkgPath}}
</body>
</html>
`))

var patternOld = regexp.MustCompile(`^/(?:([a-z0-9][-a-z0-9]+)/)?((?:v0|v[1-9][0-9]*)(?:\.0|\.[1-9][0-9]*){0,2})/([a-zA-Z][-a-zA-Z0-9]*)(?:\.git)?((?:/[a-zA-Z][-a-zA-Z0-9]*)*)$`)
var patternNew = regexp.MustCompile(`^/(?:([a-zA-Z0-9][-a-zA-Z0-9]+)/)?([a-zA-Z][-.a-zA-Z0-9]*)\.((?:v0|v[1-9][0-9]*)(?:\.0|\.[1-9][0-9]*){0,2})(?:\.git)?((?:/[a-zA-Z0-9][-.a-zA-Z0-9]*)*)$`)

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

	m := patternNew.FindStringSubmatch(req.URL.Path)
	oldFormat := false
	if m == nil {
		m = patternOld.FindStringSubmatch(req.URL.Path)
		if m == nil {
			sendNotFound(resp, "Unsupported URL pattern; see the documentation at gopkg.in for details.")
			return
		}
		m[2], m[3] = m[3], m[2]
		oldFormat = true
	}

	if strings.Contains(m[3], ".") {
		sendNotFound(resp, "Import paths take the major version only (.%s instead of .%s); see docs at gopkg.in for the reasoning.",
			m[3][:strings.Index(m[3], ".")], m[3])
		return
	}

	repo := &gopkg.Repo{
		User:      m[1],
		Name:      m[2],
		SubPath:   m[4],
		OldFormat: oldFormat,
	}

	var ok bool
	repo.MajorVersion, ok = gopkg.ParseVersion(m[3])
	if !ok {
		sendNotFound(resp, "Version %q improperly considered invalid; please warn the service maintainers.", m[3])
		return
	}

	var err error
	var refs []byte
	refs, repo.AllVersions, err = gopkg.HackedRefs(httpClient, repo)
	switch err {
	case nil:
		// all ok
	case gopkg.ErrNoRepo:
		sendNotFound(resp, "GitHub repository not found at https://%s", repo.GitHubRoot())
		return
	case gopkg.ErrNoVersion:
		v := repo.MajorVersion.String()
		sendNotFound(resp, `GitHub repository at https://%s has no branch or tag "%s", "%s.N" or "%s.N.M"`, repo.GitHubRoot(), v, v, v)
		return
	default:
		resp.WriteHeader(http.StatusBadGateway)
		resp.Write([]byte(fmt.Sprintf("Cannot obtain refs from GitHub: %v", err)))
		return
	}

	if repo.SubPath == "/git-upload-pack" {
		resp.Header().Set("Location", "https://"+repo.GitHubRoot()+"/git-upload-pack")
		resp.WriteHeader(http.StatusMovedPermanently)
		return
	}

	if repo.SubPath == "/info/refs" {
		resp.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
		resp.Write(refs)
		return
	}

	resp.Header().Set("Content-Type", "text/html")
	if req.FormValue("go-get") == "1" {
		// execute simple template when this is a go-get request
		err = gogetTemplate.Execute(resp, repo)
		if err != nil {
			log.Printf("error executing go get template: %s\n", err)
		}
		return
	}

	renderPackagePage(resp, req, repo)
}

func sendNotFound(resp http.ResponseWriter, msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	resp.WriteHeader(http.StatusNotFound)
	resp.Write([]byte(msg))
}

var httpClient = &http.Client{Timeout: 10 * time.Second}
