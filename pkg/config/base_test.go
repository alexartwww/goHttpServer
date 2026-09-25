package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// chdirTemp переключает рабочую директорию на временную и возвращает её путь.
// NewBase/NewFile/NewStreams ищут конфиг по относительным путям, поэтому тесту
// нужно управлять именно текущей директорией, а не списком файлов.
func chdirTemp(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	return dir
}

func writeFile(t *testing.T, dir, name string, body []byte) {
	t.Helper()

	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, body, 0o644))
}

func TestNewBase_Success(t *testing.T) {
	dir := chdirTemp(t)

	testConfig := Base{
		Author:  "Test Author",
		Email:   "test@example.com",
		Version: "1.0.0",
		Name:    "Test App",
		Listen:  ":8080",
		Log:     "info",
	}

	yamlData, err := yaml.Marshal(testConfig)
	require.NoError(t, err)
	writeFile(t, dir, "config.yml", yamlData)

	config, err := NewBase("config", "video-cacher")

	assert.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, testConfig.Author, config.Author)
	assert.Equal(t, testConfig.Email, config.Email)
	assert.Equal(t, testConfig.Version, config.Version)
	assert.Equal(t, testConfig.Name, config.Name)
	assert.Equal(t, testConfig.Listen, config.Listen)
	assert.Equal(t, testConfig.Log, config.Log)
}

func TestNewBase_FileNotFound(t *testing.T) {
	chdirTemp(t)

	// Имя проекта заведомо отсутствует в /etc, иначе поиск подхватил бы
	// конфиг реальной машины.
	config, err := NewBase("absent-config", "absent-project-2f8a1c")

	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "Config file not found")
}

func TestNewBase_InvalidYAML(t *testing.T) {
	dir := chdirTemp(t)
	writeFile(t, dir, "config.yml", []byte("invalid: yaml: : :"))

	config, err := NewBase("config", "video-cacher")

	assert.Error(t, err)
	assert.Nil(t, config)
}

// Побеждает первый путь из списка поиска: <name>.yml идёт раньше,
// чем config/<name>.yml.
func TestNewBase_FirstValidFile(t *testing.T) {
	dir := chdirTemp(t)

	yaml1, err := yaml.Marshal(Base{Name: "App 1", Listen: ":8081"})
	require.NoError(t, err)
	yaml2, err := yaml.Marshal(Base{Name: "App 2", Listen: ":8082"})
	require.NoError(t, err)

	writeFile(t, dir, "custom.yml", yaml1)
	writeFile(t, dir, "config/custom.yml", yaml2)

	config, err := NewBase("custom", "video-cacher")

	assert.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "App 1", config.Name)
	assert.Equal(t, ":8081", config.Listen)
}

// Конфиг с именем проекта ищется и в config/, если в корне его нет.
func TestNewBase_FallsBackToConfigDir(t *testing.T) {
	dir := chdirTemp(t)

	data, err := yaml.Marshal(Base{Name: "From config dir", Listen: ":8083"})
	require.NoError(t, err)
	writeFile(t, dir, "config/custom.yml", data)

	config, err := NewBase("custom", "video-cacher")

	assert.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "From config dir", config.Name)
}

func TestNewBase_EmptyFile(t *testing.T) {
	dir := chdirTemp(t)
	writeFile(t, dir, "config.yml", []byte(""))

	config, err := NewBase("config", "video-cacher")

	// Пустой файл — валидный YAML, даёт пустую структуру.
	assert.NoError(t, err)
	require.NotNil(t, config)
	assert.Empty(t, config.Name)
	assert.Empty(t, config.Listen)
}

func TestNewBase_FileWithOnlyComments(t *testing.T) {
	dir := chdirTemp(t)
	writeFile(t, dir, "config.yml", []byte("# This is a comment\n# Another comment"))

	config, err := NewBase("config", "video-cacher")

	assert.NoError(t, err)
	assert.NotNil(t, config)
}

func TestNewBase_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	dir := chdirTemp(t)

	testConfig := Base{
		Author:  "Performance Test",
		Email:   "perf@test.com",
		Version: "1.0.0",
		Name:    "Perf App",
		Listen:  ":9090",
		Log:     "debug",
	}

	yamlData, err := yaml.Marshal(testConfig)
	require.NoError(t, err)
	writeFile(t, dir, "config.yml", yamlData)

	start := time.Now()
	config, err := NewBase("config", "video-cacher")
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Less(t, duration, 100*time.Millisecond, "NewBase took too long")
}

// Проверяет, что функция не паникует при логировании.
func TestNewBase_WithRealLogger(t *testing.T) {
	dir := chdirTemp(t)
	writeFile(t, dir, "config.yml", []byte("name: test"))

	config, err := NewBase("config", "video-cacher")

	assert.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "test", config.Name)
}
