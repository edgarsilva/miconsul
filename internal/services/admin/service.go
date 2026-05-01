package admin

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"miconsul/internal/server"
)

type service struct {
	*server.Server
	modelNameProvider modelNameProvider
}

type modelNameProvider interface {
	ListModelNames() ([]string, error)
}

type fileModelNameProvider struct {
	modelsDir string
}

func NewService(s *server.Server) (service, error) {
	return NewServiceWithModelNameProvider(s, nil)
}

func NewServiceWithModelNameProvider(s *server.Server, provider modelNameProvider) (service, error) {
	if s == nil {
		return service{}, errors.New("admin service requires a non-nil server")
	}
	if provider == nil {
		modelsDir, err := resolveModelsDir()
		if err != nil {
			return service{}, err
		}
		provider = fileModelNameProvider{modelsDir: modelsDir}
	}

	return service{
		Server:            s,
		modelNameProvider: provider,
	}, nil
}

func (provider fileModelNameProvider) ListModelNames() ([]string, error) {
	dirEntries, err := os.ReadDir(provider.modelsDir)
	if err != nil {
		return nil, err
	}

	models := make([]string, 0, len(dirEntries))
	for _, entry := range dirEntries {
		modelName, err := findModelName(provider.modelsDir, entry)
		if err != nil {
			continue
		}
		models = append(models, modelName)
	}

	return models, nil
}

func findModelName(modelsDir string, dirEntry fs.DirEntry) (string, error) {
	filename := filepath.Join(modelsDir, dirEntry.Name())

	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	modelName := ""
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "--model:") {
			parts := strings.Split(line, ":")
			if len(parts) < 2 {
				continue
			}

			modelName = parts[1]
		}
	}

	if modelName == "" {
		return "", errors.New("model name not found in " + dirEntry.Name())
	}

	return modelName, nil
}

func resolveModelsDir() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("failed to resolve admin service source path")
	}

	modelsDir := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../models"))
	return modelsDir, nil
}
