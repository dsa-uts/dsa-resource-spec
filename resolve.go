package resource

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"time"
)

var sizePattern = regexp.MustCompile(`^([1-9][0-9]*)(KiB|MiB|GiB)$`)
var durationPattern = regexp.MustCompile(`^[1-9][0-9]*(ms|s)$`)

func sizeBytes(value string, fallback int64) (int64, error) {
	if value == "" {
		return fallback, nil
	}
	parts := sizePattern.FindStringSubmatch(value)
	if parts == nil {
		return 0, fmt.Errorf("invalid size %q", value)
	}
	n, err := strconv.ParseInt(parts[1], 10, 64)
	scale := map[string]int64{"KiB": 1 << 10, "MiB": 1 << 20, "GiB": 1 << 30}[parts[2]]
	if err != nil || n > math.MaxInt64/scale {
		return 0, fmt.Errorf("size overflows int64: %q", value)
	}
	return n * scale, nil
}

func duration(value string) (time.Duration, error) {
	if !durationPattern.MatchString(value) {
		return 0, fmt.Errorf("invalid duration %q", value)
	}
	d, err := time.ParseDuration(value)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("invalid duration %q", value)
	}
	return d, nil
}

func resolveWorkflow(root *sourceReader, dir string, raw rawWorkflow) (Workflow, error) {
	result := Workflow{Name: raw.Name, Jobs: make(map[string]Job)}
	if raw.DescriptionPath != "" {
		content, _, err := root.read(dir + "/" + raw.DescriptionPath)
		if err != nil {
			return result, err
		}
		result.Description = string(content)
	}
	if raw.Presets != nil {
		result.Presets = make([]Preset, 0, len(raw.Presets.Files))
		for _, preset := range raw.Presets.Files {
			content, executable, err := root.read(dir + "/" + preset.Source)
			if err != nil {
				return result, err
			}
			result.Presets = append(result.Presets, Preset{Path: preset.Path, Content: content, Executable: executable})
		}
	}
	for id, rawJob := range raw.Jobs {
		job, err := resolveJob(root, dir, rawJob)
		if err != nil {
			return result, fmt.Errorf("job %s: %w", id, err)
		}
		result.Jobs[id] = job
	}
	return result, nil
}

func resolveJob(root *sourceReader, dir string, raw rawJob) (Job, error) {
	job := Job{Name: raw.Name, Visibility: raw.Visibility, Depends: raw.Depends, SandboxImage: raw.SandboxImage}
	if job.Visibility == "" {
		job.Visibility = "public"
	}
	job.Limits.CPU = 1
	if raw.Limits.CPU != nil {
		job.Limits.CPU = *raw.Limits.CPU
	}
	job.Limits.PIDs = 128
	if raw.Limits.PIDs != nil {
		job.Limits.PIDs = *raw.Limits.PIDs
	}
	for _, field := range []struct {
		name, value string
		fallback    int64
		out         *int64
	}{
		{"memory", raw.Limits.Memory, 128 << 20, &job.Limits.Memory},
		{"stdout-size", raw.Limits.StdoutSize, 10 << 20, &job.Limits.StdoutSize},
		{"stderr-size", raw.Limits.StderrSize, 10 << 20, &job.Limits.StderrSize},
		{"workspace-size", raw.Limits.WorkspaceSize, 128 << 20, &job.Limits.WorkspaceSize},
		{"artifact-size", raw.Limits.ArtifactSize, 1 << 20, &job.Limits.ArtifactSize},
	} {
		value, err := sizeBytes(field.value, field.fallback)
		if err != nil {
			return job, fmt.Errorf("%s: %w", field.name, err)
		}
		*field.out = value
	}
	timeout, err := duration(raw.Limits.StepTimeout)
	if err != nil {
		return job, err
	}
	if raw.Artifacts != nil {
		job.Artifacts = &Artifacts{}
		for _, input := range raw.Artifacts.Inputs {
			job.Artifacts.Inputs = append(job.Artifacts.Inputs, ArtifactInput(input))
		}
		for _, output := range raw.Artifacts.Outputs {
			resolved := ArtifactOutput(output)
			if resolved.Visibility == "" {
				resolved.Visibility = "private"
			}
			job.Artifacts.Outputs = append(job.Artifacts.Outputs, resolved)
		}
	}
	for _, rawStep := range raw.Steps {
		step := Step{Name: rawStep.Name, Run: rawStep.Run, Timeout: timeout}
		if rawStep.Compile != nil {
			step.Compile = *rawStep.Compile
		}
		if rawStep.Timeout != "" {
			step.Timeout, err = duration(rawStep.Timeout)
			if err != nil {
				return job, err
			}
		}
		step.Stdin, err = resolveStream(root, dir, rawStep.Stdin)
		if err != nil {
			return job, err
		}
		if rawStep.Expected != nil {
			step.Expected.ExitCode = rawStep.Expected.ExitCode
			for _, field := range []struct {
				raw *rawStream
				out **OutputExpectation
			}{
				{rawStep.Expected.Stdout, &step.Expected.Stdout}, {rawStep.Expected.Stderr, &step.Expected.Stderr},
			} {
				if field.raw == nil {
					continue
				}
				content, err := resolveStream(root, dir, field.raw)
				if err != nil {
					return job, err
				}
				*field.out = &OutputExpectation{Content: content, Match: MatchMode(field.raw.Match)}
			}
		}
		job.Steps = append(job.Steps, step)
	}
	return job, nil
}

func resolveStream(root *sourceReader, dir string, stream *rawStream) ([]byte, error) {
	if stream == nil {
		return nil, nil
	}
	if stream.Value != nil {
		return []byte(*stream.Value), nil
	}
	content, _, err := root.read(dir + "/" + stream.Path)
	return content, err
}
