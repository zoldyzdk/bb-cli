package review

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

var slug = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)
var skillName = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

type Enrollment struct {
	Workspace  string `yaml:"workspace" json:"workspace"`
	Repository string `yaml:"repository" json:"repository"`
	Profile    string `yaml:"profile" json:"profile"`
}

type Project struct {
	Version       int          `yaml:"version"`
	ReportsDir    string       `yaml:"reports_dir"`
	AllowedSkills []string     `yaml:"allowed_skills"`
	Repositories  []Enrollment `yaml:"repositories"`
	Dir           string       `yaml:"-"`
}

type Policy struct {
	Version      int      `yaml:"version" json:"version"`
	Skills       []string `yaml:"skills" json:"skills"`
	Instructions string   `yaml:"instructions" json:"instructions"`
}

type Settings struct {
	ProjectDir string            `json:"project_dir"`
	Tokens     map[string]string `json:"repository_tokens"`
}

func SettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "bb-cli", "review.json"), nil
}

func LoadSettings() (*Settings, error) {
	path, err := SettingsPath()
	if err != nil {
		return nil, err
	}
	s := &Settings{Tokens: map[string]string{}}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("invalid review settings")
	}
	if s.Tokens == nil {
		s.Tokens = map[string]string{}
	}
	return s, nil
}

func SaveSettings(s *Settings) error {
	path, err := SettingsPath()
	if err != nil {
		return err
	}
	return writeJSON(path, s)
}

func LoadProject(directory string) (*Project, error) {
	dir, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "config", "repositories.yaml"))
	if err != nil {
		return nil, err
	}
	p := &Project{Dir: dir}
	if err := decodeYAML(data, p); err != nil {
		return nil, err
	}
	if p.Version != 1 {
		return nil, fmt.Errorf("unsupported project version")
	}
	if p.ReportsDir == "" {
		return nil, fmt.Errorf("reports_dir is required")
	}
	if !filepath.IsAbs(p.ReportsDir) {
		p.ReportsDir = filepath.Join(dir, p.ReportsDir)
	}
	seen := map[string]bool{}
	for _, r := range p.Repositories {
		if !ValidRepository(r.Workspace, r.Repository) || !skillName.MatchString(r.Profile) {
			return nil, fmt.Errorf("invalid repository enrollment")
		}
		key := r.Workspace + "/" + r.Repository
		if seen[key] {
			return nil, fmt.Errorf("duplicate repository enrollment: %s", key)
		}
		seen[key] = true
	}
	for _, s := range p.AllowedSkills {
		if !skillName.MatchString(s) {
			return nil, fmt.Errorf("invalid allowed skill")
		}
	}
	return p, nil
}

func ValidRepository(workspace, repository string) bool {
	return slug.MatchString(workspace) && slug.MatchString(repository)
}

func ParsePolicy(data string, project *Project) (*Policy, error) {
	p := &Policy{}
	if len(data) > 64*1024 {
		return nil, fmt.Errorf("repository policy too large")
	}
	if err := decodeYAML([]byte(data), p); err != nil {
		return nil, fmt.Errorf("invalid repository policy")
	}
	if p.Version != 1 {
		return nil, fmt.Errorf("unsupported repository policy version")
	}
	allowed := map[string]bool{}
	for _, s := range project.AllowedSkills {
		allowed[s] = true
	}
	for _, s := range p.Skills {
		if !allowed[s] {
			return nil, fmt.Errorf("repository selected unauthorized skill: %s", s)
		}
	}
	return p, nil
}

func decodeYAML(data []byte, target any) error {
	d := yaml.NewDecoder(bytes.NewReader(data))
	d.KnownFields(true)
	if err := d.Decode(target); err != nil {
		return fmt.Errorf("invalid YAML configuration: %w", err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("configuration must contain exactly one YAML document")
	}
	return nil
}
