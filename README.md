# Terraform State Importer

A comprehensive tool for migrating large Azure workloads to Terraform modules by analyzing existing Azure resources and generating import blocks.

## Table of Contents

- [Overview](#overview)
- [CLI Reference](#cli-reference)
- [Configuration Reference](#configuration-reference)
- [Usage Guide](#usage-guide)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
  - [Quick Start](#quick-start)
  - [Step-by-Step Workflow](#step-by-step-workflow)
  - [Issue Resolution Guide](#issue-resolution-guide)
- [Advanced Configuration](#advanced-configuration)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

## Overview

The Terraform State Importer simplifies the process of importing existing Azure infrastructure into Terraform modules. It automates the complex task of mapping Azure resource IDs to Terraform resources and helps resolve conflicts during the import process.

### Key Features

- **Automated Resource Discovery**: Queries Azure using Resource Graph to find existing resources
- **Intelligent Resource Mapping**: Maps Azure resources to your Terraform module configuration
- **Conflict Resolution**: Identifies and helps resolve mapping conflicts through CSV workflows
- **CSV Import/Export**: Export issues to CSV for resolution and re-import them to continue the workflow
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
4. Exporting conflicts to CSV for review and resolution
5. Re-importing resolved issues from CSV
6. Generating final import blocks for successful imports

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

### Pre-built Configuration Files

The repository includes ready-to-use configuration files for common Azure Landing Zone scenarios in the `.config/` directory:

| Configuration File | Purpose | Scope | Description |
|--------------------|---------|-------|-------------|
| `alz.management-groups.config.yaml` | Management Groups | Management Group | Import Azure Landing Zone management group hierarchy, policy definitions, policy set definitions, policy assignments, custom role definitions, and role assignments |
| `alz.connectivity.hub-and-spoke.config.yaml` | Hub and Spoke Networking | Subscription | Import Azure Landing Zone connectivity resources using hub-and-spoke topology including VNets, subnets, NSGs, route tables, VPN/ExpressRoute gateways, private DNS zones, and DDoS protection |
| `alz.connectivity.virtual-wan.config.yaml` | Virtual WAN Networking | Subscription | Import Azure Landing Zone connectivity resources using Virtual WAN topology including virtual WANs, virtual hubs, VPN/ExpressRoute gateways, firewall policies, and private DNS zones |

**Usage:**
```bash
# Using pre-built configuration for management groups
terraform-state-importer run \
  --config .config/alz.management-groups.config.yaml \
  --terraformModulePath ./my-alz-module

# Using pre-built configuration for hub-and-spoke connectivity
terraform-state-importer run \
  --config .config/alz.connectivity.hub-and-spoke.config.yaml \
  --terraformModulePath ./my-connectivity-module
```

**Customizing Pre-built Configs:**
1. Copy the appropriate config file to your working directory
2. Update subscription IDs or management group IDs
3. Modify filters and queries as needed for your environment
4. Adjust the `cloud` setting if using sovereign clouds

### Supported Azure Clouds

The tool supports all Azure cloud environments. Set the `cloud` property in your configuration file:

```yaml
# Target Cloud for the migration
cloud: "AzurePublic"  # Options: AzurePublic, AzureUSGovernment, AzureChina
```

**Available Cloud Values:**
- `AzurePublic` - Global Azure (default)
- `AzureUSGovernment` - Azure US Government
- `AzureChina` - Azure China (Mooncake)

**Note**: Ensure your Azure CLI is authenticated to the correct cloud environment before running the tool:
```bash
# For Azure US Government
az cloud set --name AzureUSGovernment
az login

# For Azure China
az cloud set --name AzureChinaCloud
az login
```

### Configuration File Structure

```yaml
# Target cloud environment
cloud: "AzurePublic"          # Options: AzurePublic, AzureGovernment, AzureChina

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

# Property mapping for cross-resource references
propertyMappings:             # Custom property mappings between resources
  - type: "azapi_resource"
    subType: "Microsoft.Network/privateDnsZones/virtualNetworkLinks"
    mappings:
      - targetProperties:
          - name: "private_dns_zone_name"
            from: "name"
        sourceLookupProperties:
          - name: "meta.address"
            target: "meta.address"
            replacements:
              - regex: "\\.module\\.virtual_network_links\\[.*\\]\\.azapi_resource\\.private_dns_zone_network_link"
                replacement: ".azapi_resource.private_dns_zone"

# Resource cleanup commands
deleteCommands:               # Commands to run for resource cleanup
  - type: "microsoft.authorization/roleassignments"
    command: "az role assignment delete --ids %s"
```

### Core Configuration Sections

#### Cloud Environment

Specify the target Azure cloud environment (optional, defaults to `AzurePublic`):

```yaml
cloud: "AzurePublic"  # Options: AzurePublic, AzureGovernment, AzureChina
```

This setting ensures the tool connects to the correct Azure cloud endpoints. Make sure your Azure CLI is authenticated to the matching cloud environment.

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

Configure custom naming patterns for resources that need special mapping logic. By default, the tool uses the `name` property from each resource. Use `nameFormats` when:
- Resource names don't have a standard `name` property
- Names are composite values (e.g., `solution_name(workspace_name)`)
- You need to match resources by ID patterns rather than exact names
- Resources are nested and need hierarchical matching

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

**SubType-Aware Matching:**


```yaml
nameFormats:
  # Most specific: matches by SubType directly
  - type: "Microsoft.Authorization/policyAssignments"
    nameFormat: "%s/providers/Microsoft.Authorization/policyAssignments/%s"
    nameMatchType: "IDExact"  # Recommended for policy artifacts
    nameFormatArguments:
      - "parent_id"
      - "name"
  
  # Alternative: explicit Type+SubType specification
  - type: "azapi_resource"
    subType: "Microsoft.Authorization/policyDefinitions"
    nameFormat: "%s/providers/Microsoft.Authorization/policyDefinitions/%s"
    nameMatchType: "IDExact"
    nameFormatArguments:
      - "parent_id"
      - "name"
```

**Type Prefiltering:**

When a resource has a SubType (e.g., `Microsoft.Authorization/policyAssignments`), the tool automatically filters Azure Resource Graph results to only consider resources with matching Azure types before applying name/ID matching. This eliminates cross-type false matches, particularly important for:
- Policy assignments vs policy definitions
- Policy definitions vs policy set definitions
- Any resources with similar names but different types

**Name Match Types:**
- `Exact`: Exact string match on resource name
- `IDEndsWith`: Azure resource ID ends with the constructed pattern
- `IDContains`: Azure resource ID contains the constructed pattern
- `IDExact`: Exact match on full Azure resource ID (recommended for policy artifacts)

#### Property Mapping

Configure custom property mappings for resources that need to reference properties from other resources in the Terraform plan. This is useful when a resource needs to look up values from parent or related resources.

**Basic Property Mapping:**
```yaml
propertyMappings:
  - type: "azapi_resource"
    subType: "Microsoft.Network/privateDnsZones/virtualNetworkLinks"
    mappings:
      - targetProperties:
          - name: "private_dns_zone_name"
            from: "name"
        sourceLookupProperties:
          - name: "meta.address"
            target: "meta.address"
            replacements:
              - regex: "\\.module\\.virtual_network_links\\[.*\\]\\.azapi_resource\\.private_dns_zone_network_link"
                replacement: ".azapi_resource.private_dns_zone"
```

**Configuration Elements:**
- `type`: The Terraform resource type to apply the mapping to
- `subType` (optional): Azure resource subtype for more specific matching (e.g., for `azapi_resource`)
- `mappings`: Array of property mapping rules
  - `targetProperties`: Properties to populate in the target resource
    - `name`: Name of the property to set
    - `from`: Source property name to read the value from
  - `sourceLookupProperties`: Properties used to locate the source resource
    - `name`: Property path to read from the source resource
    - `target`: Property path where the result will be used
    - `replacements`: Array of regex transformations to find the related resource
      - `regex`: Regular expression pattern to match in the property value
      - `replacement`: String to replace the matched pattern with

**Use Cases:**
- Looking up parent resource properties (e.g., private DNS zone name from a virtual network link)
- Cross-referencing related resources in hierarchical structures
- Resolving dependencies between nested resources
- Mapping resource relationships that aren't explicitly defined in Terraform attributes

#### Meta Properties Reference

The tool automatically adds special `meta.*` properties to each resource during Terraform plan processing. These properties can be referenced in both `propertyMappings` and `nameFormats` configurations to access resource metadata.

**Available Meta Properties:**

| Property | Description | Always Available | Example Value |
|----------|-------------|------------------|---------------|
| `meta.type` | The Terraform resource type | Yes | `azurerm_resource_group` |
| `meta.name` | The resource name from the Terraform configuration | Yes | `this` |
| `meta.address` | The full Terraform resource address including modules | Yes | `module.network.azurerm_virtual_network.main` |
| `meta.location` | The Azure region/location of the resource | If resource has `location` property | `eastus`, `uksouth` |
| `meta.subtype` | The Azure resource type for `azapi_resource` | Only for `azapi_resource` | `Microsoft.Network/privateDnsZones/virtualNetworkLinks` |
| `meta.apiversion` | The Azure API version for `azapi_resource` | Only for `azapi_resource` | `2020-06-01` |

**Usage in Property Mappings:**

Meta properties are particularly useful for looking up related resources by matching on `meta.address` or other identifying properties:

```yaml
propertyMappings:
  - type: "azapi_resource"
    subType: "Microsoft.Network/privateDnsZones/virtualNetworkLinks"
    mappings:
      - targetProperties:
          - name: "private_dns_zone_name"
            from: "name"
        sourceLookupProperties:
          - name: "meta.address"  # Use meta.address to identify the resource
            target: "meta.address"
            replacements:
              - regex: "\\.virtual_network_link$"
                replacement: ".private_dns_zone"
```

**Usage in Name Formats:**

Meta properties can be used as arguments in name format strings to construct resource names:

```yaml
nameFormats:
  - type: "azapi_resource"
    subType: "Microsoft.Network/privateDnsZones/virtualNetworkLinks"
    nameFormat: "providers/Microsoft.Network/privateDnsZones/%s/virtualNetworkLinks/%s"
    nameMatchType: "IDContains"
    nameFormatArguments:
      - "private_dns_zone_name"  # Regular property (populated via propertyMapping)
      - "name"                    # Regular property from resource
      # Note: meta.location and other meta properties can be used here too
```

**Common Patterns:**

1. **Matching by Address**: Use `meta.address` with regex replacements to find parent or related resources
2. **Type-Specific Logic**: Use `meta.type` and `meta.subtype` to apply different rules to different resource types
3. **Location-Based Matching**: Use `meta.location` when resources need to reference others in the same region
4. **Hierarchical Resources**: Use `meta.address` to navigate module hierarchies and find related resources

**Note**: All meta properties are read-only and automatically populated by the tool. They cannot be modified through configuration.

#### Policy Artifacts Configuration

Azure policy resources (assignments, definitions, set definitions) commonly share names across scopes and require precise matching. The tool provides specialized support for these resources:

**Recommended Configuration for Policy Assignments:**
```yaml
nameFormats:
  - type: "Microsoft.Authorization/policyAssignments"
    nameFormat: "%s/providers/Microsoft.Authorization/policyAssignments/%s"
    nameMatchType: "IDExact"  # Prevents cross-scope matches
    nameFormatArguments:
      - "parent_id"  # Management group or subscription scope
      - "name"       # Assignment name
```

**Recommended Configuration for Policy Definitions:**
```yaml
nameFormats:
  - type: "Microsoft.Authorization/policyDefinitions"
    nameFormat: "%s/providers/Microsoft.Authorization/policyDefinitions/%s"
    nameMatchType: "IDExact"
    nameFormatArguments:
      - "parent_id"
      - "name"
```

**Recommended Configuration for Policy Set Definitions:**
```yaml
nameFormats:
  - type: "Microsoft.Authorization/policySetDefinitions"
    nameFormat: "%s/providers/Microsoft.Authorization/policySetDefinitions/%s"
    nameMatchType: "IDExact"
    nameFormatArguments:
      - "parent_id"
      - "name"
```

**Why IDExact for Policy Artifacts?**
- Prevents matching assignments with the same name across different management groups
- Eliminates cross-type matches (assignments vs definitions)
- Ensures deterministic one-to-one resource mapping
- Critical when policy names are reused across organizational hierarchy

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

### Quick Start

For those already familiar with the tool, here's the basic workflow:

```bash
# 1. Initial analysis - generates issues.csv
terraform-state-importer run --config config.yaml --terraformModulePath ./my-module

# 2. Edit issues.csv to resolve conflicts (set Action column)

# 3. Generate import blocks with resolved issues
terraform-state-importer run --config config.yaml --terraformModulePath ./my-module --issuesCsv issues.csv

# 4. Review and apply imports
cd ./my-module
terraform plan
terraform apply
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
4. Generates output files in your working directory:
   - `issues.csv`: Mapping conflicts that need resolution
   - `issues.json`: JSON format of the same issues
   - `resources.json`: All discovered Terraform plan resources with their properties
5. Outputs summary of discovered resources and mapping conflicts

**Note**: If all resources map cleanly with no conflicts, `issues.csv` will be empty or not generated, and `imports.tf` will be created automatically. You can skip to Step 5 in this case.

#### Step 3: Review and Resolve Issues

Open the generated `issues.csv` file. You'll see three types of issues:

**Issue Types:**
- **MultipleResourceIDs**: Multiple Azure resources could map to one Terraform resource
  - *Common cause*: Resources with similar names across different regions or environments
- **NoResourceID**: Terraform resource has no matching Azure resource
  - *Common cause*: New resources in your Terraform code that haven't been deployed yet
- **UnusedResourceID**: Azure resource has no matching Terraform resource
  - *Common cause*: Resources exist in Azure but aren't yet defined in your Terraform module

**Resolution Process:**
1. Open `issues.csv` in Excel or any CSV editor
2. For each issue, set the `Action` column according to the resolution strategy
3. Save the file (you can save it with the same name or as `resolved-issues.csv`)

**Note:** The tool can read and import CSV files to deserialize issues back into the system, allowing you to:
- Resume work on previously exported issues
- Share issue resolution work across team members
- Version control issue resolutions alongside your infrastructure code
- Programmatically process and update issues before re-importing

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
2. Generates `imports.tf` file with Terraform import blocks for resources marked with `Use` action
3. Generates `destroy.tf` file with commands for resources marked with `Destroy` action (if applicable)
4. Generates `final.json` with all successfully mapped resources
5. Outputs summary of imports to be performed

#### Step 5: Execute Import

Apply the generated import blocks:

```bash
cd ./terraform-module

# Review the generated import blocks
cat imports.tf

# Run terraform plan to see what will be imported
# Resources with import blocks should show as "will be imported"
terraform plan

# If destroy.tf was generated, review and execute cleanup commands first
if [ -f destroy.tf ]; then
  cat destroy.tf
  # Execute the commands manually if needed
fi

# Apply the imports
terraform apply
```

### Issue Resolution Guide

**The CSV file contains these columns:**
- `Issue ID`: Unique identifier for the issue (e.g., `i-a1b2c3`)
- `Issue Type`: Type of mapping conflict (`MultipleResourceIDs`, `NoResourceID`, or `UnusedResourceID`)
- `Resource Address`: Full Terraform resource address (e.g., `module.network.azurerm_resource_group.main`)
- `Resource Name`: Extracted resource name used for mapping
- `Resource Type`: Terraform resource type (e.g., `azurerm_resource_group`)
- `Resource Location`: Azure region (e.g., `eastus`, `uksouth`)
- `Mapped Resource ID`: Corresponding Azure resource ID if found (e.g., `/subscriptions/.../resourceGroups/rg-name`)
- `Action`: Resolution action you choose - leave empty for first run, then set to: `Use`, `Ignore`, `Replace`, or `Destroy`
- `Action ID`: Reference to related issue ID (required only for `Replace` actions to link paired resources)

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
- Ensure adequate Azure permissions (minimum: `Reader` role on subscriptions/management groups)
- For management group queries, ensure you have access to the management group hierarchy
- Test your queries in Azure Portal's Resource Graph Explorer first

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

**Debugging with Output Files:**
- `resources.json`: Contains all Terraform plan resources with their properties and metadata
  - Useful for verifying resource names, types, and available properties
  - Shows all `meta.*` properties that can be used in configurations
  - Check this file if resources aren't matching as expected
- `issues.json`: Machine-readable version of issues.csv for automation
- `final.json`: Successfully mapped resources after issue resolution

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
