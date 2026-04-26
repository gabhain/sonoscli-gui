package scenes

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Store interface {
	List() ([]SceneMeta, error)
	Get(name string) (Scene, bool, error)
	Put(scene Scene) error
	Delete(name string) error
}

type FileStore struct {
	path string
}

func NewFileStore() (*FileStore, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return &FileStore{path: filepath.Join(dir, "sonoscli", "scenes.json")}, nil
}

func (s *FileStore) List() ([]SceneMeta, error) {
	data, err := s.readAll()
	if err != nil {
		return nil, err
	}
	metas := make([]SceneMeta, 0, len(data))
	for _, sc := range data {
		metas = append(metas, SceneMeta{Name: sc.Name, CreatedAt: sc.CreatedAt})
	}
	sort.Slice(metas, func(i, j int) bool { return metas[i].Name < metas[j].Name })
	return metas, nil
}

func (s *FileStore) Get(name string) (Scene, bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Scene{}, false, nil
	}
	data, err := s.readAll()
	if err != nil {
		return Scene{}, false, err
	}
	sc, ok := data[name]
	return sc, ok, nil
}

func (s *FileStore) Put(scene Scene) error {
	scene.Name = strings.TrimSpace(scene.Name)
	if scene.Name == "" {
		return errors.New("scene name is required")
	}
	if scene.CreatedAt.IsZero() {
		scene.CreatedAt = time.Now().UTC()
	}

	data, err := s.readAll()
	if err != nil {
		return err
	}
	data[scene.Name] = scene
	return s.writeAll(data)
}

func (s *FileStore) Delete(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("scene name is required")
	}
	data, err := s.readAll()
	if err != nil {
		return err
	}
	if _, ok := data[name]; !ok {
		return nil
	}
	delete(data, name)
	return s.writeAll(data)
}

type fileFormat struct {
	Scenes map[string]Scene `json:"scenes"`
}

func (s *FileStore) readAll() (map[string]Scene, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]Scene{}, nil
		}
		return nil, err
	}
	var ff fileFormat
	if err := json.Unmarshal(b, &ff); err != nil {
		return nil, fmt.Errorf("parse scenes store: %w", err)
	}
	if ff.Scenes == nil {
		ff.Scenes = map[string]Scene{}
	}
	return ff.Scenes, nil
}

func (s *FileStore) writeAll(data map[string]Scene) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	ff := fileFormat{Scenes: data}
	b, err := json.MarshalIndent(ff, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
