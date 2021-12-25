# About

This repo is for learning managed K8s on Digital Ocean as part of the [Digital Ocean K8s Challenege](lhttps://www.digitalocean.com/community/pages/kubernetes-challenge). I'm quite new to K8s, and to get up to speed I'll be working through [Digital Ocean's K8s Developer Starter](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers) and referencing other resources as needed (see [Reading List](#Reading). With that in mind, this document will not be so much of a traditional blog post/ Readme, but my build notes as I learn both Digital Ocean and K8s.

The goal is to deploy a scalable NoSQL database (I'll be using Redis) within K8s, but I'd also like to try to deploy a trivial application to interact with Redis and some basic observability into the performance of the Redis cluster and the K8s cluster as a whole. This will involve building out a cluster with the following resources:

- [ ] A multi-node Redis cluster service
- [x] Logging services (e.g Prometheus, Grafana, and Loki)
- [ ] Core API Service to communicate w. Redis
- [ ] A build pipeline (or at least some Terraform to spin up the K8s cluster)

## Provisioning Digital Ocean K8s w. Terraform

I'm a big fan of Terraform and will use it to manage the core infrastructure of the cluster. I'm using a DO spaces backend to store my state file. As much as I love spinning up *every* resource with Terraform, the DO Space to initialize Terraform's state (obviously) must be an exception to this rule. Initializing Terraform with DO Spaces is not dissimilar from initializing with an S3 backend as [Spaces is S3 compatible](https://www.digitalocean.com/products/spaces/).

I initialize the Terraform backend and provision a K8s cluster in a newly-created VPC on DO with the following commands and variables.

```bash
terraform init \
  -input=false \
  -backend-config=backend.tfvars

terraform plan \
  -var-file k8s_cluster.tfvars
```

```bash
# backend.tfvars
bucket                      = DIGITALOCEAN__TF_SPACES
endpoint                    = "https://${DIGITALOCEAN__REGION}.digitaloceanspaces.com"
key                         = "terraform.tfstate"
region                      = "us-east-1" # Dummy AWS region to keep the s3 backend happy...
access_key                  = DIGITALOCEAN__TF_SPACES_KEY
secret_key                  = DIGITALOCEAN__TF_SPACES_SECRET
skip_credentials_validation = true
skip_metadata_api_check     = true
```

```bash
# k8s_cluster.tfvars
```

The plan runs in about 6-8 min. Once it's successfully run, I configure my local machine to use the context for the newly provisioned cluster.

```bash
export DIGITALOCEAN__CLUSTER_ID=`(terraform output cluster-id | cut -d':' -f3 | sed 's/\"//g')`
doctl kubernetes cluster kubeconfig save $DIGITALOCEAN__CLUSTER_ID
```

The full list of modules, variables, outputs, and resources provisioned by Terraform are available within the Terraform [Readme](./terraform/dev/readme.md). For those who are interested, these are auto-generated via the [terraform-docs](https://terraform-docs.io/user-guide/introduction/) utility.

## Deploying High Availability Redis Into K8s

[Redis](https://redis.io/) is an open source in-memory data structure store used as a database, cache, and message broker. Redis provides high availability via Redis [Sentinel](https://redis.io/topics/sentinel) and automatic partitioning with Redis [Cluster](https://redis.io/topics/cluster-tutorial).





## Deploying NGINX Ingress Controller

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

## Deploying Logging Services

For this section as well I'm using the DigitalOcean Day 1 Ops guide.  Goal is to get logs for the dummy application shipped out to Grafana Loki by way of Promtail (or fluentbit...)

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts

HELM_CHART_VERSION="17.1.3"

helm install kube-prom-stack prometheus-community/kube-prometheus-stack --version "${HELM_CHART_VERSION}" \
  --namespace monitoring \
  --create-namespace \
  -f "./manifests/monitor/prom-values-v${HELM_CHART_VERSION}.yaml"

```

```bash
helm repo add grafana https://grafana.github.io/helm-charts &&\
    helm repo update

HELM_CHART_VERSION="2.4.1"

helm install loki grafana/loki-stack --version "${HELM_CHART_VERSION}" \
  --namespace=monitoring \
  --create-namespace \
  -f "./manifests/monitor/loki-values-v${HELM_CHART_VERSION}.yaml"
```

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


We're going to improve logging today by persisting the logs with DO spaces....

```bash
HELM_CHART_VERSION="2.4.1"

helm install loki grafana/loki-stack \
  --version "${HELM_CHART_VERSION}" \
  --namespace=monitoring \
  -f "./manifests/monitor/loki-values-v2.4.1.yaml"

helm upgrade loki grafana/loki-stack \
  --namespace=monitoring \
  -f "./manifests/monitor/loki-values-v2.4.1.yaml" \
  --set loki.config.storage_config.aws.access_key_id=${DO_SPACES__K8S_ACCESS_KEY} \
  --set loki.config.storage_config.aws.secret_access_key=${DO_SPACES__K8S_SECRET_KEY}
```

```bash
s3cmd getlifecycle s3://${DO_SPACES__K8S_BUCKET}

s3cmd setlifecycle \
    ./manifests/monitor/loki_do_spaces_lifecycle.xml \
    s3://${DO_SPACES__K8S_BUCKET}
```


## Deploying an Application

One of the other resources Terraform is responsible for provisioning is a container registry, see [DOCR setup](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers/tree/main/02-setup-DOCR). Because DOCR is unique within an account, I pulled my account's registry manifest and created a secret that my K8s deployments can use to pull from that registry.

```bash
doctl registry login &&\
    doctl registry kubernetes-manifest | kubectl apply -f -
```

## Reading

### General K8s

- [Video on Service Types](https://www.youtube.com/watch?v=T4Z7visMM4E)
- [Digital Ocean's K8 Developer Starter](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers)

### Redis Specific

- [Redis](https://marklu-sf.medium.com/deploy-and-operate-a-redis-cluster-in-kubernetes-94fde7853001)
- [Redis H.A Intro](https://www.youtube.com/watch?v=GEg7s3i6Jak)
- [Redis on K8s](https://www.youtube.com/watch?v=JmCn7k0PlV4)
