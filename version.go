package main

import (
	"fmt"
)

type Version struct {
	Major, Minor, Patch int
}

func (v Version) String() string {
	if v.Major < 0 {
		panic(fmt.Sprintf("cannot stringify invalid version (major is %d)", v.Major))
	}
	if v.Minor < 0 {
		return fmt.Sprintf("v%d", v.Major)
	}
	if v.Patch < 0 {
		return fmt.Sprintf("v%d.%d", v.Major, v.Minor)
	}
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v Version) Less(other Version) bool {
	if v.Major != other.Major {
		return v.Major < other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor < other.Minor
	}
	return v.Patch < other.Patch
}

func (v Version) Contains(other Version) bool {
	if v.Patch != -1 {
		return v == other
	}
	if v.Minor != -1 {
		return v.Major == other.Major && v.Minor == other.Minor
	}
	return v.Major == other.Major
}

func (v Version) IsValid() bool {
	return v == InvalidVersion
}

var InvalidVersion = Version{-1, -1, -1}

func parseVersion(s string) (Version, bool) {
	if len(s) < 2 {
		return InvalidVersion, false
	}
	if s[0] != 'v' {
		return InvalidVersion, false
	}
	v := Version{-1, -1, -1}
	i := 1
	v.Major, i = parseVersionPart(s, i)
	if i < 0 {
		return InvalidVersion, false
	}
	if i == len(s) {
		return v, true
	}
	v.Minor, i = parseVersionPart(s, i)
	if i < 0 {
		return InvalidVersion, false
	}
	if i == len(s) {
		return v, true
	}
	v.Patch, i = parseVersionPart(s, i)
	if i < 0 || i < len(s) {
		return InvalidVersion, false
	}
	return v, true
}

func parseVersionPart(s string, i int) (part int, newi int) {
	dot := i
	for dot < len(s) && s[dot] != '.' {
		dot++
	}
	if dot == i || dot-i > 1 && s[i] == '0' {
		return -1, -1
	}
	for i < len(s) {
		if s[i] < '0' || s[i] > '9' {
			return -1, -1
		}
		part *= 10
		part += int(s[i] - '0')
		if part < 0 {
			return -1, -1
		}
		i++
		if i+1 < len(s) && s[i] == '.' {
			return part, i + 1
		}
	}
	return part, i
}

// VersionList implements sort.Interface
type VersionList []Version

func (vl VersionList) Len() int           { return len(vl) }
func (vl VersionList) Less(i, j int) bool { return vl[i].Less(vl[j]) }
func (vl VersionList) Swap(i, j int)      { vl[i], vl[j] = vl[j], vl[i] }
