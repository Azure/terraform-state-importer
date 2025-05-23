# Terraform State Importer

This is a tool for running basic analysis and creating import blocks for migrating large Azure workloads to a new module in Terraform.

## Overview

The tools requires two main inputs:

- A set of subscription IDs where your deployed resources reside
- A Terraform module with variables supplied, which is the module you want to import the resources into

There are two main phases to using the tool:

1. **Resource ID Mapping**: We need to map Azure resource IDs to the resources in the module.
1. **Resource Attribute Mapping**: We need to map the attributes of the Azure resources to the variables in the module.

The tool will help you with the first phase an we then provide guidance for the second phase.

## Usage

### Setup

The following steps are required to start running the tool:

1. Create a new folder in your local machine to store the tool and the configuration files.

    ```powershell
    mkdir ~/terraform-state-importer
    cd ~/terraform-state-importer
    ```

1. Download the latest tool binary from the [releases page](TBC)

    ```powershell
    Invoke-WebRequest -Uri "<URL to the latest release>" -OutFile terraform-state-importer.zip
    Expand-Archive terraform-state-importer.zip -DestinationPath ~/terraform-state-importer
    ```

1. Create a file called `config.yaml` and copy the content from the [config.yaml](TBC) in the repository.

    ```powershell
    Invoke-WebRequest -Uri "<URL to the config.yaml>" -OutFile config.yaml
    ```

1. Identify the subscriptions IDs where your deployed resources reside. You can find these in the Azure portal by going to Subscriptions and copying the Subscription ID.

    Add the subscription IDs to your YAML `config.yaml` file under the `subscriptionIds` key.

    ```yaml
    subscriptionIds:
      - "00000000-0000-0000-0000-000000000001"
      - "00000000-0000-0000-0000-000000000002"
    ```

1. Create a folder with your destination module. This is the module you want to import the resources into. The tool will run a Terraform plan against this module and analyze the plan to determine which resources can be mapped to existing Azure resource IDs.

    In most cases, this will be a pre-existing repository that you just need to clone to your local machine.

    ```powershell
    git clone <URL to your module repository> ~/terraform-module
    ```

1. Now you can run the tool. The tool will query Azure for all resources in the specified subscriptions and run a Terraform plan against the module you provided.

    ```powershell
    ./terraform-state-importer --terraformModulePath ~/terraform-module --config ./config.yaml
    ```

1. The tool will run and output an `issues.csv` file into the folder where your terraform module resides. This file is required for the next phase of the tool.

### Issue Resolution

The tool will provide a list of issues in CSV format. You should open the CSV file in Excel, ensuring you don't let Excel alter the formatting of the file.

There are three types of issue that you'll see listed in the CSV file:

* `MultipleResourceIDs`: This means that the tool found multiple resource IDs that could be mapped to the same resource in the module. You will need to decide which resource ID to use.
* `NoResourceID`: This means that the tool could not find a resource ID that could be mapped to the resource in the module. You will need to decide how to resolve this issue.
* `UnusedResourceID`: This means that the tool found an Azure Resource ID that it could not map to any item in the module. You will need to decide how to resolve this issue.

For each issue, you will need to decide how to resolve it.

In the CSV file, you will find an Action column. Each issue must have an action assigned to it for the tool to be able to generate the final import blocks. The available actions depends on the type of issue. The following sections decsribe each issues type and the available actions.

Follow the steps below to resolve each issue, then:

#### MultipleResourceIDs

This issue means that the tool found multiple resource IDs that could be mapped to the same resource in the module. You will need to decide which resource ID to use.

1. Each matching resource ID will be listed as a separate row in the CSV file. This could be 2 or more rows.
1. Examine the resource IDs in the `Mapped Resource ID` column. Compare this to the Resource Address from the Terraform plan and find the one which corresponds to the resource in the module.
1. In the line that has the correct resource ID, set the `Action` to `Use`.
1. In the other lines, set the `Action` to `Ignore`.

#### NoResourceID

This issue means that the tool could not find a resource ID that could be mapped to the resource in the module. You will need to decide how to resolve this issue.

There are three possible actions you can take:

##### Update your Terraform module

This is the most common solution to this issue. You will need to update your module input variables to ensure that the resource ID matches your existing resource ID in Azure.

In this case, you don't need to make any changes to the CSV file, just leave the `Action` column emty. The tool will recheck it on the next run and remove it from the issues list.

##### Ignore

This means that you don't want to import this resource into the module. The tool will not generate an import block for this resource and the resource will be created by the new module.

1. Set the `Action` to `Ignore` for this issue.

##### Destroy and Recreate (Replace)

This means that you want to destroy the existing resource in Azure and recreate it with the module. To do this, you need to tell the tool which resource it will replace.

1. Take note of the `Issue ID` and set the `Action` to `Replace`, so you know you have processed this issue.
1. Find the issue in the CSV file that you want to replace it with and set the `Action` to `Replace` and set the `Action ID` to the `Issue ID` you recorded in the previous step.

#### UnusedResourceID

In most cases you will resolve this issue by following the steps for the `NoResourceID` issue, since there is usually a 1 to 1 mapping between the two issues.

This issue means that the tool found an Azure Resource ID that it could not map to any resource declaration in the module. You will need to decide how to resolve this issue.

##### Update your Terraform module

This is the most common solution to this issue. You will need to update you module input variables to ensure that the resource ID matches your existing resource ID in Azure.

In this case, you don't need to make any changes to the CSV file, just leave the `Action` column emty. The tool will recheck it on the next run and remove it from the issues list.

##### Ignore

There may be some resource IDs that you want to ignore and not import into the module.

1. Set the `Action` to `Ignore` for this issue.

### Generate Import Blocks

Once you think you have resolved all the issues in the CSV file, you need to run the tool again.

1. Save the CSV file in CSV format as a different file name.
1. Run the tool again, this time using the `--issuesCsv` option to specify the path to the CSV file you just saved.

    ```powershell
    ./terraform-state-importer ~/terraform-module --config ./config.yaml --issuesCsv <path to your issues.csv>
    ```

    The tool will read your CSV file and validate that all issues have resolution actions. If any are missing, you'll be prompted to resolve them before the tool can continue.

1. Head back to the [Issue Resolution](#issue-resolution) section and follow the steps to resolve any issues that are still present in the CSV file.

1. Once all the issues have been reolved or actioned, the tool will generate a set of import blocks for the resources that can be imported into the module.
1. You can now commit and push your changes to the module repository and run your continuous delivery pipeline to plan the changes.

### Attribute Mapping

When you run your Terraform plan, you will likely see that some resources will be updated in place. This is because the attributes in your module do not match the attributes of the existing resources in Azure.

You will need to examine the plan and decide which attributes you want to update.

1. Run a Terraform plan against your module and examine the output.
1. For each resource that is being updated, you will need to decide if you want to update the attribute in the module or leave it as is.
1. If you want to update the attribute in the module, you will need to update the module input variable to match the existing resource in Azure.
1. Once you have updated them all, run another Terraform plan and check the output.
1. If you are happy with the plan, you can now run a Terraform apply to import the resources into the module and deploy any new resources in your module.
