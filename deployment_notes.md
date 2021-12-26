# Supplemental Deployment Notes

## Deploying Logging Services

For this section I'm using the DigitalOcean Day 2 Operations Guide. This section pretty closely aligns with the following section. Rather than rehashing every step, I'd encourage you to refer to the following:

- [Section 4 - Setting Up a Prometheus Stack](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers/tree/main/04-setup-prometheus-stack)
- [Section 5 - Setting Up a Grafana Loki Stack](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers/tree/main/05-setup-loki-stack)
  
The goal for this section is to **get metrics and logs from the `redis` namespace available in Prometheus, Grafana, and Loki**.


Using [prom-stack-values-v17.1.3.yaml](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers/blob/main/04-setup-prometheus-stack/assets/manifests/prom-stack-values-v17.1.3.yaml) as a starting point, I modified the storage and metric configuration for my Redis deployment.
 
```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts

HELM_CHART_VERSION="17.1.3"

helm install kube-prom-stack prometheus-community/kube-prometheus-stack \
  --version "${HELM_CHART_VERSION}" \
  --namespace monitoring \
  --create-namespace \
  -f "./manifests/monitor/prom-values-v${HELM_CHART_VERSION}.yaml"
```

Using [loki-stack-values-v2.4.1.yaml](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers/blob/main/05-setup-loki-stack/assets/manifests/loki-stack-values-v2.4.1.yaml) as a starting point, I 

```bash
helm repo add grafana https://grafana.github.io/helm-charts &&\
    helm repo update

LOKI_HELM_CHART_VERSION="2.4.1"

helm install loki grafana/loki-stack \
  --version "${HELM_CHART_VERSION}" \
  --namespace=monitoring \
  -f "./manifests/monitor/loki-values-v${LOKI_HELM_CHART_VERSION}.yaml"

helm upgrade loki grafana/loki-stack \
  --namespace=monitoring \
  -f "./manifests/monitor/loki-values-v${LOKI_HELM_CHART_VERSION}.yaml" \
  --set loki.config.storage_config.aws.access_key_id=${DO_SPACES__K8S_ACCESS_KEY} \
  --set loki.config.storage_config.aws.secret_access_key=${DO_SPACES__K8S_SECRET_KEY}
```

```bash
s3cmd setlifecycle \
    ./manifests/monitor/loki_do_spaces_lifecycle.xml \
    s3://${DO_SPACES__K8S_BUCKET}
```







## Deploying NGINX Ingress Controller

For this section I'm following along pretty closely with the DigitalOcean K8s developer's guide. The application shown ad  the end-product of this repo required a few iterations, but for brevity's sake I've condensed this section to just the final iteration. I would strongly recommend referring to the configuring Nginx [section](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers/blob/main/03-setup-ingress-controller/nginx.md) of the K8s developer starter guide to get a sense of how I started.

But to ensure that my DNS configuration was correct, I used a demo application from the ops guide, the manifests for which are found in `./manifests/test-application`

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

## Deploying an Application

One of the other resources Terraform is responsible for provisioning is a container registry, see [DOCR setup](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers/tree/main/02-setup-DOCR). Because DOCR is unique within an account, I pulled my account's registry manifest and created a secret that my K8s deployments can use to pull from that registry.

```bash
doctl registry login &&\
    doctl registry kubernetes-manifest | kubectl apply -f -
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

## Reading

### General K8s

- [Video on Service Types](https://www.youtube.com/watch?v=T4Z7visMM4E)
- [Digital Ocean's K8 Developer Starter](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers)

### Redis Specific

- [Redis](https://marklu-sf.medium.com/deploy-and-operate-a-redis-cluster-in-kubernetes-94fde7853001)
- [Redis H.A Intro](https://www.youtube.com/watch?v=GEg7s3i6Jak)
- [Redis on K8s](https://www.youtube.com/watch?v=JmCn7k0PlV4)
- [Helm w. Bitnami](https://docs.bitnami.com/tutorials/deploy-redis-sentinel-production-cluster/)
