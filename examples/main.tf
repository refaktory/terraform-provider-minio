terraform {
  required_providers {
    minio = {
      version = "0.1"
      source  = "foundational-solutions/minio"
    }
  }
}

provider "minio" {
  endpoint = "localhost:9000"
  access_key = "00000000"
  secret_key = "00000000"
  ssl = false
}

resource "minio_bucket" "bucket" {
  name = "bucket"
}


resource "minio_canned_policy" "policy1" {
  name = "policy1"
  policy = <<EOT
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "s3:GetObject"
      ],
      "Effect": "Allow",
      "Resource": [
        "arn:aws:s3:::my-bucketname/*"
      ]
    }
  ]
}
EOT
}

resource "minio_group" "group1" {
  name = "group1"
  policies = [minio_canned_policy.policy1.name]
}

resource "minio_group" "group2" {
  name = "group2"
}


resource "minio_user" "user1" {
  access_key = "00000001"
  secret_key = "00000001"
  policies = [
    minio_canned_policy.policy1.name,
    data.minio_canned_policy.console_admin.name,
  ]
  groups = [
    minio_group.group2.name,
  ]
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
