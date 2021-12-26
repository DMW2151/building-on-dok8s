# About

This repo is for learning managed K8s on Digital Ocean as part of the [Digital Ocean K8s Challenege](lhttps://www.digitalocean.com/community/pages/kubernetes-challenge). I'm quite new to K8s, and to get up to speed I'll be working through [Digital Ocean's K8s Developer Starter](https://github.com/digitalocean/Kubernetes-Starter-Kit-Developers) and referencing other resources as needed.

With that in mind, this document will not be so much of a traditional Readme, but my moreso build notes as I learn both Digital Ocean and K8s.

The goal is to deploy a scalable NoSQL database (I'll be using Redis) within K8s, but I'd also like to try to deploy a trivial application to interact with Redis and get some basic observability into the performance of the Redis cluster and the K8s cluster as a whole. This will involve building out a cluster with the following resources:

- [x] A multi-node Redis cluster service
- [x] Logging services (e.g Prometheus, Grafana, and Loki)
- [x] Core API Service to communicate w. Redis
- [x] A build pipeline (or at least some Terraform to spin up the K8s cluster)

This document will cover the first of those bullets. Time permitting, I will have supplemental [deployment notes](./deployment_notes.md) which will cover the remaining bullets.

## Provisioning Digital Ocean K8s with Terraform

I'm a big fan of Terraform and will use it to manage the core infrastructure of the cluster. I'm using a DO spaces backend to store my state file. As much as I love spinning up *every* resource with Terraform, the DO Space to initialize Terraform's state (obviously) must be an exception to this rule. Initializing Terraform with DO Spaces was not too difficult as [Spaces is S3 compatible](https://www.digitalocean.com/products/spaces/).

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
...
```

The full list of modules, variables, outputs, and resources provisioned by Terraform are available within the Terraform [Readme](./terraform/dev/readme.md). For those who are interested, these are auto-generated via the [terraform-docs](https://terraform-docs.io/user-guide/introduction/) utility.

The plan runs in about 6-8 min. Once it's successfully run, I configure my local machine to use the context for the newly provisioned cluster. In a more serious deployment, we'd likely want to lock this down a bit more, but for my purposes today, this is fine.

```bash
export DIGITALOCEAN__CLUSTER_ID=`(terraform output cluster-id | cut -d':' -f3 | sed 's/\"//g')`
doctl kubernetes cluster kubeconfig save $DIGITALOCEAN__CLUSTER_ID
```

## Deploying Redis Into Our New K8s Cluster

[Redis](https://redis.io/) is an open source in-memory data structure store used as a database, cache, and message broker. Redis provides high availability via [Redis Sentinel](https://redis.io/topics/sentinel) and automatic partitioning with [Redis Cluster](https://redis.io/topics/cluster-tutorial).

For this deployment, I want to make sure that the service is highly available and that data is durable between instances. I'll make a few choices when deploying the `bitnami` Redis Cluster [Helm chart](https://github.com/bitnami/charts/tree/master/bitnami/redis) which I'll highlight as I go.

### Configuring Redis

#### High Availability

Using the Helm Chart values from [Bitnami](https://raw.githubusercontent.com/bitnami/charts/master/bitnami/redis/values.yaml) as a starting point, I looked to improve cluster availability, persistence, logging, and monitoring.

To allow for high-availability, I enabled the most basic Sentinel [configuration](https://redis.io/topicssentinel#example-2-basic-setup-with-three-boxes) possible. Under this configuration, each pod runs a Sentinel (S) container and a Redis (R) container, with one of the pods acting as a master node (M) which replicates all writes to the other nodes.

```bash
# Basic Sentinel Deployment - configuration: quorum = 2

       +----+
       | M1 | <----- Write
       | S1 | <----- Read
       +----+
          |
          | Replication
          |
+----+    ↓    +----+
| R2 |<---+--->| R3 |  <--X-- Write
| S2 |         | S3 |  <----- Read
+----+         +----+
```

```yaml
# values-production.yaml

sentinel:
  ## @param sentinel.enabled Use Redis&trade; Sentinel on Redis&trade; pods.
  enabled: True
```

#### Data Persistance

While Redis is traditionally used as an in-memory cache or messaging bus, Redis offers two persistence models. More about these models is available [here](https://redis.io/topics/persistence).

1. Append Only File (AOF) - Under this model, each incoming transaction (sorta) is written to the cluster and an append-only file on disk. In the event of a failure, no data is lost, but AOF can slow the cluster during times of high load.

2. Redis DB (RDB) - Under this model, the entire DB is dumped to storage every *K* writes or *N* seconds. In the event of a failure, data since the last RDB dump will be lost.

For the sake of argument, let's assume our application data cannot tolerate a few minutes of data loss (e.g. RDB only). I enable both mode with the following options.

```yaml
# values-production.yaml

# Enable AOF and RDB persistence See:
#   - https://redis.io/topics/persistence#append-only-file
#   - https://raw.githubusercontent.com/redis/redis/6.2/redis.conf
commonConfiguration: |-
  appendonly yes
  save 3600 1
  save 300 100
  save 60 10000
```

#### Metrics and Monitoring

In [deployment notes](./deployment_notes.md), I'll discuss scraping metrics from a Redis Prometheus exporter and configuring them to send to Grafana. For the time being, I'll ignore any nuance here and just enable `metrics`.

```yaml
# values-production.yaml

## @param metrics.enabled Start a sidecar prometheus exporter to expose Redis&trade; metrics
metrics:
  enabled: True
```

### Deploying Redis

Let's deploy the Helm chart with the following `helm repo add` and `helm install` commands.

```bash
BITNAMI_REDIS_CHART_VERSION="15.6.8"

helm repo add bitnami https://charts.bitnami.com/bitnami

# Note that `helm upgrade` requires slightly different parameters
helm install redis bitnami/redis \
  --create-namespace \
  --namespace redis \
  --version "$BITNAMI_REDIS_CHART_VERSION" \
  --values ./manifests/redis/values-production.yaml
```

A few minutes after a successful deployment, I can verify that all expected services, statefulsets, and containers are up and running.

```bash
# Check all resources in namespace
kubectl get all -n redis

NAME               READY   STATUS    RESTARTS   AGE
pod/redis-node-0   3/3     Running   0          50m
pod/redis-node-1   3/3     Running   0          51m
pod/redis-node-2   3/3     Running   0          52m

NAME                     TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)              AGE
service/redis            ClusterIP   10.245.198.243   <none>        6379/TCP,26379/TCP   4h48m
service/redis-headless   ClusterIP   None             <none>        6379/TCP,26379/TCP   4h48m
service/redis-metrics    ClusterIP   10.245.237.0     <none>        9121/TCP             3h43m

NAME                          READY   AGE
statefulset.apps/redis-node   3/3     4h48m

# Confirm each pod/redis-node-* has 3 containers (metrics exporter, sentinel, redis)
kubectl get pods -n redis redis-node-0 -o yaml | grep -E 'image:' | sort | uniq -

image: docker.io/bitnami/redis-exporter:1.31.4-debian-10-r11
image: docker.io/bitnami/redis-sentinel:6.2.6-debian-10-r54
image: docker.io/bitnami/redis:6.2.6-debian-10-r53
```

### Testing Redis Data Replication

The next step I'd like to take to confirm that my deployment went properly is to check the following properties regarding data persistence and replication. I expect each of the following to be true:

1. The master node allows writes and replicates that information to slave nodes
2. The slave nodes disallow writes
3. Redis' AOF and RDB files are present at `/data`

I strongly suspect that `redis-node-0` will start as our master node. I'll `kubectl exec` into this pod and use the redis-cli command, `INFO` to confirm this.

```bash
export REDIS_PASSWORD=$(kubectl get secret --namespace redis redis -o jsonpath="{.data.redis-password}" | base64 --decode)

# Check `INFO replication` on redis-node-0; confirm redis-node-1 and redis-node-2 listed as connected
kubectl exec \
  --stdin \
  --tty \
  --container redis \
  --namespace redis \
  redis-node-0 -- /bin/sh -c 'export REDISCLI_AUTH=$REDIS_PASSWORD; redis-cli -c INFO replication'

role:master
connected_slaves:2
slave0:ip=redis-node-1.redis-headless.redis.svc.cluster.local,port=6379,state=online,offset=723454,lag=1
slave1:ip=redis-node-2.redis-headless.redis.svc.cluster.local,port=6379,state=online,offset=723672,lag=1
```

Beautiful, I can now use the same pattern as above, changing only the `redis-cli` command, to confirm the master can execute writes.

```bash
# On `redis-node-0`
SET foo bar EX 3600
OK
```

By changing `redis-node-0` to `redis-node-1`, I can confirm that no other node can receive writes.

```bash
# On `redis-node-1` (or any node except the current master, `redis-node-0`)
SET foo bar EX 3600
"(error) READONLY You can't write against a read only replica."
```

I can also confirm these nodes read values set in `redis-node-0` . In this case, I check that the key `foo` I is both present and ticking closer to expiration on `redis-node-1`

```bash
# On `redis-node-1` (or any node)
GET foo
"bar"

TTL foo
(integer) 3479
```

Finally, I'd like to check persistance. To do this, I check the contents of the `/data` folder. I see both an `*.aof` file and a `*.rdb` file. As an additional test, I could fire off several thousand writes to make sure they are updating, but for the time being this is satisfactory.

```bash
$ ls -lh /data
total 24K
-rw-r--r-- 1 1001 1001 176 Dec 24 23:36 appendonly.aof
-rw-r--r-- 1 1001 1001 175 Dec 24 22:26 dump.rdb
drwxrws--- 2 root 1001 16K Dec 24 19:25 lost+found
```

### Testing Metrics Export

To confirm that there's a metrics exporter running, I'll port-forward `9121` of `svc/redis-metrics` to my local machine and `tail` the `/metrics` endpoint to show the `redis_uptime_in_seconds` metric. I wait a moment and run the command again to find the value has increased by a few seconds. 

```bash
# Port forward `svc/redis-metrics` (i.e. the metrics exporter) -> localhost
kubectl port-forward \
  --namespace redis \
  svc/redis-metrics 9121:9121
```

```bash
# Check a metric, in this case `redis_uptime_in_seconds`
curl --silent -XGET http://localhost:9121/metrics | tail -n 1
redis_uptime_in_seconds 5586

# /metrics endpoint should update at least every few seconds, expect `redis_uptime_in_seconds` higher on second call...
curl --silent -XGET http://localhost:9121/metrics | tail -n 1
redis_uptime_in_seconds 5692
```

Looks good to me!

### Testing Sentinel FailOver

I'd also like to test cluster failover with Sentinel by killing the current master node. There may be a more elegant way to do this, but nothing makes as much sense to me as just coming in with a `kubectl delete`

```bash
kubectl delete pod redis-node-0 \
  --namespace redis
```

Several moments later, I check the status of the pods. I can see that a new pod has just started in the namespace, replacing the old `redis-node-0`.

```bash
kubectl get pods \
  --namespace redis
NAME           READY   STATUS    RESTARTS   AGE
redis-node-0   1/3     Running   0          23s
redis-node-1   3/3     Running   0          22m
redis-node-2   3/3     Running   0          23m
```

Using the same `INFO replication` command from [earlier](###Testing%20Redis%20Data%20Replication), I check the `replication` status of the just-restarted `redis-node-0` and find the following:

```bash
role:slave
master_host:redis-node-1.redis-headless.redis.svc.cluster.local
master_port:6379
```

Excellent, this suggests that our "new" node has joined the cluster as a slave of `redis-node-1.redis-headless.redis.svc.cluster.local`. Just as writes were initially restricted to `redis-node-0`, I'd now expect writes to be restrited to `redis-node-1`.

## Conclusion

From my perspective, this is a great starting point for learning K8s. For the purposes of the Digital Ocean K8s Challenge, this is where I'll end.

As I mentioned earlier, I'd like to build logging, monitoring, and a simple application on top of this Redis cluster. I'm sure that will reveal any glaring flaws in this configuration. The notes for that ongoing work are [here](./deployment_notes.md).
