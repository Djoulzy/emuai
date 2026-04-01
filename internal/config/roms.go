package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Address uint16

type ROMSet struct {
	Chargen *Asset   `yaml:"chargen,omitempty"`
	Slots   SlotROMs `yaml:"slots,omitempty"`
	ROMs    []ROM    `yaml:"roms"`
}

type SlotROMs struct {
	Slot1 *Asset `yaml:"slot1,omitempty"`
	Slot2 *Asset `yaml:"slot2,omitempty"`
	Slot3 *Asset `yaml:"slot3,omitempty"`
	Slot4 *Asset `yaml:"slot4,omitempty"`
	Slot5 *Asset `yaml:"slot5,omitempty"`
	Slot6 *Asset `yaml:"slot6,omitempty"`
	Slot7 *Asset `yaml:"slot7,omitempty"`
}

type Asset struct {
	Path string `yaml:"path"`
}

type ROM struct {
	Name  string  `yaml:"name,omitempty"`
	Path  string  `yaml:"path"`
	Start Address `yaml:"start"`
}

func LoadROMSet(path string) (*ROMSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("rom config: read %q: %w", path, err)
	}

	var cfg ROMSet
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("rom config: parse %q: %w", path, err)
	}

	baseDir := filepath.Dir(path)
	if err := cfg.Validate(baseDir); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (s *ROMSet) Validate(baseDir string) error {
	if s == nil {
		return fmt.Errorf("rom config: config is nil")
	}

	if len(s.ROMs) == 0 {
		return fmt.Errorf("rom config: at least one ROM entry is required")
	}

	if s.Chargen != nil {
		if err := validateAssetPath("chargen", s.Chargen.Path, baseDir); err != nil {
			return err
		}
	}

	for _, slot := range s.ConfiguredSlots() {
		asset := s.Slots.Asset(slot)
		if asset == nil {
			continue
		}
		if err := validateAssetPath(fmt.Sprintf("slots.slot%d", slot), asset.Path, baseDir); err != nil {
			return err
		}
	}

	for idx, rom := range s.ROMs {
		if err := validateAssetPath(fmt.Sprintf("rom[%d]", idx), rom.Path, baseDir); err != nil {
			return err
		}
	}

	return nil
}

func (s *ROMSet) ResolveChargenPath(baseDir string) string {
	if s == nil || s.Chargen == nil {
		return ""
	}

	return resolveAssetPath(baseDir, s.Chargen.Path)
}

func (s *ROMSet) ConfiguredSlots() []int {
	if s == nil {
		return nil
	}

	slots := make([]int, 0, 7)
	for slot := 1; slot <= 7; slot++ {
		if s.Slots.Asset(slot) != nil {
			slots = append(slots, slot)
		}
	}
	return slots
}

func (s *ROMSet) ResolveSlotROMPath(slot int, baseDir string) string {
	if s == nil {
		return ""
	}

	asset := s.Slots.Asset(slot)
	if asset == nil {
		return ""
	}

	return resolveAssetPath(baseDir, asset.Path)
}

func (s SlotROMs) Asset(slot int) *Asset {
	switch slot {
	case 1:
		return s.Slot1
	case 2:
		return s.Slot2
	case 3:
		return s.Slot3
	case 4:
		return s.Slot4
	case 5:
		return s.Slot5
	case 6:
		return s.Slot6
	case 7:
		return s.Slot7
	default:
		return nil
	}
}

func (r ROM) ResolvePath(baseDir string) string {
	return resolveAssetPath(baseDir, r.Path)
}

func resolveAssetPath(baseDir string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Clean(filepath.Join(baseDir, path))
}

func validateAssetPath(label string, path string, baseDir string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("rom config: %s path is required", label)
	}

	resolved := resolveAssetPath(baseDir, path)
	info, err := os.Stat(resolved)
	if err != nil {
		return fmt.Errorf("rom config: %s path %q: %w", label, resolved, err)
	}
	if info.IsDir() {
		return fmt.Errorf("rom config: %s path %q is a directory", label, resolved)
	}

	return nil
}

func (a *Address) UnmarshalYAML(node *yaml.Node) error {
	if node == nil {
		return fmt.Errorf("rom config: missing address")
	}

	value := strings.TrimSpace(node.Value)
	if value == "" {
		return fmt.Errorf("rom config: empty address")
	}

	parsed, err := strconv.ParseUint(value, 0, 16)
	if err != nil {
		return fmt.Errorf("rom config: invalid address %q: %w", value, err)
	}

	*a = Address(parsed)
	return nil
}

func (a Address) Uint16() uint16 {
	return uint16(a)
}
