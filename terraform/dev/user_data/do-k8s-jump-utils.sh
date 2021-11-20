#! /bin/sh

# Install kubectl
# Ref: https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/

# NOTE: As of writing (11/18/2021) $KUBECTL__RELEASE_VERSION == 1.22.4; if 1.23.X
# is released without the K8s minor version bump this init script will break; requires
# kubectl within 1 minor version of K8s version...
export KUBECTL__RELEASE_VERSION=$(curl -L -s https://dl.k8s.io/release/stable.txt)

curl -LO "https://dl.k8s.io/release/${KUBECTL__RELEASE_VERSION}/bin/linux/amd64/kubectl" &&\
curl -LO "https://dl.k8s.io/${KUBECTL__RELEASE_VERSION}/bin/linux/amd64/kubectl.sha256" &&\
echo "$(<kubectl.sha256) kubectl" | sha256sum --check

sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Install Helm
curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 &&\
    chmod 700 get_helm.sh &&\
    ./get_helm.sh

# Install doctl + utils; requires 1-time auth w. DO-token
# Ref: https://docs.digitalocean.com/reference/doctl/how-to/install/
sudo snap install doctl &&\
    sudo snap connect doctl:kube-config &&\
    sudo snap connect doctl:ssh-keys :ssh-keys &&\
    sudo snap connect doctl:dot-docker


