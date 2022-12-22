package main

import (
	"testing"
)

func Test_versionRE(t *testing.T) {
	test := func(s string, match bool) {
		t.Helper()
		if got := versionRE.MatchString(s); got != match {
			t.Errorf("got %t; want %t", got, match)
		}
	}

	test("tip", true)
	test("top", false)
	test("1", true)
	test("1.", false)
	test("1.18", true)
	test("1.18rc", false)
	test("1.18rc1", true)
	test("1.18alpha1", false)
	test("1.18beta1", true)
	test("1.18.", false)
	test("1.18.10", true)
	test("1.18.10.", false)
}
