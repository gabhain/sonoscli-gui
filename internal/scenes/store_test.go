package scenes

import (
	"path/filepath"
	"testing"
)

func TestFileStoreCRUD(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := &FileStore{path: filepath.Join(dir, "scenes.json")}

	if metas, err := s.List(); err != nil || len(metas) != 0 {
		t.Fatalf("expected empty list, got metas=%v err=%v", metas, err)
	}

	scene := Scene{Name: "Morning"}
	if err := s.Put(scene); err != nil {
		t.Fatalf("put: %v", err)
	}

	got, ok, err := s.Get("Morning")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok")
	}
	if got.Name != "Morning" {
		t.Fatalf("name: %q", got.Name)
	}
	if got.CreatedAt.IsZero() {
		t.Fatalf("expected createdAt to be set")
	}

	metas, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(metas) != 1 || metas[0].Name != "Morning" {
		t.Fatalf("unexpected metas: %v", metas)
	}

	if err := s.Delete("Morning"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, ok, err = s.Get("Morning")
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	if ok {
		t.Fatalf("expected missing after delete")
	}
}
