package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func createTempStorage(t *testing.T) (*Storage, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	tokenFile := filepath.Join(tmpDir, "auth.yml")

	storage := &Storage{
		tokenDir:  tmpDir,
		tokenFile: tokenFile,
	}

	return storage, func() {
		_ = os.Remove(tokenFile)
	}
}

// TestStorage tests storage.
func TestStorage(t *testing.T) {
	validToken := &StoredToken{
		Token:     "valid.jwt.token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	expiredToken := &StoredToken{
		Token:     "expired.jwt.token",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	t.Run("SaveToken and LoadToken success", func(t *testing.T) {
		store, cleanup := createTempStorage(t)
		defer cleanup()

		err := store.SaveToken(validToken)
		require.NoError(t, err)

		loaded, err := store.LoadToken()
		require.NoError(t, err)
		require.Equal(t, validToken.Token, loaded.Token)
	})

	t.Run("LoadToken should fail if expired", func(t *testing.T) {
		store, cleanup := createTempStorage(t)
		defer cleanup()

		data, err := yaml.Marshal(expiredToken)
		require.NoError(t, err)

		err = os.WriteFile(store.tokenFile, data, 0600)
		require.NoError(t, err)

		loaded, err := store.LoadToken()
		require.Error(t, err)
		require.Nil(t, loaded)
		require.Contains(t, err.Error(), "token has expired")
	})

	t.Run("LoadToken should fail if file missing", func(t *testing.T) {
		store, _ := createTempStorage(t)

		_, err := store.LoadToken()
		require.Error(t, err)
		require.Contains(t, err.Error(), "not authenticated")
	})

	t.Run("RemoveToken deletes token", func(t *testing.T) {
		store, cleanup := createTempStorage(t)
		defer cleanup()

		err := store.SaveToken(validToken)
		require.NoError(t, err)

		err = store.RemoveToken()
		require.NoError(t, err)

		_, err = os.Stat(store.tokenFile)
		require.True(t, os.IsNotExist(err))
	})

	t.Run("RemoveToken errors if already deleted", func(t *testing.T) {
		store, _ := createTempStorage(t)

		err := store.RemoveToken()
		require.Error(t, err)
		require.Contains(t, err.Error(), "not logged in")
	})
}
