# About

This repo is for learning managed K8s on DO. Working through [Digital Oceans' Developer Starter](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers) for the DO Kubernetes Challenege and referencing other resources as needed.

The goal is to deploy a [Gazetteer](https://en.wikipedia.org/wiki/Gazetteer) API and get really good observability into its performance. This will involve building out a cluster with the following resources:

- [ ] An Nginx Ingress controller
- [ ] An ElasticSearch stateful set for the application data
- [ ] A core API service
- [ ] An ElasticSearch stateful set, Kibana deployment, and Fluentd daemon set for logging

## Provisioning K8s [11/17/2021 - 11/XX/2021]

Terraform only creates a few key resources for this deployment, the cluster's state is primarily managed via Helm. I'm using a DO spaces backend to store my state files and use the following command to init the backend and apply the plan. The `init` assumes the existence of a file similar to `terraform.tfbackend`, which contains references to pre-existing spaces, access keys, etc.

```bash
terraform init -input=false -reconfigure -backend-config=terraform.tfbackend &&\
    terraform apply
```

```bash
# terraform.tfbackend
bucket = "${DIGITALOCEAN__TF_SPACES}"
endpoint = "https://${DIGITALOCEAN__REGION}.digitaloceanspaces.com"
key    = "terraform.tfstate"
region = "us-east-1" # dummy aws region to keep the s3 backend happy...
access_key = "${DIGITALOCEAN__TF_SPACES_KEY}"
secret_key = "${DIGITALOCEAN__TF_SPACES_SECRET}"
skip_credentials_validation = true
skip_metadata_api_check = true
```

The plan runs in ~5-10 min and the new registry and cluster are up and running. I configure my local machine with the cluster certificate using the following command, this swaps my current k8s config to my cluster, `do-nyc1-core-dev`, and lasts a week before needing to be refreshed.

```bash
export DIGITALOCEAN__CLUSTER_ID=`(terraform output cluster-id | cut -d':' -f3 | sed 's/\"//g')`
doctl kubernetes cluster kubeconfig save $DIGITALOCEAN__CLUSTER_ID
```

One of the other resources terraform is responsible for provisioning is a container registry, see [DOCR setup](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers/tree/main/02-setup-DOCR). Because DOCR is unique within an account, I pulled my account's registry manifest and created a secret that my K8s deployments can use to pull from that registry.

```bash
doctl registry login &&\
    doctl registry kubernetes-manifest | kubectl apply -f -
```

## Deploying Core API [11/XX/2021 - XX/XX/2021]

## Deploying Logging Services [11/XX/2021 - XX/XX/2021]

Leaning heavily on this [article](https://www.digitalocean.com/community/tutorials/how-to-set-up-an-elasticsearch-fluentd-and-kibana-efk-logging-stack-on-kubernetes) for configuring ElasticSearch, Fluentd, and Kibana.

### Reading List

- [Video on Service Types](https://www.youtube.com/watch?v=T4Z7visMM4E)

## Deploying App DB [11/XX/2021 - XX/XX/2021]
