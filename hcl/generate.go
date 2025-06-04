package hcl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"

	"github.com/azure/terraform-state-importer/types"
)

type IHclClient interface {
	WriteImportBlocks(resources []types.ImportBlock, fileName string)
	WriteDestroyBlocks(resources []types.DestroyBlock, fileName string)
	CleanFiles(filesToRemove []string)
}

type HclClient struct {
	TerraformModulePath string
	DeleteCommands      []types.DeleteCommand
	Logger              *logrus.Logger
}

func NewHclClient(terraformModulePath string, deleteCommands []types.DeleteCommand, logger *logrus.Logger) *HclClient {
	return &HclClient{
		TerraformModulePath: terraformModulePath,
		DeleteCommands:      deleteCommands,
		Logger:              logger,
	}
}

const defaultDeleteCommand = `$resourceID = (az resource show --ids %s | ConvertFrom-Json | Select-Object -ExpandProperty id)
if ($resourceID -ne $null) {
	Write-Host "Deleting resource..."
	az resource delete --ids $resourceID --verbose
} else {
	Write-Host "Resource not found, skipping deletion."
}`

func (hclClient *HclClient) WriteImportBlocks(importBlocks []types.ImportBlock, fileName string) {
	hclFilePath := filepath.Join(hclClient.TerraformModulePath, fileName)
	hclFile := hclwrite.NewEmptyFile()

	for _, importBlock := range importBlocks {
		resourceBlock := hclFile.Body().AppendNewBlock("import", nil)
		resourceBlock.Body().SetAttributeValue("id", cty.StringVal(importBlock.ID))
		traversal := hcl.Traversal{
			hcl.TraverseRoot{Name: importBlock.To},
		}
		resourceBlock.Body().SetAttributeTraversal("to", traversal)
		hclFile.Body().AppendNewline()
	}

	err := os.WriteFile(hclFilePath, hclFile.Bytes(), 0644)
	if err != nil {
		hclClient.Logger.Fatal("Error writing file: ", err)
	}

	hclClient.Logger.Infof("HCL imports file %s written to: %s", fileName, hclFilePath)
}

func (hclClient *HclClient) WriteDestroyBlocks(destroyBlocks []types.DestroyBlock, fileName string) {
	hclFilePath := filepath.Join(hclClient.TerraformModulePath, fileName)
	hclFile := hclwrite.NewEmptyFile()

	for i, destroyBlock := range destroyBlocks {
		resourceBlock := hclFile.Body().AppendNewBlock("resource", []string{"terraform_data", fmt.Sprintf("destroy_%03d", i+1)})

		provisionerBlock := resourceBlock.Body().AppendNewBlock("provisioner", []string{"local-exec"})

		destroyCommandTemplate := defaultDeleteCommand
		for _, deleteCommand := range hclClient.DeleteCommands {
			if strings.ToLower(deleteCommand.Type) == strings.ToLower(destroyBlock.Type) {
				destroyCommandTemplate = deleteCommand.Command
			}
		}

		destroyCommand := fmt.Sprintf(destroyCommandTemplate, destroyBlock.ID)
		resourceBlock.Body().SetAttributeValue("triggers_replace", cty.StringVal(destroyCommand))
		provisionerBlock.Body().SetAttributeValue("command", cty.StringVal(destroyCommand))
		provisionerBlock.Body().SetAttributeValue("interpreter", cty.ListVal([]cty.Value{cty.StringVal("pwsh"), cty.StringVal("-Command")}))
		hclFile.Body().AppendNewline()
	}

	err := os.WriteFile(hclFilePath, hclFile.Bytes(), 0644)
	if err != nil {
		hclClient.Logger.Fatal("Error writing file: ", err)
	}

	hclClient.Logger.Infof("HCL imports file %s written to: %s", fileName, hclFilePath)
}

func (hclClient *HclClient) CleanFiles(filesToRemove []string) {
	for _, fileName := range filesToRemove {
		filePath := filepath.Join(hclClient.TerraformModulePath, fileName)
		if _, err := os.Stat(filePath); err == nil {
			hclClient.Logger.Debugf("File %s already exists, it will be deleted", filePath)
			if err := os.Remove(filePath); err != nil {
				hclClient.Logger.Fatalf("Error deleting existing imports file: %v", err)
			}
		}
	}
}
