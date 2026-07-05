package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/config"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/kubeconfig"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/tencentcloud"
	"github.com/jackc/pgx/v5/pgxpool"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

const confirmValue = "DELETE"

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), durationEnv("OPL_STAGING_CLEANUP_TIMEOUT", 30*time.Minute))
	defer cancel()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	if os.Getenv("OPL_STAGING_CLEANUP_CONFIRM") != confirmValue {
		return fmt.Errorf("staging cleanup requires OPL_STAGING_CLEANUP_CONFIRM=%s", confirmValue)
	}
	cfg := config.Load()
	restConfig, err := kubeconfig.LoadRESTConfig(cfg.TencentDeployKubeconfigRef)
	if err != nil {
		return err
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	if cfg.KubernetesNamespace == "" {
		return errors.New("OPL_K8S_NAMESPACE is required")
	}

	ids := stringSet{}
	ids.addAll(splitCSV(os.Getenv("OPL_STAGING_CLEANUP_NODEPOOL_IDS")))
	discovered, err := discoverFabricNodePoolsFromNodes(ctx, client)
	if err != nil {
		return err
	}
	ids.addAll(discovered)
	dbIDs, err := discoverFabricNodePoolsFromPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("postgres nodepool discovery skipped: %v", err)
	} else {
		ids.addAll(dbIDs)
	}

	if err := deleteFabricKubernetesRuntime(ctx, client, cfg.KubernetesNamespace); err != nil {
		return err
	}
	if err := deleteNodePools(ctx, cfg, ids.sorted()); err != nil {
		return err
	}
	log.Printf("staging cleanup complete namespace=%s nodePools=%s", cfg.KubernetesNamespace, strings.Join(ids.sorted(), ","))
	return nil
}

func deleteFabricKubernetesRuntime(ctx context.Context, client kubernetes.Interface, namespace string) error {
	if err := deleteDeployments(ctx, client, namespace); err != nil {
		return err
	}
	if err := deleteServices(ctx, client, namespace); err != nil {
		return err
	}
	if err := deleteSecrets(ctx, client, namespace); err != nil {
		return err
	}
	if err := deletePVCs(ctx, client, namespace); err != nil {
		return err
	}
	if err := deleteWorkspaceIngress(ctx, client, namespace); err != nil {
		return err
	}
	return nil
}

func deleteDeployments(ctx context.Context, client kubernetes.Interface, namespace string) error {
	items, err := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{LabelSelector: labels.Set{"app.kubernetes.io/name": "opl-workspace"}.String()})
	if err != nil {
		return err
	}
	for _, item := range items.Items {
		if !isFabricDeployment(item) {
			continue
		}
		log.Printf("delete deployment %s/%s compute=%s nodePool=%s", namespace, item.Name, item.Annotations["oplcloud.cn/compute-id"], item.Annotations["oplcloud.cn/node-pool-id"])
		if err := client.AppsV1().Deployments(namespace).Delete(ctx, item.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func deleteServices(ctx context.Context, client kubernetes.Interface, namespace string) error {
	items, err := client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{LabelSelector: labels.Set{"app.kubernetes.io/name": "opl-workspace"}.String()})
	if err != nil {
		return err
	}
	for _, item := range items.Items {
		log.Printf("delete service %s/%s", namespace, item.Name)
		if err := client.CoreV1().Services(namespace).Delete(ctx, item.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func deleteSecrets(ctx context.Context, client kubernetes.Interface, namespace string) error {
	items, err := client.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{LabelSelector: labels.Set{"app.kubernetes.io/name": "opl-workspace"}.String()})
	if err != nil {
		return err
	}
	for _, item := range items.Items {
		log.Printf("delete secret %s/%s", namespace, item.Name)
		if err := client.CoreV1().Secrets(namespace).Delete(ctx, item.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func deletePVCs(ctx context.Context, client kubernetes.Interface, namespace string) error {
	items, err := client.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{LabelSelector: labels.Set{"app.kubernetes.io/name": "opl-workspace-storage"}.String()})
	if err != nil {
		return err
	}
	for _, item := range items.Items {
		log.Printf("delete pvc %s/%s storage=%s", namespace, item.Name, item.Annotations["oplcloud.cn/storage-id"])
		if err := client.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, item.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func deleteWorkspaceIngress(ctx context.Context, client kubernetes.Interface, namespace string) error {
	const name = "opl-fabric-workspace-gateway"
	if err := client.NetworkingV1().Ingresses(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	log.Printf("delete ingress %s/%s", namespace, name)
	return nil
}

func discoverFabricNodePoolsFromNodes(ctx context.Context, client kubernetes.Interface) ([]string, error) {
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	ids := stringSet{}
	for _, node := range nodes.Items {
		if node.Labels["oplfabric.cn/compute-id"] == "" {
			continue
		}
		if id := node.Labels["node.tke.cloud.tencent.com/machineset"]; id != "" {
			ids.add(id)
			continue
		}
		if instanceID := node.Labels["cloud.tencent.com/node-instance-id"]; strings.HasPrefix(instanceID, "np-") {
			parts := strings.Split(instanceID, "-")
			if len(parts) >= 2 {
				ids.add(parts[0] + "-" + parts[1])
			}
		}
	}
	return ids.sorted(), nil
}

func discoverFabricNodePoolsFromPostgres(ctx context.Context, databaseURL string) ([]string, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("DATABASE_URL is empty")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	defer pool.Close()
	rows, err := pool.Query(ctx, `
SELECT DISTINCT node_pool_id
FROM compute_allocations
WHERE node_pool_id <> ''
  AND (isolation_mode = 'workspace_exclusive_cvm' OR capacity_pool_id IN ('tencent-cpu-compute-pool', 'tencent-gpu-compute-pool'))
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := stringSet{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids.add(id)
	}
	return ids.sorted(), rows.Err()
}

func deleteNodePools(ctx context.Context, cfg config.Config, ids []string) error {
	if len(ids) == 0 {
		log.Printf("no fabric nodepools discovered")
		return nil
	}
	provider := tencentcloud.NodePoolProvider{Config: tencentcloud.NodePoolResolverConfig{
		ClusterID:        cfg.TencentDeployClusterID,
		Region:           cfg.TencentTKERegion,
		SecretID:         cfg.TencentMutationSecretID,
		SecretKey:        cfg.TencentMutationSecretKey,
		SubnetIDs:        defaultString(cfg.TencentCVMSubnetIDs, "cleanup-placeholder"),
		SecurityGroupIDs: defaultString(cfg.TencentCVMSecurityGroupIDs, "cleanup-placeholder"),
		MutationAllowed:  true,
	}}
	for _, id := range ids {
		log.Printf("delete nodepool %s", id)
		if err := provider.DeleteNodePool(ctx, id); err != nil {
			if isNotFoundLike(err) {
				log.Printf("delete nodepool %s skipped: already absent: %v", id, err)
				continue
			}
			return fmt.Errorf("delete nodepool %s: %w", id, err)
		}
	}
	return nil
}

func isNotFoundLike(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "notfound") ||
		strings.Contains(message, "not found") ||
		strings.Contains(message, "not exist") ||
		strings.Contains(message, "notexists")
}

func isFabricDeployment(item appsv1.Deployment) bool {
	return item.Labels["app.kubernetes.io/name"] == "opl-workspace" &&
		(item.Annotations["oplcloud.cn/compute-id"] != "" || item.Labels["oplcloud.cn/compute-key"] != "")
}

type stringSet map[string]struct{}

func (s stringSet) add(value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		s[value] = struct{}{}
	}
}

func (s stringSet) addAll(values []string) {
	for _, value := range values {
		s.add(value)
	}
}

func (s stringSet) sorted() []string {
	values := make([]string, 0, len(s))
	for value := range s {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
