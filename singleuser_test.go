package gopkg

import (
	"net/url"
	"testing"
)

var singleUserRegexpTests = []struct {
	full, name, version string
	valid               bool
}{
	{"pkg.v3.2.1", "pkg", "v3.2.1", true},
	{"pkg.v3.2", "pkg", "v3.2", true},
	{"pkg.v3", "pkg", "v3", true},
	{"abc/def.v3.2", "def", "v3.2", true},
	{"abc/other.v3/def", "other", "v3", true},
	{"pkg", "", "", false},
}

// Test the patternSingleUser regexp.
func TestSingleUserRegexp(t *testing.T) {
	for _, tst := range singleUserRegexpTests {
		m := patternSingleUser.FindStringSubmatch(tst.full)

		// Validate the return value.
		if !tst.valid && m != nil {
			t.Logf("%q\n", tst.full)
			t.Fatal("want no match, got", m)
		} else if tst.valid && m == nil {
			t.Logf("%q\n", tst.full)
			t.Fatal("want match, got nil")
		}
		if !tst.valid {
			continue
		}

		// Validate length of matched data.
		if len(m) != 5 {
			t.Logf("%q\n", tst.full)
			for i, m := range m {
				t.Logf("%d. %q\n", i, m)
			}
			t.Fatal("expected 5 values, but regex matched", len(m))
		}

		// Validate the actual data.
		if m[2] != tst.name {
			t.Logf("%q\n", tst.full)
			t.Fatalf("expected name %q, got %q", tst.name, m[2])
		} else if m[3] != tst.version {
			t.Logf("%q\n", tst.full)
			t.Fatalf("expected version %q, got %q", tst.version, m[3])
		}
	}
}

var singleUserTests = []struct {
	url, github, user string
	valid             bool
}{
	{"pkg.v3", "github.com/bob/pkg", "bob", true},
	{"folder/pkg.v3", "github.com/bob/folder-pkg", "carol", true},
	{"multi/folder/pkg.v3", "github.com/bob/multi-folder-pkg", "george", true},
	{"folder/pkg.v3/subpkg", "github.com/bob/folder-pkg", "henry", true},
	{"pkg.v3/folder/subpkg", "github.com/bob/pkg", "henry", true},
	{"pkg.v3.1/no/minor", "", "", false},
	{"", "", "", false},
	{"", "", "", false},
	{"a", "", "", false},
	{"a", "", "", false},
	{"a/b", "", "", false},
	{"a/b/", "", "", false},
	{"a/b.v3/", "", "", false},
	{"a.v3/b/c.v3", "", "", false},
}

// Tests the SingleUser URL matcher.
func TestSingleUser(t *testing.T) {
	for _, tst := range singleUserTests {
		// Parse test case URL.
		u, err := url.Parse(tst.url)
		if err != nil {
			t.Fatal(err)
		}

		// Create a matcher and perform matching.
		matcher := SingleUser(tst.user)
		repo, err := matcher.Match(u)
		if tst.valid && err != nil {
			t.Log(u)
			t.Fatal("Test is valid but matcher returned:", err)
		} else if !tst.valid && err == nil {
			t.Log(u)
			t.Fatal("Test is invalid but matcher returned nil error!")
		}
		if !tst.valid {
			continue
		}

		// Validate repo structure.
		if repo == nil {
			t.Log(u)
			t.Fatal("nil repo")
		}
		if repo.Name == "" {
			t.Log(u)
			t.Fatal("missing repo name")
		}
		var zeroValueVersion Version
		if repo.MajorVersion == zeroValueVersion {
			t.Log(u)
			t.Fatal("zero-value repo version")
		}
		if !repo.MajorVersion.IsValid() {
			t.Log(u)
			t.Fatal("invalid repo version")
		}
	}
}
