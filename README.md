# Sample Runtime Extension for Cluster API Runtime SDK

# capi-custom-extension

capi-custom-extension is a runtime extension of Cluster API (CAPI) that implements the logic for the BeforeClusterUpgradeHook. This extension provides additional functionality for handling cluster upgrades before the actual upgrade process takes place.

## Prerequisites

Before using this extension, ensure that you have the following prerequisites in place:

- A ClusterUpgrade resource must be created. Here's an example of a ClusterUpgrade resource manifest:

  ```yaml
  apiVersion: cluster.aruba.it/v1alpha1
  kind: ClusterUpgrade
  metadata:
    name: clusterupgrade-sample-v1.21.2-v1.22.0
    namespace: default
  spec:
    nodes_ip:
    - 10.244.0.12
  ```

- The ClusterUpgrade resource should be in the desired state with relevant annotations, labels, and status conditions based on your use case.

## Usage

To test the extension, run the following command:

```shell
kubectl create --raw '/api/v1/namespaces/default/services/https:sample-service:443/proxy/hooks.runtime.cluster.x-k8s.io/v1alpha1/beforeclusterupgrade/before-cluster-upgrade' -f <(echo '{"apiVersion":"hooks.runtime.cluster.x-k8s.io/v1alpha1","kind":"BeforeClusterUpgradeRequest","cluster":{"metadata":{"namespace":"default","name":"clusterupgrade-sample"}},"fromKubernetesVersion":"v1.21.2","toKubernetesVersion":"v1.22.0"}')
```

This command creates a BeforeClusterUpgradeRequest for the specified cluster and triggers the extension logic.

## Build and Deployment

To build the project, execute the following commands:

```shell
docker build --platform=linux/amd64 -t YOUR_DOCKER_IMAGE_NAME --build-arg builder_image=golang:1.19.3 .
```

Replace `YOUR_DOCKER_IMAGE_NAME` with the desired name for your Docker image. This command builds the project and creates the Docker image.

To push the Docker image to a repository, use the following command:

```shell
docker push YOUR_DOCKER_IMAGE_NAME
```

Replace `YOUR_DOCKER_IMAGE_NAME` with the actual name of your Docker image. This command pushes the image to the specified repository.

To deploy the extension to your cluster, run the following command:

```shell
kubectl -k apply -k config/default
```

This command applies the Kubernetes manifests located in the `config/default` directory to deploy the extension. Make sure to modify the `extension_image_patch.yaml` file in the `config/default` directory to update the image reference accordingly.

Note: Replace `YOUR_DOCKER_IMAGE_NAME` with the actual name of your Docker image in the `extension_image_patch.yaml` file.
