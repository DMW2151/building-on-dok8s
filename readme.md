# About

This repo is for learning managed K8s on DO. Working through [Digital Ocean's K8s Developer Starter](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers) for the DO Kubernetes Challenege and referencing other resources as needed.

The goal is to deploy a [Gazetteer](https://en.wikipedia.org/wiki/Gazetteer) API and get really good observability into its performance. This will involve building out a cluster with the following resources:

- [x] An Nginx Ingress controller
- [ ] An ${DB} service + stateful set for the application data
- [ ] A core API service that talks to ${DB}
- [ ] An ElasticSearch service + stateful set, Kibana deployment, and Fluentd daemon set for logging

## Provisioning Digital Ocean K8s [11/17/2021 - 11/20/2021]

Terraform only creates a few resources for this deployment, the cluster's state is primarily managed via Helm. I'm using a DO spaces backend to store my state files and use the following command to init the backend and apply the plan. The `init` assumes the existence of a file similar to `terraform.tfbackend`, which contains references to pre-existing spaces, access keys, etc.

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

One of the other resources Terraform is responsible for provisioning is a container registry, see [DOCR setup](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers/tree/main/02-setup-DOCR). Because DOCR is unique within an account, I pulled my account's registry manifest and created a secret that my K8s deployments can use to pull from that registry.

```bash
doctl registry login &&\
    doctl registry kubernetes-manifest | kubectl apply -f -
```

## Deploying NGINX Ingress Controller [11/18/2021 - 11/19/2021]

For this section I'm following along pretty closely with the DigitalOcean Day 1 Ops guide. As the core application is developed, the Nginx configuration will evolve. But to ensure that my DNS configuration was correct, I used a demo application from the ops guide, the manifests for which are found in `./manifests/test-application`

```bash
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx &&\
    helm repo update

NGINX_CHART_VERSION="4.0.6"
CERT_MANAGER_HELM_CHART_VERSION="1.5.4"

helm install ingress-nginx ingress-nginx/ingress-nginx \
    --version "$NGINX_CHART_VERSION" \
    --namespace ingress-nginx \
    --create-namespace \
    -f "./manifests/ingress/nginx-values-v${NGINX_CHART_VERSION}.yaml"

helm install cert-manager jetstack/cert-manager --version "$CERT_MANAGER_HELM_CHART_VERSION" \
  --namespace cert-manager \
  --create-namespace \
  -f "./manifests/ingress/cert-manager-values-v${CERT_MANAGER_HELM_CHART_VERSION}.yaml"
```

## Deploying Core API [11/19/2021 - 11/20/2021]

Starting with a demo app....

```bash
# DNS
LOAD_BALANCER_IP=$(doctl compute load-balancer list --format IP --no-header)

doctl compute domain records create maphub.dev \
    --record-type "A" \
    --record-name "quote" \
    --record-data "$LOAD_BALANCER_IP" \
    --record-ttl "30"
```

```bash
kubectl create ns backend

kubectl apply -f ./manifests/test-application/quote_deployment.yaml &&\
    ./manifests/test-application/quote_service.yaml &&\
    ./manifests/test-application/quote_host_nginx.yaml
```

## Deploying Logging Services [11/XX/2021 - XX/XX/2021]

## Deploying App DB [11/XX/2021 - XX/XX/2021]

### Reading / Watching List

- [Video on Service Types](https://www.youtube.com/watch?v=T4Z7visMM4E)
- [Digital Ocean's K8 Developer Starter](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers)
