package api

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"

	"trivy-ui/kubernetes"
)

func TestTrendDataReadyWaitsForReportsAndReplicaSets(t *testing.T) {
	registry := GetDefaultRegistry()
	registry.mu.Lock()
	saved := registry.clients
	registry.clients = map[string]*ClusterClient{}
	registry.mu.Unlock()
	t.Cleanup(func() {
		registry.mu.Lock()
		registry.clients = saved
		registry.mu.Unlock()
	})

	if trendDataReady() {
		t.Fatal("no clusters yet: not ready")
	}

	replicaSets := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}
	dynamicClient := fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{replicaSets: "ReplicaSetList"})
	client := kubernetes.NewClientFromDynamic(dynamicClient)
	cluster := &ClusterClient{Name: "nuc", Client: client, SyncState: "FullySynced"}
	registry.mu.Lock()
	registry.clients["nuc"] = cluster
	registry.mu.Unlock()
	if trendDataReady() {
		t.Fatal("ReplicaSet lookup not started: not ready")
	}

	resolver := client.StartWorkloadResolver("nuc")
	t.Cleanup(resolver.Stop)
	cluster.mu.Lock()
	cluster.SyncState = "Syncing"
	cluster.mu.Unlock()
	deadline := time.Now().Add(5 * time.Second)
	for !resolver.Ready() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !resolver.Ready() {
		t.Fatal("ReplicaSet informer did not sync")
	}
	if trendDataReady() {
		t.Fatal("reports still syncing: not ready")
	}

	cluster.mu.Lock()
	cluster.SyncState = "FullySynced"
	cluster.mu.Unlock()
	if !trendDataReady() {
		t.Fatal("reports and ReplicaSets synced: ready")
	}
}

func TestWaitForTrendDataGivesUp(t *testing.T) {
	if waitForTrendData(func() bool { return false }, time.Millisecond, 20*time.Millisecond) {
		t.Fatal("must give up when the data never becomes ready")
	}
	calls := 0
	if !waitForTrendData(func() bool { calls++; return calls > 2 }, time.Millisecond, time.Second) {
		t.Fatal("must continue once the data is ready")
	}
}
