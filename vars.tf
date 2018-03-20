variable "github_account" {}
variable "github_token" {}
variable "aws_ssh_key_name" {}
variable "local_ssh_key_path" {}
variable "aws_access_key" {}
variable "aws_secret_key" {}
variable "aws_region" { default = "us-west-1" }
variable "panos_ami" { default = "ami-5d59583d" }
variable "panos_username" { default = "admin" }
variable "linux_ami" { default = "ami-824c4ee2" }
variable "linux_instance_type" { default = "t2.small" }
