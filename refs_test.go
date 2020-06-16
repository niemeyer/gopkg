package main

import (
	"bytes"
	"fmt"
	"sort"

	. "gopkg.in/check.v1"
)

var _ = Suite(&RefsSuite{})

type RefsSuite struct{}

type refsTest struct {
	summary  string
	original string
	version  string
	changed  string
	versions []string
}

var refsTests = []refsTest{{
	"Version v0 works even without any references",
	reflines(
		"hash1 HEAD",
	),
	"v0",
	reflines(
		"hash1 HEAD",
	),
	nil,
}, {
	"Preserve original capabilities",
	reflines(
		"hash1 HEAD\x00caps",
	),
	"v0",
	reflines(
		"hash1 HEAD\x00caps",
	),
	nil,
}, {
	"Matching major version branch",
	reflines(
		"00000000000000000000000000000000000hash1 HEAD",
		"00000000000000000000000000000000000hash2 refs/heads/v0",
		"00000000000000000000000000000000000hash3 refs/heads/v1",
		"00000000000000000000000000000000000hash4 refs/heads/v2",
	),
	"v1",
	reflines(
		"00000000000000000000000000000000000hash3 HEAD\x00symref=HEAD:refs/heads/v1",
		"00000000000000000000000000000000000hash3 refs/heads/master",
		"00000000000000000000000000000000000hash2 refs/heads/v0",
		"00000000000000000000000000000000000hash3 refs/heads/v1",
		"00000000000000000000000000000000000hash4 refs/heads/v2",
	),
	[]string{"v0", "v1", "v2"},
}, {
	"Matching minor version branch",
	reflines(
		"00000000000000000000000000000000000hash1 HEAD",
		"00000000000000000000000000000000000hash2 refs/heads/v1.1",
		"00000000000000000000000000000000000hash3 refs/heads/v1.3",
		"00000000000000000000000000000000000hash4 refs/heads/v1.2",
	),
	"v1",
	reflines(
		"00000000000000000000000000000000000hash3 HEAD\x00symref=HEAD:refs/heads/v1.3",
		"00000000000000000000000000000000000hash3 refs/heads/master",
		"00000000000000000000000000000000000hash2 refs/heads/v1.1",
		"00000000000000000000000000000000000hash3 refs/heads/v1.3",
		"00000000000000000000000000000000000hash4 refs/heads/v1.2",
	),
	[]string{"v1.1", "v1.2", "v1.3"},
}, {
	"Disable original symref capability",
	reflines(
		"00000000000000000000000000000000000hash1 HEAD\x00foo symref=bar baz",
		"00000000000000000000000000000000000hash2 refs/heads/v1",
	),
	"v1",
	reflines(
		"00000000000000000000000000000000000hash2 HEAD\x00symref=HEAD:refs/heads/v1 foo oldref=bar baz",
		"00000000000000000000000000000000000hash2 refs/heads/master",
		"00000000000000000000000000000000000hash2 refs/heads/v1",
	),
	[]string{"v1"},
}, {
	"Replace original master branch",
	reflines(
		"00000000000000000000000000000000000hash1 HEAD",
		"00000000000000000000000000000000000hash1 refs/heads/master",
		"00000000000000000000000000000000000hash2 refs/heads/v1",
	),
	"v1",
	reflines(
		"00000000000000000000000000000000000hash2 HEAD\x00symref=HEAD:refs/heads/v1",
		"00000000000000000000000000000000000hash2 refs/heads/master",
		"00000000000000000000000000000000000hash2 refs/heads/v1",
	),
	[]string{"v1"},
}, {
	"Matching tag",
	reflines(
		"00000000000000000000000000000000000hash1 HEAD",
		"00000000000000000000000000000000000hash2 refs/tags/v0",
		"00000000000000000000000000000000000hash3 refs/tags/v1",
		"00000000000000000000000000000000000hash4 refs/tags/v2",
	),
	"v1",
	reflines(
		"00000000000000000000000000000000000hash3 HEAD",
		"00000000000000000000000000000000000hash3 refs/heads/master",
		"00000000000000000000000000000000000hash2 refs/tags/v0",
		"00000000000000000000000000000000000hash3 refs/tags/v1",
		"00000000000000000000000000000000000hash4 refs/tags/v2",
	),
	[]string{"v0", "v1", "v2"},
}, {
	"Tag peeling",
	reflines(
		"00000000000000000000000000000000000hash1 HEAD",
		"00000000000000000000000000000000000hash2 refs/heads/master",
		"00000000000000000000000000000000000hash3 refs/tags/v1",
		"00000000000000000000000000000000000hash4 refs/tags/v1^{}",
		"00000000000000000000000000000000000hash5 refs/tags/v2",
	),
	"v1",
	reflines(
		"00000000000000000000000000000000000hash4 HEAD",
		"00000000000000000000000000000000000hash4 refs/heads/master",
		"00000000000000000000000000000000000hash3 refs/tags/v1",
		"00000000000000000000000000000000000hash4 refs/tags/v1^{}",
		"00000000000000000000000000000000000hash5 refs/tags/v2",
	),
	[]string{"v1", "v1", "v2"},
}, {
	"Matching unstable versions",
	reflines(
		"00000000000000000000000000000000000hash1 HEAD",
		"00000000000000000000000000000000000hash2 refs/heads/master",
		"00000000000000000000000000000000000hash3 refs/heads/v1",
		"00000000000000000000000000000000000hash4 refs/heads/v1.1-unstable",
		"00000000000000000000000000000000000hash5 refs/heads/v1.3-unstable",
		"00000000000000000000000000000000000hash6 refs/heads/v1.2-unstable",
		"00000000000000000000000000000000000hash7 refs/heads/v2",
	),
	"v1-unstable",
	reflines(
		"00000000000000000000000000000000000hash5 HEAD\x00symref=HEAD:refs/heads/v1.3-unstable",
		"00000000000000000000000000000000000hash5 refs/heads/master",
		"00000000000000000000000000000000000hash3 refs/heads/v1",
		"00000000000000000000000000000000000hash4 refs/heads/v1.1-unstable",
		"00000000000000000000000000000000000hash5 refs/heads/v1.3-unstable",
		"00000000000000000000000000000000000hash6 refs/heads/v1.2-unstable",
		"00000000000000000000000000000000000hash7 refs/heads/v2",
	),
	[]string{"v1", "v1.1-unstable", "v1.2-unstable", "v1.3-unstable", "v2"},
}}

func reflines(lines ...string) string {
	var buf bytes.Buffer
	buf.WriteString("001e# service=git-upload-pack\n0000")
	for _, l := range lines {
		buf.WriteString(fmt.Sprintf("%04x%s\n", len(l)+5, l))
	}
	buf.WriteString("0000")
	return buf.String()
}

func (s *RefsSuite) TestChangeRefs(c *C) {
	for _, test := range refsTests {
		c.Logf(test.summary)

		v, ok := parseVersion(test.version)
		if !ok {
			c.Fatalf("Test has an invalid version: %q", test.version)
		}

		changed, versions, err := changeRefs([]byte(test.original), v)
		c.Assert(err, IsNil)

		c.Assert(string(changed), Equals, test.changed)

		sort.Sort(versions)

		var vs []string
		for _, v := range versions {
			vs = append(vs, v.String())
		}
		c.Assert(vs, DeepEquals, test.versions)
	}
}
