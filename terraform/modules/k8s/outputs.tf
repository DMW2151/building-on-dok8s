output "cluster-id" {
  description = "The uniform resource name (URN) of the Kubernetes cluster."
  value = digitalocean_kubernetes_cluster.core-dev.urn
}