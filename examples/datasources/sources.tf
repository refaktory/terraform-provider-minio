terraform {
  required_providers {
    minio = {
      version = "0.1"
      source  = "foundational-solutions/minio"
    }
  }
}

provider "minio" {
  # The Minio server endpoint.
  # NOTE: do NOT add an http:// or https:// prefix!
  # Set the `ssl = true/false` setting instead.
  endpoint = "localhost:9000"
  # Specify your minio user access key here.
  access_key = "00000000"
  # Specify your minio user secret key here.
  secret_key = "00000000"
  # If true, the server will be contacted via https://
  ssl = false
}

data "minio_bucket" "data_bucket1" {
  name = "bucket"
}

data "minio_user" "data_user1" {
  access_key = "00000001"
}

data "minio_canned_policy" "console_admin" {
  name = "consoleAdmin"
}

data "minio_group" "mygroup" {
  name = "group1"
}
