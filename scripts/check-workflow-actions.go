package main

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: check-workflow-actions.go WORKFLOW_DIRECTORY")
		os.Exit(2)
	}

	err := filepath.WalkDir(os.Args[1], func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".yml" && ext != ".yaml" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var document any
		if err := yaml.Unmarshal(data, &document); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		emitUses(document)
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func emitUses(value any) {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			emitUses(item)
		}
	case map[string]any:
		for key, item := range typed {
			if key == "uses" {
				fmt.Println(item)
			}
			emitUses(item)
		}
	}
}
