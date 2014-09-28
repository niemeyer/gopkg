package gopkg

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var ErrNotPackageURL = errors.New("not a valid package URL")

// Matcher is responsible for matching a URL and generating a Repo structure
// with Name and (optionally) User fields filled out.
type Matcher interface {
	// Match should match the given URL and return a repo structure with at
	// least the Name and MajorVersion fields filled in.
	//
	// If the URL is not a valid package URL repo=nil, err=ErrNotPackageURL
	// should be returned.
	Match(url *url.URL) (repo *Repo, err error)
}

// MatcherFunc implements the Matcher interface by simply invoking the
// function.
type MatcherFunc func(url *url.URL) (repo *Repo, err error)

// Match simply invokes the function, m.
func (m MatcherFunc) Match(url *url.URL) (repo *Repo, err error) {
	return m(url)
}

var patternOld = regexp.MustCompile(`^/(?:([a-z0-9][-a-z0-9]+)/)?((?:v0|v[1-9][0-9]*)(?:\.0|\.[1-9][0-9]*){0,2})/([a-zA-Z][-a-zA-Z0-9]*)(?:\.git)?((?:/[a-zA-Z][-a-zA-Z0-9]*)*)$`)
var patternNew = regexp.MustCompile(`^/(?:([a-zA-Z0-9][-a-zA-Z0-9]+)/)?([a-zA-Z][-.a-zA-Z0-9]*)\.((?:v0|v[1-9][0-9]*)(?:\.0|\.[1-9][0-9]*){0,2})(?:\.git)?((?:/[a-zA-Z0-9][-.a-zA-Z0-9]*)*)$`)

// DefaultMatcher implements the default gopkg URL scheme.
var DefaultMatcher = MatcherFunc(defaultMatcher)

func defaultMatcher(url *url.URL) (repo *Repo, err error) {
	m := patternNew.FindStringSubmatch(url.Path)
	oldFormat := false
	if m == nil {
		m = patternOld.FindStringSubmatch(url.Path)
		if m == nil {
			// Not a valid package URL.
			return nil, ErrNotPackageURL
		}
		m[2], m[3] = m[3], m[2]
		oldFormat = true
	}

	if strings.Contains(m[3], ".") {
		err = fmt.Errorf("Import paths take the major version only (.%s instead of .%s); see docs at gopkg.in for the reasoning.",
			m[3][:strings.Index(m[3], ".")], m[3])
		return
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
		err = fmt.Errorf("Version %q improperly considered invalid; please warn the service maintainers.", m[3])
		return
	}
	return
}
