//go:build unit

package service

import (
	"archive/tar"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type updateServiceCacheStub struct {
	data string
}

func (s *updateServiceCacheStub) GetUpdateInfo(context.Context) (string, error) {
	if s.data == "" {
		return "", errors.New("cache miss")
	}
	return s.data, nil
}

func (s *updateServiceCacheStub) SetUpdateInfo(_ context.Context, data string, _ time.Duration) error {
	s.data = data
	return nil
}

type updateServiceGitHubClientStub struct {
	release *GitHubRelease
	repo    string
}

func (s *updateServiceGitHubClientStub) FetchLatestRelease(_ context.Context, repo string) (*GitHubRelease, error) {
	s.repo = repo
	return s.release, nil
}

func (s *updateServiceGitHubClientStub) DownloadFile(context.Context, string, string, int64) error {
	panic("DownloadFile should not be called when no update is available")
}

func (s *updateServiceGitHubClientStub) FetchChecksumFile(context.Context, string) ([]byte, error) {
	panic("FetchChecksumFile should not be called when no update is available")
}

func TestUpdateServicePerformUpdateNoUpdateReturnsSentinel(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{
			release: &GitHubRelease{
				TagName: "v0.1.132",
				Name:    "v0.1.132",
			},
		},
		"0.1.132",
		"release",
		"",
	)

	err := svc.PerformUpdate(context.Background())

	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNoUpdateAvailable))
	require.ErrorIs(t, err, ErrNoUpdateAvailable)
}

func TestUpdateServiceUsesDefaultNiuniuRepository(t *testing.T) {
	client := &updateServiceGitHubClientStub{
		release: &GitHubRelease{TagName: "v0.1.133", Name: "v0.1.133"},
	}
	svc := NewUpdateService(&updateServiceCacheStub{}, client, "0.1.132", "release", "")

	_, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.Equal(t, "dabinDev/niuniuapi", client.repo)
}

func TestUpdateServiceUsesConfiguredRepository(t *testing.T) {
	client := &updateServiceGitHubClientStub{
		release: &GitHubRelease{TagName: "v0.1.133", Name: "v0.1.133"},
	}
	svc := NewUpdateService(&updateServiceCacheStub{}, client, "0.1.132", "release", "owner/custom-release-channel")

	_, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.Equal(t, "owner/custom-release-channel", client.repo)
}

func TestUpdateServiceUsesNiuniuBinaryNameForOwnReleaseChannel(t *testing.T) {
	svc := NewUpdateService(&updateServiceCacheStub{}, &updateServiceGitHubClientStub{}, "0.1.132", "release", "")

	require.Equal(t, "niuniuapi", svc.binaryName())
	require.Equal(t, ".niuniuapi-update-*", svc.tempDirPattern())
}

func TestUpdateServiceExtractsNiuniuBinaryFromReleaseArchive(t *testing.T) {
	svc := NewUpdateService(&updateServiceCacheStub{}, &updateServiceGitHubClientStub{}, "0.1.132", "release", "")
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "niuniuapi_linux_amd64.tar")
	destPath := filepath.Join(tempDir, "candidate")

	createTarArchive(t, archivePath, "niuniuapi", "own release binary")

	require.NoError(t, svc.extractBinary(archivePath, destPath))
	got, err := os.ReadFile(destPath)
	require.NoError(t, err)
	require.Equal(t, "own release binary", string(got))
}

func TestUpdateServiceRejectsArchiveWithoutNiuniuBinary(t *testing.T) {
	svc := NewUpdateService(&updateServiceCacheStub{}, &updateServiceGitHubClientStub{}, "0.1.132", "release", "")
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "upstream_linux_amd64.tar")
	destPath := filepath.Join(tempDir, "candidate")

	createTarArchive(t, archivePath, "sub2api", "upstream binary")

	err := svc.extractBinary(archivePath, destPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "niuniuapi")
}

func createTarArchive(t *testing.T, path, name, content string) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, f.Close()) }()

	tw := tar.NewWriter(f)
	defer func() { require.NoError(t, tw.Close()) }()

	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: name,
		Mode: 0o755,
		Size: int64(len(content)),
	}))
	_, err = io.Copy(tw, strings.NewReader(content))
	require.NoError(t, err)
}
