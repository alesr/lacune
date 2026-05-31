package gcbench

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// parseGoModVersion reads the "go" directive from go.mod and reports the
// version string plus whether Green Tea is the default GC for that version
// (Go >= 1.26).
func parseGoModVersion(targetPath string) (string, bool, error) {
	file, err := os.Open(filepath.Join(targetPath, "go.mod"))
	if err != nil {
		return "", false, fmt.Errorf("unable to read go.mod: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "go ") {
			parts := strings.Split(line, " ")

			if len(parts) >= 2 {
				versionStr := parts[1]
				vParts := strings.Split(versionStr, ".")

				if len(vParts) < 2 {
					return versionStr, false, nil
				}

				// Strip pre-release suffixes (e.g. "1.26rc1" -> "26").
				var cleanMinor strings.Builder

				for _, char := range vParts[1] {
					if char >= '0' && char <= '9' {
						cleanMinor.WriteString(string(char))
					} else {
						break
					}
				}

				minor, _ := strconv.Atoi(cleanMinor.String())
				return versionStr, minor >= 26, nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", false, fmt.Errorf("unable to read go.mod: %w", err)
	}

	return "", false, fmt.Errorf("explicit 'go' declaration directive not found in go.mod")
}
