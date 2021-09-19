# terraform-provider-minio

A [Terraform](https://terraform.io) provider for [Minio](https://min.io), a 
self-hosted object storage server that is compatible with S3.


Check out the documenation on the [Terraform Registry - refaktory/minio](https://registry.terraform.io/providers/refaktory/minio/latest/docs) for more information and usage examples.

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
- [ ] Serviceaccount


## Usage

Consult the 
[published documenation](https://registry.terraform.io/providers/refaktory/minio/latest/docs) 
on the registry for usage documenation. 

Additional examples are available in the `./examples` directory.


## Local Development

### Configure Terraform to use the locally built provider

Add this configuration into `$HOME/.terraformrc`:

(**Notice** the trailing `/bin` in the path)

```
provider_installation {
  dev_overrides {
    "refaktory-dev/minio" = "/PATH/TO/LOCAL/REPO/bin"
  }

  direct {}
}
```

### Build

Build the provider with `make build`

### Test manually

* Start a local minio instance in a separate terminal 
  (and keep it running)
  `docker-compose up`

* Use the provider
  `cd ./examples && terraform apply`

### Deploy

Steps to deploy:

* `make prepare-release`
* `git tag v0.X.0`
* `git push`

Actual publishing is handled by the Github action defined in 
`./.github/workflows/release.yml`.

The module is managed on the Terraform registry at 
https://registry.terraform.io/publish/provider.

## About

Developed by [refaktory](https://refaktory.net).
