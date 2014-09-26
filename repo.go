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

// Root returns the absolute package root, without a schema.
func (repo *Repo) Root() string {
	return repo.VersionRoot(repo.MajorVersion)
}

// Path returns the absolute package path, without a schema.
func (repo *Repo) Path() string {
	return repo.VersionRoot(repo.MajorVersion) + repo.SubPath
}

// VersionRoot returns the absolute package root for the provided version,
// without a schema.
func (repo *Repo) VersionRoot(version Version) string {
	version.Minor = -1
	version.Patch = -1
	v := version.String()
	if repo.OldFormat {
		if repo.User == "" {
			return "/" + v + "/" + repo.Name
		} else {
			return "/" + repo.User + "/" + v + "/" + repo.Name
		}
	} else {
		if repo.User == "" {
			return "/" + repo.Name + "." + v
		} else {
			return "/" + repo.User + "/" + repo.Name + "." + v
		}
	}
}
