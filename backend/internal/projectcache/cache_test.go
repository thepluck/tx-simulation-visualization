package projectcache

import (
	"path/filepath"
	"testing"
)

func TestCacheAddListAndPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.json")
	cache := New(path, 3)

	if err := cache.Add("~/alpha"); err != nil {
		t.Fatal(err)
	}
	if err := cache.Add("~/beta"); err != nil {
		t.Fatal(err)
	}
	if err := cache.Add("~/alpha"); err != nil {
		t.Fatal(err)
	}

	projects, err := New(path, 3).List()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"~/alpha", "~/beta"}
	if len(projects) != len(want) {
		t.Fatalf("projects = %#v, want %#v", projects, want)
	}
	for i := range want {
		if projects[i] != want[i] {
			t.Fatalf("projects = %#v, want %#v", projects, want)
		}
	}
}

func TestCacheLimit(t *testing.T) {
	cache := New(filepath.Join(t.TempDir(), "projects.json"), 2)

	for _, path := range []string{"~/alpha", "~/beta", "~/gamma"} {
		if err := cache.Add(path); err != nil {
			t.Fatal(err)
		}
	}

	projects, err := cache.List()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"~/gamma", "~/beta"}
	if len(projects) != len(want) {
		t.Fatalf("projects = %#v, want %#v", projects, want)
	}
	for i := range want {
		if projects[i] != want[i] {
			t.Fatalf("projects = %#v, want %#v", projects, want)
		}
	}
}
