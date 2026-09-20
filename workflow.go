package resource

import (
	_ "crypto/sha256"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/distribution/reference"
	"golang.org/x/mod/semver"
)

var identifier = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var fullVersion = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:[-+].*)?$`)

func validateResource(resource *Resource) error {
	if !identifier.MatchString(resource.Metadata.ID) || resource.Metadata.Name == "" || !fullVersion.MatchString(resource.Metadata.Version) || !semver.IsValid(resource.Metadata.Version) {
		return fmt.Errorf("invalid resource metadata")
	}
	if len(resource.Workflows) == 0 {
		return fmt.Errorf("resource requires workflows")
	}
	for id, workflow := range resource.Workflows {
		if !identifier.MatchString(id) {
			return fmt.Errorf("invalid workflow ID %q", id)
		}
		if err := validateWorkflow(workflow); err != nil {
			return fmt.Errorf("workflow %s: %w", id, err)
		}
	}
	return nil
}

func validateWorkflow(workflow Workflow) error {
	if len(workflow.Jobs) == 0 {
		return fmt.Errorf("workflow requires jobs")
	}
	if workflow.Presets != nil {
		paths := map[string]bool{}
		for _, preset := range workflow.Presets.Files {
			if err := validateRuntimePath(preset.Path); err != nil {
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
	if !identifier.MatchString(id) {
		return fmt.Errorf("invalid job ID %q", id)
	}
	if job.Visibility != "public" && job.Visibility != "private" {
		return fmt.Errorf("invalid job visibility")
	}
	limits := job.Limits
	if limits.CPU != 1 || limits.PIDs < 1 || limits.Memory <= 0 || limits.StdoutSize <= 0 || limits.StderrSize <= 0 || limits.WorkspaceSize <= 0 || limits.ArtifactSize <= 0 {
		return fmt.Errorf("invalid job limits")
	}
	if len(job.Steps) == 0 {
		return fmt.Errorf("job requires steps")
	}
	seen := map[string]bool{}
	for _, dependency := range job.Depends {
		if seen[dependency] {
			return fmt.Errorf("duplicate dependency: %s", dependency)
		}
		seen[dependency] = true
	}

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
		if step.Timeout <= 0 {
			return fmt.Errorf("invalid step timeout")
		}
		if step.Expected.ExitCode < 0 || step.Expected.ExitCode > 255 {
			return fmt.Errorf("invalid expected exit code")
		}
		for _, output := range []*OutputExpectation{step.Expected.Stdout, step.Expected.Stderr} {
			if output != nil && output.Match != MatchExact && output.Match != MatchEasy && output.Match != MatchSorted {
				return fmt.Errorf("invalid output match mode")
			}
		}

		if strings.TrimSpace(step.Run) == "" || strings.ContainsRune(step.Run, 0) {
			return fmt.Errorf("invalid run script")
		}
	}
	return nil
}

func validateArtifacts(job Job, jobs map[string]Job) error {
	names, paths := map[string]bool{}, map[string]bool{}
	for _, output := range job.Artifacts.Outputs {
		if !identifier.MatchString(output.Name) {
			return fmt.Errorf("invalid artifact name")
		}
		switch output.Visibility {
		case "public":
			if !slices.Contains([]string{"image/png", "image/jpeg", "text/plain", "application/json"}, output.ContentType) {
				return fmt.Errorf("invalid public artifact content type")
			}
		case "private":
			if output.ContentType != "" {
				return fmt.Errorf("private artifact cannot specify content type")
			}
		default:
			return fmt.Errorf("invalid artifact visibility")
		}

		if err := validateRuntimePath(output.Path); err != nil {
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
		if !identifier.MatchString(input.Name) {
			return fmt.Errorf("invalid artifact name")
		}
		if err := validateRuntimePath(input.Path); err != nil {
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
