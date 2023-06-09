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
	"strconv"
	"strings"
	"time"

	capov1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1alpha6"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	runtimehooksv1 "sigs.k8s.io/cluster-api/exp/runtime/hooks/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Handler is the handler for the lifecycle hooks.
type Handler struct {
	Client client.Client
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
	response.Status = runtimehooksv1.ResponseStatusSuccess
	response.RetryAfterSeconds = 60

	//osc := &capov1.OpenStackCluster{}
	//err := h.Client.Get(context.Background(), client.ObjectKey{Name: request.Cluster.Name, Namespace: "default"}, osc)
	//if err != nil || osc == nil {
	//	log.Error(err, err.Error())
	//	return
	//}
	//var inventoryInline string
	//var playbookInline string
	//inventoryInline += osc.Spec.ControlPlaneEndpoint.Host + " ansible_user=ubuntu ansible_ssh_private_key_file=arutest" + "\n"

	//osmList := &capov6.OpenStackMachineList{}
	//err = lifecycleHandler.Client.List(context.Background(), osmList, client.InNamespace("default"))
	//if err != nil || len(osmList.Items) == 0 {
	//	setupLog.Error(err, err.Error())
	//	return
	//}
	//var inventoryInline string
	//var playbookInline string
	//for _, osm := range osmList.Items {
	//	setupLog.Info(osm.Name)
	//	if isChildOf(context.Background(), osm) {
	//		for _, addr := range osm.Status.Addresses {
	//			//os.Getenv("CIDRID")
	//			//if !strings.HasPrefix(addr.Address, "10.6.") {
	//			inventoryInline += addr.Address + " ansible_user=ubuntu ansible_ssh_private_key_file=/arutest" + "\n"
	//			//}
	//		}
	//	}
	//}

	/*ansi := &ansible.AnsibleRun{}
	content, err := os.ReadFile("handlers/lifecycle/play.yaml")
	if err != nil {
		log.Error(err, err.Error())
		return
	}

	name := "test-go-ansible"
	namespace := "default"
	playbookInline = string(content)
	ansi.Name = name
	ansi.Namespace = namespace
	ansi.Spec.ForProvider.InventoryInline = &inventoryInline
	ansi.Spec.ForProvider.PlaybookInline = &playbookInline
	*/
	setupLog := ctrl.Log.WithName("setup")

	osmList := &capov1.OpenStackMachineList{}
	err := h.Client.List(context.Background(), osmList, client.InNamespace("default"))
	if err != nil || len(osmList.Items) == 0 {
		setupLog.Error(err, err.Error())
		return
	}
	var nodesIp string
	for _, osm := range osmList.Items {
		setupLog.Info(osm.Name)
		if isChildOf(context.Background(), osm, request.Cluster.Name) {
			for _, addr := range osm.Status.Addresses {
				//os.Getenv("CIDRID")
				//	if !strings.HasPrefix(addr.Address, "10.6.") {
				nodesIp += addr.Address + " "
				//	}
			}
		}
	}

	// Using a unstructured object.
	u := &unstructured.Unstructured{}
	u.Object = map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": request.Cluster.Name + strconv.FormatInt(time.Now().Unix(), 16),
		},
		"spec": map[string]interface{}{
			"nodes_ip": strings.Fields(nodesIp),
		},
	}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "cluster.aruba.it",
		Kind:    "ClusterUpgrade",
		Version: "v1alpha1",
	})

	err = h.Client.Create(context.Background(), u)

	if err != nil {
		log.Error(err, err.Error())
		return
	}

	return
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
