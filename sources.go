package resource

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
)

// Record the exact inputs that the resolver reads, including shared materials.
// Map encoding gives a stable order despite workflow/job map iteration order.
type sourceReader struct {
	root   *os.Root
	hashes map[string]string
}

func (r *sourceReader) read(name string) ([]byte, bool, error) {
	data, executable, err := readMaterial(r.root, name)
	if err == nil {
		r.hashes[name] = fmt.Sprintf("%t:%x", executable, sha256.Sum256(data))
	}
	return data, executable, err
}

func (r *sourceReader) hash() string {
	data, _ := json.Marshal(r.hashes)
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}
