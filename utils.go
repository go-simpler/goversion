package main

import (
	"os"
	"strconv"
	"strings"
)

// cutFromPath cuts the given value from a $PATH-like string.
func cutFromPath(path, value string) string {
	var list []string
	for _, v := range strings.Split(path, string(os.PathListSeparator)) {
		if v != value {
			list = append(list, v)
		}
	}
	return strings.Join(list, string(os.PathListSeparator))
}

// latestPatches filters the sorted versions list,
// returning only the latest patch for each minor version.
func latestPatches(vs []string) []string {
	if l := len(vs); l == 0 || l == 1 {
		return vs
	}

	latest := []string{vs[0]}
	prev, _, _ := parseVersion(vs[0])

	for i := 1; i < len(vs); i++ {
		curr, _, _ := parseVersion(vs[i])
		if prev != curr {
			prev = curr
			latest = append(latest, vs[i])
		}
	}

	return latest
}

// the following code is a modified version of the functions from
// https://github.com/golang/website/blob/master/internal/dl/dl.go

func versionLess(a, b string) bool {
	// put gotip at the top of the list.
	if a == "tip" {
		return true
	} else if b == "tip" {
		return false
	}
	maja, mina, ta := parseVersion(a)
	majb, minb, tb := parseVersion(b)
	if maja == majb {
		if mina == minb {
			if ta == "" {
				return true
			} else if tb == "" {
				return false
			}
			return ta >= tb
		}
		return mina >= minb
	}
	return maja >= majb
}

func parseVersion(v string) (maj, min int, tail string) {
	if i := strings.Index(v, "beta"); i > 0 {
		tail = v[i:]
		v = v[:i]
	}
	if i := strings.Index(v, "rc"); i > 0 {
		tail = v[i:]
		v = v[:i]
	}
	p := strings.Split(strings.TrimPrefix(v, "1."), ".")
	maj, _ = strconv.Atoi(p[0])
	if len(p) < 2 {
		return
	}
	min, _ = strconv.Atoi(p[1])
	return
}
