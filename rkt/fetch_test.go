package main

import (
	"reflect"
	"testing"

	"github.com/appc/spec/discovery"
)

func TestNewDiscoveryApp(t *testing.T) {
	tests := []struct {
		in string

		w *discovery.App
	}{
		// not a valid AC name
		{
			"bad AC name",
			nil,
		},
		// simple case - default arch, os should be substituted
		{
			"foo.com/bar",
			&discovery.App{
				Name: "foo.com/bar",
				Labels: map[string]string{
					"arch": defaultArch,
					"os":   defaultOS,
				},
			},
		},
		// overriding arch, os should work
		{
			"www.abc.xyz/my/app,os=freebsd,arch=i386",
			&discovery.App{
				Name: "www.abc.xyz/my/app",
				Labels: map[string]string{
					"arch": "i386",
					"os":   "freebsd",
				},
			},
		},
		// setting version should work
		{
			"yes.com/no:v1.2.3",
			&discovery.App{
				Name: "yes.com/no",
				Labels: map[string]string{
					"version": "v1.2.3",
					"arch":    defaultArch,
					"os":      defaultOS,
				},
			},
		},
		// arbitrary user-supplied labels
		{
			"example.com/foo/haha,val=one",
			&discovery.App{
				Name: "example.com/foo/haha",
				Labels: map[string]string{
					"val":  "one",
					"arch": defaultArch,
					"os":   defaultOS,
				},
			},
		},
		// combinations
		{
			"one.two/:three,os=four,foo=five,arch=six",
			&discovery.App{
				Name: "one.two/",
				Labels: map[string]string{
					"version": "three",
					"os":      "four",
					"foo":     "five",
					"arch":    "six",
				},
			},
		},
	}
	for i, tt := range tests {
		g := newDiscoveryApp(tt.in)
		if !reflect.DeepEqual(g, tt.w) {
			t.Errorf("#%d: got %v, want %v", i, g, tt.w)
		}
	}
}
