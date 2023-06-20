/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package lifecycle contains the handlers for the lifecycle hooks.
package lifecycle

import (
	"context"
	"log"
	"strings"

	capov1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha6"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamic "k8s.io/client-go/dynamic"
	runtimehooksv1 "sigs.k8s.io/cluster-api/exp/runtime/hooks/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const clusterUpgradeGroup string = "cluster.aruba.it"
const clusterUpgradeKind string = "ClusterUpgrade"
const clusterUpgradeVersion string = "v1alpha1"

// Handler is the handler for the lifecycle hooks.
type Handler struct {
	Client        client.Client
	DynamicClient dynamic.Interface
}

// DoBeforeClusterCreate implements the BeforeClusterCreate hook.
func (h *Handler) DoBeforeClusterCreate(ctx context.Context, request *runtimehooksv1.BeforeClusterCreateRequest, response *runtimehooksv1.BeforeClusterCreateResponse) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("BeforeClusterCreate is called")
	response.Status = runtimehooksv1.ResponseStatusSuccess
	return
}

// DoBeforeClusterUpgrade implements the BeforeClusterUpgrade hook.
func (h *Handler) DoBeforeClusterUpgrade(ctx context.Context, request *runtimehooksv1.BeforeClusterUpgradeRequest, response *runtimehooksv1.BeforeClusterUpgradeResponse) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("BeforeClusterUpgrade is called")

	setupLog := ctrl.Log.WithName("setup")
	//check if a ClusterUpgrade resource already exists for the specific cluster and upgrade (formVersion -> toVersion)
	clusterpgrade, error := h.getClusterUpgrade(context.Background(), h.DynamicClient, request.Cluster.Name, request.Cluster.Namespace, request.ToKubernetesVersion)
	if error != nil {
		setupLog.Error(error, error.Error())
		response.Status = runtimehooksv1.ResponseStatusFailure
		response.Message = "Error retrieving ClusterUpgrade list"
		return
	}
	if len(clusterpgrade) > 0 {
		log.Info("There are ClusterUpgrade resource for cluster " + request.Cluster.Name)
		response.Status = runtimehooksv1.ResponseStatusSuccess
		//if a ClusterUpgrade resource exists and its run status is != Successful, the upgrade must be blocked
		if !runsSuccessful(clusterpgrade[0]) {
			response.RetryAfterSeconds = 30
		}
		return
	}

	osmList := &capov1.OpenStackMachineList{}
	err := h.Client.List(context.Background(), osmList, client.InNamespace("default"))
	if err != nil || len(osmList.Items) == 0 {
		setupLog.Error(err, err.Error())
		response.Status = runtimehooksv1.ResponseStatusFailure
		response.Message = "Error retrieving Machine list"
		return
	}
	var nodesIp []string = extractControPlaneNodesIp(osmList, request.Cluster.Name)

	// Using a unstructured object.
	u := &unstructured.Unstructured{}
	u.Object = map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      request.Cluster.Name + "-" + request.ToKubernetesVersion,
			"namespace": request.Cluster.Namespace,
		},
		"spec": map[string]interface{}{
			"nodes_ip": nodesIp,
			"upgraded": false,
		},
	}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   clusterUpgradeGroup,
		Kind:    clusterUpgradeKind,
		Version: clusterUpgradeVersion,
	})

	err = h.Client.Create(context.Background(), u)

	if err != nil {
		log.Error(err, err.Error())
		//TODO manage error
		return
	}
	response.Status = runtimehooksv1.ResponseStatusSuccess
	response.RetryAfterSeconds = 30
	return
}

func runsSuccessful(u unstructured.Unstructured) bool {
	conditions, found, err := unstructured.NestedSlice(u.Object, "status", "conditions")
	if err != nil {
		log.Fatalf("Failed to get field: %v", err)
	}
	if found {
		for _, condition := range conditions {
			if conditionMap, ok := condition.(map[string]interface{}); ok {
				conditionType := conditionMap["type"].(string)
				if conditionType == "Successful" {
					return true
				}
			}
		}
	}
	return false
}

// DoAfterControlPlaneInitialized implements the AfterControlPlaneInitialized hook.
func (h *Handler) DoAfterControlPlaneInitialized(ctx context.Context, request *runtimehooksv1.AfterControlPlaneInitializedRequest, response *runtimehooksv1.AfterControlPlaneInitializedResponse) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("AfterControlPlaneInitialized is called")
	response.Status = runtimehooksv1.ResponseStatusSuccess
	return
}

// DoAfterControlPlaneUpgrade implements the AfterControlPlaneUpgrade hook.
func (h *Handler) DoAfterControlPlaneUpgrade(ctx context.Context, request *runtimehooksv1.AfterControlPlaneUpgradeRequest, response *runtimehooksv1.AfterControlPlaneUpgradeResponse) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("AfterControlPlaneUpgrade is called")
	response.Status = runtimehooksv1.ResponseStatusSuccess
	return
}

// DoAfterClusterUpgrade implements the AfterClusterUpgrade hook.
func (h *Handler) DoAfterClusterUpgrade(ctx context.Context, request *runtimehooksv1.AfterClusterUpgradeRequest, response *runtimehooksv1.AfterClusterUpgradeResponse) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("AfterClusterUpgrade is called")
	setupLog := ctrl.Log.WithName("setup")
	//check if a ClusterUpgrade resource already exists for the specific cluster and upgrade (formVersion -> toVersion)
	clusterpgrade, error := h.getClusterUpgrade(context.Background(), h.DynamicClient, request.Cluster.Name, request.Cluster.Namespace, request.KubernetesVersion)
	if error != nil {
		setupLog.Error(error, error.Error())
		response.Status = runtimehooksv1.ResponseStatusFailure
		response.Message = "Error retrieving ClusterUpgrade list"
		return
	}
	if len(clusterpgrade) == 0 {
		log.Info("There are no ClusterUpgrade resource for cluster " + request.Cluster.Name + "and version: " + request.KubernetesVersion)
		response.Status = runtimehooksv1.ResponseStatusFailure
		response.Message = "There are no ClusterUpgrade resource for cluster " + request.Cluster.Name + "and version: " + request.KubernetesVersion
		return
	}

	patchData := []byte(`{"spec": {"upgraded": "true"}}`)
	gvr := schema.GroupVersionResource{
		Group:    clusterUpgradeGroup,
		Version:  clusterUpgradeVersion,
		Resource: "clusterupgrades",
	}
	_, err := h.DynamicClient.Resource(gvr).Namespace(clusterpgrade[0].GetNamespace()).Patch(context.Background(), clusterpgrade[0].GetName(), "application/merge-patch+json", patchData, v1.PatchOptions{})
	if err != nil {
		setupLog.Error(error, "failed to patch resource: %v")
		response.Status = runtimehooksv1.ResponseStatusFailure
		return
	}

	response.Status = runtimehooksv1.ResponseStatusSuccess
	return
}

// DoBeforeClusterDelete implements the BeforeClusterDelete hook.
func (h *Handler) DoBeforeClusterDelete(ctx context.Context, request *runtimehooksv1.BeforeClusterDeleteRequest, response *runtimehooksv1.BeforeClusterDeleteResponse) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("BeforeClusterDelete is called")
	response.Status = runtimehooksv1.ResponseStatusSuccess
	return
}

func extractControPlaneNodesIp(machineList *capov1.OpenStackMachineList, clusterName string) []string {
	var nodesIp string
	for _, osm := range machineList.Items {
		if isChildOf(context.Background(), osm, clusterName) {
			for _, addr := range osm.Status.Addresses {
				//os.Getenv("CIDRID")
				if !strings.HasPrefix(addr.Address, "10.6.") {
					nodesIp += addr.Address + " "
				}
			}
		}
	}
	return strings.Fields(nodesIp)
}

func (h *Handler) getClusterUpgrade(ctx context.Context, client dynamic.Interface, clusterName string, namespace string, tok8sVersion string) ([]unstructured.Unstructured, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Cluster name=" + clusterName + ", namespace=" + namespace)
	gvr := schema.GroupVersionResource{
		Group:    clusterUpgradeGroup,
		Version:  clusterUpgradeVersion,
		Resource: "clusterupgrades",
	}
	list, err := client.Resource(gvr).Namespace(namespace).List(ctx, v1.ListOptions{FieldSelector: "metadata.name=" + clusterName + "-" + tok8sVersion})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func isChildOf(ctx context.Context, osm capov1.OpenStackMachine, clusterName string) bool {
	// os.GetEnv("CPID")
	cpIdentifier := "control-plane"
	//idx := "deploymentId"
	log := ctrl.LoggerFrom(ctx)
	log.Info(clusterName)

	if strings.Contains(osm.Name, cpIdentifier) && strings.Contains(osm.Name, clusterName) { //&& osm.Labels[idx] == cluster.Labels[idx] {
		return true
	}

	return false
}
