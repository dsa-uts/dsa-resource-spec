package resource

import "time"

type Metadata struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Workflow struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Presets     []Preset       `json:"presets"`
	Jobs        map[string]Job `json:"jobs"`
}

type Preset struct {
	Content    []byte `json:"content"`
	Executable bool   `json:"executable"`
	Path       string `json:"path"`
}

type Job struct {
	Name         string     `json:"name"`
	Visibility   string     `json:"visibility"`
	Depends      []string   `json:"depends"`
	SandboxImage string     `json:"sandbox-image"`
	Limits       Limits     `json:"limits"`
	Artifacts    *Artifacts `json:"artifacts"`
	Steps        []Step     `json:"steps"`
}

type Limits struct {
	CPU           int   `json:"cpu"`
	Memory        int64 `json:"memory"`
	PIDs          int   `json:"pids"`
	StdoutSize    int64 `json:"stdout-size"`
	StderrSize    int64 `json:"stderr-size"`
	WorkspaceSize int64 `json:"workspace-size"`
	ArtifactSize  int64 `json:"artifact-size"`
}

type Artifacts struct {
	Inputs  []ArtifactInput  `json:"inputs"`
	Outputs []ArtifactOutput `json:"outputs"`
}

type ArtifactInput struct {
	FromJob string `json:"from-job"`
	Name    string `json:"name"`
	Path    string `json:"path"`
}

type ArtifactOutput struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Visibility  string `json:"visibility"`
	ContentType string `json:"content-type"`
}

type Step struct {
	Name     string        `json:"name"`
	Compile  bool          `json:"compile"`
	Run      string        `json:"run"`
	Stdin    []byte        `json:"stdin"`
	Timeout  time.Duration `json:"timeout"`
	Expected Expected      `json:"expected"`
}

// MatchMode selects the output comparison rule.
type MatchMode string

const (
	MatchExact  MatchMode = "exact"
	MatchEasy   MatchMode = "easy"
	MatchSorted MatchMode = "sorted"
)

type OutputExpectation struct {
	Content []byte    `json:"content"`
	Match   MatchMode `json:"match"`
}

type Expected struct {
	ExitCode int                `json:"exit-code"`
	Stdout   *OutputExpectation `json:"stdout"`
	Stderr   *OutputExpectation `json:"stderr"`
}

// ImageBuild holds CI settings; build files are not read by this package.
type ImageBuild struct {
	Context    string   `json:"context"`
	Dockerfile string   `json:"dockerfile"`
	Image      string   `json:"image"`
	Platforms  []string `json:"platforms"`
}
