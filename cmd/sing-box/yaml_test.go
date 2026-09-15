package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sagernet/sing-box/include"

	"github.com/stretchr/testify/require"
)

const yamlTestConfig = `
log:
  level: info
outbounds:
  - type: direct
    tag: direct
`

const jsonTestConfig = `{
  "log": {
    "level": "info"
  },
  "outbounds": [
    {
      "type": "direct",
      "tag": "direct"
    }
  ]
}`

func TestReadConfigYAML(t *testing.T) {
	t.Parallel()

	globalCtx = include.Context(context.Background())

	tempDir := t.TempDir()
	yamlPath := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(yamlPath, []byte(yamlTestConfig), 0o644)
	require.NoError(t, err)
	jsonPath := filepath.Join(tempDir, "config.json")
	err = os.WriteFile(jsonPath, []byte(jsonTestConfig), 0o644)
	require.NoError(t, err)

	yamlEntry, err := readConfigAt(yamlPath)
	require.NoError(t, err)
	jsonEntry, err := readConfigAt(jsonPath)
	require.NoError(t, err)
	require.Equal(t, jsonEntry.options.Log, yamlEntry.options.Log)
	require.Equal(t, jsonEntry.options.Outbounds, yamlEntry.options.Outbounds)
}
