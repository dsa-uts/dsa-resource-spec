package resource

import "time"

type Metadata struct {
	ID          string  `json:"id" validate:"identifier"`
	Name        string  `json:"name" validate:"required"`
	Description string  `json:"description" validate:"max=64"`
	Version     Version `json:"version" validate:"validateFn"`
}

type Workflow struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Presets     []Preset       `json:"presets" validate:"unique=Path,dive"`
	Jobs        map[string]Job `json:"jobs" validate:"min=1,dive,keys,identifier,endkeys"`
}

type Preset struct {
	Content    []byte      `json:"content"`
	Executable bool        `json:"executable"`
	Path       RuntimePath `json:"path" validate:"validateFn"`
}

type Job struct {
	Name         string       `json:"name"`
	Visibility   string       `json:"visibility" validate:"oneof=public private"`
	Depends      []string     `json:"depends" validate:"unique"`
	SandboxImage SandboxImage `json:"sandbox-image" validate:"validateFn"`
	Limits       Limits       `json:"limits"`
	Artifacts    *Artifacts   `json:"artifacts" validate:"omitnil"`
	Steps        []Step       `json:"steps" validate:"min=1,dive"`
}

type Limits struct {
	CPU           int   `json:"cpu" validate:"gte=1,lte=2"`
	Memory        int64 `json:"memory" validate:"gt=0,lte=536870912"`
	PIDs          int   `json:"pids" validate:"gte=1,lte=256"`
	StdoutSize    int64 `json:"stdout-size" validate:"gt=0,lte=32768"`
	StderrSize    int64 `json:"stderr-size" validate:"gt=0,lte=32768"`
	WorkspaceSize int64 `json:"workspace-size" validate:"gt=0,lte=134217728"`
	ArtifactSize  int64 `json:"artifact-size" validate:"gt=0,lte=1048576"`
}

type Artifacts struct {
	Inputs  []ArtifactInput  `json:"inputs" validate:"unique=Path,dive"`
	Outputs []ArtifactOutput `json:"outputs" validate:"unique=Name,unique=Path,dive"`
}

type ArtifactInput struct {
	FromJob string      `json:"from-job"`
	Name    string      `json:"name" validate:"identifier"`
	Path    RuntimePath `json:"path" validate:"validateFn"`
}

type ArtifactOutput struct {
	Name        string      `json:"name" validate:"identifier"`
	Path        RuntimePath `json:"path" validate:"validateFn"`
	Visibility  string      `json:"visibility" validate:"oneof=public private"`
	ContentType string      `json:"content-type" validate:"required_if=Visibility public,excluded_if=Visibility private,omitempty,oneof=image/png image/jpeg text/plain application/json"`
}

type Step struct {
	Name     string        `json:"name"`
	Compile  bool          `json:"compile"`
	Run      string        `json:"run" validate:"notblank,excludesall=\x00"`
	Stdin    []byte        `json:"stdin"`
	Timeout  time.Duration `json:"timeout" validate:"gt=0"`
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
	Match   MatchMode `json:"match" validate:"oneof=exact easy sorted"`
}

type Expected struct {
	ExitCode *int               `json:"exit-code" validate:"omitnil,gte=0,lte=255"` // nil means the exit code is not checked.
	Stdout   *OutputExpectation `json:"stdout" validate:"omitnil"`
	Stderr   *OutputExpectation `json:"stderr" validate:"omitnil"`
}

// ImageBuild holds CI settings; build files are not read by this package.
type ImageBuild struct {
	Context    string   `json:"context"`
	Dockerfile string   `json:"dockerfile"`
	Image      string   `json:"image"`
	Platforms  []string `json:"platforms"`
}
