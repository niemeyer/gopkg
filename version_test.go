package main

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&VersionSuite{})

type VersionSuite struct{}

var versionParseTests = []struct {
	major int
	minor int
	patch int
	dev   bool
	s     string
}{
	{-1, -1, -1, false, "v"},
	{-1, -1, -1, false, "v-1"},
	{-1, -1, -1, false, "v-deb"},
	{-1, -1, -1, false, "v01"},
	{-1, -1, -1, false, "v1.01"},
	{-1, -1, -1, false, "a1"},
	{-1, -1, -1, false, "v1a"},
	{-1, -1, -1, false, "v1..2"},
	{-1, -1, -1, false, "v1.2.3.4"},
	{-1, -1, -1, false, "v1."},
	{-1, -1, -1, false, "v1.2."},
	{-1, -1, -1, false, "v1.2.3."},

	{0, -1, -1, false, "v0"},
	{0, -1, -1, true, "v0-unstable"},
	{1, -1, -1, false, "v1"},
	{1, -1, -1, true, "v1-unstable"},
	{1, 2, -1, false, "v1.2"},
	{1, 2, -1, true, "v1.2-unstable"},
	{1, 2, 3, false, "v1.2.3"},
	{1, 2, 3, true, "v1.2.3-unstable"},
	{12, 34, 56, false, "v12.34.56"},
	{12, 34, 56, true, "v12.34.56-unstable"},
}

func (s *VersionSuite) TestParse(c *C) {
	for _, t := range versionParseTests {
		got, ok := parseVersion(t.s)
		if t.major == -1 {
			if ok || got != InvalidVersion {
				c.Fatalf("version %q is invalid but parsed as %#v", t.s, got)
			}
		} else {
			want := Version{t.major, t.minor, t.patch, t.dev}
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
	oneUnstable                  bool
	twoMajor, twoMinor, twoPatch int
	twoUnstable, less            bool
}{
	{0, 0, 0, false, 0, 0, 0, false, false},
	{1, 0, 0, false, 1, 0, 0, false, false},
	{1, 0, 0, false, 1, 1, 0, false, true},
	{1, 0, 0, false, 2, 0, 0, false, true},
	{0, 1, 0, false, 0, 1, 0, false, false},
	{0, 1, 0, false, 0, 1, 1, false, true},
	{0, 0, 0, false, 0, 2, 0, false, true},
	{0, 0, 1, false, 0, 0, 1, false, false},
	{0, 0, 1, false, 0, 0, 2, false, true},

	{0, 0, 0, false, 0, 0, 0, true, false},
	{0, 0, 0, true, 0, 0, 0, false, true},
	{0, 0, 1, true, 0, 0, 0, false, false},
}

func (s *VersionSuite) TestLess(c *C) {
	for _, t := range versionLessTests {
		one := Version{t.oneMajor, t.oneMinor, t.onePatch, t.oneUnstable}
		two := Version{t.twoMajor, t.twoMinor, t.twoPatch, t.twoUnstable}
		if one.Less(two) != t.less {
			c.Fatalf("version %s < %s returned %v", one, two, !t.less)
		}
	}
}

var versionContainsTests = []struct {
	oneMajor, oneMinor, onePatch int
	oneUnstable                  bool
	twoMajor, twoMinor, twoPatch int
	twoUnstable, contains        bool
}{
	{12, 34, 56, false, 12, 34, 56, false, true},
	{12, 34, 56, false, 12, 34, 78, false, false},
	{12, 34, -1, false, 12, 34, 56, false, true},
	{12, 34, -1, false, 12, 78, 56, false, false},
	{12, -1, -1, false, 12, 34, 56, false, true},
	{12, -1, -1, false, 78, 34, 56, false, false},

	{12, -1, -1, true, 12, -1, -1, false, false},
	{12, -1, -1, false, 12, -1, -1, true, false},
}

func (s *VersionSuite) TestContains(c *C) {
	for _, t := range versionContainsTests {
		one := Version{t.oneMajor, t.oneMinor, t.onePatch, t.oneUnstable}
		two := Version{t.twoMajor, t.twoMinor, t.twoPatch, t.twoUnstable}
		if one.Contains(two) != t.contains {
			c.Fatalf("version %s.Contains(%s) returned %v", one, two, !t.contains)
		}
	}
}

func (s *VersionSuite) TestIsValid(c *C) {
	c.Assert(InvalidVersion.IsValid(), Equals, false)
	c.Assert(Version{0, 0, 0, false}.IsValid(), Equals, true)
}
