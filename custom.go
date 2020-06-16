package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const (
	GitSiteDomain = "github.com" // the target domain
	MyPkgDomain   = "gopkg.in"   // the proxy domain
	MyHomePage    = "https://labix.org/gopkg.in"
)

// autocert whitelist
var MyWhiteList = []string{
	"localhost",
	MyPkgDomain,
	"p1." + MyPkgDomain,
	"p2." + MyPkgDomain,
	"p3." + MyPkgDomain,
}

var (
	pattern = regexp.MustCompile(`^/(?:([a-zA-Z0-9][-a-zA-Z0-9]+)/)?([a-zA-Z][-.a-zA-Z0-9]*)\.` +
		`((?:v0|v[1-9][0-9]*)(?:\.0|\.[1-9][0-9]*){0,2}(?:-unstable)?)(?:\.git)?((?:/[a-zA-Z0-9][-.a-zA-Z0-9]*)*)$`)
	redirects = map[repoBase]repoBase{
		// https://github.com/go-fsnotify/fsnotify/issues/1
		{"", "fsnotify"}: {"fsnotify", "fsnotify"},
	}
)

func GetDefaultDir(site, name string) string {
	switch site {
	default:
		return "mirrors/"
	case "github.com":
		return "go-" + name + "/"
	}
}

func ParseUrlPath(path string) (matches []string, major Version, err error) {
	matches, major = pattern.FindStringSubmatch(path), Version{}
	if matches == nil {
		tpl := "Unsupported URL pattern; see the documentation at gopkg.in for details."
		err = fmt.Errorf(tpl)
		return
	}
	if strings.Contains(matches[3], ".") {
		tpl := "Import paths take the major version only (.%s instead of .%s); see docs at gopkg.in for the reasoning."
		first := matches[3][:strings.Index(matches[3], ".")]
		err = fmt.Errorf(tpl, first, matches[3])
		return
	}
	var ok bool
	if major, ok = parseVersion(matches[3]); !ok {
		tpl := "Version %q improperly considered invalid; please warn the service maintainers."
		err = fmt.Errorf(tpl, matches[3])
	}
	return
}

// parse url and create repo
func CreateRepo(url *url.URL) (*Repo, error) {
	matches, major, err := ParseUrlPath(url.Path)
	if err != nil {
		return nil, err
	}
	repo := &Repo{
		Domain:       GitSiteDomain,
		User:         matches[1],
		Name:         matches[2],
		SubPath:      matches[4],
		MajorVersion: major,
		FullVersion:  InvalidVersion,
	}
	if ok, user, name := Redirect(repo.User, repo.Name); ok {
		repo.RedirUser, repo.RedirName = repo.User, repo.Name
		repo.User, repo.Name = user, name
	}
	return repo, nil
}

// for redir user and name
type repoBase struct {
	user string
	name string
}

func Redirect(user, name string) (bool, string, string) {
	if r, ok := redirects[repoBase{user, name}]; ok {
		return true, r.user, r.name
	}
	return false, "", ""
}
