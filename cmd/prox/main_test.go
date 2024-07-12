package main

import (
	"github.com/fgrosse/prox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"os"
	"testing"
)

func TestParseProcessesFile(t *testing.T) {
	cases := map[string]struct {
		dir  string
		path string
	}{
		"default Procfile": {dir: "_testdata/default-to-Procfile"},
		"default Proxfile": {dir: "_testdata/default-to-Proxfile"},
		"Proxfile":         {dir: "_testdata", path: "Proxfile"},
		"Procfile":         {path: "_testdata/Procfile"},
		"Procfile.dev":     {path: "_testdata/Procfile.dev"},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			logger = zaptest.NewLogger(t)
			env := prox.Environment{}

			if c.dir != "" {
				changeDirectory(t, c.dir)
			}

			pp, err := processes(env, c.path)
			require.NoError(t, err)
			assert.Len(t, pp, 3)
		})
	}
}

func changeDirectory(t *testing.T, dir string) {
	t.Helper()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	err = os.Chdir(dir)
	require.NoError(t, err)

	t.Cleanup(func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	})
}
