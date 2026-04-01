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
	Chargen *Asset `yaml:"chargen,omitempty"`
	ROMs    []ROM  `yaml:"roms"`
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
