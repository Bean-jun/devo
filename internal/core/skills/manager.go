package skills

import (
	"devo/internal/config"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const skillFileName = "SKILL.md"

func DefaultGlobalSkillsDir() string {
	return config.GlobalSkillsDir()
}

func DefaultProjectSkillsDir(workingDir string) string {
	return config.ProjectSkillsDir(workingDir)
}

type Manager struct {
	mu sync.RWMutex

	globalSkillsDir  string
	projectSkillsDir string

	globalSkills  map[string]*Skill
	projectSkills map[string]*Skill
	projectDir    string
}

func NewManager(globalSkillsDir string) *Manager {
	return &Manager{
		globalSkillsDir: globalSkillsDir,
		globalSkills:    make(map[string]*Skill),
		projectSkills:   make(map[string]*Skill),
	}
}

func (m *Manager) SetProjectDir(workingDir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.projectDir = workingDir
	m.projectSkills = make(map[string]*Skill)
	m.projectSkillsDir = DefaultProjectSkillsDir(workingDir)

	if err := m.scanGlobal(); err != nil {
		return fmt.Errorf("scan global skills: %w", err)
	}
	if err := m.scanProject(); err != nil {
		return fmt.Errorf("scan project skills: %w", err)
	}

	if err := m.applyConfig(); err != nil {
		return fmt.Errorf("apply project config: %w", err)
	}

	return nil
}

func (m *Manager) ReloadSkills() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.scanGlobal(); err != nil {
		return fmt.Errorf("reload global skills: %w", err)
	}
	if err := m.scanProject(); err != nil {
		return fmt.Errorf("reload project skills: %w", err)
	}

	if err := m.applyConfig(); err != nil {
		return fmt.Errorf("reapply config: %w", err)
	}

	return nil
}

func (m *Manager) scanGlobal() error {
	skills, err := scanSkillsDir(m.globalSkillsDir, SourceGlobal)
	if err != nil {
		if os.IsNotExist(err) {
			m.globalSkills = make(map[string]*Skill)
			return nil
		}
		return err
	}

	m.globalSkills = make(map[string]*Skill)
	for _, s := range skills {
		m.globalSkills[s.Name] = s
	}
	return nil
}

func (m *Manager) scanProject() error {
	if m.projectSkillsDir == "" {
		return nil
	}

	skills, err := scanSkillsDir(m.projectSkillsDir, SourceProject)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	m.projectSkills = make(map[string]*Skill)
	for _, s := range skills {
		m.projectSkills[s.Name] = s
	}
	return nil
}

func scanSkillsDir(dir string, source SkillSource) ([]*Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var skills []*Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(dir, entry.Name())
		skillFile := filepath.Join(skillDir, skillFileName)

		data, err := os.ReadFile(skillFile)
		if err != nil {
			continue
		}

		content := string(data)
		fm := parseFrontmatter(content)
		name := fm["name"]
		if name == "" {
			name = extractSkillName(content, entry.Name())
		}
		description := fm["description"]

		skills = append(skills, &Skill{
			Name:         name,
			Description:  description,
			Source:       source,
			Priority:     sourcePriority(source),
			Location:     skillDir,
			Instructions: content,
			Enabled:      true,
			InstalledAt:  time.Now(),
		})
	}

	return skills, nil
}

func parseFrontmatter(content string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return result
	}
	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIdx = i
			break
		}
	}
	if endIdx <= 1 {
		return result
	}
	for i := 1; i < endIdx; i++ {
		line := lines[i]
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			if key != "" {
				result[key] = val
			}
		}
	}
	return result
}

func extractSkillName(content, fallback string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			name := strings.TrimPrefix(trimmed, "# ")
			name = strings.TrimSpace(name)
			if name != "" {
				return name
			}
		}
	}
	return fallback
}

func (m *Manager) GetAllSkills() []*Skill {
	m.mu.RLock()
	defer m.mu.RUnlock()

	merged := m.mergeSkills()
	result := make([]*Skill, 0, len(merged))
	for _, s := range merged {
		result = append(result, s)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Enabled != result[j].Enabled {
			return result[i].Enabled
		}
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	return result
}

func (m *Manager) mergeSkills() map[string]*Skill {
	merged := make(map[string]*Skill)

	for name, s := range m.globalSkills {
		merged[name] = s
	}
	for name, s := range m.projectSkills {
		merged[name] = s
	}

	return merged
}

func (m *Manager) applyConfig() error {
	if m.projectDir == "" {
		return nil
	}

	cfg, err := config.LoadProjectConfig(m.projectDir)
	if err != nil {
		return err
	}

	if cfg == nil {
		for _, s := range m.globalSkills {
			s.Enabled = true
		}
		for _, s := range m.projectSkills {
			s.Enabled = true
		}
		return nil
	}

	enabledSet := make(map[string]bool, len(cfg.Skills))
	for _, name := range cfg.Skills {
		enabledSet[name] = true
	}

	for _, s := range m.globalSkills {
		s.Enabled = enabledSet[s.Name]
	}
	for _, s := range m.projectSkills {
		s.Enabled = enabledSet[s.Name]
	}

	return nil
}

func (m *Manager) EnableSkill(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.projectDir == "" {
		return fmt.Errorf("no project directory set")
	}

	cfg, err := config.LoadProjectConfig(m.projectDir)
	if err != nil {
		return err
	}
	if cfg == nil {
		cfg = &config.ProjectConfig{}
	}

	found := false
	for _, n := range cfg.Skills {
		if n == name {
			found = true
			break
		}
	}
	if !found {
		cfg.Skills = append(cfg.Skills, name)
	}

	if err := config.SaveProjectConfig(m.projectDir, cfg); err != nil {
		return err
	}

	if s, ok := m.globalSkills[name]; ok {
		s.Enabled = true
	}
	if s, ok := m.projectSkills[name]; ok {
		s.Enabled = true
	}

	return nil
}

func (m *Manager) DisableSkill(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.projectDir == "" {
		return fmt.Errorf("no project directory set")
	}

	cfg, err := config.LoadProjectConfig(m.projectDir)
	if err != nil {
		return err
	}
	if cfg == nil {
		cfg = &config.ProjectConfig{}
	}

	filtered := make([]string, 0, len(cfg.Skills))
	for _, n := range cfg.Skills {
		if n != name {
			filtered = append(filtered, n)
		}
	}
	cfg.Skills = filtered

	if err := config.SaveProjectConfig(m.projectDir, cfg); err != nil {
		return err
	}

	if s, ok := m.globalSkills[name]; ok {
		s.Enabled = false
	}
	if s, ok := m.projectSkills[name]; ok {
		s.Enabled = false
	}

	return nil
}

func (m *Manager) GetEnabledSkills() []*Skill {
	m.mu.RLock()
	defer m.mu.RUnlock()

	merged := m.mergeSkills()
	result := make([]*Skill, 0, len(merged))
	for _, s := range merged {
		if s.Enabled {
			result = append(result, s)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	return result
}

func (m *Manager) GetActiveSkillsPrompt() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	merged := m.mergeSkills()

	var catalogParts []string

	for _, skill := range merged {
		if !skill.Enabled {
			continue
		}

		desc := skill.Description
		if desc == "" {
			desc = "No description"
		}
		catalogParts = append(catalogParts, fmt.Sprintf("- **%s**: %s", skill.Name, desc))
	}

	if len(catalogParts) == 0 {
		return ""
	}

	result := "## Available Skills\nYou have access to the following skills. Use the `use_skill` tool to load full instructions for any skill.\n\n" +
		strings.Join(catalogParts, "\n") + "\n"

	return result
}

func (m *Manager) InstallSkill(sourcePath string) (*Skill, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("source path not found: %w", err)
	}

	var skillDir string
	if info.IsDir() {
		skillDir = sourcePath
	} else {
		skillDir = filepath.Dir(sourcePath)
	}

	skillFile := filepath.Join(skillDir, skillFileName)
	data, err := os.ReadFile(skillFile)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrSkillInvalid, skillDir)
	}

	content := string(data)
	fm := parseFrontmatter(content)
	skillName := fm["name"]
	if skillName == "" {
		skillName = extractSkillName(content, filepath.Base(skillDir))
	}
	description := fm["description"]

	destDir := filepath.Join(m.globalSkillsDir, skillName)
	if err := copyDir(skillDir, destDir); err != nil {
		return nil, fmt.Errorf("copy skill: %w", err)
	}

	skill := &Skill{
		Name:         skillName,
		Description:  description,
		Source:       SourceGlobal,
		Priority:     sourcePriority(SourceGlobal),
		Location:     destDir,
		Instructions: content,
		Enabled:      true,
		InstalledAt:  time.Now(),
	}

	m.globalSkills[skillName] = skill
	return skill, nil
}

func (m *Manager) SaveSkill(name string, instructions string) (*Skill, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	skillDir := filepath.Join(m.globalSkillsDir, name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return nil, fmt.Errorf("create skill dir: %w", err)
	}

	fm := parseFrontmatter(instructions)
	description := fm["description"]
	fullContent := instructions
	if len(fm) == 0 {
		fullContent = fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s", name, name, instructions)
		description = name
	}

	skillFile := filepath.Join(skillDir, skillFileName)
	if err := os.WriteFile(skillFile, []byte(fullContent), 0644); err != nil {
		return nil, fmt.Errorf("write skill file: %w", err)
	}

	skill := &Skill{
		Name:         name,
		Description:  description,
		Source:       SourceGlobal,
		Priority:     sourcePriority(SourceGlobal),
		Location:     skillDir,
		Instructions: fullContent,
		Enabled:      true,
		InstalledAt:  time.Now(),
	}

	m.globalSkills[name] = skill
	return skill, nil
}

func (m *Manager) DeleteSkill(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	skill, ok := m.projectSkills[name]
	if !ok {
		skill, ok = m.globalSkills[name]
	}
	if !ok {
		return ErrSkillNotFound
	}

	if skill.Location != "" {
		if err := os.RemoveAll(skill.Location); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("delete skill dir: %w", err)
		}
	}

	delete(m.projectSkills, name)
	delete(m.globalSkills, name)

	if m.projectDir != "" {
		cfg, err := config.LoadProjectConfig(m.projectDir)
		if err != nil {
			return err
		}
		if cfg != nil {
			filtered := make([]string, 0, len(cfg.Skills))
			for _, n := range cfg.Skills {
				if n != name {
					filtered = append(filtered, n)
				}
			}
			cfg.Skills = filtered
			if err := config.SaveProjectConfig(m.projectDir, cfg); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *Manager) ListSkillResources(location string) (scripts, references, assets []string) {
	entries, err := os.ReadDir(location)
	if err != nil {
		return nil, nil, nil
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == ".git" || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		subEntries, err := os.ReadDir(filepath.Join(location, entry.Name()))
		if err != nil {
			continue
		}

		var files []string
		for _, f := range subEntries {
			if !f.IsDir() {
				files = append(files, entry.Name()+"/"+f.Name())
			}
		}

		switch entry.Name() {
		case "scripts":
			scripts = files
		case "references":
			references = files
		case "assets":
			assets = files
		}
	}

	return
}

func (m *Manager) GetSkill(name string) (*Skill, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if s, ok := m.projectSkills[name]; ok {
		return s, nil
	}
	if s, ok := m.globalSkills[name]; ok {
		return s, nil
	}
	return nil, ErrSkillNotFound
}

func (m *Manager) IsSkillAllowed(name string) bool {
	return true
}

func (m *Manager) Rescan() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.scanGlobal(); err != nil {
		return err
	}
	if err := m.scanProject(); err != nil {
		return err
	}
	return nil
}

func (m *Manager) RescanWithConfig(cfg *config.ProjectConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	enabledSet := make(map[string]bool, len(cfg.Skills))
	for _, name := range cfg.Skills {
		enabledSet[name] = true
	}

	for _, s := range m.globalSkills {
		s.Enabled = enabledSet[s.Name]
	}
	for _, s := range m.projectSkills {
		s.Enabled = enabledSet[s.Name]
	}

	return nil
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}
