---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "minio_bucket Data Source - terraform-provider-minio"
subcategory: ""
description: |-
  
---

# minio_bucket (Data Source)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- **name** (String) The name of the bucket. Can not be changed without recreating.

### Optional

- **id** (String) The ID of this resource.
- **versioning_enabled** (Boolean) Enables versioning. Note: this is only available if the Minio server is run with erasure codes enabled. See https://docs.min.io/docs/minio-erasure-code-quickstart-guide

