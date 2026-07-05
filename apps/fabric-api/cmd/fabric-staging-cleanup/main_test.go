package main

import (
	"context"
	"errors"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDiscoverFabricNodePoolsOnlyUsesFabricComputeLabels(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "fabric-a", Labels: map[string]string{
			"oplfabric.cn/compute-id":               "compute-a",
			"node.tke.cloud.tencent.com/machineset": "np-fabric-a",
		}}},
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "fabric-b", Labels: map[string]string{
			"oplfabric.cn/compute-id":            "compute-b",
			"cloud.tencent.com/node-instance-id": "np-fabric-b-abcd",
		}}},
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "shared", Labels: map[string]string{
			"oplcloud.cn/pool-id":                   "pool-basic-2c4g",
			"node.tke.cloud.tencent.com/machineset": "np-shared",
		}}},
	)

	got, err := discoverFabricNodePoolsFromNodes(context.Background(), client)
	if err != nil {
		t.Fatalf("discoverFabricNodePoolsFromNodes: %v", err)
	}
	want := []string{"np-fabric", "np-fabric-a"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("nodepool ids = %#v, want %#v", got, want)
	}
}

func TestDeleteDeploymentsFiltersFabricWorkspaceDeployments(t *testing.T) {
	client := fake.NewSimpleClientset(
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      "fabric-compute",
			Namespace: "oplfabric",
			Labels:    map[string]string{"app.kubernetes.io/name": "opl-workspace"},
			Annotations: map[string]string{
				"oplcloud.cn/compute-id": "compute-1",
			},
		}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      "other",
			Namespace: "oplfabric",
			Labels:    map[string]string{"app.kubernetes.io/name": "not-fabric"},
		}},
	)

	if err := deleteDeployments(context.Background(), client, "oplfabric"); err != nil {
		t.Fatalf("deleteDeployments: %v", err)
	}
	if _, err := client.AppsV1().Deployments("oplfabric").Get(context.Background(), "fabric-compute", metav1.GetOptions{}); err == nil {
		t.Fatal("fabric deployment should be deleted")
	}
	if _, err := client.AppsV1().Deployments("oplfabric").Get(context.Background(), "other", metav1.GetOptions{}); err != nil {
		t.Fatalf("non-fabric deployment should remain: %v", err)
	}
}

func TestStringSetSortsAndDedupes(t *testing.T) {
	set := stringSet{}
	set.addAll(splitCSV("np-b, np-a,,np-b"))
	got := set.sorted()
	want := []string{"np-a", "np-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sorted = %#v, want %#v", got, want)
	}
}

func TestIsNotFoundLikeRecognizesDeletedCloudResources(t *testing.T) {
	for _, err := range []error{
		errors.New("ResourceNotFound.NodePool not found"),
		errors.New("node pool does not exist"),
	} {
		if !isNotFoundLike(err) {
			t.Fatalf("isNotFoundLike(%q) = false, want true", err)
		}
	}
	if isNotFoundLike(errors.New("UnauthorizedOperation")) {
		t.Fatal("authorization errors must not be treated as not found")
	}
}
