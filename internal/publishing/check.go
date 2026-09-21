package publishing

import resource "github.com/dsa-uts/dsa-resource-spec/v2"

// Check validates the current resources and release index.
func Check(root string) error {
	if _, err := resource.LoadManifest(root); err != nil {
		return err
	}
	_, err := readIndex(root)
	return err
}
