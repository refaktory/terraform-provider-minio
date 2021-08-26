terraform {
  required_providers {
    minio = {
      version = "0.1"
      source  = "foundational/minio"
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

resource "minio_user" "user1" {
  access_key = "00000001"
  secret_key = "00000000"
  policies = [minio_canned_policy.policy1.name, "consoleAdmin"]
}
