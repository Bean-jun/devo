package skills

import (
	"errors"
	"time"
)

type SkillSource string

const (
	SourceProject   SkillSource = "project"
	SourceGlobal    SkillSource = "global"
	SourceCommunity SkillSource = "community"
)

type Skill struct {
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Source       SkillSource `json:"source"`
	Priority     int         `json:"priority"`
	Location     string      `json:"location"`
	Instructions string      `json:"instructions"`
	Enabled      bool        `json:"enabled"`
	InstalledAt  time.Time   `json:"installed_at,omitempty"`
}

var (
	ErrSkillNotFound    = errors.New("skill not found")
	ErrSkillInvalid     = errors.New("invalid skill: missing SKILL.md")
	ErrSkillDirNotFound = errors.New("skill directory not found")
)

func sourcePriority(source SkillSource) int {
	switch source {
	case SourceProject:
		return 100
	case SourceGlobal:
		return 50
	case SourceCommunity:
		return 10
	}
	return 0
}
