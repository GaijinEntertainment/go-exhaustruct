package fileutil_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dev.gaijin.team/go/exhaustruct/v4/internal/fileutil"
)

var errPrimaryFailed = errors.New("primary failed")

func TestReader_ReadFile_NilPrimary(t *testing.T) {
	t.Parallel()

	reader := fileutil.NewReader(nil)
	filename := filepath.Join("testdata", "sample.go")

	content, err := reader.ReadFile(filename)
	require.NoError(t, err)
	assert.Contains(t, string(content), "package sample")
	assert.Contains(t, string(content), "SampleStruct")
}

func TestReader_ReadFile_PrimarySuccess(t *testing.T) {
	t.Parallel()

	expectedContent := []byte("custom content")
	primary := func(_ string) ([]byte, error) {
		return expectedContent, nil
	}

	reader := fileutil.NewReader(primary)

	content, err := reader.ReadFile("any-file.go")
	require.NoError(t, err)
	assert.Equal(t, expectedContent, content)
}

func TestReader_ReadFile_PrimaryFailsFallsBack(t *testing.T) {
	t.Parallel()

	primary := func(_ string) ([]byte, error) {
		return nil, errPrimaryFailed
	}

	reader := fileutil.NewReader(primary)
	filename := filepath.Join("testdata", "sample.go")

	// Should fall back to os.ReadFile
	content, err := reader.ReadFile(filename)
	require.NoError(t, err)
	assert.Contains(t, string(content), "package sample")
}

func TestReader_ReadFile_BothFail(t *testing.T) {
	t.Parallel()

	primary := func(_ string) ([]byte, error) {
		return nil, errPrimaryFailed
	}

	reader := fileutil.NewReader(primary)

	_, err := reader.ReadFile("nonexistent-file.go")
	require.Error(t, err)
}

func TestReader_ReadFile_NilPrimaryNonexistent(t *testing.T) {
	t.Parallel()

	reader := fileutil.NewReader(nil)

	_, err := reader.ReadFile("nonexistent-file.go")
	require.Error(t, err)
}
