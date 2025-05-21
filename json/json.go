package json

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

type IJsonClient interface {
	Export(resources any, fileName string)
	Import(fileName string) map[string]any
}

type JsonClient struct {
	WorkingFolderPath string
	Logger            *logrus.Logger
}

func NewJsonClient(workingFolderPath string, logger *logrus.Logger) *JsonClient {
	return &JsonClient{
		WorkingFolderPath: workingFolderPath,
		Logger:            logger,
	}
}

func (jsonClient *JsonClient) Export(resources any, fileName string) {
	jsonResources, err := json.Marshal(resources)
	if err != nil {
		jsonClient.Logger.Fatal("Error during Marshal(): ", err)
	}
	jsonFilePath := filepath.Join(jsonClient.WorkingFolderPath, fileName)
	err = os.WriteFile(jsonFilePath, jsonResources, 0644)
	if err != nil {
		jsonClient.Logger.Fatal("Error writing file: ", err)
	}
}

func (jsonClient *JsonClient) Import(fileName string) map[string]any {
	jsonFilePath := filepath.Join(jsonClient.WorkingFolderPath, fileName)

	content, err := os.ReadFile(jsonFilePath)
	if err != nil {
		jsonClient.Logger.Fatal("Error when opening file: ", err)
	}

	var payload map[string]any
	err = json.Unmarshal(content, &payload)
	if err != nil {
		jsonClient.Logger.Fatal("Error during Unmarshal(): ", err)
	}
	return payload
}
