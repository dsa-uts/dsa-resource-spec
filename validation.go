package resource

import (
	_ "crypto/sha256"
	"fmt"
	"io/fs"
	"reflect"
	"regexp"
	"strings"

	"github.com/distribution/reference"
	"github.com/go-playground/validator/v10"
	"github.com/go-playground/validator/v10/non-standard/validators"
	"golang.org/x/mod/semver"
)

var fieldValidator = func() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterAlias("identifier", "hostname,lowercase,excludes=.,max=63")
	if err := v.RegisterValidation("notblank", validators.NotBlank); err != nil {
		panic(err)
	}
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		return strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
	})
	return v
}()

var fullVersion = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:[-+].*)?$`)

// Version is a v-prefixed semantic version with all three numeric components.
type Version string

func (v Version) Validate() error {
	if !fullVersion.MatchString(string(v)) || !semver.IsValid(string(v)) {
		return fmt.Errorf("invalid version %q", v)
	}
	return nil
}

// RuntimePath is a POSIX relative path, independent of the local filesystem.
type RuntimePath string

func (p RuntimePath) Validate() error {
	s := string(p)
	if s == "." || !fs.ValidPath(s) || strings.ContainsAny(s, "\\:\x00") {
		return fmt.Errorf("invalid relative path %q", p)
	}
	return nil
}

// SandboxImage is a fully qualified container image with a tag or digest.
type SandboxImage string

func (s SandboxImage) Validate() error {
	r, err := reference.ParseNamed(string(s))
	if err != nil {
		return fmt.Errorf("invalid fully qualified image %q: %w", s, err)
	}
	_, tag := r.(reference.Tagged)
	_, digest := r.(reference.Digested)
	if !tag && !digest {
		return fmt.Errorf("image requires tag or digest: %s", s)
	}
	return nil
}
