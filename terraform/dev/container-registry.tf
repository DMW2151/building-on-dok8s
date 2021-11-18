# Create a DOCR for $0.00/month to store application containers; assumes
# ONE and only ONE repo with max size of 500MB, so we'll have to make sure
# to keep the application slim!

# Resource: https://registry.terraform.io/providers/digitalocean/digitalocean/latest/docs/resources/container_registry
resource "digitalocean_container_registry" "internal_registry" {
  name                   = "dmw2151-internal"
  subscription_tier_slug = "starter"
}