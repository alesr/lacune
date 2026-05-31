package gcbench

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeGoMod(t *testing.T, dir, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o644))
}

func TestParseGoModVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		goMod         string
		wantVersion   string
		wantGreenTea  bool
		wantErrSubstr string
	}{
		{
			name:         "go 1.24",
			goMod:        "module example.com/app\n\ngo 1.24\n",
			wantVersion:  "1.24",
			wantGreenTea: false,
		},
		{
			name:         "go 1.25",
			goMod:        "module example.com/app\n\ngo 1.25\n",
			wantVersion:  "1.25",
			wantGreenTea: false,
		},
		{
			name:         "go 1.25.6",
			goMod:        "module example.com/app\n\ngo 1.25.6\n",
			wantVersion:  "1.25.6",
			wantGreenTea: false,
		},
		{
			name:         "go 1.26",
			goMod:        "module example.com/app\n\ngo 1.26\n",
			wantVersion:  "1.26",
			wantGreenTea: true,
		},
		{
			name:         "go 1.26rc1",
			goMod:        "module example.com/app\n\ngo 1.26rc1\n",
			wantVersion:  "1.26rc1",
			wantGreenTea: true,
		},
		{
			name:          "missing go directive",
			goMod:         "module example.com/app\n",
			wantErrSubstr: "explicit 'go' declaration directive not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			if tt.goMod != "" {
				writeGoMod(t, dir, tt.goMod)
			}

			version, greenTea, err := parseGoModVersion(dir)

			if tt.wantErrSubstr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrSubstr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantVersion, version)
			assert.Equal(t, tt.wantGreenTea, greenTea)
		})
	}

	t.Run("no go.mod file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()

		_, _, err := parseGoModVersion(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unable to read go.mod")
	})
}
