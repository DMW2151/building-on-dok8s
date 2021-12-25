
variable "deployer_or_whitelisted_cidr_range" {
  type        = string
  description = "An IP or CIDR range to whitelist for SSH into the core VPC jump instance"
}
