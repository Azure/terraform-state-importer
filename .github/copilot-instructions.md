# Terraform State Importer

Terraform State Importer is a Go CLI tool that helps migrate Azure workloads to Terraform by analyzing existing Azure resources and generating import blocks for new Terraform modules.

**Always reference these instructions first and fallback to search or bash commands only when you encounter unexpected information that does not match the info here.**

## Working Effectively

### Prerequisites and Installation
Install required dependencies in this exact order:
- **Install Go 1.24.3+**: The project requires Go 1.24.3 or newer. Verify with `go version`
- **Install Terraform CLI**:
  ```bash
  wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
  echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
  sudo apt update && sudo apt install -y terraform
  terraform version
  ```
- **Azure CLI**: Usually pre-installed. Verify with `az version`
- **Azure Authentication**: Run `az login` before using the tool

### Build and Test Commands
- **Download dependencies**: `go mod download` -- takes 5-10 seconds
- **Build**: `go build -v ./...` -- takes 30 seconds first time, 5 seconds subsequent builds. NEVER CANCEL - set timeout to 60+ minutes
- **Test**: `go test -v ./...` -- takes 5 seconds. NEVER CANCEL - set timeout to 30+ minutes  
- **Create binary**: `go build -o terraform-state-importer .`
- **Code formatting**: `go fmt ./...` -- always run before committing
- **Static analysis**: `go vet ./...` -- always run before committing

### Running the Tool
- **Test CLI**: `./terraform-state-importer --help`
- **Main command**: `./terraform-state-importer run --help`
- **Basic usage**: `./terraform-state-importer run --terraformModulePath <path> --config <config.yaml>`

## Validation Scenarios

**CRITICAL**: Always test these scenarios after making changes:

### Core Functionality Test
1. **Build the tool**: `go build -o terraform-state-importer .`
2. **Test help output**: `./terraform-state-importer --help`
3. **Test run command help**: `./terraform-state-importer run --help`  
4. **Create minimal Terraform module**:
   ```bash
   mkdir -p /tmp/terraform-test
   cd /tmp/terraform-test
   cat > main.tf << 'EOF'
   terraform {
     required_providers {
       azurerm = {
         source  = "hashicorp/azurerm"
         version = "~>4.0"
       }
     }
   }
   provider "azurerm" {
     features {}
   }
   resource "azurerm_resource_group" "example" {
     name     = "rg-example"
     location = "East US"
   }
   EOF
   ```
5. **Create test config**:
   ```bash
   cat > test-config.yaml << 'EOF'
   subscriptionIds:
     - "00000000-0000-0000-0000-000000000000"
   ignoreResourceIDPatterns: []
   ignoreResourceTypePatterns: []
   resourceGraphQueries:
     - name: "Resource Groups"
       scope: "Subscription"
       query: |
         resources
         | where type == "microsoft.resources/resourcegroups"
         | project id, name, type, location, subscriptionId, resourceGroup
   nameFormats: []
   deleteCommands: []
   EOF
   ```
6. **Test without Azure auth**: `./terraform-state-importer run --terraformModulePath /tmp/terraform-test --config ./test-config.yaml --skipInitPlanShow`
   - Should fail with proper Azure authentication error message
7. **Run all tests**: `go test -v ./...` -- must pass completely

### Authentication Test
- **Verify Azure CLI**: `az version`
- **Test auth setup**: `az account show` -- should show current subscription or require login
- **Expected auth error**: When running without `az login`, tool should show clear error message about Azure credentials

## Configuration and Structure

### Important Directories
- **`/cmd/`**: CLI command definitions using Cobra framework
- **`/analyzer/`**: Core mapping logic with comprehensive tests
- **`/azure/`**: Azure Resource Graph integration
- **`/terraform/`**: Terraform CLI integration (calls `terraform init`, `plan`, `show`)
- **`/types/`**: Shared data structures
- **`/filepathparser/`**: Path handling utilities with tests
- **`/.config/`**: Example configuration files for Azure Landing Zone scenarios

### Configuration Files
The tool requires YAML configuration with these mandatory sections:
- **subscriptionIds**: Array of Azure subscription IDs to analyze
- **resourceGraphQueries**: Queries to find Azure resources
- **ignoreResourceIDPatterns**: Regex patterns to exclude Azure resources  
- **ignoreResourceTypePatterns**: Terraform resource types to ignore
- **nameFormats**: Custom name mapping rules
- **deleteCommands**: Azure CLI commands for resource cleanup

Example configurations are in `.config/alz.*.config.yaml` files.

### Key Project Files
- **`go.mod`**: Specifies Go 1.24.3 requirement
- **`main.go`**: Entry point, calls cmd.Execute()
- **`.github/workflows/go.yml`**: CI pipeline (build and test)
- **`.github/workflows/release.yml`**: Release pipeline using GoReleaser

## Build Timing and Timeouts

**CRITICAL TIMING REQUIREMENTS**:
- **Initial build with dependencies**: 30 seconds - NEVER CANCEL, set timeout to 60+ minutes
- **Subsequent builds**: 5 seconds - NEVER CANCEL, set timeout to 30+ minutes  
- **Tests**: 5 seconds - NEVER CANCEL, set timeout to 30+ minutes
- **Module downloads**: 5-10 seconds first time
- **Binary compilation**: 2-3 seconds
- **Code formatting**: < 1 second
- **Static analysis**: < 1 second

**NEVER CANCEL any go build, go test, or go mod commands**. Builds may occasionally take longer due to network or system conditions.

## Common Development Tasks

### Before Committing Changes
Always run these commands in order:
1. **Format code**: `go fmt ./...`
2. **Static analysis**: `go vet ./...` 
3. **Build**: `go build -v ./...`
4. **Test**: `go test -v ./...`
5. **Test CLI**: `go build -o terraform-state-importer . && ./terraform-state-importer --help`

### Adding New Features
1. **Understand the workflow**: Tool queries Azure Resource Graph → runs Terraform plan → maps resources → generates import blocks
2. **Key interfaces**: Check `/analyzer/mapping.go` for core mapping logic
3. **Add tests**: Follow patterns in `analyzer/mapping_test.go` and `filepathparser/parser_test.go`
4. **Configuration changes**: Update example configs in `.config/` directory if needed

### Debugging
- **Enable debug logging**: `./terraform-state-importer run --verbosity debug`
- **JSON logging**: `./terraform-state-importer run --structuredLogs`
- **Skip Terraform operations**: Use `--skipInitPlanShow` to test Azure integration only
- **Test specific modules**: Use `--terraformModulePath` to point to test directories

## Dependencies and External Tools

### Required External Commands
- **`terraform`**: Must be in PATH, used for init/plan/show operations
- **`az`**: Azure CLI, must be authenticated with `az login`
- **`go`**: Go 1.24.3+, check with `go version`

### Authentication Requirements
- **Azure credentials**: The tool uses DefaultAzureCredential chain:
  1. Environment variables (AZURE_TENANT_ID, AZURE_CLIENT_ID, etc.)
  2. Workload Identity (in Kubernetes)
  3. Managed Identity
  4. Azure CLI (`az login`)
  5. Azure Developer CLI

### Network Requirements  
- **Azure API access**: Tool calls Azure Resource Manager and Resource Graph APIs
- **Terraform registry access**: For provider downloads during `terraform init`
- **Go module proxy**: For downloading Go dependencies

## Troubleshooting

### Build Issues
- **"command not found: go"**: Install Go 1.24.3+
- **Module download failures**: Check network connectivity, retry `go mod download`
- **Build timeout**: Increase timeout, never cancel builds

### Runtime Issues
- **"terraform: command not found"**: Install Terraform CLI
- **"az: command not found"**: Install or check Azure CLI
- **Azure credential errors**: Run `az login` and verify with `az account show`
- **Config file errors**: Validate YAML syntax, check required sections exist

### Testing Issues  
- **Test failures**: Check if tests require specific Azure resources or configs
- **Timing issues**: Some tests may be sensitive to system performance
- **Authentication in CI**: Tests should not require real Azure credentials