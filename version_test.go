package gopkg

import (
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&VersionSuite{})

type VersionSuite struct{}

var versionParseTests = []struct {
	major, minor, patch int
	s                   string
}{
	{-1, -1, -1, "v"},
	{-1, -1, -1, "v-1"},
	{-1, -1, -1, "v01"},
	{-1, -1, -1, "v1.01"},
	{-1, -1, -1, "a1"},
	{-1, -1, -1, "v1a"},
	{-1, -1, -1, "v1..2"},
	{-1, -1, -1, "v1.2.3.4"},
	{-1, -1, -1, "v1."},
	{-1, -1, -1, "v1.2."},
	{-1, -1, -1, "v1.2.3."},

	{0, -1, -1,
		"v0"},
	{1, -1, -1,
		"v1"},
	{1, 2, -1,
		"v1.2"},
	{1, 2, 3,
		"v1.2.3"},
	{12, 34, 56,
		"v12.34.56"},
}

func (s *VersionSuite) TestParse(c *C) {
	for _, t := range versionParseTests {
		got, ok := ParseVersion(t.s)
		if t.major == -1 {
			if ok || got != InvalidVersion {
				c.Fatalf("version %q is invalid but parsed as %#v", t.s, got)
			}
		} else {
			want := Version{t.major, t.minor, t.patch}
			if got != want {
				c.Fatalf("version %q must parse as %#v, got %#v", t.s, want, got)
			}
			if got.String() != t.s {
				c.Fatalf("version %q got parsed as %#v and stringified as %q", t.s, got, got.String())
			}
		}
	}
}

var versionLessTests = []struct {
	oneMajor, oneMinor, onePatch int
	twoMajor, twoMinor, twoPatch int
	less                         bool
}{
	{0, 0, 0, 0, 0, 0, false},
	{1, 0, 0, 1, 0, 0, false},
	{1, 0, 0, 1, 1, 0, true},
	{1, 0, 0, 2, 0, 0, true},
	{0, 1, 0, 0, 1, 0, false},
	{0, 1, 0, 0, 1, 1, true},
	{0, 0, 0, 0, 2, 0, true},
	{0, 0, 1, 0, 0, 1, false},
	{0, 0, 1, 0, 0, 2, true},
}

func (s *VersionSuite) TestLess(c *C) {
	for _, t := range versionLessTests {
		one := Version{t.oneMajor, t.oneMinor, t.onePatch}
		two := Version{t.twoMajor, t.twoMinor, t.twoPatch}
		if one.Less(two) != t.less {
			c.Fatalf("version %s < %s returned %v", one, two, !t.less)
		}
	}
}

var versionContainsTests = []struct {
	oneMajor, oneMinor, onePatch int
	twoMajor, twoMinor, twoPatch int
	contains                     bool
}{
	{12, 34, 56, 12, 34, 56, true},
	{12, 34, 56, 12, 34, 78, false},
	{12, 34, -1, 12, 34, 56, true},
	{12, 34, -1, 12, 78, 56, false},
	{12, -1, -1, 12, 34, 56, true},
	{12, -1, -1, 78, 34, 56, false},
}

func (s *VersionSuite) TestContains(c *C) {
	for _, t := range versionContainsTests {
		one := Version{t.oneMajor, t.oneMinor, t.onePatch}
		two := Version{t.twoMajor, t.twoMinor, t.twoPatch}
		if one.Contains(two) != t.contains {
			c.Fatalf("version %s.Contains(%s) returned %v", one, two, !t.contains)
		}
	}
}

func (s *VersionSuite) TestIsValid(c *C) {
	c.Assert(InvalidVersion.IsValid(), Equals, false)
	c.Assert(Version{0, 0, 0}.IsValid(), Equals, true)
}
