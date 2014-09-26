package gopkg

import (
	"fmt"
)

// Version represents a version number.
// An element that is not present is represented as -1.
type Version struct {
	Major, Minor, Patch int
}

// String returns a string representation of this version like:
//  v1.2.1
//
// If minor or patch components of this version are less than zero then a
// simplified string is returned, for example:
//  v1 (missing minor)
//  v1.2 (missing revision)
//
// If the major version is invalid (less than zero) a panic will occur.
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

// Less returns whether v is less than other.
func (v Version) Less(other Version) bool {
	if v.Major != other.Major {
		return v.Major < other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor < other.Minor
	}
	return v.Patch < other.Patch
}

// Contains returns whether version v contains version other.
// Version v is defined to contain version other when they both have the same Major
// version and v.Minor and v.Patch are either undefined or are equal to other's.
//
// For example, Version{1, 1, -1} contains both Version{1, 1, -1} and Version{1, 1, 2},
// but not Version{1, -1, -1} or Version{1, 2, -1}.
func (v Version) Contains(other Version) bool {
	if v.Patch != -1 {
		return v == other
	}
	if v.Minor != -1 {
		return v.Major == other.Major && v.Minor == other.Minor
	}
	return v.Major == other.Major
}

// IsValid is short-handed for:
//  v != InvalidVersion
func (v Version) IsValid() bool {
	return v != InvalidVersion
}

// InvalidVersion represents a version that can't be parsed.
var InvalidVersion = Version{-1, -1, -1}

// ParseVersion parses a version string prefixed with 'v'. If the version
// string cannot be parsed then (InvalidVersion, false) is returned.
func ParseVersion(s string) (Version, bool) {
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
