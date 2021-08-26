# terraform-provider-minio

A [Terraform](https://terraform.io) provider for [Minio](https://minio.io), a 
self-hosted object storage server that is compatible with S3.

## Features

- [ ] Buckets
  * [x] Create/delete
  * [x] Versioning config
  * [ ] Encryption config
  * [ ] Replication config
  * [ ] Lifecycle config
  * [ ] Access rules
- [ ] Users
  * [x] Create/delete
  * [x] Assign policies
  * [ ] Assign groups
- [x] Canned policies
- [ ] Groups
  
  Group management is not yet provided by the official API sdk - see https://github.com/minio/madmin-go/issues/25
- [ ] Objects
  Support for creating files with a given content.

## Local Development

### Configure Terraform to use the locally built provider

Add this configuration into `$HOME/.terraformrc`:

(**Notice** the trailing `/bin` in the path)

```
provider_installation {
  dev_overrides {
    "foundational/minio" = "/PATH/TO/LOCAL/REPO/bin"
  }

  direct {}
}
```

### Build

Build the provider with `make build`

### Test manually

* Start a local minio instance
  `docker-compose up`

* Use the provider
  `cd ./examples && terraform apply`
