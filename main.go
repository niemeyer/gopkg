package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
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
	if len(flag.Args()) > 0 {
		return fmt.Errorf("too many arguments: %s", flag.Args()[0])
	}

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

var tmplProxy = template.Must(template.New("").Parse(`
<html>
<head>
<meta name="go-import" content="{{.PkgRoot}} git {{.GitRoot}}">
</head>
<body>
<script>
window.location = "http://godoc.org/{{.PkgPath}}" + window.location.hash;
</script>
</body>
</html>
`))

type Repo struct {
	User       string // username or organization, includes forward slash
	Pkg        string // repository name
	VersionStr string // version string ("v1")
	Path       string // path inside repository
	Compat     bool   // when true, use old format

	Version  Version     // requested version (major only)
	Versions VersionList // available versions
}

func (repo *Repo) GitRoot() string {
	return "https://" + repo.PkgRoot()
}

func (repo *Repo) HubRoot() string {
	if len(repo.User) == 0 {
		return "https://github.com/go-" + repo.Pkg + "/" + repo.Pkg
	}
	return "https://github.com/" + repo.User + repo.Pkg
}

func (repo *Repo) PkgBase() string {
	return "gopkg.in/" + repo.User + repo.Pkg
}

func (repo *Repo) PkgRoot() string {
	if repo.Compat {
		return "gopkg.in/" + repo.User + repo.VersionStr + "/" + repo.Pkg
	}
	return repo.PkgBase() + "." + repo.VersionStr
}

func (repo *Repo) PkgPath() string {
	return repo.PkgRoot() + repo.Path
}

var patternOld = regexp.MustCompile(`^/([a-z0-9][-a-z0-9]+/)?((?:v0|v[1-9][0-9]*)(?:\.0|\.[1-9][0-9]*){0,2})/([a-zA-Z][-a-zA-Z0-9]*)(?:\.git)?((?:/[a-zA-Z][-a-zA-Z0-9]*)*)$`)
var patternNew = regexp.MustCompile(`^/([a-z0-9][-a-z0-9]+/)?([a-zA-Z][-a-zA-Z0-9]*)\.((?:v0|v[1-9][0-9]*)(?:\.0|\.[1-9][0-9]*){0,2})(?:\.git)?((?:/[a-zA-Z][-a-zA-Z0-9]*)*)$`)

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
	compat := false
	if m == nil {
		m = patternOld.FindStringSubmatch(req.URL.Path)
		if m == nil {
			sendNotFound(resp, "Unsupported URL pattern; see the documentation at gopkg.in for details.")
			return
		}
		m[2], m[3] = m[3], m[2]
		compat = true
	}

	if strings.Contains(m[3], ".") {
		sendNotFound(resp, "Import paths take the major version only (.%s instead of .%s); see docs at gopkg.in for the reasoning.",
			m[3][:strings.Index(m[3], ".")], m[3])
		return
	}

	repo := &Repo{
		User:       m[1],
		Pkg:        m[2],
		VersionStr: m[3],
		Path:       m[4],
		Compat:     compat,
	}

	var ok bool
	repo.Version, ok = parseVersion(repo.VersionStr)
	if !ok {
		sendNotFound(resp, "Version %q improperly considered invalid; please warn the service maintainers.", m[3])
		return
	}

	var err error
	var refs []byte
	refs, repo.Versions, err = hackedRefs(repo)
	switch err {
	case nil:
		// all ok
	case ErrNoRepo:
		sendNotFound(resp, "GitHub repository not found at %s", repo.HubRoot())
		return
	case ErrNoVersion:
		v := repo.Version.String()
		if repo.Version.Minor == -1 {
			sendNotFound(resp, `GitHub repository at %s has no branch or tag "%s", "%s.N" or "%s.N.M"`, repo.HubRoot(), v, v, v)
		} else if repo.Version.Patch == -1 {
			sendNotFound(resp, `GitHub repository at %s has no branch or tag "%s" or "%s.N"`, repo.HubRoot(), v, v)
		} else {
			sendNotFound(resp, `GitHub repository at %s has no branch or tag "%s"`, repo.HubRoot(), v)
		}
		return
	default:
		resp.WriteHeader(http.StatusBadGateway)
		resp.Write([]byte(fmt.Sprintf("Cannot obtain refs from GitHub: %v", err)))
		return
	}

	if m[4] == "/git-upload-pack" {
		resp.Header().Set("Location", repo.HubRoot()+"/git-upload-pack")
		resp.WriteHeader(http.StatusMovedPermanently)
		return
	}

	if m[4] == "/info/refs" {
		resp.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
		resp.Write(refs)
		return
	}

	resp.Header().Set("Content-Type", "text/html")
	if req.FormValue("go-get") == "1" {
		// execute simple template when this is a go-get request
		err = tmplProxy.Execute(resp, repo)
		if err != nil {
			log.Printf("error executing tmplProxy: %s\n", err)
		}
		return
	}

	renderInterface(resp, req, repo)
}

func sendNotFound(resp http.ResponseWriter, msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	resp.WriteHeader(http.StatusNotFound)
	resp.Write([]byte(msg))
}

// TODO Timeouts for these http interactions. Use the new support coming in 1.3.

const refsSuffix = ".git/info/refs?service=git-upload-pack"

var ErrNoRepo = errors.New("repository not found in github")
var ErrNoVersion = errors.New("version reference not found in github")

func hackedRefs(repo *Repo) (data []byte, versions []Version, err error) {
	resp, err := http.Get(repo.HubRoot() + refsSuffix)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot talk to GitHub: %v", err)
	}
	switch resp.StatusCode {
	case 200:
		defer resp.Body.Close()
	case 401, 404:
		return nil, nil, ErrNoRepo
	default:
		return nil, nil, fmt.Errorf("error from GitHub: %v", resp.Status)
	}

	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading from GitHub: %v", err)
	}

	var mrefi, mrefj int
	var vrefi, vrefj int
	var vrefv = InvalidVersion
	var unversioned = true

	versions = make([]Version, 0)
	sdata := string(data)
	for i, j := 0, 0; i < len(data); i = j {
		size, err := strconv.ParseInt(sdata[i:i+4], 16, 32)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot parse refs line size: %s", string(data[i:i+4]))
		}
		if size == 0 {
			size = 4
		}
		j = i + int(size)
		if j > len(sdata) {
			return nil, nil, fmt.Errorf("incomplete refs data received from GitHub")
		}
		if sdata[0] == '#' {
			continue
		}

		hashi := i + 4
		hashj := strings.IndexByte(sdata[hashi:j], ' ')
		if hashj < 0 || hashj != 40 {
			continue
		}
		hashj += hashi

		namei := hashj + 1
		namej := strings.IndexAny(sdata[namei:j], "\n\x00")
		if namej < 0 {
			namej = j
		} else {
			namej += namei
		}

		name := sdata[namei:namej]

		if name == "refs/heads/master" {
			mrefi = hashi
			mrefj = hashj
		}

		if strings.HasPrefix(name, "refs/heads/v") || strings.HasPrefix(name, "refs/tags/v") {
			v, ok := parseVersion(name[strings.IndexByte(name, 'v'):])
			if ok && repo.Version.Contains(v) && (!vrefv.IsValid() || vrefv.Less(v)) {
				vrefv = v
				vrefi = hashi
				vrefj = hashj
			}
			if ok {
				unversioned = false
				versions = append(versions, v)
			}
		}
	}

	// If there were absolutely no versions, and v0 was requested, accept the master as-is.
	if unversioned && repo.Version == (Version{0, -1, -1}) {
		return data, nil, nil
	}

	if mrefi == 0 || vrefi == 0 {
		return nil, nil, ErrNoVersion
	}

	copy(data[mrefi:mrefj], data[vrefi:vrefj])
	return data, versions, nil
}
