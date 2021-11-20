
# Resource: https://registry.terraform.io/providers/digitalocean/digitalocean/latest/docs/resources/ssh_key
resource "digitalocean_ssh_key" "default" {
  name       = "Digital Ocean Core VPC Jump"
  public_key = file("~/.ssh/do-public-jump-1.pub")
}

# Resource: https://registry.terraform.io/providers/digitalocean/digitalocean/latest/docs/resources/droplet
resource "digitalocean_droplet" "jump" {
  name  = "vpc-jump-1"
  image = "ubuntu-20-04-x64"
  size  = "s-1vcpu-1gb"

  # Init
  user_data = file("${path.module}/user_data/do-k8s-jump-utils.sh")

  # Networking + Security
  region   = digitalocean_vpc.core.region
  vpc_uuid = digitalocean_vpc.core.id
  ssh_keys = [
    digitalocean_ssh_key.default.fingerprint
  ]


}