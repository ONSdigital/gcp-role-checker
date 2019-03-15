# Google Cloud Platform IAM role checker

This tool allows you to gain an insight into what permissions a given user account, service account or group (collectively known as members in GCP) has across the resources within your organisation.

## Prerequisites

- Go 1.8+
- Python 3.6+
- Google Cloud SDK

## Installation

```
go get github.com/ONSdigital/gcp-role-checker
```

## Basic usage

```
gcloud auth login

cd ${GOPATH}/src/github.com/ONSdigital/gcp-role-checker

python scripts/audit_service_accounts.py organizations/<my organisation id>
```

### Functionality

This script will query a collection of Google's APIs to gather:

- All projects in organisation
- All folders in organisation
- All built-in IAM roles
- All custom IAM roles defined on the organisation and any projects and folders within it
- All permission scopes associated with the roles
- All IAM bindings (I.e. user/service account -> role) on the organisation and any projects and folders within it

It will output the raw results to JSON files.

The `audit_service_accounts.py` script will then use these files to print out a view of the data.

## Parameters

### --project_labels
*Optional*

Allows the projects queried to be restricted by labels.

```
python scripts/audit_service_accounts.py \
    --project_labels=env:dev,project:foo \
    organizations/999999999999
```

Labels are defined as key:value. Multiple labels can be added by comma-separating them; these will be ANDed together.

This may be used to only audit projects used in production.

### --data_dir
*Optional*

The directory used by the Go script to output the collected data. B

Default: a `data` folder in the package root.

### --limit
*Optional*

Limits the number of members to print in the output.

Default: `10`

### --member_type
*Optional*

Filters the members by their type.

Supported values:

- service_account
- user_account
- group


### --skip_collect
*Optional*

Specifies whether to skip the data gathering step. Use this to avoid unnecessarily hammering the Google APIs.

Is this is flagged then gathered data must already be available in the `--data_dir` directory.

```
python scripts/audit_service_accounts.py --skip_collect organizations/999999999999
```

### --sort_type
*Optional*

Alters the sort function used to order the members.

Supported values:

#### `total_sum`

Sorts by the sum of all permissions an account holds across all resources. Useful to find members that have broad permissions across the organisation

#### `top_sum`

Sorts by the sum permissions an account has on its most-privileged resource. This will tend to highlight members with roles/owner access so best to use in conjunction with the `--member_type` parameter

Default: `total_sum`
