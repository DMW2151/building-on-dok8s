# Create core network resources for DMW2151 x DO

# Resource: https://registry.terraform.io/providers/digitalocean/digitalocean/latest/docs/resources/vpc
resource "digitalocean_vpc" "core" {
  name     = "dmw2151-core-dev"
  region   = "nyc1"
  ip_range = "192.168.0.0/24"
}

# Need to think of a better way to do this pattern - we should be able to SSH into the instance from 
# a CIDR, but the deployer might be GitHub...and that make it a mess...
# Resource: https://registry.terraform.io/providers/digitalocean/digitalocean/latest/docs/resources/firewall
resource "digitalocean_firewall" "ssh-from-deployer" {
  name = "only-ssh-deployer"

  droplet_ids = [
    digitalocean_droplet.jump.id
  ]

  inbound_rule {
    protocol   = "tcp"
    port_range = "22"
    source_addresses = [
      var.deployer_or_whitelisted_cidr_range
    ]
  }

  outbound_rule {
    protocol              = "icmp"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "tcp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }

}