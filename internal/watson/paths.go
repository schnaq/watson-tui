package watson

import (
	"os"
	"path/filepath"
)

// Dir resolves the Watson data directory: explicit override > $WATSON_DIR > OS app dir.
func Dir(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	if d := os.Getenv("WATSON_DIR"); d != "" {
		return d, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "watson"), nil
}
