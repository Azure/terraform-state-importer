package hcl

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"

	"github.com/azure/terraform-state-importer/types"
)

type IHclClient interface {
	WriteImportBlocks(resources []types.ImportBlock, fileName string)
	WriteDestroyBlocks(resources []types.DestroyBlock, fileName string)
}

type HclClient struct {
	TerraformModulePath string
	Logger              *logrus.Logger
}

func NewHclClient(terraformModulePath string, logger *logrus.Logger) *HclClient {
	return &HclClient{
		TerraformModulePath: terraformModulePath,
		Logger:              logger,
	}
}

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

	for _, destroyBlock := range destroyBlocks {
		resourceBlock := hclFile.Body().AppendNewBlock("terraform_data", nil)
		provisionerBlock := resourceBlock.Body().AppendNewBlock("provisioner", []string{"local-exec"})
		destroyCommand := fmt.Sprintf(`$resourceID = (az resource show --ids %s | ConvertFrom-Json | Select-Object -ExpandProperty id)
		if ($resourceID -ne $null) {
			az resource delete --ids $resourceID --no-wait --yes
		} else {
			Write-Host "Resource %s not found, skipping deletion."
		}`, destroyBlock.ID, destroyBlock.ID)
		provisionerBlock.Body().SetAttributeValue("command", cty.StringVal(destroyCommand))
		provisionerBlock.Body().SetAttributeValue("shell", cty.StringVal("pwsh"))
		hclFile.Body().AppendNewline()
	}

	err := os.WriteFile(hclFilePath, hclFile.Bytes(), 0644)
	if err != nil {
		hclClient.Logger.Fatal("Error writing file: ", err)
	}

	hclClient.Logger.Infof("HCL imports file %s written to: %s", fileName, hclFilePath)
}
