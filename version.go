package main

import (
	"fmt"
)

// Version represents a version number.
// An element that is not present is represented as -1.
type Version struct {
	Major    int
	Minor    int
	Patch    int
	Unstable bool
}

const unstableSuffix = "-unstable"

func (v Version) String() string {
	if v.Major < 0 {
		panic(fmt.Sprintf("cannot stringify invalid version (major is %d)", v.Major))
	}
	suffix := ""
	if v.Unstable {
		suffix = unstableSuffix
	}
	if v.Minor < 0 {
		return fmt.Sprintf("v%d%s", v.Major, suffix)
	}
	if v.Patch < 0 {
		return fmt.Sprintf("v%d.%d%s", v.Major, v.Minor, suffix)
	}
	return fmt.Sprintf("v%d.%d.%d%s", v.Major, v.Minor, v.Patch, suffix)
}

// Less returns whether v is less than other.
func (v Version) Less(other Version) bool {
	if v.Major != other.Major {
		return v.Major < other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor < other.Minor
	}
	if v.Patch != other.Patch {
		return v.Patch < other.Patch
	}
	return v.Unstable && !other.Unstable
}

// Contains returns whether version v contains version other.
// Version v is defined to contain version other when they both have the same Major
// version and v.Minor and v.Patch are either undefined or are equal to other's.
//
// For example, Version{1, 1, -1} contains both Version{1, 1, -1} and Version{1, 1, 2},
// but not Version{1, -1, -1} or Version{1, 2, -1}.
//
// Unstable versions (-unstable) only contain unstable versions, and stable
// versions only contain stable versions.
func (v Version) Contains(other Version) bool {
	if v.Unstable != other.Unstable {
		return false
	}
	if v.Patch != -1 {
		return v == other
	}
	if v.Minor != -1 {
		return v.Major == other.Major && v.Minor == other.Minor
	}
	return v.Major == other.Major
}

func (v Version) IsValid() bool {
	return v != InvalidVersion
}

// InvalidVersion represents a version that can't be parsed.
var InvalidVersion = Version{-1, -1, -1, false}

func parseVersion(s string) (v Version, ok bool) {
	v = InvalidVersion
	if len(s) < 2 {
		return
	}
	if s[0] != 'v' {
		return
	}
	vout := InvalidVersion
	unstable := false
	i := 1
	for _, vptr := range []*int{&vout.Major, &vout.Minor, &vout.Patch} {
		*vptr, unstable, i = parseVersionPart(s, i)
		if i < 0 {
			return
		}
		if i == len(s) {
			vout.Unstable = unstable
			return vout, true
		}
	}
	return
}

func parseVersionPart(s string, i int) (part int, unstable bool, newi int) {
	j := i
	for j < len(s) && s[j] != '.' && s[j] != '-' {
		j++
	}
	if j == i || j-i > 1 && s[i] == '0' {
		return -1, false, -1
	}
	c := s[i]
	for {
		if c < '0' || c > '9' {
			return -1, false, -1
		}
		part *= 10
		part += int(c - '0')
		if part < 0 {
			return -1, false, -1
		}
		i++
		if i == len(s) {
			return part, false, i
		}
		c = s[i]
		if i+1 < len(s) {
			if c == '.' {
				return part, false, i + 1
			}
			if c == '-' && s[i:] == unstableSuffix {
				return part, true, i + len(unstableSuffix)
			}
		}
	}
	panic("unreachable")
}

// VersionList implements sort.Interface
type VersionList []Version

func (vl VersionList) Len() int           { return len(vl) }
func (vl VersionList) Less(i, j int) bool { return vl[i].Less(vl[j]) }
func (vl VersionList) Swap(i, j int)      { vl[i], vl[j] = vl[j], vl[i] }
