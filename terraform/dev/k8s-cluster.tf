# Provision a fresh K8s cluster on Digital Ocean with very little customization

# Resource: https://registry.terraform.io/providers/digitalocean/digitalocean/latest/docs/data-sources/kubernetes_versions 
data "digitalocean_kubernetes_versions" "k8-21-1-x" {
  version_prefix = "1.21."
}

# Tags for resources - be careful with applying these; don't want to overwrite any mandatory labels or
# resource tags like `k8.*`
# Resource: https://registry.terraform.io/providers/digitalocean/digitalocean/latest/docs/resources/tag
resource "digitalocean_tag" "development" {
  name = "development"
}

# Creates a k8s cluster with 3vCPUs, 6 GB total memory, 150 GB total storage...
# Resource: https://registry.terraform.io/providers/digitalocean/digitalocean/latest/docs/resources/kubernetes_cluster
resource "digitalocean_kubernetes_cluster" "core-dev" {

  # Basic
  name    = "core-dev"
  region  = "nyc1"
  version = data.digitalocean_kubernetes_versions.k8-21-1-x.latest_version

  # Networking
  vpc_uuid = digitalocean_vpc.core.id

  # Maintenance
  auto_upgrade = true
  ha           = false

  maintenance_policy {
    start_time = "04:00"
    day        = "sunday"
  }

  node_pool {
    name       = "k8s-default"
    size       = "s-2vcpu-2gb"
    auto_scale = true
    min_nodes  = 3
    max_nodes  = 5
  }

  tags = [
    digitalocean_tag.development.name
  ]
}