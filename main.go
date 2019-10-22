package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

var (
	httpFlag  = flag.String("http", ":8080", "Serve HTTP at given address")
	httpsFlag = flag.String("https", "", "Serve HTTPS at given address")
	certFlag  = flag.String("cert", "", "Use the provided TLS certificate")
	keyFlag   = flag.String("key", "", "Use the provided TLS key")
	acmeFlag  = flag.String("acme", "", "Auto-request TLS certs and store in given directory")
)

var httpServer = &http.Server{
	ReadTimeout:  30 * time.Second,
	WriteTimeout: 5 * time.Minute,
}

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

var bulkClient = &http.Client{
	Timeout: 5 * time.Minute,
}

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
	if *acmeFlag != "" && *httpsFlag == "" {
		return fmt.Errorf("cannot use -acme without -https")
	}
	if *acmeFlag != "" && (*certFlag != "" || *keyFlag != "") {
		return fmt.Errorf("cannot provide -acme with -key or -cert")
	}
	if *acmeFlag == "" && (*httpsFlag != "" || *certFlag != "" || *keyFlag != "") && (*httpsFlag == "" || *certFlag == "" || *keyFlag == "") {
		return fmt.Errorf("-https -cert and -key must be used together")
	}

	ch := make(chan error, 2)

	if *acmeFlag != "" {
		// So a potential error is seen upfront.
		if err := os.MkdirAll(*acmeFlag, 0700); err != nil {
			return err
		}
	}

	if *httpFlag != "" && (*httpsFlag == "" || *acmeFlag == "") {
		server := *httpServer
		server.Addr = *httpFlag
		go func() {
			ch <- server.ListenAndServe()
		}()
	}
	if *httpsFlag != "" {
		server := *httpServer
		server.Addr = *httpsFlag
		if *acmeFlag != "" {
			m := autocert.Manager{
				ForceRSA:    true,
				Prompt:      autocert.AcceptTOS,
				Cache:       autocert.DirCache(*acmeFlag),
				RenewBefore: 24 * 30 * time.Hour,
				HostPolicy: autocert.HostWhitelist(
					"localhost",
					"gopkg.in",
					"p1.gopkg.in",
					"p2.gopkg.in",
					"p3.gopkg.in",
				),
				Email: "gustavo@niemeyer.net",
			}
			server.TLSConfig = &tls.Config{
				GetCertificate: m.GetCertificate,
			}
			go func() {
				ch <- http.ListenAndServe(":80", m.HTTPHandler(nil))
			}()
		}
		go func() {
			ch <- server.ListenAndServeTLS(*certFlag, *keyFlag)
		}()

	}
	return <-ch
}

var gogetTemplate = template.Must(template.New("").Parse(`
<html>
<head>
<meta name="go-import" content="{{.Original.GopkgRoot}} git https://{{.Original.GopkgRoot}}">
{{$root := .GitHubRoot}}{{$tree := .GitHubTree}}<meta name="go-source" content="{{.Original.GopkgRoot}} _ https://{{$root}}/tree/{{$tree}}{/dir} https://{{$root}}/blob/{{$tree}}{/dir}/{file}#L{line}">
</head>
<body>
go get {{.GopkgPath}}
</body>
</html>
`))

// Repo represents a source code repository on GitHub.
type Repo struct {
	User         string
	Name         string
	SubPath      string
	OldFormat    bool // The old /v2/pkg format.
	MajorVersion Version

	// FullVersion is the best version in AllVersions that matches MajorVersion.
	// It defaults to InvalidVersion if there are no matches.
	FullVersion Version

	// AllVersions holds all versions currently available in the repository,
	// either coming from branch names or from tag names. Version zero (v0)
	// is only present in the list if it really exists in the repository.
	AllVersions VersionList

	// When there is a redirect in place, these are from the original request.
	RedirUser string
	RedirName string
}

// SetVersions records in the relevant fields the details about which
// package versions are available in the repository.
func (repo *Repo) SetVersions(all []Version) {
	repo.AllVersions = all
	for _, v := range repo.AllVersions {
		if v.Major == repo.MajorVersion.Major && v.Unstable == repo.MajorVersion.Unstable && repo.FullVersion.Less(v) {
			repo.FullVersion = v
		}
	}
}

// When there is a redirect in place, this will return the original repository
// but preserving the data for the new repository.
func (repo *Repo) Original() *Repo {
	if repo.RedirName == "" {
		return repo
	}
	orig := *repo
	orig.User = repo.RedirUser
	orig.Name = repo.RedirName
	return &orig
}

type repoBase struct {
	user string
	name string
}

var redirect = map[repoBase]repoBase{
	// https://github.com/go-fsnotify/fsnotify/issues/1
	{"", "fsnotify"}: {"fsnotify", "fsnotify"},
}

// GitHubRoot returns the repository root at GitHub, without a schema.
func (repo *Repo) GitHubRoot() string {
	if repo.User == "" {
		return "github.com/go-" + repo.Name + "/" + repo.Name
	} else {
		return "github.com/" + repo.User + "/" + repo.Name
	}
}

// GitHubTree returns the repository tree name at GitHub for the selected version.
func (repo *Repo) GitHubTree() string {
	if repo.FullVersion == InvalidVersion {
		return "master"
	}
	return repo.FullVersion.String()
}

// GopkgRoot returns the package root at gopkg.in, without a schema.
func (repo *Repo) GopkgRoot() string {
	return repo.GopkgVersionRoot(repo.MajorVersion)
}

// GopkgPath returns the package path at gopkg.in, without a schema.
func (repo *Repo) GopkgPath() string {
	return repo.GopkgVersionRoot(repo.MajorVersion) + repo.SubPath
}

// GopkgVersionRoot returns the package root in gopkg.in for the
// provided version, without a schema.
func (repo *Repo) GopkgVersionRoot(version Version) string {
	version.Minor = -1
	version.Patch = -1
	v := version.String()
	if repo.OldFormat {
		if repo.User == "" {
			return "gopkg.in/" + v + "/" + repo.Name
		} else {
			return "gopkg.in/" + repo.User + "/" + v + "/" + repo.Name
		}
	} else {
		if repo.User == "" {
			return "gopkg.in/" + repo.Name + "." + v
		} else {
			return "gopkg.in/" + repo.User + "/" + repo.Name + "." + v
		}
	}
}


var patternOld = regexp.MustCompile(`^/(?:([a-z0-9][-a-z0-9]+)/)?((?:v0|v[1-9][0-9]*)(?:\.0|\.[1-9][0-9]*){0,2}(?:-unstable)?)/([a-zA-Z][-a-zA-Z0-9]*)(?:\.git)?((?:/[a-zA-Z][-a-zA-Z0-9]*)*)$`)
var patternNew = regexp.MustCompile(`^/(?:([a-zA-Z0-9][-a-zA-Z0-9]+)/)?([a-zA-Z][-.a-zA-Z0-9]*)\.((?:v0|v[1-9][0-9]*)(?:\.0|\.[1-9][0-9]*){0,2}(?:-unstable)?)(?:\.git)?((?:/[a-zA-Z0-9][-.a-zA-Z0-9]*)*)$`)

func handler(resp http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/health-check" {
		resp.Write([]byte("ok"))
		return
	}

	log.Printf("%s requested %s", req.RemoteAddr, req.URL)

	if req.URL.Path == "/" {
		resp.Header().Set("Location", "https://labix.org/gopkg.in")
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
		// "/v2/name" <= "/name.v2"
		m[2], m[3] = m[3], m[2]
		oldFormat = true
	}

	if strings.Contains(m[3], ".") {
		sendNotFound(resp, "Import paths take the major version only (.%s instead of .%s); see docs at gopkg.in for the reasoning.",
			m[3][:strings.Index(m[3], ".")], m[3])
		return
	}

	repo := &Repo{
		User:        m[1],
		Name:        m[2],
		SubPath:     m[4],
		OldFormat:   oldFormat,
		FullVersion: InvalidVersion,
	}

	if r, ok := redirect[repoBase{repo.User, repo.Name}]; ok {
		repo.RedirUser, repo.RedirName = repo.User, repo.Name
		repo.User, repo.Name = r.user, r.name
	}

	var ok bool
	repo.MajorVersion, ok = parseVersion(m[3])
	if !ok {
		sendNotFound(resp, "Version %q improperly considered invalid; please warn the service maintainers.", m[3])
		return
	}

	var changed []byte
	var versions VersionList
	original, err := fetchRefs(repo)
	if err == nil {
		changed, versions, err = changeRefs(original, repo.MajorVersion)
		repo.SetVersions(versions)
	}

	switch err {
	case nil:
		// all ok
	case ErrNoRepo:
		sendNotFound(resp, "GitHub repository not found at https://%s", repo.GitHubRoot())
		return
	case ErrNoVersion:
		major := repo.MajorVersion
		suffix := ""
		if major.Unstable {
			major.Unstable = false
			suffix = unstableSuffix
		}
		v := major.String()
		sendNotFound(resp, `GitHub repository at https://%s has no branch or tag "%s%s", "%s.N%s" or "%s.N.M%s"`, repo.GitHubRoot(), v, suffix, v, suffix, v, suffix)
		return
	default:
		resp.WriteHeader(http.StatusBadGateway)
		resp.Write([]byte(fmt.Sprintf("Cannot obtain refs from GitHub: %v", err)))
		return
	}

	if repo.SubPath == "/git-upload-pack" {
		proxyUploadPack(resp, req, repo)
		return
	}

	if repo.SubPath == "/info/refs" {
		resp.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
		resp.Write(changed)
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

const refsSuffix = ".git/info/refs?service=git-upload-pack"

func proxyUploadPack(resp http.ResponseWriter, req *http.Request, repo *Repo) {
	preq, err := http.NewRequest(req.Method, "https://"+repo.GitHubRoot()+"/git-upload-pack", req.Body)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("Cannot create GitHub request: %v", err)))
		return
	}
	preq.Header = req.Header
	presp, err := bulkClient.Do(preq)
	if err != nil {
		resp.WriteHeader(http.StatusBadGateway)
		resp.Write([]byte(fmt.Sprintf("Cannot obtain data pack from GitHub: %v", err)))
		return
	}
	defer presp.Body.Close()

	header := resp.Header()
	for key, values := range presp.Header {
		header[key] = values
	}
	resp.WriteHeader(presp.StatusCode)

	// Ignore errors. Dropped connections are usual and will make this fail.
	_, err = io.Copy(resp, presp.Body)
	if err != nil {
		log.Printf("Error copying data from GitHub: %v", err)
	}
}

var ErrNoRepo = errors.New("repository not found in GitHub")
var ErrNoVersion = errors.New("version reference not found in GitHub")

func fetchRefs(repo *Repo) (data []byte, err error) {
	resp, err := httpClient.Get("https://" + repo.GitHubRoot() + refsSuffix)
	if err != nil {
		return nil, fmt.Errorf("cannot talk to GitHub: %v", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		// ok
	case 401, 404:
		return nil, ErrNoRepo
	default:
		return nil, fmt.Errorf("error from GitHub: %v", resp.Status)
	}

	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading from GitHub: %v", err)
	}
	return data, err
}

func changeRefs(data []byte, major Version) (changed []byte, versions VersionList, err error) {
	var hlinei, hlinej int // HEAD reference line start/end
	var mlinei, mlinej int // master reference line start/end
	var vrefhash string
	var vrefname string
	var vrefv = InvalidVersion

	// Record all available versions, the locations of the master and HEAD lines,
	// and details of the best reference satisfying the requested major version.
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

		if name == "HEAD" {
			hlinei = i
			hlinej = j
		}
		if name == "refs/heads/master" {
			mlinei = i
			mlinej = j
		}

		if strings.HasPrefix(name, "refs/heads/v") || strings.HasPrefix(name, "refs/tags/v") {
			if strings.HasSuffix(name, "^{}") {
				// Annotated tag is peeled off and overrides the same version just parsed.
				name = name[:len(name)-3]
			}
			v, ok := parseVersion(name[strings.IndexByte(name, 'v'):])
			if ok && major.Contains(v) && (v == vrefv || !vrefv.IsValid() || vrefv.Less(v)) {
				vrefv = v
				vrefhash = sdata[hashi:hashj]
				vrefname = name
			}
			if ok {
				versions = append(versions, v)
			}
		}
	}

	// If there were absolutely no versions, and v0 was requested, accept the master as-is.
	if len(versions) == 0 && major == (Version{0, -1, -1, false}) {
		return data, nil, nil
	}

	// If the file has no HEAD line or the version was not found, report as unavailable.
	if hlinei == 0 || vrefhash == "" {
		return nil, nil, ErrNoVersion
	}

	var buf bytes.Buffer
	buf.Grow(len(data) + 256)

	// Copy the header as-is.
	buf.Write(data[:hlinei])

	// Extract the original capabilities.
	caps := ""
	if i := strings.Index(sdata[hlinei:hlinej], "\x00"); i > 0 {
		caps = strings.Replace(sdata[hlinei+i+1:hlinej-1], "symref=", "oldref=", -1)
	}

	// Insert the HEAD reference line with the right hash and a proper symref capability.
	var line string
	if strings.HasPrefix(vrefname, "refs/heads/") {
		if caps == "" {
			line = fmt.Sprintf("%s HEAD\x00symref=HEAD:%s\n", vrefhash, vrefname)
		} else {
			line = fmt.Sprintf("%s HEAD\x00symref=HEAD:%s %s\n", vrefhash, vrefname, caps)
		}
	} else {
		if caps == "" {
			line = fmt.Sprintf("%s HEAD\n", vrefhash)
		} else {
			line = fmt.Sprintf("%s HEAD\x00%s\n", vrefhash, caps)
		}
	}
	fmt.Fprintf(&buf, "%04x%s", 4+len(line), line)

	// Insert the master reference line.
	line = fmt.Sprintf("%s refs/heads/master\n", vrefhash)
	fmt.Fprintf(&buf, "%04x%s", 4+len(line), line)

	// Append the rest, dropping the original master line if necessary.
	if mlinei > 0 {
		buf.Write(data[hlinej:mlinei])
		buf.Write(data[mlinej:])
	} else {
		buf.Write(data[hlinej:])
	}

	return buf.Bytes(), versions, nil
}
