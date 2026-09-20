package resource

// Private authoring types mirror the YAML schema.
type rawWorkflow struct {
	Name            string            `yaml:"name,omitempty"`
	DescriptionPath string            `yaml:"description-path,omitempty"`
	Presets         *rawPresets       `yaml:"presets,omitempty"`
	Jobs            map[string]rawJob `yaml:"jobs"`
}

type rawPresets struct {
	Files []rawPreset `yaml:"files"`
}

type rawPreset struct {
	Source string `yaml:"source"`
	Path   string `yaml:"path"`
}

type rawJob struct {
	Name         string        `yaml:"name,omitempty"`
	Visibility   string        `yaml:"visibility,omitempty"`
	Depends      []string      `yaml:"depends,omitempty"`
	SandboxImage string        `yaml:"sandbox-image"`
	Limits       rawLimits     `yaml:"limits"`
	Artifacts    *rawArtifacts `yaml:"artifacts,omitempty"`
	Steps        []rawStep     `yaml:"steps"`
}

type rawLimits struct {
	CPU           *int   `yaml:"cpu,omitempty"`
	Memory        string `yaml:"memory"`
	PIDs          *int   `yaml:"pids,omitempty"`
	StepTimeout   string `yaml:"step-timeout"`
	StdoutSize    string `yaml:"stdout-size,omitempty"`
	StderrSize    string `yaml:"stderr-size,omitempty"`
	WorkspaceSize string `yaml:"workspace-size,omitempty"`
	ArtifactSize  string `yaml:"artifact-size,omitempty"`
}

type rawArtifacts struct {
	Inputs  []rawArtifactInput  `yaml:"inputs,omitempty"`
	Outputs []rawArtifactOutput `yaml:"outputs,omitempty"`
}

type rawArtifactInput struct {
	FromJob string `yaml:"from-job"`
	Name    string `yaml:"name"`
	Path    string `yaml:"path"`
}

type rawArtifactOutput struct {
	Name        string `yaml:"name"`
	Path        string `yaml:"path"`
	Visibility  string `yaml:"visibility,omitempty"`
	ContentType string `yaml:"content-type,omitempty"`
}

type rawStep struct {
	Name     string       `yaml:"name,omitempty"`
	Compile  *bool        `yaml:"compile,omitempty"`
	Run      string       `yaml:"run"`
	Stdin    *rawStream   `yaml:"stdin,omitempty"`
	Timeout  string       `yaml:"timeout,omitempty"`
	Expected *rawExpected `yaml:"expected,omitempty"`
}

type rawStream struct {
	Value *string `yaml:"value,omitempty"`
	Path  string  `yaml:"path,omitempty"`
	Match string  `yaml:"match,omitempty"`
}

type rawExpected struct {
	ExitCode *int       `yaml:"exit-code,omitempty"`
	Stdout   *rawStream `yaml:"stdout,omitempty"`
	Stderr   *rawStream `yaml:"stderr,omitempty"`
}
type rawMetadata struct {
	ID      string `yaml:"id"`
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type rawImageBuild struct {
	Context    string   `yaml:"context"`
	Dockerfile string   `yaml:"dockerfile"`
	Image      string   `yaml:"image"`
	Platforms  []string `yaml:"platforms"`
}
