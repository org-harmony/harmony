package web

import (
	"github.com/org-harmony/harmony/core/trans"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestNewTemplater(t *testing.T) {
	t.Run("new templater with base template", func(t *testing.T) {
		dir, baseDir := setupDirectories(t)
		cfg := setupConfig(dir, baseDir)

		templater, err := NewTemplater(cfg.UI, trans.NewTranslator(), FromBaseTemplate())
		assert.NoError(t, err)
		assert.NotNil(t, templater)
	})

	t.Run("new templater without options", func(t *testing.T) {
		dir, baseDir := setupDirectories(t)
		cfg := setupConfig(dir, baseDir)

		templater, err := NewTemplater(cfg.UI, trans.NewTranslator())
		assert.Error(t, err)
		assert.Nil(t, templater)
	})

	t.Run("new templater with landing page template", func(t *testing.T) {
		dir, baseDir := setupDirectories(t)
		cfg := setupConfig(dir, baseDir)
		setupLandingPage(t, dir, cfg.UI.Templates)

		templater, err := NewTemplater(cfg.UI, trans.NewTranslator(), FromLandingPageTemplate())
		assert.NoError(t, err)
		assert.NotNil(t, templater)
	})
}

func TestTemplaterTemplate(t *testing.T) {
	dir, baseDir := setupDirectories(t)
	cfg := setupConfig(dir, baseDir)
	setupLandingPage(t, dir, cfg.UI.Templates)

	t.Run("fetch base template", func(t *testing.T) {
		templater, _ := NewTemplater(cfg.UI, trans.NewTranslator(), FromBaseTemplate())
		template, err := templater.Template(BaseTemplate)
		assert.NoError(t, err)
		assert.NotNil(t, template)
	})

	t.Run("fetch landing page template", func(t *testing.T) {
		templater, _ := NewTemplater(cfg.UI, trans.NewTranslator(), FromLandingPageTemplate())
		template, err := templater.Template(LandingPageTemplate)
		assert.NoError(t, err)
		assert.NotNil(t, template)
	})

	t.Run("fetch non-existent template", func(t *testing.T) {
		templater, _ := NewTemplater(cfg.UI, trans.NewTranslator(), FromBaseTemplate())
		template, err := templater.Template("non-existent-template")
		assert.Error(t, err)
		assert.Nil(t, template)
	})
}

func TestCtrlTmplUtilFunc(t *testing.T) {
	dir, baseDir := setupDirectories(t)
	cfg := setupConfig(dir, baseDir)
	translator := trans.NewTranslator()

	funcMap := ctrlTmplUtilFunc(translator, cfg.UI)

	t.Run("test translation functions", func(t *testing.T) {
		translateFunc, exists := funcMap["t"]
		assert.True(t, exists)
		assert.NotNil(t, translateFunc)

		translateFormatFunc, exists := funcMap["tf"]
		assert.True(t, exists)
		assert.NotNil(t, translateFormatFunc)
	})

	t.Run("test asset function", func(t *testing.T) {
		assetFunc, exists := funcMap["asset"]
		assert.True(t, exists)
		assert.NotNil(t, assetFunc)

		assetFuncS, ok := assetFunc.(func(string) string)
		assert.True(t, ok)

		result := assetFuncS("image.png")
		assert.Equal(t, filepath.Join(cfg.UI.AssetsUri, "image.png"), result)
	})
}

// setupLandingPageContent creates a landing page template file in the given directory.
func setupLandingPage(t *testing.T, dir string, tCfg *TemplatesCfg) {
	landingPageContent := `{{ define "landing-page" }}
	<html>
	<head>
		<title>Harmony</title>
	</head>
	<body>
		<h1>Harmony</h1>
	</body>
	{{ end }}`

	landingPagePath := filepath.Join(dir, "landing_page.go.html")
	err := os.WriteFile(landingPagePath, []byte(landingPageContent), 0644)
	require.NoError(t, err)

	tCfg.LandingPageFilepath = filepath.Join(dir, "landing_page.go.html")
}
