package resource

import (
	_ "crypto/sha256"
	"fmt"
	"slices"
	"strings"

	"github.com/distribution/reference"
)

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
	Name         string     `yaml:"name,omitempty" json:"name,omitempty"`
	Visibility   string     `yaml:"visibility,omitempty" json:"visibility,omitempty"`
	Depends      []string   `yaml:"depends,omitempty" json:"depends,omitempty"`
	SandboxImage string     `yaml:"sandbox-image" json:"sandbox-image"`
	Limits       Limits     `yaml:"limits" json:"limits"`
	Artifacts    *Artifacts `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
	Steps        []Step     `yaml:"steps" json:"steps"`
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

func applyWorkflowDefaults(workflow Workflow) {
	for id, job := range workflow.Jobs {
		if job.Visibility == "" {
			job.Visibility = "public"
			workflow.Jobs[id] = job
		}
	}
}

func validateWorkflow(workflow Workflow) error {
	if workflow.Presets != nil {
		paths := map[string]bool{}
		for _, preset := range workflow.Presets.Files {
			if err := relative(preset.Path); err != nil {
				return err
			}
			if paths[preset.Path] {
				return fmt.Errorf("duplicate Preset path: %s", preset.Path)
			}
			paths[preset.Path] = true
		}
	}
	for id, job := range workflow.Jobs {
		if err := validateJob(id, job, workflow.Jobs); err != nil {
			return fmt.Errorf("job %s: %w", id, err)
		}
	}
	return validateAcyclic(workflow.Jobs)
}

func validateJob(id string, job Job, jobs map[string]Job) error {
	if _, err := imageReference(job.SandboxImage); err != nil {
		return err
	}
	for _, dependency := range job.Depends {
		producer, exists := jobs[dependency]
		if !exists || dependency == id {
			return fmt.Errorf("invalid dependency: %s", dependency)
		}
		if job.Visibility == "public" && producer.Visibility != "public" {
			return fmt.Errorf("public Job depends on private Job")
		}
	}
	if job.Artifacts != nil {
		if err := validateArtifacts(job, jobs); err != nil {
			return err
		}
	}
	for _, step := range job.Steps {
		if strings.TrimSpace(step.Run) == "" || strings.ContainsRune(step.Run, 0) {
			return fmt.Errorf("invalid run script")
		}
	}
	return nil
}

func validateArtifacts(job Job, jobs map[string]Job) error {
	names, paths := map[string]bool{}, map[string]bool{}
	for _, output := range job.Artifacts.Outputs {
		if err := relative(output.Path); err != nil {
			return err
		}
		if names[output.Name] || paths[output.Path] {
			return fmt.Errorf("duplicate Artifact name/path")
		}
		names[output.Name] = true
		paths[output.Path] = true
	}
	paths = map[string]bool{}
	for _, input := range job.Artifacts.Inputs {
		if err := relative(input.Path); err != nil {
			return err
		}
		if paths[input.Path] {
			return fmt.Errorf("duplicate Artifact path: %s", input.Path)
		}
		paths[input.Path] = true
		if !slices.Contains(job.Depends, input.FromJob) {
			return fmt.Errorf("Artifact producer must be a direct dependency")
		}
		producer := jobs[input.FromJob]
		if producer.Artifacts == nil || !slices.ContainsFunc(producer.Artifacts.Outputs, func(output ArtifactOutput) bool {
			return output.Name == input.Name
		}) {
			return fmt.Errorf("unknown Artifact output: %s", input.Name)
		}
	}
	return nil
}

func validateAcyclic(jobs map[string]Job) error {
	const (
		visiting = 1
		visited  = 2
	)
	state := map[string]int{}
	var visit func(string) error
	visit = func(id string) error {
		switch state[id] {
		case visiting:
			return fmt.Errorf("dependency cycle at job %s", id)
		case visited:
			return nil
		}
		state[id] = visiting
		for _, dependency := range jobs[id].Depends {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		state[id] = visited
		return nil
	}
	for id := range jobs {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func imageReference(s string) (reference.Named, error) {
	r, err := reference.ParseNamed(s)
	if err != nil {
		return nil, fmt.Errorf("invalid fully qualified image %q: %w", s, err)
	}
	_, tag := r.(reference.Tagged)
	_, digest := r.(reference.Digested)
	if !tag && !digest {
		return nil, fmt.Errorf("image requires tag or digest: %s", s)
	}
	return r, nil
}
