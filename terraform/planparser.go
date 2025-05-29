package terraform

import (
	"bufio"
	"os"
	"path/filepath"
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

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "  ~ resource ") || strings.HasPrefix(line, "-/+ resource") {
			keepLines = true
		}

		if keepLines {
			updateLines = append(updateLines, line)
		}

		if keepLines && strings.HasPrefix(line, "    }") {
			keepLines = false
			updateLines = append(updateLines, "")
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
