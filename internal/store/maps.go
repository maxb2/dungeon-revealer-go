package store

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/xid"
)

type Token struct {
	ID          string  `json:"id"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Radius      float64 `json:"radius"`
	Label       string  `json:"label"`
	Color       string  `json:"color"`
	Visible     bool    `json:"visible"`
	Moveable    bool    `json:"moveable"`
	ImageID     string  `json:"imageId,omitempty"`
	Shape       string  `json:"shape,omitempty"`
	LabelSize   float64 `json:"labelSize,omitempty"`
	SightRadius float64 `json:"sightRadius,omitempty"`
}

type WallPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Wall struct {
	ID     string      `json:"id"`
	Points []WallPoint `json:"points"`
}

type Map struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Ext             string    `json:"ext"`
	Tokens          []Token   `json:"tokens,omitempty"`
	Walls           []Wall    `json:"walls,omitempty"`
	DynamicLighting bool      `json:"dynamicLighting,omitempty"`
	GridSize        float64   `json:"gridSize,omitempty"`
	GridOffsetX     float64   `json:"gridOffsetX,omitempty"`
	GridOffsetY     float64   `json:"gridOffsetY,omitempty"`
	GridEnabled     bool      `json:"gridEnabled,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

func (m *Map) GridSizeOrDefault() float64 {
	if m.GridSize > 0 {
		return m.GridSize
	}
	return 50.0
}

type MapStore struct {
	dataDir string
	active  string // active map ID
}

func NewMapStore(dataDir string) *MapStore {
	mapsDir := filepath.Join(dataDir, "maps")
	os.MkdirAll(mapsDir, 0755)
	return &MapStore{dataDir: dataDir}
}

func (s *MapStore) mapsDir() string {
	return filepath.Join(s.dataDir, "maps")
}

func (s *MapStore) mapDir(id string) string {
	return filepath.Join(s.mapsDir(), id)
}

func (s *MapStore) metaPath(id string) string {
	return filepath.Join(s.mapDir(id), "map.json")
}

func (s *MapStore) Create(title string, ext string, imageData io.Reader) (*Map, error) {
	id := xid.New().String()
	dir := s.mapDir(id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create map dir: %w", err)
	}

	// Normalize extension
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	// Write image file
	imgPath := filepath.Join(dir, "map"+ext)
	f, err := os.Create(imgPath)
	if err != nil {
		return nil, fmt.Errorf("create image file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, imageData); err != nil {
		return nil, fmt.Errorf("write image: %w", err)
	}

	m := &Map{
		ID:        id,
		Title:     title,
		Ext:       ext,
		CreatedAt: time.Now(),
	}

	if err := s.writeMeta(m); err != nil {
		return nil, err
	}

	return m, nil
}

func (s *MapStore) Get(id string) (*Map, error) {
	return s.readMeta(id)
}

func (s *MapStore) List() ([]*Map, error) {
	entries, err := os.ReadDir(s.mapsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var maps []*Map
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		m, err := s.readMeta(e.Name())
		if err != nil {
			continue
		}
		maps = append(maps, m)
	}

	sort.Slice(maps, func(i, j int) bool {
		return maps[i].CreatedAt.After(maps[j].CreatedAt)
	})

	return maps, nil
}

func (s *MapStore) Delete(id string) error {
	if s.active == id {
		s.active = ""
	}
	return os.RemoveAll(s.mapDir(id))
}

func (s *MapStore) SetActive(id string) {
	s.active = id
}

func (s *MapStore) ActiveID() string {
	return s.active
}

func (s *MapStore) ImagePath(id string) (string, error) {
	m, err := s.readMeta(id)
	if err != nil {
		return "", err
	}
	return filepath.Join(s.mapDir(id), "map"+m.Ext), nil
}

func (s *MapStore) FogProgressPath(id string) string {
	return filepath.Join(s.mapDir(id), "fog-progress.png")
}

func (s *MapStore) FogLivePath(id string) string {
	return filepath.Join(s.mapDir(id), "fog-live.png")
}

func (s *MapStore) AddToken(mapID string, t Token) (*Token, error) {
	m, err := s.readMeta(mapID)
	if err != nil {
		return nil, err
	}
	t.ID = xid.New().String()
	m.Tokens = append(m.Tokens, t)
	if err := s.writeMeta(m); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *MapStore) UpdateToken(mapID string, t Token) error {
	m, err := s.readMeta(mapID)
	if err != nil {
		return err
	}
	for i, tok := range m.Tokens {
		if tok.ID == t.ID {
			m.Tokens[i] = t
			return s.writeMeta(m)
		}
	}
	return fmt.Errorf("token %s not found", t.ID)
}

func (s *MapStore) DeleteToken(mapID, tokenID string) error {
	m, err := s.readMeta(mapID)
	if err != nil {
		return err
	}
	for i, tok := range m.Tokens {
		if tok.ID == tokenID {
			m.Tokens = append(m.Tokens[:i], m.Tokens[i+1:]...)
			return s.writeMeta(m)
		}
	}
	return fmt.Errorf("token %s not found", tokenID)
}

func (s *MapStore) GetTokens(mapID string) ([]Token, error) {
	m, err := s.readMeta(mapID)
	if err != nil {
		return nil, err
	}
	return m.Tokens, nil
}

func (s *MapStore) GetVisibleTokens(mapID string) ([]Token, error) {
	m, err := s.readMeta(mapID)
	if err != nil {
		return nil, err
	}
	var visible []Token
	for _, t := range m.Tokens {
		if t.Visible {
			visible = append(visible, t)
		}
	}
	return visible, nil
}

func (s *MapStore) AddWall(mapID string, w Wall) (*Wall, error) {
	m, err := s.readMeta(mapID)
	if err != nil {
		return nil, err
	}
	w.ID = xid.New().String()
	m.Walls = append(m.Walls, w)
	if err := s.writeMeta(m); err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *MapStore) DeleteWall(mapID, wallID string) error {
	m, err := s.readMeta(mapID)
	if err != nil {
		return err
	}
	for i, w := range m.Walls {
		if w.ID == wallID {
			m.Walls = append(m.Walls[:i], m.Walls[i+1:]...)
			return s.writeMeta(m)
		}
	}
	return fmt.Errorf("wall %s not found", wallID)
}

func (s *MapStore) UpdateWall(mapID, wallID string, points []WallPoint) error {
	m, err := s.readMeta(mapID)
	if err != nil {
		return err
	}
	for i, w := range m.Walls {
		if w.ID == wallID {
			m.Walls[i].Points = points
			return s.writeMeta(m)
		}
	}
	return fmt.Errorf("wall %s not found", wallID)
}

func (s *MapStore) GetWalls(mapID string) ([]Wall, error) {
	m, err := s.readMeta(mapID)
	if err != nil {
		return nil, err
	}
	return m.Walls, nil
}

func (s *MapStore) UpdateGridSettings(mapID string, gridSize, offsetX, offsetY float64, enabled bool) error {
	m, err := s.readMeta(mapID)
	if err != nil {
		return err
	}
	m.GridSize = gridSize
	m.GridOffsetX = offsetX
	m.GridOffsetY = offsetY
	m.GridEnabled = enabled
	return s.writeMeta(m)
}

func (s *MapStore) SetDynamicLighting(mapID string, enabled bool) error {
	m, err := s.readMeta(mapID)
	if err != nil {
		return err
	}
	m.DynamicLighting = enabled
	return s.writeMeta(m)
}

func (s *MapStore) writeMeta(m *Map) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal map meta: %w", err)
	}
	return os.WriteFile(s.metaPath(m.ID), data, 0644)
}

func (s *MapStore) readMeta(id string) (*Map, error) {
	data, err := os.ReadFile(s.metaPath(id))
	if err != nil {
		return nil, fmt.Errorf("read map meta: %w", err)
	}
	var m Map
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("unmarshal map meta: %w", err)
	}
	return &m, nil
}
