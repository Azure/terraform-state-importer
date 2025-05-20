# Terraform State Importer

This is a tool for running basic analysis and creating import blocks for migrating large Azure workloads to a new module in Terraform.

## Usage

The tools requires two main inputs:

- A set of subscription IDs where your deployed resources reside
- A Terraform module with variables supplied, which is the module you want to import the resources into

There are three main phases to the tool:

1. **Discovery** and **Analysis**: The tool will query Azure for all resources in the specified subscriptions. The tool run a Terraform plan against the module you provided, and will analyze the plan to determine which resources can be mapped to existing Azure resource IDs.
1. **Issue Resolution**: For any resources that cannnot be mapped, the tools will provide a list of issues. You can then rewiew the issues and decide how to resolve them.
1. **Import Generation**: The tool will generate a set of import blocks for the resources that can be imported into the module. You can then run these in your Terraform pipeline to import the resources into the module.

### Discovery and Analysis

The following steps are required to run the discovery and analysis phase of the tool:

1. Download the latest tool binary from the [releases page](TBC)
1. Create a file called `config.yaml` and copy the content from the `config.yaml` in the repository.
1. Identify the subscriptions IDs where your deployed resources reside. You can find these in the Azure portal by going to Subscriptions and copying the Subscription ID.

    Add the subscription IDs to your YAML `config.yaml` file under the `subscriptionIds` key.

    ```yaml
    subscriptionIds:
      - "00000000-0000-0000-0000-000000000001"
      - "00000000-0000-0000-0000-000000000002"
    ```

1. Create the folder with your destination module. This is the module you want to import the resources into. The tool will run a Terraform plan against this module and analyze the plan to determine which resources can be mapped to existing Azure resource IDs.

    In most cases, this will be a pre-existing repository that you just need to clone to your local machine.

1. Now you can run the tool. The tool will query Azure for all resources in the specified subscriptions and run a Terraform plan against the module you provided.

    ```bash
    ./terraform-state-importer --terraformModulePath <path to your terraform module> --config <path to your config.yaml>
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

1. Save the CSV file in CSV format. We recommend saving this in a separate folder from the original file, so you don't accidentally overwrite the original file.
1. Run the tool again, this time using the `--issuesCsv` option to specify the path to the CSV file you just saved.

    ```bash
    ./terraform-state-importer --terraformModulePath <path to your terraform module> --config <path to your config.yaml> --issuesCsv <path to your issues.csv>
    ```

    The tool will read your CSV file and validate that all issues have resolution actions. If any are missing, you'll be prompted to resolve them before the tool can continue.

1. The tool will then generate a set of import blocks for the resources that can be imported into the module.
1. The tool will clean up any files it created during the discovery and analysis phase, including the `issues.csv` file.
1. You can now commit and push your changes to the module repository and run your continuous delivery pipeline to plan and apply the changes.

#### MultipleResourceIDs

This issue means that the tool found multiple resource IDs that could be mapped to the same resource in the module. You will need to decide which resource ID to use.

1. Each matching resource ID will be listed as a separate row in the CSV file. This could be 2 or more rows.
1. Examine the resource IDs in the `Mapped Resource ID` column. Compare this to the Resource Address from the Terraform plan and find the one which corresponds to the resource in the module.
1. In the line that has the correct resource ID, set the `Action` to `Use`.
1. In the other lines, set the `Action` to `Ignore`.

#### NoResourceID

This issue means that the tool could not find a resource ID that could be mapped to the resource in the module. You will need to decide how to resolve this issue.

There are two possible actions you can take:

##### Ignore

This means that you don't want to import this resource into the module. The tool will not generate an import block for this resource and the resource will be created by the new module.

1. Set the `Action` to `Ignore` for this issue.

##### Destroy and Recreate (Replace)

This means that you want to destroy the existing resource in Azure and recreate it with the module. To do this, you need to tell the tool which resource it will replace.

1. Take note of the `Issue ID` and set the `Action` to `Replace`, so you know you have processed this issue.
1. Find the issue in the CSV file that you want to replace it with and set the `Action` to `Replace` and set the `Action ID` to the `Issue ID` you recorded in the previous step.

#### UnusedResourceID

This issue means that the tool found an Azure Resource ID that it could not map to any resource declaration in the module. You will need to decide how to resolve this issue.

In most cases you will resolve this issue by following the steps for the `NoResourceID` Destroy and Recreate (Replace). However, there may be some resource IDs that you want to ignore and not import into the module.

1. Set the `Action` to `Ignore` for this issue.

