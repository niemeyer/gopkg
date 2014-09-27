package gopkg

import (
	"fmt"
	"log"
	"net/http"
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

// Handler is responsible for handling gopkg HTTP requests.
type Handler struct {
	// The literal host string (e.g. "gopkg.in"). It must be present.
	Host string

	// The URL matcher, if nil then DefaultMatcher is used.
	Matcher

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
//
// Handle will panic if the handler's Host string is empty.
func (h *Handler) Handle(resp http.ResponseWriter, req *http.Request) (repo *Repo, handled bool) {
	// The host string must be non-empty.
	if h.Host == "" {
		panic("Handle(): Empty Host string in Handler.")
	}

	// If the Matcher is nil then we use DefaultMatcher.
	matcher := h.Matcher
	if matcher == nil {
		matcher = DefaultMatcher
	}

	// Perform URL matching.
	var err error
	repo, err = matcher.Match(req.URL)
	if err == ErrNotPackageURL {
		// Not a valid package URL.
		return nil, false
	}
	if err != nil {
		// Some other URL matching error, send it to the client.
		sendNotFound(resp, err.Error())
		return nil, true
	}

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
			"Host": h.Host,
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
