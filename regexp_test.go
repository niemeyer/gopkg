package main

import (
	. "gopkg.in/check.v1"
)

var _ = Suite(&RegexpSuite{})

type RegexpSuite struct{}

// Test for the patternNew regexp defined in the main.go source file.
var regexpPatternNewTests = []struct {
	path   string
	expect []string
}{
	// Basic paths:
	{"/pkg.v3", []string{"/pkg.v3", "", "pkg", "v3", ""}},
	{"/user/pkg.v3", []string{"/user/pkg.v3", "user", "pkg", "v3", ""}},

	// Subpackage paths:
	{"/pkg.v3/sub/pkg", []string{"/pkg.v3/sub/pkg", "", "pkg", "v3", "/sub/pkg"}},
	{"/user/pkg.v3/subpkg", []string{"/user/pkg.v3/subpkg", "user", "pkg", "v3", "/subpkg"}},

	// Invalid paths:
	{"/a", []string{}},
	{"/a/b", []string{}},
	{"/a/b/pkg.v3", []string{}},
}

func (s *RegexpSuite) TestPatternNew(c *C) {
	for _, t := range regexpPatternNewTests {
		m := patternNew.FindStringSubmatch(t.path)
		if len(m) != len(t.expect) {
			c.Fatalf("got %#v expected %#v", m, t.expect)
		}
		for i, s := range m {
			if s != t.expect[i] {
				c.Fatalf("got %#v expected %#v", m, t.expect)
			}
		}
	}
}
