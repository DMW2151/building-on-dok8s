## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_digitalocean"></a> [digitalocean](#requirement\_digitalocean) | ~> 2.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_digitalocean"></a> [digitalocean](#provider\_digitalocean) | 2.16.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [digitalocean_container_registry.internal_registry](https://registry.terraform.io/providers/digitalocean/digitalocean/latest/docs/resources/container_registry) | resource |
| [digitalocean_kubernetes_cluster.core-dev](https://registry.terraform.io/providers/digitalocean/digitalocean/latest/docs/resources/kubernetes_cluster) | resource |
| [digitalocean_tag.development](https://registry.terraform.io/providers/digitalocean/digitalocean/latest/docs/resources/tag) | resource |
| [digitalocean_vpc.core](https://registry.terraform.io/providers/digitalocean/digitalocean/latest/docs/resources/vpc) | resource |
| [digitalocean_kubernetes_versions.k8-21-1-x](https://registry.terraform.io/providers/digitalocean/digitalocean/latest/docs/data-sources/kubernetes_versions) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_deployer_or_whitelisted_cidr_range"></a> [deployer\_or\_whitelisted\_cidr\_range](#input\_deployer\_or\_whitelisted\_cidr\_range) | An IP or CIDR range to whitelist for SSH into the core VPC jump instance | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_cluster-id"></a> [cluster-id](#output\_cluster-id) | The uniform resource name (URN) of the Kubernetes cluster. |
