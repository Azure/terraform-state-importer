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

Blah, blah

### Issue Resolution

The tool will provide a list of issues in CSV format. You should open the CSV file in Excel, ensuring you don't let Excel alter the formatting of the file.

There are three types of issue that you'll see listed in the CSV file:

* `MultipleResourceIDs`: This means that the tool found multiple resource IDs that could be mapped to the same resource in the module. You will need to decide which resource ID to use.
* `NoResourceID`: This means that the tool could not find a resource ID that could be mapped to the resource in the module. You will need to decide how to resolve this issue.
* `UnusedResourceID`: This means that the tool found an Azure Resource ID that it could not map to any item in the module. You will need to decide how to resolve this issue.


