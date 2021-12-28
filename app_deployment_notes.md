# Setting Up Ingress, Deploying First Apps, using Private Registry

## Intro

This repo is for learning managed K8s on Digital Ocean as part of the [Digital Ocean K8s Challenege](lhttps://www.digitalocean.com/community/pages/kubernetes-challenge). I'm quite new to K8s, and to get up to speed I'll be working through [Digital Ocean's K8s Developer Starter](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers) and referencing other resources as needed.

The goal is to deploy a scalable NoSQL database (I'll be using Redis) within K8s, but I'd also like to try to deploy a trivial application to interact with Redis and get some basic observability into the performance of the Redis cluster and the K8s cluster as a whole. This will involve building out a cluster with the following resources:

- [x] A multi-node, highly available Redis cluster
- [x] A build pipeline (or at least some Terraform to spin up the K8s cluster)
- [ ] **Core API Service to communicate w. Redis**
- [ ] Logging services (e.g Prometheus, Grafana, and Loki)

This document will cover the third of these bullets. I have supplemental deployment notes which will cover the remaining bullets

- [Part 1 - Deploying K8s and H.A. Redis](./app_deployment_notes.md)
- [Part 3 - Deploying Logging Services](./logs_deployment_notes)

## Setting Up Ingress and Deploying an Application to K8s

For this section I'm following along pretty closely with the DigitalOcean Day 2 Operations Guide. I would strongly recommend referring to the following sections:

- [Configuring DOCR](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers/blob/main/02-setup-DOCR/README.md)
- [Configuring An Ingress Controller](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers/tree/main/03-setup-ingress-controller)

I chose to proceed through this section using Nginx because I have at least *some* experience working with Nginx configurations outside of K8s. Ambassador is a relatively new option as an ingress controller, and I wanted to keep the "new things" I was exposing myself to this week to a manageable level.

The goals for this section are as follows:

- Deploy Nginx to K8s w. functional SSL termination s.t. my cluster can run services on `<service>.maphub.dev`
- Deploy an application to K8s from my internal DOCR to `<service>.maphub.dev`

## Deploying an Ingress Controller

Before doing any application development work, I wanted to get a bit more familiar with K8s. Rather than opting for the Digital Ocean 1-click installation of an ingress controller, I'm going to try to install these basic services using Helm charts where possible. Following along with the DO Developer's guide, I added the Kubernetes maintained Nginx chart to my cluster using the following:

```bash
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx &&\
    helm repo update

NGINX_CHART_VERSION="4.0.6"

helm install ingress-nginx ingress-nginx/ingress-nginx \
    --version "$NGINX_CHART_VERSION" \
    --namespace ingress-nginx \
    --create-namespace \
    -f "./manifests/ingress/nginx-values-v${NGINX_CHART_VERSION}.yaml"
```

As with most files throughout this project, I made some slight modifications to what the guide provided. In this case, however, I ended up returning to this chart and upgrading it several times due to mistakes I made.

- My first mistake was uncommenting the `sysctl` commands in the reference yaml file, these (theoretically) can improve Nginx performance, but at the scale I'm operating at, they're not worth the additional complexity.  

- Later, I found that I was unable to access a backend service that I configured because of a stray annotation in [nginx-values-v4.0.6.yaml](./manifests/ingress/nginx-values-v4.0.6.yaml). It seems this [issue](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers/issues/91) speaks to the trouble I was having, but I still don't quite understand the explanation given [here](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers/blob/main/03-setup-ingress-controller/nginx.md#step-6---enabling-proxy-protocol) with respect to a L4 LB? Why does that matter. Doesn't DO create an L7 LB?

## Moving A Site to DOK8S and Deploying a Sample Application

I've held `maphub.dev` for about a year for a few toy projects, this is the only domain I own that I can fiddle with, so I [moved the nameservers](https://www.digitalocean.com/community/tutorials/how-to-point-to-digitalocean-nameservers-from-common-domain-registrars) over to Digital Ocean and proceeded to create records associating our recently created load balancer with `echo.maphub.dev` as suggested in the Developer Guide.

```bash
# Reference: doctl version 1.64.0-release
LOAD_BALANCER_IP=$(doctl compute load-balancer list --format IP --no-header)

doctl compute domain records create maphub.dev \
    --record-type "A" \
    --record-name "echo" \
    --record-data "$LOAD_BALANCER_IP" \
    --record-ttl "30"
```

I ran `curl -XGET http://echo.maphub.dev` and checked the logs of the `ingress-nginx` pod to confirm traffic was being routed into the cluster. There are two glaring problems, both of which I'll address in the coming sections.

- `echo.maphub.dev` points to a load balancer, but resolves to no actual service. We need to create an `echo` service.
- `http://echo.maphub.dev` is not doing any SSL termination, we'd like our controller to allow for `https` requests to our domain.

## Deploying an Application

Following along with the Developer's Guide, I deployed a token application to `echo.maphub.dev` into a newly created namespace, `backend`. The `*.yaml` files for this test application are in `./manifests/test-application/`, but are largely superseded by the subsequent sections. The important thing for this section was that the following returned `200`.

```bash
# curl request to our new service - using HTTP
curl -Li -XGET http://echo.maphub.dev -d '{"Hello": "World"}'

HTTP/1.1 200
date: Mon, 27 Dec 2021 00:51:33 GMT
content-type: text/plain
content-length: 427
vary: Accept-Encoding
```

The same day, I also deployed two more practical applications:

- A [deployment](https://github.com/DMW2151/expert-garbanzo/tree/feature/vanilla-redis/hslservices/cmd/mqtt) that ingests data from an MQTT stream and writes events to Redis. This deployment is not exposed to the outside world and is just a single pod listening for updates from a remote address (see: [mqtt-ingestion](./manifests/mqtt-ingestion/mqtt_broker_deployment.yaml)).

- A service that reads the keys that `mqtt-ingestion` set. This service is exposed to the outside world (for now) at `https://hsl.maphub.dev/locations/<route_number>`

To prevent this post from dragging on further, I'll present my notes for the above inline without much comment...

```bash
# LogIn to DOCR && Create a Secret s.t. any new deployment/pod can pull from DOCR
doctl registry login &&\
    doctl registry kubernetes-manifest | kubectl apply -f -

# Using `kubectl patch` add the annotation to all new deployment/pods
kubectl patch serviceaccount default \
    -p '{"imagePullSecrets": [{"name": "registry-dmw2151-internal"}]}'


# The next two deployments depend on a secret in the `backend` namespace; find the
# REDISCLIENT_AUTH secret in `redis` and "copy" it over to backend...

# NOTE: If a container already consumes a Secret in an environment variable, a Secret update will not be seen by the container unless it is restarted. There are third party solutions for triggering restarts when secrets change

kubectl create secret generic redisauth \
    --from-literal=redisclientauth=$REDIS_PASSWORD \
    --dry-run=client \
    --namespace backend -o yaml | kubectl apply -f -

# Build, Tag, and Push our Ingestion Service and API
docker build . -f ./cmd/mqtt/Dockerfile -t hsl-redis-bridge
    docker tag hsl-redis-bridge registry.digitalocean.com/dmw2151-internal/hsl-redis-bridge &&\
    docker push registry.digitalocean.com/dmw2151-internal/hsl-redis-bridge

docker build . -t hsl &&\
    docker tag hsl registry.digitalocean.com/dmw2151-internal/hsl &&\
    docker push registry.digitalocean.com/dmw2151-internal/hsl

kubectl rollout restart deployment hsl-api -n backend  

kubectl rollout restart deployment mqtt-redis-connector -n backend  

# Apply deployment for ingestion service
kubectl apply -f ./manifests/mqtt-ingestion/mqtt_broker_deployment.yaml

# Apply deployment + service for locations API
kubectl apply -f ./manifests/hsl/hsl_deployment.yaml \
    -f ./manifests/hsl/hsl_service.yaml
```

## Handling for SSL Termination

This section and the preceeding sections blurred together a bit. I found myself circling back to this section to make updates a few times. I'll present this as if I knew what I was doing from the start (i.e. starting with the expectation of needing a wildcard cert). I followed along with this [manual](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers/blob/main/03-setup-ingress-controller/guides/wildcard_certificates.md#installing-cert-manager), and will try to note the key points below.

```bash
# Applied a Helm chart for Cert-Manager
helm repo add jetstack https://charts.jetstack.io

CERT_MANAGER_HELM_CHART_VERSION="1.5.4"

helm upgrade cert-manager jetstack/cert-manager --version "$CERT_MANAGER_HELM_CHART_VERSION" \
  --namespace cert-manager \
  --create-namespace \
  -f "./manifests/ingress/cert-manager-values-v${CERT_MANAGER_HELM_CHART_VERSION}.yaml"
```

[Cert Manager](https://cert-manager.io/docs/concepts/) relies on a series of custom resources (CRD) to get a cert from a CA. One of the unique elements of this process is that Cert-Manager needs to be able to control the DNS records for a domain. As a result, I create a secret (my D.O token) and pass it into the cluster for CM to reference.

```bash
# Add DO Token as Secret for CM to manage DNS records
kubectl create secret generic "digitalocean-dns" \
  --namespace backend \
  --from-literal=access-token="$DO_API_TOKEN"

# Apply Issuer
kubectl apply -f ./manifests/ingress/cert-manager-nginx-issuer.yaml

# Apply Cert
kubectl apply -f ./manifests/ingress/cert-manager-nginx-certificate.yaml

# Apply Ingress Rules
kubectl apply -f ./manifests/ingress/cert-manager-nginx-certificate.yaml
```

I now had the certs in place that allowed me to make HTTPS requests to my endpoint....

```bash

curl https://hsl.maphub.dev/locations/107 | jq '.'

HTTP/2 200 
date: Mon, 27 Dec 2021 18:34:32 GMT
content-type: application/json
content-length: 44

{
  "Route": "107",
  "Status": "OK",
  "Vehicles": []
}
```


## Misc

- [Nginx Metrics](https://docs.nginx.com/nginx-ingress-controller/logging-and-monitoring/prometheus/)

cross namespace 