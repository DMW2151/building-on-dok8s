# Create core network resources for DMW2151 x DO

# Resource: https://registry.terraform.io/providers/digitalocean/digitalocean/latest/docs/resources/vpc
resource "digitalocean_vpc" "core" {

  # Basic
  name     = "dmw2151-core-dev"
  region   = "nyc1"
  ip_range = "192.168.0.0/24"
  
}