package attenuator

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

func collectInventory(root, output string) (Inventory, error) {
	root = filepath.Clean(root)
	output = filepath.Clean(output)
	result := Inventory{RootReadmeExcluded: true}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != root && (path == output || pathWithin(path, output)) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path != root && excludedInventoryDirectory(entry.Name()) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path == filepath.Join(root, "README.md") {
			return nil
		}
		if path != root && entry.IsDir() {
			result.Directories++
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		result.Files++
		result.Bytes += len(data)
		switch filepath.Ext(path) {
		case ".go":
			result.GoFiles++
			result.GoLines += physicalLines(data)
		case ".gooo":
			result.GoooFiles++
			result.GoooLines += physicalLines(data)
		}
		return nil
	})
	return result, err
}

func pathWithin(path, parent string) bool {
	relative, err := filepath.Rel(parent, path)
	if err != nil || relative == "." {
		return relative == "."
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func excludedInventoryDirectory(name string) bool {
	switch name {
	case ".git", ".cache", "cache", "tmp", "temp", "output", "out", "vendor", "toolchain", "toolchains", ".toolchain":
		return true
	default:
		return false
	}
}

func physicalLines(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	lines := bytes.Count(data, []byte{'\n'})
	if data[len(data)-1] != '\n' {
		lines++
	}
	return lines
}
