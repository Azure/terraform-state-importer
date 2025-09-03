# Terraform State Importer

A comprehensive tool for migrating large Azure workloads to Terraform modules by analyzing existing Azure resources and generating import blocks.

## Overview

The Terraform State Importer simplifies the process of importing existing Azure infrastructure into Terraform modules. It automates the complex task of mapping Azure resource IDs to Terraform resources and helps resolve conflicts during the import process.

### Key Features

- **Automated Resource Discovery**: Queries Azure using Resource Graph to find existing resources
- **Intelligent Resource Mapping**: Maps Azure resources to your Terraform module configuration
- **Conflict Resolution**: Identifies and helps resolve mapping conflicts through CSV workflows
- **Import Block Generation**: Creates ready-to-use Terraform import blocks
- **Flexible Configuration**: Supports both subscription and management group scopes

### How It Works

The tool operates in two main phases:

1. **Resource ID Mapping**: Maps Azure resource IDs to resources defined in your Terraform module
2. **Resource Attribute Mapping**: Provides guidance for aligning Azure resource attributes with module variables

The primary workflow involves:
1. Discovering Azure resources using configurable Resource Graph queries
2. Analyzing your Terraform module through `terraform plan`
3. Mapping resources and identifying conflicts
4. Resolving conflicts through an interactive CSV workflow
5. Generating final import blocks for successful imports

## CLI Reference

### Basic Command Structure

```bash
terraform-state-importer [global-flags] run [command-flags]
```

### Global Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--config` | | Path to configuration YAML file | `$HOME/.terraform-state-importer.yaml` |
| `--verbosity` | `-v` | Logging level (trace, debug, info, warn, error, fatal, panic) | `info` |
| `--structuredLogs` | | Output logs in JSON format for structured logging | `false` |
| `--help` | `-h` | Show help information | |

### Run Command Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--terraformModulePath` | `-t` | Path to the Terraform module to import resources into | `.` (current directory) |
| `--workingFolderPath` | `-w` | Working directory for temporary files and outputs | `.` (current directory) |
| `--issuesCsv` | `-c` | Path to resolved issues CSV file for generating import blocks | (empty - analysis mode) |
| `--planAsTextOnly` | `-p` | Generate only a text-based Terraform plan without analysis | `false` |
| `--planSubscriptionID` | `-s` | Override subscription ID for Terraform plan operations | (uses az cli default) |
| `--skipInitPlanShow` | `-x` | Skip terraform init, plan, and show steps (for debugging) | `false` |
| `--skipInitOnly` | `-k` | Skip only the terraform init step | `false` |

### Command Usage Examples

#### Initial Analysis
Run analysis to discover resources and generate issues for resolution:

```bash
terraform-state-importer run \
  --terraformModulePath ./my-terraform-module \
  --config ./config.yaml \
  --verbosity debug
```

#### Generate Import Blocks
After resolving issues in the CSV file, generate final import blocks:

```bash
terraform-state-importer run \
  --terraformModulePath ./my-terraform-module \
  --config ./config.yaml \
  --issuesCsv ./resolved-issues.csv
```

#### Generate Plan Only
Generate a text plan without running full analysis:

```bash
terraform-state-importer run \
  --planAsTextOnly \
  --terraformModulePath ./my-terraform-module
```

#### Advanced Configuration
Use custom working directory and override subscription:

```bash
terraform-state-importer run \
  --terraformModulePath /path/to/module \
  --workingFolderPath /tmp/import-work \
  --planSubscriptionID "12345678-1234-1234-1234-123456789012" \
  --config ./custom-config.yaml
```

## Configuration Reference

The tool uses a YAML configuration file to define Azure resource queries, filtering rules, and mapping behaviors.

### Configuration File Structure

```yaml
# Azure scope configuration (choose one)
subscriptionIDs:              # Target specific subscriptions
  - "subscription-id-1"
  - "subscription-id-2"

managementGroupIDs:           # Target management groups (alternative to subscriptionIDs)
  - "mg-id-1"

# Resource filtering
ignoreResourceIDPatterns:     # Azure resource ID patterns to ignore
  - "/subscriptions/.*/providers/Microsoft.Authorization/policyAssignments"
  - "resourceGroups/NetworkWatcherRG"

ignoreResourceTypePatterns:   # Terraform resource types to ignore
  - "random_uuid.telemetry"
  - "module.management_groups"

# Azure resource queries
resourceGraphQueries:         # Custom Resource Graph queries
  - name: "Resource Groups"
    scope: "Subscription"     # "Subscription" or "ManagementGroup"
    query: |
      resourcecontainers
      | where type == "microsoft.resources/subscriptions/resourcegroups"
      | project id, name, type, location, subscriptionId, resourceGroup = name

# Resource naming patterns
nameFormats:                  # Custom name mapping rules
  - type: "azurerm_log_analytics_solution"
    nameFormat: "%s(%s)"
    nameMatchType: "Exact"    # "Exact", "IDEndsWith", "IDContains"
    nameFormatArguments:
      - "solution_name"
      - "workspace_name"

# Resource cleanup commands
deleteCommands:               # Commands to run for resource cleanup
  - type: "microsoft.authorization/roleassignments"
    command: "az role assignment delete --ids %s"
```

### Core Configuration Sections

#### Azure Scope Configuration

Define which Azure resources to target:

**Subscription-scoped:**
```yaml
subscriptionIDs:
  - "12345678-1234-1234-1234-123456789012"
  - "87654321-4321-4321-4321-210987654321"
```

**Management Group-scoped:**
```yaml
managementGroupIDs:
  - "alz"
  - "production"
```

#### Resource Filtering

**Ignore Azure Resources by Pattern:**
```yaml
ignoreResourceIDPatterns:
  - "/subscriptions/.*/providers/Microsoft.Authorization/policyAssignments"  # All policy assignments
  - "resourceGroups/Default-.*"                                               # Default resource groups
  - "/providers/Microsoft.OperationsManagement/solutions/.*"                  # Monitoring solutions
```

**Ignore Terraform Resources by Pattern:**
```yaml
ignoreResourceTypePatterns:
  - "random_uuid.telemetry"      # Telemetry resources
  - "modtm"                      # Module telemetry
  - "terraform_data"             # Terraform data resources
  - "time_sleep"                 # Time delay resources
  - "module.management_groups.*" # Entire modules
```

#### Resource Graph Queries

Define how to discover Azure resources using Kusto Query Language (KQL):

**Basic Resource Query:**
```yaml
resourceGraphQueries:
  - name: "Virtual Machines"
    scope: "Subscription"
    query: |
      resources
      | where type == "microsoft.compute/virtualmachines"
      | project id, name, type, location, subscriptionId, resourceGroup
```

**Complex Query with Transformations:**
```yaml
resourceGraphQueries:
  - name: "Virtual Network Subnets"
    scope: "Subscription"
    query: |
      resources
      | where type == "microsoft.network/virtualnetworks"
      | project id, name, type, location, subscriptionId, resourceGroup, subnets = properties.subnets
      | mv-expand subnets
      | project name = tostring(subnets.name), id = tostring(subnets.id), type = tostring(subnets.type), location, subscriptionId, resourceGroup
```

**Query Output Requirements:**
All queries must return these columns:
- `id`: Azure resource ID
- `name`: Resource name
- `type`: Azure resource type
- `location`: Azure region
- `subscriptionId`: Subscription ID
- `resourceGroup`: Resource group name

#### Name Format Mapping

Configure custom naming patterns for resources that need special mapping logic:

**Exact Name Matching:**
```yaml
nameFormats:
  - type: "azurerm_log_analytics_solution"
    nameFormat: "%s(%s)"
    nameMatchType: "Exact"
    nameFormatArguments:
      - "solution_name"
      - "workspace_name"
```

**ID Pattern Matching:**
```yaml
nameFormats:
  - type: "Microsoft.Authorization/policyAssignments"
    nameFormat: "%s/providers/Microsoft.Authorization/policyAssignments/%s"
    nameMatchType: "IDEndsWith"
    nameFormatArguments:
      - "parent_id"
      - "name"
```

**Name Match Types:**
- `Exact`: Exact string match
- `IDEndsWith`: Azure resource ID ends with pattern
- `IDContains`: Azure resource ID contains pattern

#### Delete Commands

Define cleanup commands for resources that may need to be deleted before import:

```yaml
deleteCommands:
  - type: "microsoft.authorization/roleassignments"
    command: "az role assignment delete --ids %s"
  - type: "microsoft.network/virtualnetworkpeerings"
    command: "az network vnet peering delete --ids %s"
```

The `%s` placeholder is replaced with the resource ID during execution.

### Example Configurations

#### Subscription-scoped Configuration
```yaml
subscriptionIDs:
  - "12345678-1234-1234-1234-123456789012"

ignoreResourceIDPatterns:
  - "resourceGroups/NetworkWatcherRG"
  - "Microsoft.OperationsManagement/solutions/ChangeTracking"

ignoreResourceTypePatterns:
  - "random_uuid.telemetry"
  - "module.management_groups"

resourceGraphQueries:
  - name: "Resource Groups"
    scope: "Subscription"
    query: |
      resourcecontainers
      | where type == "microsoft.resources/subscriptions/resourcegroups"
      | project id, name, type, location, subscriptionId, resourceGroup = name

  - name: "Top Level Resources"
    scope: "Subscription"
    query: |
      resources
      | project name, id, type, location, subscriptionId, resourceGroup
```

#### Management Group-scoped Configuration
```yaml
managementGroupIDs:
  - "alz"

ignoreResourceIDPatterns:
  - "/subscriptions/.*/providers/Microsoft.Authorization/policyAssignments"

ignoreResourceTypePatterns:
  - "random_uuid.telemetry"
  - "module.connectivity"

resourceGraphQueries:
  - name: "Management Groups"
    scope: "ManagementGroup"
    query: |
      resourcecontainers
      | where type == "microsoft.management/managementgroups"
      | project id, name, type, location, subscriptionId, resourceGroup

  - name: "Policy Definitions"
    scope: "ManagementGroup"
    query: |
      policyresources
      | where type == "microsoft.authorization/policydefinitions"
      | project id, name, type, location, subscriptionId, resourceGroup

nameFormats:
  - type: "Microsoft.Authorization/policyAssignments"
    nameFormat: "%s/providers/Microsoft.Authorization/policyAssignments/%s"
    nameMatchType: "IDEndsWith"
    nameFormatArguments:
      - "parent_id"
      - "name"
```

## Usage Guide

### Prerequisites

- Azure CLI installed and authenticated (`az login`)
- Terraform installed and configured
- Go 1.19+ (for building from source) or download pre-built binary
- Appropriate Azure permissions to read resources in target subscriptions/management groups

### Installation

#### Download Pre-built Binary
```bash
# Download latest release (replace with actual URL when available)
wget https://github.com/Azure/terraform-state-importer/releases/latest/download/terraform-state-importer-linux-amd64.tar.gz
tar -xzf terraform-state-importer-linux-amd64.tar.gz
chmod +x terraform-state-importer
```

#### Build from Source
```bash
git clone https://github.com/Azure/terraform-state-importer.git
cd terraform-state-importer
go build -o terraform-state-importer .
```

### Step-by-Step Workflow

#### Step 1: Prepare Your Environment

**Create a working directory:**
```bash
mkdir ~/terraform-import-project
cd ~/terraform-import-project
```

**Create your configuration file:**
```bash
cat > config.yaml << 'EOF'
subscriptionIDs:
  - "12345678-1234-1234-1234-123456789012"  # Replace with your subscription ID

ignoreResourceIDPatterns:
  - "resourceGroups/NetworkWatcherRG"
  - "resourceGroups/Default-.*"

ignoreResourceTypePatterns:
  - "random_uuid.telemetry"
  - "modtm"

resourceGraphQueries:
  - name: "Resource Groups"
    scope: "Subscription"
    query: |
      resourcecontainers
      | where type == "microsoft.resources/subscriptions/resourcegroups"
      | project id, name, type, location, subscriptionId, resourceGroup = name
      
  - name: "All Resources"
    scope: "Subscription" 
    query: |
      resources
      | project name, id, type, location, subscriptionId, resourceGroup
EOF
```

**Prepare your Terraform module:**
```bash
# Clone or copy your existing Terraform module
git clone <your-terraform-module-repo> ./terraform-module
cd ./terraform-module

# Ensure your module is ready
terraform init
cd ..
```

#### Step 2: Run Initial Analysis

Run the tool to discover resources and generate the issues CSV:

```bash
terraform-state-importer run \
  --terraformModulePath ./terraform-module \
  --config ./config.yaml \
  --verbosity info
```

**What happens:**
1. Tool queries Azure for resources using your configuration
2. Runs `terraform plan` on your module
3. Maps Azure resources to planned Terraform resources  
4. Generates `issues.csv` in your working directory
5. Outputs summary of discovered resources and mapping conflicts

#### Step 3: Review and Resolve Issues

Open the generated `issues.csv` file. You'll see three types of issues:

**Issue Types:**
- **MultipleResourceIDs**: Multiple Azure resources could map to one Terraform resource
- **NoResourceID**: Terraform resource has no matching Azure resource
- **UnusedResourceID**: Azure resource has no matching Terraform resource

**Resolution Process:**
1. Open `issues.csv` in Excel or any CSV editor
2. For each issue, set the `Action` column according to the resolution strategy
3. Save the file as `resolved-issues.csv`

#### Step 4: Generate Import Blocks

Run the tool again with your resolved issues:

```bash
terraform-state-importer run \
  --terraformModulePath ./terraform-module \
  --config ./config.yaml \
  --issuesCsv ./resolved-issues.csv
```

**What happens:**
1. Tool validates all issues have resolutions
2. Generates `import.tf` file with import blocks
3. Creates any necessary delete commands

#### Step 5: Execute Import

Apply the generated import blocks:

```bash
cd ./terraform-module

# Review the generated import blocks
cat import.tf

# Execute imports
terraform plan  # Should show resources being imported
terraform apply # Apply the imports
```

### Issue Resolution Guide

**The CSV file contains these columns:**
- `Issue ID`: Unique identifier for the issue
- `Issue Type`: Type of mapping conflict
- `Terraform Address`: Resource address in your Terraform plan
- `Mapped Resource ID`: Corresponding Azure resource ID (if found)
- `Action`: Resolution action you choose (to be filled)
- `Action ID`: Reference to related issue (used for Replace actions)

#### MultipleResourceIDs Issues

**Problem**: Multiple Azure resources match a single Terraform resource

**Resolution Options:**
1. **Use**: Select the correct Azure resource to import
2. **Ignore**: Skip the incorrect matches

**Steps:**
1. Review all rows with the same `Terraform Address`
2. Identify the correct `Mapped Resource ID` based on your requirements
3. Set `Action` to `Use` for the correct resource
4. Set `Action` to `Ignore` for all other matches

**Example:**
```csv
Issue ID,Issue Type,Terraform Address,Mapped Resource ID,Action,Action ID
1,MultipleResourceIDs,module.network.azurerm_resource_group.main,/subscriptions/.../resourceGroups/prod-rg,Use,
2,MultipleResourceIDs,module.network.azurerm_resource_group.main,/subscriptions/.../resourceGroups/test-rg,Ignore,
```

#### NoResourceID Issues

**Problem**: Terraform resource has no matching Azure resource

**Resolution Options:**
1. **Leave blank**: Update your Terraform module to match existing resources, re-run analysis
2. **Ignore**: Terraform will create a new resource
3. **Replace**: Link to an unused Azure resource (see Replace workflow below)

**Steps for Replace workflow:**
1. Find corresponding `UnusedResourceID` issue for the resource you want to use
2. Set `Action` to `Replace` for the `NoResourceID` issue
3. Set `Action ID` to the `Issue ID` of the related `UnusedResourceID` issue
4. Set `Action` to `Replace` for the `UnusedResourceID` issue
5. Set `Action ID` to the `Issue ID` of the related `NoResourceID` issue

**Example:**
```csv
Issue ID,Issue Type,Terraform Address,Mapped Resource ID,Action,Action ID
3,NoResourceID,module.network.azurerm_resource_group.new,N/A,Replace,4
4,UnusedResourceID,N/A,/subscriptions/.../resourceGroups/existing-rg,Replace,3
```

#### UnusedResourceID Issues

**Problem**: Azure resource exists but has no matching Terraform resource

**Resolution Options:**
1. **Leave blank**: Update your Terraform module to include this resource, re-run analysis  
2. **Ignore**: Leave the Azure resource unmanaged by Terraform
3. **Replace**: Link to a Terraform resource (see Replace workflow above)

### Advanced Configuration

#### Working with Management Groups

For management group-scoped resources (policies, management groups):

```yaml
managementGroupIDs:
  - "alz"
  - "production"

resourceGraphQueries:
  - name: "Management Groups"
    scope: "ManagementGroup"
    query: |
      resourcecontainers
      | where type == "microsoft.management/managementgroups"
      | project id, name, type, location, subscriptionId, resourceGroup

nameFormats:
  - type: "Microsoft.Authorization/policyAssignments"
    nameFormat: "%s/providers/Microsoft.Authorization/policyAssignments/%s"
    nameMatchType: "IDEndsWith"
    nameFormatArguments:
      - "parent_id"
      - "name"
```

#### Custom Resource Queries

For complex resource relationships:

```yaml
resourceGraphQueries:
  - name: "Virtual Network Peerings"
    scope: "Subscription"
    query: |
      resources
      | where type == "microsoft.network/virtualnetworks"
      | project id, name, type, location, subscriptionId, resourceGroup, peerings = properties.virtualNetworkPeerings
      | mv-expand peerings
      | project name = tostring(peerings.name), 
                id = tostring(peerings.id), 
                type = "microsoft.network/virtualnetworkpeerings",
                location, subscriptionId, resourceGroup
```

#### Performance Optimization

For large environments:

```bash
# Skip terraform init if already done
terraform-state-importer run --skipInitOnly --config config.yaml

# Skip plan generation for import-only mode
terraform-state-importer run --skipInitPlanShow --issuesCsv resolved-issues.csv

# Use structured logging for automation
terraform-state-importer run --structuredLogs --verbosity debug
```

### Troubleshooting

#### Common Issues

**Authentication Errors:**
```bash
# Ensure Azure CLI is logged in
az login
az account show  # Verify correct subscription
```

**Terraform Plan Failures:**
```bash
# Verify module can plan successfully
cd terraform-module
terraform init
terraform plan
```

**Resource Query Failures:**
- Verify subscription IDs are correct
- Check Resource Graph query syntax
- Ensure adequate Azure permissions

**Import Block Generation Issues:**
- Verify all issues in CSV have actions assigned
- Check CSV formatting (no extra commas, proper encoding)
- Ensure Action IDs reference valid Issue IDs for Replace actions

#### Debug Mode

Use verbose logging to troubleshoot:

```bash
terraform-state-importer run \
  --verbosity debug \
  --structuredLogs \
  --config config.yaml 2>&1 | tee debug.log
```

### Best Practices

#### Configuration Management
- Store configuration files in version control
- Use environment-specific configurations
- Document custom Resource Graph queries
- Test queries independently in Azure Portal

#### Workflow Management
- Run analysis on a copy of your Terraform module first
- Back up existing state files before importing
- Use `terraform plan` to preview imports before applying
- Import resources in small batches for complex environments

#### Resource Organization
- Group related resources in the same import session
- Use consistent naming patterns
- Document any manual interventions required
