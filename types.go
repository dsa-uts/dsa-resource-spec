// Package resource validates and eagerly reads locally available resource definitions.
package resource

type Definition struct {
	Resource  Metadata            `yaml:"resource" json:"resource"`
	Workflows map[string]Workflow `yaml:"workflows" json:"workflows"`
}

type Metadata struct {
	ID      string `yaml:"id" json:"id"`
	Name    string `yaml:"name" json:"name"`
	Version string `yaml:"version" json:"version"`
}

type Workflow struct {
	Name            string         `yaml:"name,omitempty" json:"name,omitempty"`
	DescriptionPath string         `yaml:"description-path,omitempty" json:"description-path,omitempty"`
	Presets         *Presets       `yaml:"presets,omitempty" json:"presets,omitempty"`
	Jobs            map[string]Job `yaml:"jobs" json:"jobs"`
}

type Presets struct {
	Files []Preset `yaml:"files" json:"files"`
}

type Preset struct {
	Source string `yaml:"source" json:"source"`
	Path   string `yaml:"path" json:"path"`
}

type Job struct {
	Name             string     `yaml:"name,omitempty" json:"name,omitempty"`
	Visibility       string     `yaml:"visibility,omitempty" json:"visibility,omitempty"`
	Depends          []string   `yaml:"depends,omitempty" json:"depends,omitempty"`
	SandboxImage     string     `yaml:"sandbox-image" json:"sandbox-image"`
	WorkingDirectory string     `yaml:"working-directory,omitempty" json:"working-directory,omitempty"`
	Limits           Limits     `yaml:"limits" json:"limits"`
	Artifacts        *Artifacts `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
	Steps            []Step     `yaml:"steps" json:"steps"`
}

type Limits struct {
	CPU           *int   `yaml:"cpu,omitempty" json:"cpu,omitempty"`
	Memory        string `yaml:"memory" json:"memory"`
	PIDs          *int   `yaml:"pids,omitempty" json:"pids,omitempty"`
	StepTimeout   string `yaml:"step-timeout" json:"step-timeout"`
	StdoutSize    string `yaml:"stdout-size,omitempty" json:"stdout-size,omitempty"`
	StderrSize    string `yaml:"stderr-size,omitempty" json:"stderr-size,omitempty"`
	WorkspaceSize string `yaml:"workspace-size,omitempty" json:"workspace-size,omitempty"`
	ArtifactSize  string `yaml:"artifact-size,omitempty" json:"artifact-size,omitempty"`
}

type Artifacts struct {
	Inputs  []ArtifactInput  `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Outputs []ArtifactOutput `yaml:"outputs,omitempty" json:"outputs,omitempty"`
}

type ArtifactInput struct {
	FromJob string `yaml:"from-job" json:"from-job"`
	Name    string `yaml:"name" json:"name"`
	Path    string `yaml:"path" json:"path"`
}

type ArtifactOutput struct {
	Name        string `yaml:"name" json:"name"`
	Path        string `yaml:"path" json:"path"`
	Visibility  string `yaml:"visibility,omitempty" json:"visibility,omitempty"`
	ContentType string `yaml:"content-type,omitempty" json:"content-type,omitempty"`
}

type Step struct {
	Name     string    `yaml:"name,omitempty" json:"name,omitempty"`
	Compile  *bool     `yaml:"compile,omitempty" json:"compile,omitempty"`
	Run      string    `yaml:"run" json:"run"`
	Stdin    *Stream   `yaml:"stdin,omitempty" json:"stdin,omitempty"`
	Timeout  string    `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Expected *Expected `yaml:"expected,omitempty" json:"expected,omitempty"`
}

type Stream struct {
	Value *string `yaml:"value,omitempty" json:"value,omitempty"`
	Path  string  `yaml:"path,omitempty" json:"path,omitempty"`
	Match string  `yaml:"match,omitempty" json:"match,omitempty"`
}

type Expected struct {
	ExitCode *int    `yaml:"exit-code,omitempty" json:"exit-code,omitempty"`
	Stdout   *Stream `yaml:"stdout,omitempty" json:"stdout,omitempty"`
	Stderr   *Stream `yaml:"stderr,omitempty" json:"stderr,omitempty"`
}

// Resource holds the validated definition and every referenced file, keyed by its resource-relative path.
// It does not retain the input filesystem or perform subsequent I/O.
type Resource struct {
	Definition Definition
	Files      map[string][]byte
}
