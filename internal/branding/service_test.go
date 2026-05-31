package branding

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sqlite "github.com/company/smartticket/internal/database/moderncsqlite"
	"github.com/company/smartticket/internal/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Branding{}))
	return NewService(db, t.TempDir())
}

func strptr(s string) *string { return &s }

func TestGet_LazilyCreatesDefaults(t *testing.T) {
	s := newTestService(t)

	b, err := s.Get()
	require.NoError(t, err)
	require.Equal(t, uint(singletonID), b.ID)
	require.Equal(t, defaults.AppName, b.AppName)
	require.Equal(t, defaults.PrimaryColor, b.PrimaryColor)

	// A second call returns the same singleton, not a new row.
	again, err := s.Get()
	require.NoError(t, err)
	require.Equal(t, b.ID, again.ID)
}

func TestUpdate_PatchesProvidedFieldsOnly(t *testing.T) {
	s := newTestService(t)

	b, err := s.Update(&UpdateRequest{
		AppName:      strptr("Acme Helpdesk"),
		PrimaryColor: strptr("#3b82f6"),
	})
	require.NoError(t, err)
	require.Equal(t, "Acme Helpdesk", b.AppName)
	require.Equal(t, "#3b82f6", b.PrimaryColor)
	// Untouched fields keep their defaults.
	require.Equal(t, defaults.WorkspaceName, b.WorkspaceName)
}

func TestUpdate_RejectsInvalidColor(t *testing.T) {
	s := newTestService(t)
	_, err := s.Update(&UpdateRequest{PrimaryColor: strptr("not-a-color")})
	require.Error(t, err)
}

func TestUpdate_AcceptsShortAndAlphaHex(t *testing.T) {
	s := newTestService(t)
	for _, c := range []string{"#fff", "#f59e0b", "#f59e0bcc"} {
		_, err := s.Update(&UpdateRequest{PrimaryColor: strptr(c)})
		require.NoErrorf(t, err, "color %s should be valid", c)
	}
}

func TestSaveAndDeleteLogo(t *testing.T) {
	s := newTestService(t)

	b, err := s.SaveLogo("brand.png", "image/png", strings.NewReader("PNGDATA"), 7)
	require.NoError(t, err)
	require.NotEmpty(t, b.LogoPath)
	require.Equal(t, ".png", b.LogoExt)

	// The file exists on disk and is served with the right content type.
	path, ct, err := s.LogoFile()
	require.NoError(t, err)
	require.Equal(t, "image/png", ct)
	_, statErr := os.Stat(path)
	require.NoError(t, statErr)
	require.Equal(t, "logo.png", filepath.Base(path))

	// Delete removes both the record and the file.
	b, err = s.DeleteLogo()
	require.NoError(t, err)
	require.Empty(t, b.LogoPath)
	_, statErr = os.Stat(path)
	require.True(t, os.IsNotExist(statErr))

	_, _, err = s.LogoFile()
	require.Error(t, err)
}

func TestSaveLogo_RejectsBadExtension(t *testing.T) {
	s := newTestService(t)
	_, err := s.SaveLogo("evil.exe", "application/octet-stream", strings.NewReader("x"), 1)
	require.Error(t, err)
}

func TestSaveLogo_ReplacesPreviousExtension(t *testing.T) {
	s := newTestService(t)

	_, err := s.SaveLogo("a.png", "image/png", strings.NewReader("one"), 3)
	require.NoError(t, err)
	first := filepath.Join(s.dataPath, "branding", "logo.png")
	_, statErr := os.Stat(first)
	require.NoError(t, statErr)

	// Uploading a different format should remove the old logo file.
	b, err := s.SaveLogo("b.svg", "image/svg+xml", strings.NewReader("<svg/>"), 6)
	require.NoError(t, err)
	require.Equal(t, ".svg", b.LogoExt)
	_, statErr = os.Stat(first)
	require.True(t, os.IsNotExist(statErr), "old .png logo should be removed")
}
