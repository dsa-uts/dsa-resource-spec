package resource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

func TestOutputSizeLimits(t *testing.T) {
	for _, field := range []string{"stdout-size", "stderr-size"} {
		for _, size := range []int64{0, 1, 4, 99, 100, 119, 120, 127, 128, 129, 1024} {
			t.Run(fmt.Sprintf("%s/%dKiB", field, size), func(t *testing.T) {
				input := fmt.Sprintf(`resource: {id: sample, name: Sample, version: v1.0.0}
workflows:
  main:
    jobs:
      run:
        sandbox-image: registry.example.com/sample:v1
        limits: {step-timeout: 1s, %s: %dKiB}
        steps: [{run: echo ok}]
`, field, size)
				_, err := decodeDefinition([]byte(input))
				valid := size > 0 && size <= 128
				if (err == nil) != valid {
					t.Fatalf("YAML size %dKiB: %v", size, err)
				}
				job, err := resolveJob(nil, "", rawJob{
					SandboxImage: "registry.example.com/sample:v1",
					Limits:       rawLimits{StepTimeout: "1s"},
					Steps:        []rawStep{{Run: "echo ok"}},
				})
				if err != nil {
					t.Fatal(err)
				}
				if job.Limits.StdoutSize != 4<<10 || job.Limits.StderrSize != 4<<10 {
					t.Fatalf("unexpected default output limits: %+v", job.Limits)
				}
				if field == "stdout-size" {
					job.Limits.StdoutSize = size << 10
				} else {
					job.Limits.StderrSize = size << 10
				}
				data, err := json.Marshal(Resource{
					Metadata:  Metadata{ID: "sample", Name: "Sample", Version: "v1.0.0"},
					Workflows: map[string]Workflow{"main": {Jobs: map[string]Job{"run": job}}},
				})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := DecodeResource(bytes.NewReader(data)); (err == nil) != valid {
					t.Fatalf("JSON size %dKiB: %v", size, err)
				}
			})
		}
	}
}
