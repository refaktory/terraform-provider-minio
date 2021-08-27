# terraform-provider-minio

A [Terraform](https://terraform.io) provider for [Minio](https://minio.io), a 
self-hosted object storage server that is compatible with S3.

## Features

### Resources

- [ ] Buckets
  - [x] Create/delete
  - [x] Versioning config
  - [ ] Encryption config
  - [ ] Replication config
  - [ ] Lifecycle config
  - [ ] Access rules
- [x] Users
  - [x] Create/delete
  - [x] Assign policies
  - [x] Assign groups
- [ ] Serviceaccounts
- [x] Canned policies
- [x] Groups
  - [x] Create/delete
  - [x] Assign policies
- [ ] Objects
  - [  ] Create files with a given content

### Datasources

- [x] Bucket
- [x] Canned policy
- [x] Group
- [x] User

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
