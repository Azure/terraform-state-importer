package terraform

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func (planClient *PlanClient) ExtractUpdateResourcesFromPlan(sourcePlanFileName string, outputPlanFileName string) {
	planFilePath := filepath.Join(planClient.WorkingFolderPath, sourcePlanFileName)
	outputPlanFilePath := filepath.Join(planClient.WorkingFolderPath, outputPlanFileName)

	file, err := os.Open(planFilePath)
	if err != nil {
		planClient.Logger.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	updateLines := []string{}

	keepLines := false
	resourceCommentLines := []string{}

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "  #") {
			resourceCommentLines = append(resourceCommentLines, line)
		}

		if strings.HasPrefix(line, "  ~ resource ") || strings.HasPrefix(line, "-/+ resource") {
			shouldIgnore := false
			for _, pattern := range planClient.IgnoreResourceTypePatterns {
				if shouldIgnore {
					break
				}
				for _, commentLine := range resourceCommentLines {
					matched, err := regexp.MatchString(pattern, commentLine)
					if err != nil {
						planClient.Logger.Debugf("Error matching pattern %s: %v", pattern, err)
						continue
					}
					if matched {
						shouldIgnore = true
						break
					}
				}
			}
			if shouldIgnore {
				planClient.Logger.Tracef("Ignoring Resource: %s", line)
				keepLines = false
			} else {
				planClient.Logger.Tracef("Keeping Resource: %s", line)
				keepLines = true
			}
		}

		if keepLines {
			updateLines = append(updateLines, line)
		}

		if keepLines && strings.HasPrefix(line, "    }") {
			keepLines = false
			updateLines = append(updateLines, "")
		}

		if strings.HasPrefix(line, "    }") {
			resourceCommentLines = []string{}
		}
	}

	if err := scanner.Err(); err != nil {
		planClient.Logger.Fatal(err)
	}

	content := strings.Join(updateLines, "\n")

	file, err = os.Create(outputPlanFilePath)
	if err != nil {
		planClient.Logger.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(content)
	if err != nil {
		planClient.Logger.Fatalf("Failed to write to file: %v", err)
	}
}
