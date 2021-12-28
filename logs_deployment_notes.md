# Supplemental Deployment Notes

## Deploying Logging Services

For this section I'm using the DigitalOcean Day 2 Operations Guide. Rather than rehashing every step, I'd encourage you to refer to the following:

- [Section 4 - Setting Up a Prometheus Stack](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers/tree/main/04-setup-prometheus-stack)
- [Section 5 - Setting Up a Grafana Loki Stack](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers/tree/main/05-setup-loki-stack)
  
The goal for this section is primarily to get metrics and logs from the `redis` namespace available in Prometheus, Grafana, and Loki.

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

With the storage configuration I'm using (see: `storageSpec`) a new Digital Ocean Volume is created as a persistent volume for Prometheus to store metric history. 

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

In the below, I use [s3cmd](https://docs.digitalocean.com/products/spaces/resources/s3cmd/) to set a lifecycle policy on my DOK8s cluster's logs.

```bash
s3cmd setlifecycle \
    ./manifests/monitor/loki_do_spaces_lifecycle.xml \
    s3://${DO_SPACES__K8S_BUCKET}
```

### Verifying Installation

kubectl port-forward svc/kube-prom-stack-kube-prome-prometheus 9090:9090 -n monitoring

expect to see redis, loki, and promtail. This section proved to be quite challenging, although I could get loki and promtail showing, configuring the servicemonitor for redis took multiple attempts


```
- name: "redis-monitor"
      selector:
        matchLabels:
          app.kubernetes.io/component: metrics
```

```bash
kubectl get svc --selector=app.kubernetes.io/component=metrics -n redis
```

or the `podAnnotations` in the redis config...

### Misc.

kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/download/v0.4.2/components.yaml