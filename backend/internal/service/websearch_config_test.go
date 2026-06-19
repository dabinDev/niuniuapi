package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type missingWebSearchSettingRepo struct{}

func (missingWebSearchSettingRepo) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}

func (missingWebSearchSettingRepo) GetValue(context.Context, string) (string, error) {
	return "", ErrSettingNotFound
}

func (missingWebSearchSettingRepo) Set(context.Context, string, string) error {
	return nil
}

func (missingWebSearchSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (missingWebSearchSettingRepo) SetMultiple(context.Context, map[string]string) error {
	return nil
}

func (missingWebSearchSettingRepo) GetAll(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}

func (missingWebSearchSettingRepo) Delete(context.Context, string) error {
	return nil
}

func TestGetWebSearchEmulationConfigMissingSettingReturnsDefault(t *testing.T) {
	webSearchEmulationSF.Forget(sfKeyWebSearchConfig)
	webSearchEmulationCache.Store((*cachedWebSearchEmulationConfig)(nil))
	t.Cleanup(func() {
		webSearchEmulationSF.Forget(sfKeyWebSearchConfig)
		webSearchEmulationCache.Store((*cachedWebSearchEmulationConfig)(nil))
	})

	svc := NewSettingService(missingWebSearchSettingRepo{}, &config.Config{})
	cfg, err := svc.GetWebSearchEmulationConfig(context.Background())

	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.False(t, cfg.Enabled)
	require.Empty(t, cfg.Providers)
}
