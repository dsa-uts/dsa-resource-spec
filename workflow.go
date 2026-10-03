package resource

import (
	"fmt"
	"slices"
)

func validateResource(resource *Resource) error {
	if err := fieldValidator.Struct(resource); err != nil {
		return err
	}
	for id, workflow := range resource.Workflows {
		if err := validateWorkflow(workflow); err != nil {
			return fmt.Errorf("workflow %s: %w", id, err)
		}
	}
	return nil
}

func validateWorkflow(workflow Workflow) error {
	for id, job := range workflow.Jobs {
		if err := validateJob(id, job, workflow.Jobs); err != nil {
			return fmt.Errorf("job %s: %w", id, err)
		}
	}
	return validateAcyclic(workflow.Jobs)
}

func validateJob(id string, job Job, jobs map[string]Job) error {
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
		return validateArtifacts(job, jobs)
	}
	return nil
}

func validateArtifacts(job Job, jobs map[string]Job) error {
	for _, input := range job.Artifacts.Inputs {
		if !slices.Contains(job.Depends, input.FromJob) {
			return fmt.Errorf("Artifact producer must be a direct dependency")
		}
		producer := jobs[input.FromJob]
		if producer.Artifacts == nil || !slices.ContainsFunc(producer.Artifacts.Outputs, func(output ArtifactOutput) bool { return output.Name == input.Name }) {
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
