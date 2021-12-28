# Supplemental Deployment Notes - Logging

![metrics](/_imgs/redis_grafana.png)

## Intro

This repo is for learning managed K8s on Digital Ocean as part of the [Digital Ocean K8s Challenege](lhttps://www.digitalocean.com/community/pages/kubernetes-challenge). The goal is to deploy a scalable NoSQL database (I'll be using Redis) within K8s, but I'd also like to try to deploy a trivial application to interact with Redis and get some basic observability into the performance of the Redis cluster and the K8s cluster as a whole. This will involve building out a cluster with the following resources:

- [x] A multi-node, highly available Redis cluster
- [x] A build pipeline (or at least some Terraform to spin up the K8s cluster)
- [x] Core API Service to communicate w. Redis
- [ ] **Logging services (e.g Prometheus, Grafana, and Loki)**

The goal for this section is primarily to get metrics and logs from the `redis` namespace available in Prometheus, Grafana, and Loki. See Also:

- [Part 1 - Deploying K8s and H.A. Redis](./readme.md)
- [Part 2 - Deploying Core Application](./app_deployment_notes)

## Deploying Logging Services

For this section I'm using the DigitalOcean Day 2 Operations Guide. Rather than rehashing every step, I'd encourage you to refer to the following:

- [Section 4 - Setting Up a Prometheus Stack](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers/tree/main/04-setup-prometheus-stack)
- [Section 5 - Setting Up a Grafana Loki Stack](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers/tree/main/05-setup-loki-stack)
  
### Adding Prometheus and Grafana

Using [prom-stack-values-v17.1.3.yaml](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers/blob/main/04-setup-prometheus-stack/assets/manifests/prom-stack-values-v17.1.3.yaml) as a starting point, I modified the storage and metric configuration to make sense with my already existing Redis deployment.

Under the `additionalServiceMonitors` key, I added additional configurations to get values from Loki, Redis, and Promtail endpoints. We'll test that these endpoints are functional at the end of this section.

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts

PROM_HELM_CHART_VERSION="17.1.3"

helm upgrade kube-prom-stack prometheus-community/kube-prometheus-stack \
  --version "${PROM_HELM_CHART_VERSION}" \
  --namespace monitoring \
  --create-namespace \
  --set grafana.adminPassword="prom-operator" \
  -f "./manifests/monitor/prom-values-v${PROM_HELM_CHART_VERSION}.yaml"
```

With the storage configuration I'm using (see: `storageSpec`) a new Digital Ocean Volume is created as a persistent volume for Prometheus to store metric history. The advantages of this are discussed in the developer guide, although I should look into storing them using the 120GB already provisioned for the cluster instead of extra block storage.

### Adding Loki

Using [loki-stack-values-v2.4.1.yaml](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers/blob/main/05-setup-loki-stack/assets/manifests/loki-stack-values-v2.4.1.yaml) as a starting point.

```bash
helm repo add grafana https://grafana.github.io/helm-charts &&\
    helm repo update

LOKI_HELM_CHART_VERSION="2.4.1"

helm upgrade loki grafana/loki-stack \
  --version "${HELM_CHART_VERSION}" \
  --namespace monitoring \
  -f "./manifests/monitor/loki-values-v${LOKI_HELM_CHART_VERSION}.yaml" \
  --set loki.config.storage_config.aws.access_key_id=${DO_SPACES__K8S_ACCESS_KEY} \
  --set loki.config.storage_config.aws.secret_access_key=${DO_SPACES__K8S_SECRET_KEY}
```

As with Prometheus, I had to make some special adjustments to allow for logs (instead of metrics) to be persisted. In this case, Loki can be configured to allow storage on D.O Spaces.

In the below, I use [s3cmd](https://docs.digitalocean.com/products/spaces/resources/s3cmd/) to set a lifecycle policy on my DOK8s cluster's logs. As in part 1 with terraform, we can rely on the fact that D.O spaces is s3 compatible and use tools developed for AWS S3 to modify our resources.

```bash
s3cmd setlifecycle \
    ./manifests/monitor/loki_do_spaces_lifecycle.xml \
    s3://${DO_SPACES__K8S_BUCKET}
```

### Verifying Service Monitors

![prom](/_imgs/prom_svc_monitors.png)

The developer guide explained how to add `additionalServiceMonitors` for Prometheus, but it wasn't quite clear to me how annotations were used to identify the specfic endpoint to scrape. Using the below I port forwarded Prometheus to `localhost`
with the expectation of seeing redis, loki, and promtail service monitors.

```bash
kubectl port-forward svc/kube-prom-stack-kube-prome-prometheus \
  9090:9090 -n monitoring
```

This section proved to be quite challenging, although I could get loki and promtail showing, configuring the `servicemonitor` for redis took multiple attempts. What finally clicked for me was realizing that `service` was not a special key under `matchLabels`, but instead just *a* key to match. I swapped out the key and everything worked as expected.

```yaml
- name: "redis-monitor"
    selector:
      matchLabels:
        # Incorrectly assumed this was right for hours -> `service: redis-metrics`
        app.kubernetes.io/component: metrics
```

```bash
# Checked that `app.kubernetes.io/component` would pick up my redis-metrics service
kubectl get svc \
  --selector=app.kubernetes.io/component=metrics \
  --namespace redis

NAME            TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)    AGE
redis-metrics   ClusterIP   10.245.237.0   <none>        9121/TCP   2d8h
```

I upgraded the Prometheus chart and everything flowed through as expected. From there I added a Prometheus and Loki data-source for Grafana (shown at top) and filmed & submitted my D.O Challenge video.


### Miscellaneous/Notes


```bash
kubectl apply \
  -f https://github.com/kubernetes-sigs/metrics-server/releases/download/v0.4.2/components.yaml
```
