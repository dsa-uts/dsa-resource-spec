package resource

import (
	"fmt"
	"slices"
	"strings"
)

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
