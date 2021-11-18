terraform {

  # Full Digital Ocean! Store Backend in Spaces!! Run with following cmd
  # where `terraform.tfbackend` contains keys as shown in the readme...
  backend "s3" {}

  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }
}

# Configure the DigitalOcean Provider - Assumes `DIGITALOCEAN_TOKEN` set externally
provider "digitalocean" {}
