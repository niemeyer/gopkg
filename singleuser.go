package gopkg

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var patternSingleUser = regexp.MustCompile(`^(?:([a-zA-Z0-9][-a-zA-Z0-9]+)/)?([a-zA-Z][-.a-zA-Z0-9]*)\.((?:v0|v[1-9][0-9]*)(?:\.0|\.[1-9][0-9]*){0,2})(?:\.git)?((?:/[a-zA-Z0-9][-.a-zA-Z0-9]*)*)$`)

// SingleUser returns a URL Matcher that operates on a single GitHub user or
// organization. For instance if the service was running at example.com and the
// user string was "bob", it would match URLS in the pattern of:
//
//  example.com/pkg.v3 → github.com/bob/pkg (branch/tag v3, v3.N, or v3.N.M)
//  example.com/folder/pkg.v3 → github.com/bob/folder-pkg (branch/tag v3, v3.N, or v3.N.M)
//  example.com/multi/folder/pkg.v3 → github.com/bob/multi-folder-pkg (branch/tag v3, v3.N, or v3.N.M)
//  example.com/folder/pkg.v3/subpkg → github.com/bob/folder-pkg (branch/tag v3, v3.N, or v3.N.M)
//  example.com/pkg.v3/folder/subpkg → github.com/bob/pkg (branch/tag v3, v3.N, or v3.N.M)
//
func SingleUser(user string) Matcher {
	f := func(url *url.URL) (repo *Repo, err error) {
		// Split the path elements. If any element is an empty string then it
		// is because there are two consecutive slashes ("/a//b/c") or the path
		// ends with a trailing slash ("example.com/pkg.v1/").
		//
		// If more than one element contains a version match then it's invalid
		// as well ("example.com/foo.v1/bar.v1/something.v2").
		var (
			s           = strings.Split(url.Path, "/")
			versionElem = -1   // Index of version element in s.
			version     string // e.g. "v3".
			pkgName     string // e.g. "pkg" from "foo/bar/pkg.v3/sub".
		)
		for index, elem := range s {
			if len(elem) == 0 {
				// Path has two consecutive slashes ("/a//b/c") or ends with
				// trailing slash.
				return nil, ErrNotPackageURL
			}
			m := patternSingleUser.FindStringSubmatch(elem)
			if m != nil {
				if versionElem != -1 {
					// Multiple versions in path.
					return nil, ErrNotPackageURL
				}
				pkgName = m[2]
				version = m[3]
				versionElem = index
			}
		}
		if versionElem == -1 {
			// No version in path.
			return nil, ErrNotPackageURL
		}

		// Check for invalid requests for e.g. "pkg.v3.1"
		if strings.Contains(version, ".") {
			err = fmt.Errorf("Import paths take the major version only (.%s instead of .%s); see docs at gopkg.in for the reasoning.",
				version[:strings.Index(version, ".")], version)
			return
		}

		// Everything in the path up to the path element index [found] is part
		// of the repository name. We replace all slashes with dashes (the same
		// thing GitHub does if you try to create a repository with slashes in
		// the name).
		repo = &Repo{
			User:    user,
			Name:    strings.Join(append(s[:versionElem], pkgName), "-"),
			SubPath: strings.Join(s[versionElem+1:], "/"),
		}

		// Parse package version.
		var ok bool
		repo.MajorVersion, ok = ParseVersion(version)
		if !ok {
			err = fmt.Errorf("Version %q improperly considered invalid; please warn the service maintainers.", version)
			return
		}
		return
	}
	return MatcherFunc(f)
}
