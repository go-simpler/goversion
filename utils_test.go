package main

import (
	"os"
	"strings"
	"testing"

	"github.com/go-simpler/assert"
	. "github.com/go-simpler/assert/dotimport"
)

func Test_cutFromPath(t *testing.T) {
	join := func(values ...string) string {
		return strings.Join(values, string(os.PathListSeparator))
	}
	path := join("foo", "bar", "baz")
	got := cutFromPath(path, "bar")
	assert.Equal[E](t, got, join("foo", "baz"))
}

func Test_latestPatches(t *testing.T) {
	got := latestPatches([]string{
		"tip",
		"1.20rc3",
		"1.20rc2",
		"1.20rc1",
		"1.19.5",
		"1.19.4",
		"1.19.3",
		"1.19.2",
		"1.19.1",
	})
	assert.Equal[E](t, got, []string{
		"tip",
		"1.20rc3",
		"1.19.5",
	})
}
