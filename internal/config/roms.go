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
	ROMs []ROM `yaml:"roms"`
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

	for idx, rom := range s.ROMs {
		if strings.TrimSpace(rom.Path) == "" {
			return fmt.Errorf("rom config: rom[%d] path is required", idx)
		}

		resolved := rom.ResolvePath(baseDir)
		info, err := os.Stat(resolved)
		if err != nil {
			return fmt.Errorf("rom config: rom[%d] path %q: %w", idx, resolved, err)
		}
		if info.IsDir() {
			return fmt.Errorf("rom config: rom[%d] path %q is a directory", idx, resolved)
		}
	}

	return nil
}

func (r ROM) ResolvePath(baseDir string) string {
	if filepath.IsAbs(r.Path) {
		return r.Path
	}

	return filepath.Clean(filepath.Join(baseDir, r.Path))
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
