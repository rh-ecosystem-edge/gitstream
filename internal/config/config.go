package config

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/creasty/defaults"
	"gopkg.in/yaml.v3"
)

type Downstream struct {
	CreateDraftPRs bool   `yaml:"create_draft_prs"`
	GitHubRepoName string `yaml:"github_repo_name"`
	LocalRepoPath  string `yaml:"local_repo_path" default:"."`
	MainBranch     string `yaml:"main_branch" default:"main"`
	MaxOpenItems   int    `yaml:"max_open_items" default:"-1"`
}

type Diff struct {
	CommitsSince *time.Time `yaml:"commits_since"`
}

type Sync struct {
	BeforeCommit [][]string `yaml:"before_commit"`
}

type Upstream struct {
	Ref string `default:"main"`
	URL string
}

type Config struct {
	CommitMarkup string `yaml:"commit_markup" default:"Upstream-Commit"`
	Downstream   Downstream
	Diff         Diff
	LogLevel     int `yaml:"log_level"`
	Sync         Sync
	Upstream     Upstream
}

func ReadConfig(rd io.Reader) (*Config, error) {
	cfg := Config{}

	if err := defaults.Set(&cfg); err != nil {
		return nil, fmt.Errorf("could not set default values: %v", err)
	}

	return &cfg, yaml.NewDecoder(rd).Decode(&cfg)
}

func ReadConfigFile(path string) (*Config, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not read the config file: %v", err)
	}
	defer fd.Close()

	return ReadConfig(fd)
}
