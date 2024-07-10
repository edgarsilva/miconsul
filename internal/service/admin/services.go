package admin

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"miconsul/internal/server"
	"os"
	"path/filepath"
	"strings"
)

type service struct {
	*server.Server
}

func NewService(s *server.Server) service {
	return service{
		Server: s,
	}
}

func FindModelName(dirEntry fs.DirEntry) (string, error) {
	filename := dirEntry.Name()

	f, err := os.Open("internal/model/" + filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	modelName := ""
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
		if strings.Contains(line, "--model:") {
			parts := strings.Split(line, ":")
			if len(parts) < 2 {
				continue
			}

			modelName = parts[1]
		}
	}

	if modelName == "" {
		return "", errors.New("model name not found in " + filename)
	}

	return modelName, nil
}

func FindFile(path string) (string, error) {
	err := filepath.WalkDir(path, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Println(err)
	}

	return "", nil
}
