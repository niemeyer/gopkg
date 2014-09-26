package gopkg

// Repo represents a source code repository on GitHub.
type Repo struct {
	User         string
	Name         string
	SubPath      string
	OldFormat    bool // The old /v2/pkg format.
	MajorVersion Version
	AllVersions  VersionList
}

// GitHubRoot returns the repository root at GitHub, without a schema.
func (repo *Repo) GitHubRoot() string {
	if repo.User == "" {
		return "github.com/go-" + repo.Name + "/" + repo.Name
	} else {
		return "github.com/" + repo.User + "/" + repo.Name
	}
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

