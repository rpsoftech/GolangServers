package daemon

import "testing"

func TestIgnored(t *testing.T) {
	cases := []struct {
		pat   string
		rel   string
		isDir bool
		want  bool
	}{
		{"**/*.log", "a.log", false, true},
		{"**/*.log", "x/y/a.log", false, true},
		{"**/*.log", "x/a.txt", false, false},
		{"/dist", "dist", true, true},
		{"/dist", "dist/app.js", false, true},
		{"/dist", "src/dist", true, false},
		{"/dist", "src/dist/app.js", false, false},
		{"tmp/", "tmp", true, true},
		{"tmp/", "tmp", false, false}, // a file named tmp
		{"tmp/", "a/b/tmp/x", false, true},
		{"tmp/", "a/tmp", true, true},
		{"*.swp", "a.swp", false, true},
		{"*.swp", "d/.a.swp", false, true},
		{"*.swp", "a.txt", false, false},
		{"a/b/c", "a/b/c", true, true},
		{"a/b/c", "a/b/c/d.txt", false, true},
		{"a/b/c", "x/a/b/c", true, false},
		{"a/b/c", "a/b", true, false},
		{"a/**/c", "a/c", true, true},
		{"a/**/c", "a/x/y/c", true, true},
		{"a/**/c", "b/c", true, false},
		{"src/**", "src/x/y.go", false, true},
		{"cache", "x/cache/f", false, true}, // bare name at any depth
		{"/", "x", false, false},
	}
	for _, c := range cases {
		if got := ignored(c.rel, c.isDir, []string{c.pat}); got != c.want {
			t.Errorf("pattern %q path %q dir=%v: got %v want %v", c.pat, c.rel, c.isDir, got, c.want)
		}
	}
}

func TestDefaultIgnores(t *testing.T) {
	for _, rel := range []string{".git/HEAD", "a/node_modules/x.js", ".pm2go/x"} {
		if !ignored(rel, false, nil) {
			t.Errorf("%q should be ignored by default", rel)
		}
	}
	if ignored("src/main.go", false, nil) {
		t.Error("src/main.go ignored")
	}
}
