package app

import (
	goversion "go/version"
	"os"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
)

func isValid(version string) bool {
	return goversion.IsValid("go"+version) || version == "tip"
}

func exe() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func cutFromPath(path, value string) string {
	oldPath := strings.Split(path, string(os.PathListSeparator))
	newPath := slices.DeleteFunc(oldPath, func(v string) bool {
		return v == value
	})
	return strings.Join(newPath, string(os.PathListSeparator))
}

func latestPatches(versions []string) []string {
	sorted := sort.SliceIsSorted(versions, func(i, j int) bool {
		return versionLess(versions[i], versions[j])
	})
	if !sorted {
		panic("version list is not sorted")
	}

	if len(versions) <= 1 {
		return versions
	}

	latest := []string{versions[0]}
	prev, _, _ := parseVersion(versions[0])

	for i := 1; i < len(versions); i++ {
		curr, _, _ := parseVersion(versions[i])
		if prev != curr {
			prev = curr
			latest = append(latest, versions[i])
		}
	}

	return latest
}

// the following code is a modified version of the functions from
// https://github.com/golang/website/blob/master/internal/dl/dl.go

func versionLess(a, b string) bool {
	if a == "tip" {
		return true
	}
	if b == "tip" {
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

func parseVersion(v string) (major, minor int, tail string) {
	if i := strings.Index(v, "beta"); i > 0 {
		tail = v[i:]
		v = v[:i]
	}
	if i := strings.Index(v, "rc"); i > 0 {
		tail = v[i:]
		v = v[:i]
	}
	p := strings.Split(strings.TrimPrefix(v, "1."), ".")
	major, _ = strconv.Atoi(p[0])
	if len(p) < 2 {
		return
	}
	minor, _ = strconv.Atoi(p[1])
	return
}
