terraform {
  required_providers {
    minio = {
      # ATTENTION: use the current version here!
      version = "0.1.0-alpha5"
      source  = "refaktory/minio"
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

# Create a bucket.
resource "minio_bucket" "bucket" {
  name = "bucket"
}

# Create a policy.
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

# Create a user group and assign the specified policies.
resource "minio_group" "group1" {
  name = "group1"
  policies = [minio_canned_policy.policy1.name]
}

resource "minio_group" "group2" {
  name = "group2"
}


# Import an existing policy.
# (the consoleAdmin policy is created by Minio automatically)
data "minio_canned_policy" "console_admin" {
  name = "consoleAdmin"
}

# Create a user with specified access credentials, policies and group membership.
resource "minio_user" "user1" {
  access_key = "00000001"
  secret_key = "00000001"
  policies = [
    minio_canned_policy.policy1.name,
    # Note: using a data source here!
    data.minio_canned_policy.console_admin.name,
  ]
  groups = [
    minio_group.group2.name,
  ]
}
