package gopkg

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"text/template"
)

var gogetTemplate = template.Must(template.New("").Parse(`
<html>
<head>
<meta name="go-import" content="{{.Host}}{{.Repo.Root}} git https://{{.Host}}{{.Repo.Root}}">
</head>
<body>
go get {{.Host}}{{.Repo.Path}}
</body>
</html>
`))

var patternOld = regexp.MustCompile(`^/(?:([a-z0-9][-a-z0-9]+)/)?((?:v0|v[1-9][0-9]*)(?:\.0|\.[1-9][0-9]*){0,2})/([a-zA-Z][-a-zA-Z0-9]*)(?:\.git)?((?:/[a-zA-Z][-a-zA-Z0-9]*)*)$`)
var patternNew = regexp.MustCompile(`^/(?:([a-zA-Z0-9][-a-zA-Z0-9]+)/)?([a-zA-Z][-.a-zA-Z0-9]*)\.((?:v0|v[1-9][0-9]*)(?:\.0|\.[1-9][0-9]*){0,2})(?:\.git)?((?:/[a-zA-Z0-9][-.a-zA-Z0-9]*)*)$`)

// Handler is responsible for handling gopkg HTTP requests.
type Handler struct {
	// The HTTP client used to make request to GitHub. If nil then
	// http.DefaultClient will be used.
	Client *http.Client
}

// Handle effectively asks gopkg to handle the HTTP request if it can. The
// return parameter handled informs you of whether or not the request was
// completely handled and a response was written.
//
// The returned repo contains information regarding the repository of the
// requested package. If the request was not handled and the returned repo is
// non-nil, then it means for example that:
//
//  gopkg.in/pkg.v1
//
// was requested by the client but the client is not the Go tool nor git, but
// rather something else (e.g. a web browser) so you could for example respond
// with a package page.
func (h *Handler) Handle(resp http.ResponseWriter, req *http.Request) (repo *Repo, handled bool) {
	m := patternNew.FindStringSubmatch(req.URL.Path)
	oldFormat := false
	if m == nil {
		m = patternOld.FindStringSubmatch(req.URL.Path)
		if m == nil {
			// Not a valid package URL.
			return nil, false
		}
		m[2], m[3] = m[3], m[2]
		oldFormat = true
	}

	if strings.Contains(m[3], ".") {
		sendNotFound(resp, "Import paths take the major version only (.%s instead of .%s); see docs at gopkg.in for the reasoning.",
			m[3][:strings.Index(m[3], ".")], m[3])
		return nil, true
	}

	repo = &Repo{
		User:      m[1],
		Name:      m[2],
		SubPath:   m[4],
		OldFormat: oldFormat,
	}

	var ok bool
	repo.MajorVersion, ok = ParseVersion(m[3])
	if !ok {
		sendNotFound(resp, "Version %q improperly considered invalid; please warn the service maintainers.", m[3])
		return nil, true
	}

	var err error
	var refs []byte
	refs, repo.AllVersions, err = HackedRefs(h.Client, repo)
	switch err {
	case nil:
		// all ok
	case ErrNoRepo:
		sendNotFound(resp, "GitHub repository not found at https://%s", repo.GitHubRoot())
		return nil, true
	case ErrNoVersion:
		v := repo.MajorVersion.String()
		sendNotFound(resp, `GitHub repository at https://%s has no branch or tag "%s", "%s.N" or "%s.N.M"`, repo.GitHubRoot(), v, v, v)
		return nil, true
	default:
		resp.WriteHeader(http.StatusBadGateway)
		resp.Write([]byte(fmt.Sprintf("Cannot obtain refs from GitHub: %v", err)))
		return nil, true
	}

	if repo.SubPath == "/git-upload-pack" {
		resp.Header().Set("Location", "https://"+repo.GitHubRoot()+"/git-upload-pack")
		resp.WriteHeader(http.StatusMovedPermanently)
		return repo, true
	}

	if repo.SubPath == "/info/refs" {
		resp.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
		resp.Write(refs)
		return repo, true
	}

	resp.Header().Set("Content-Type", "text/html")
	if req.FormValue("go-get") == "1" {
		// execute simple template when this is a go-get request
		err = gogetTemplate.Execute(resp, map[string]interface{}{
			"Repo": repo,
			"Host": req.URL.Host,
		})
		if err != nil {
			log.Printf("error executing go get template: %s\n", err)
		}
		return repo, true
	}
	return repo, false
}

func sendNotFound(resp http.ResponseWriter, msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	resp.WriteHeader(http.StatusNotFound)
	resp.Write([]byte(msg))
}
