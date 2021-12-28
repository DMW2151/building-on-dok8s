# Resource: https://registry.terraform.io/providers/digitalocean/digitalocean/latest/docs/resources/container_registry
resource "digitalocean_container_registry" "internal_registry" {

  # Basics
  name                   = "dmw2151-internal"
  subscription_tier_slug = "basic"

} 