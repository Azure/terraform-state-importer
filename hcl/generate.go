package hcl

import (
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"

	"github.com/azure/terraform-state-importer/types"
)

type IHclClient interface {
	Export(resources []types.ImportBlock, fileName string)
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

func (hclClient *HclClient) Export(importBlocks []types.ImportBlock, fileName string) {
	hclFilePath := filepath.Join(hclClient.TerraformModulePath, fileName)
	hclFile := hclwrite.NewEmptyFile()

	for _, importBlock := range importBlocks {
		resourceBlock := hclFile.Body().AppendNewBlock("import", nil)
		resourceBlock.Body().SetAttributeValue("id", cty.StringVal(importBlock.ID))
		resourceBlock.Body().SetAttributeValue("to", cty.StringVal(importBlock.To))
	}

	err := os.WriteFile(hclFilePath, hclFile.Bytes(), 0644)
	if err != nil {
		hclClient.Logger.Fatal("Error writing file: ", err)
	}
}
