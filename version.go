package main

import (
	"fmt"
	"strconv"
	"strings"
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
var TooHighVersion = Version{-2, -2, -2}

func parseVersion(s string) (Version, bool) {
	if len(s) < 2 {
		return InvalidVersion, false
	}
	if s[0] != 'v' {
		return InvalidVersion, false
	}
	v := Version{-1, -1, -1}

	parts := strings.Split(s[1:], ".")
	if len(parts) == 0 || len(parts) > 3 {
		return InvalidVersion, false
	}
	for i, part := range parts {
		if len(part) == 0 {
			return InvalidVersion, false
		}
		if len(part) > 1 && part[0] == '0' {
			return InvalidVersion, false
		}
		num, err := strconv.ParseInt(part, 10, 32)
		if err != nil {
			if err.(*strconv.NumError).Err == strconv.ErrRange {
				return TooHighVersion, false
			}
			return InvalidVersion, false
		}
		switch i {
		case 0:
			v.Major = int(num)
		case 1:
			v.Minor = int(num)
		case 2:
			v.Patch = int(num)
		}
	}

	if v == InvalidVersion {
		return InvalidVersion, false
	}
	return v, true
}
