package k8s

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

type Provider struct {
	Client               kubernetes.Interface
	Namespace            string
	WorkspaceImage       string
	StorageClassName     string
	WorkspaceDomain      string
	IngressClassName     string
	WorkspaceWebUIPort   int32
	WorkspaceDataDir     string
	WorkspaceProjectsDir string
	CodexHome            string
	CodexModel           string
	CodexReasoningEffort string
	CodexBaseURL         string
	CodexAPIKey          string
	CodexModelProvider   string
	CodexProviderName    string
}

type CreateComputeInput struct {
	ID                   string
	WorkspaceName        string
	ProductPresetID      string
	ComputeShapeJSON     string
	ProviderInstanceType string
	CapacityPoolID       string
	IsolationMode        string
	NodePoolID           string
	RuntimeRef           string
	CPU                  int
	MemoryGB             int
}

type CreateComputeResult struct {
	ProviderRef string
	ServiceRef  string
}

type CreateStorageVolumeInput struct {
	ID     string
	SizeGB int
}

type CreateStorageVolumeResult struct {
	ProviderRef string
}

type AttachStorageInput struct {
	ID         string
	ComputeRef string
	StorageRef string
	MountPath  string
	SubPath    string
}

type AttachStorageResult struct {
	ProviderRef string
}

type CreateWorkspaceEntryInput struct {
	ID          string
	WorkspaceID string
	Host        string
	Path        string
	ServiceRef  string
}

type DestroyComputeInput struct {
	ProviderRef string
	RuntimeRef  string
}

type DestroyStorageInput struct {
	ProviderRef string
}

type DetachStorageInput struct {
	ProviderRef string
}

func (p Provider) CreateCompute(ctx context.Context, input CreateComputeInput) (CreateComputeResult, error) {
	name := k8sName(input.ID)
	computeKey := labelValue(input.ID)
	labels := map[string]string{
		"app.kubernetes.io/name":     "opl-workspace",
		"app.kubernetes.io/instance": name,
		"oplcloud.cn/compute-key":    computeKey,
	}
	annotations := map[string]string{
		"oplcloud.cn/compute-id": input.ID,
	}
	if input.CapacityPoolID != "" {
		annotations["oplcloud.cn/capacity-pool-id"] = input.CapacityPoolID
	}
	if input.IsolationMode != "" {
		annotations["oplcloud.cn/isolation-mode"] = input.IsolationMode
	}
	if input.NodePoolID != "" {
		annotations["oplcloud.cn/node-pool-id"] = input.NodePoolID
	}
	if input.RuntimeRef != "" {
		annotations["oplcloud.cn/runtime-ref"] = input.RuntimeRef
	}
	if input.ProviderInstanceType != "" {
		annotations["oplcloud.cn/provider-instance-type"] = input.ProviderInstanceType
	}
	codexSecretName := ""
	if secret := p.codexSecret(name, labels); secret != nil {
		if _, err := p.Client.CoreV1().Secrets(p.Namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
			return CreateComputeResult{}, err
		}
		codexSecretName = secret.Name
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: p.Namespace, Labels: labels, Annotations: annotations},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr[int32](1),
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels, Annotations: annotations},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: ptr(false),
					Containers: []corev1.Container{{
						Name:  "workspace",
						Image: p.WorkspaceImage,
						Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: p.workspaceWebUIPort()}},
						Resources: corev1.ResourceRequirements{
							Requests: resourceList(input),
							Limits:   resourceList(input),
						},
						EnvFrom: p.codexEnvFrom(codexSecretName),
						Env: []corev1.EnvVar{
							{Name: "OPL_COMPUTE_ID", Value: input.ID},
							{Name: "OPL_WORKSPACE_NAME", Value: input.WorkspaceName},
							{Name: "OPL_PRODUCT_PRESET_ID", Value: input.ProductPresetID},
							{Name: "OPL_COMPUTE_SHAPE_JSON", Value: input.ComputeShapeJSON},
							{Name: "OPL_CAPACITY_POOL_ID", Value: input.CapacityPoolID},
							{Name: "OPL_ISOLATION_MODE", Value: input.IsolationMode},
							{Name: "OPL_PROJECTS_DIR", Value: defaultString(p.WorkspaceProjectsDir, "/projects")},
							{Name: "OPL_WEBUI_AUTH_MODE", Value: "none"},
							{Name: "OPL_WORKSPACE_ROOT", Value: defaultString(p.WorkspaceProjectsDir, "/projects")},
							{Name: "CODEX_HOME", Value: defaultString(p.CodexHome, "/data/codex")},
						},
					}},
				},
			},
		},
	}
	if _, err := p.Client.AppsV1().Deployments(p.Namespace).Create(ctx, deploy, metav1.CreateOptions{}); err != nil {
		if codexSecretName != "" {
			if deleteErr := p.Client.CoreV1().Secrets(p.Namespace).Delete(ctx, codexSecretName, metav1.DeleteOptions{}); deleteErr != nil {
				return CreateComputeResult{}, errors.Join(err, fmt.Errorf("cleanup secret %q: %w", codexSecretName, deleteErr))
			}
		}
		return CreateComputeResult{}, err
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: p.Namespace, Labels: labels},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports:    []corev1.ServicePort{{Name: "http", Port: p.workspaceWebUIPort(), TargetPort: intstr.FromInt(int(p.workspaceWebUIPort()))}},
		},
	}
	if _, err := p.Client.CoreV1().Services(p.Namespace).Create(ctx, service, metav1.CreateOptions{}); err != nil {
		if deleteErr := p.Client.AppsV1().Deployments(p.Namespace).Delete(ctx, name, metav1.DeleteOptions{}); deleteErr != nil {
			return CreateComputeResult{}, errors.Join(err, fmt.Errorf("cleanup deployment %q: %w", name, deleteErr))
		}
		if codexSecretName != "" {
			if deleteErr := p.Client.CoreV1().Secrets(p.Namespace).Delete(ctx, codexSecretName, metav1.DeleteOptions{}); deleteErr != nil {
				return CreateComputeResult{}, errors.Join(err, fmt.Errorf("cleanup secret %q: %w", codexSecretName, deleteErr))
			}
		}
		return CreateComputeResult{}, err
	}

	return CreateComputeResult{ProviderRef: "deployment/" + name, ServiceRef: "service/" + name}, nil
}

func (p Provider) CreateStorageVolume(ctx context.Context, input CreateStorageVolumeInput) (CreateStorageVolumeResult, error) {
	name := k8sName(input.ID) + "-data"
	sizeGB := input.SizeGB
	if sizeGB <= 0 {
		sizeGB = 10
	}
	storageClassName := defaultString(p.StorageClassName, "cbs")
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: p.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "opl-workspace-storage",
				"oplcloud.cn/storage-key":      labelValue(input.ID),
				"oplcloud.cn/storage-retained": "true",
			},
			Annotations: map[string]string{"oplcloud.cn/storage-id": input.ID},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &storageClassName,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(fmt.Sprintf("%dGi", sizeGB))},
			},
		},
	}
	if _, err := p.Client.CoreV1().PersistentVolumeClaims(p.Namespace).Create(ctx, pvc, metav1.CreateOptions{}); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return CreateStorageVolumeResult{ProviderRef: "pvc/" + name}, nil
		}
		return CreateStorageVolumeResult{}, err
	}
	return CreateStorageVolumeResult{ProviderRef: "pvc/" + name}, nil
}

func (p Provider) AttachStorage(ctx context.Context, input AttachStorageInput) (AttachStorageResult, error) {
	deploymentName := resourceName(input.ComputeRef)
	pvcName := resourceName(input.StorageRef)
	deploy, err := p.Client.AppsV1().Deployments(p.Namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return AttachStorageResult{}, err
	}
	volumeName := "workspace-data"
	mountPath := defaultString(input.MountPath, defaultString(p.WorkspaceDataDir, "/data"))
	deploy.Spec.Template.Spec.Volumes = upsertVolume(deploy.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName},
		},
	})
	container := &deploy.Spec.Template.Spec.Containers[0]
	container.VolumeMounts = upsertVolumeMount(container.VolumeMounts, corev1.VolumeMount{
		Name:      volumeName,
		MountPath: mountPath,
		SubPath:   input.SubPath,
	})
	if _, err := p.Client.AppsV1().Deployments(p.Namespace).Update(ctx, deploy, metav1.UpdateOptions{}); err != nil {
		return AttachStorageResult{}, err
	}
	return AttachStorageResult{ProviderRef: "deployment/" + deploymentName + ":pvc/" + pvcName}, nil
}

func (p Provider) DetachStorage(ctx context.Context, input DetachStorageInput) error {
	deploymentName, pvcName := splitAttachmentRef(input.ProviderRef)
	deploy, err := p.Client.AppsV1().Deployments(p.Namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	volumeName := "workspace-data"
	if pvcName != "" {
		for _, volume := range deploy.Spec.Template.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == pvcName {
				volumeName = volume.Name
				break
			}
		}
	}
	deploy.Spec.Template.Spec.Volumes = removeVolume(deploy.Spec.Template.Spec.Volumes, volumeName)
	if len(deploy.Spec.Template.Spec.Containers) > 0 {
		container := &deploy.Spec.Template.Spec.Containers[0]
		container.VolumeMounts = removeVolumeMount(container.VolumeMounts, volumeName)
	}
	_, err = p.Client.AppsV1().Deployments(p.Namespace).Update(ctx, deploy, metav1.UpdateOptions{})
	return err
}

func (p Provider) CreateWorkspaceEntry(ctx context.Context, input CreateWorkspaceEntryInput) error {
	name := "opl-fabric-workspace-gateway"
	host := defaultString(input.Host, p.WorkspaceDomain)
	path := defaultString(input.Path, "/w/"+input.WorkspaceID+"/")
	serviceName := resourceName(input.ServiceRef)
	pathType := networkingv1.PathTypePrefix
	backend := networkingv1.IngressBackend{
		Service: &networkingv1.IngressServiceBackend{
			Name: serviceName,
			Port: networkingv1.ServiceBackendPort{Number: p.workspaceWebUIPort()},
		},
	}
	ingress, err := p.Client.NetworkingV1().Ingresses(p.Namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		ingress = &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: p.Namespace,
				Labels:    map[string]string{"app.kubernetes.io/name": "opl-workspace-gateway"},
			},
			Spec: networkingv1.IngressSpec{
				IngressClassName: optionalString(p.IngressClassName),
				Rules: []networkingv1.IngressRule{{
					Host: host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{},
					},
				}},
			},
		}
		ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths = []networkingv1.HTTPIngressPath{{
			Path:     path,
			PathType: &pathType,
			Backend:  backend,
		}}
		_, err = p.Client.NetworkingV1().Ingresses(p.Namespace).Create(ctx, ingress, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	ensureIngressClass(ingress, p.IngressClassName)
	rule := ensureIngressRule(ingress, host)
	rule.HTTP.Paths = upsertIngressPath(rule.HTTP.Paths, networkingv1.HTTPIngressPath{
		Path:     path,
		PathType: &pathType,
		Backend:  backend,
	})
	_, err = p.Client.NetworkingV1().Ingresses(p.Namespace).Update(ctx, ingress, metav1.UpdateOptions{})
	return err
}

func (p Provider) DestroyCompute(ctx context.Context, input DestroyComputeInput) error {
	deploymentName := resourceName(input.ProviderRef)
	serviceName := resourceName(input.RuntimeRef)
	if serviceName == "" {
		serviceName = deploymentName
	}
	if err := p.Client.AppsV1().Deployments(p.Namespace).Delete(ctx, deploymentName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := p.Client.CoreV1().Services(p.Namespace).Delete(ctx, serviceName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := p.Client.CoreV1().Secrets(p.Namespace).Delete(ctx, deploymentName+"-env", metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func (p Provider) DestroyStorage(ctx context.Context, input DestroyStorageInput) error {
	name := resourceName(input.ProviderRef)
	if err := p.Client.CoreV1().PersistentVolumeClaims(p.Namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func k8sName(id string) string {
	return boundedName("opl", id, 63)
}

func labelValue(id string) string {
	return boundedName("compute", id, 63)
}

func boundedName(prefix, id string, limit int) string {
	clean := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + 32
		}
		return '-'
	}, id)
	clean = strings.Trim(clean, "-")
	if clean == "" {
		clean = "resource"
	}
	hash := shortHash(id)
	maxClean := limit - len(prefix) - len(hash) - 2
	if maxClean < 1 {
		maxClean = 1
	}
	if len(clean) > maxClean {
		clean = strings.Trim(clean[:maxClean], "-")
	}
	if clean == "" {
		clean = "resource"
	}
	return fmt.Sprintf("%s-%s-%s", prefix, clean, hash)
}

func shortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:8]
}

func resourceList(input CreateComputeInput) corev1.ResourceList {
	resources := corev1.ResourceList{}
	if input.CPU > 0 {
		resources[corev1.ResourceCPU] = resource.MustParse(fmt.Sprintf("%d", input.CPU))
	}
	if input.MemoryGB > 0 {
		resources[corev1.ResourceMemory] = resource.MustParse(fmt.Sprintf("%dGi", input.MemoryGB))
	}
	return resources
}

func (p Provider) workspaceWebUIPort() int32 {
	if p.WorkspaceWebUIPort > 0 {
		return p.WorkspaceWebUIPort
	}
	return 3000
}

func (p Provider) codexSecret(name string, labels map[string]string) *corev1.Secret {
	data := map[string][]byte{}
	put := func(key, value string) {
		if value != "" {
			data[key] = []byte(value)
		}
	}
	put("OPL_CODEX_MODEL", p.CodexModel)
	put("OPL_CODEX_REASONING_EFFORT", p.CodexReasoningEffort)
	put("OPL_CODEX_BASE_URL", p.CodexBaseURL)
	put("OPL_CODEX_API_KEY", p.CodexAPIKey)
	put("OPL_CODEX_MODEL_PROVIDER", p.CodexModelProvider)
	put("OPL_CODEX_PROVIDER_NAME", p.CodexProviderName)
	if len(data) == 0 {
		return nil
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name + "-env", Namespace: p.Namespace, Labels: labels},
		Type:       corev1.SecretTypeOpaque,
		Data:       data,
	}
}

func (p Provider) codexEnvFrom(secretName string) []corev1.EnvFromSource {
	if secretName == "" {
		return nil
	}
	return []corev1.EnvFromSource{{
		SecretRef: &corev1.SecretEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
		},
	}}
}

func resourceName(ref string) string {
	value := strings.TrimSpace(ref)
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ":")
	value = parts[0]
	if strings.HasPrefix(ref, "pvc/") && len(parts) > 1 {
		value = parts[1]
	}
	return strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(value, "deployment/"), "service/"), "pvc/")
}

func splitAttachmentRef(ref string) (deploymentName, pvcName string) {
	parts := strings.Split(ref, ":")
	if len(parts) > 0 {
		deploymentName = resourceName(parts[0])
	}
	if len(parts) > 1 {
		pvcName = resourceName(parts[1])
	}
	return deploymentName, pvcName
}

func upsertVolume(volumes []corev1.Volume, next corev1.Volume) []corev1.Volume {
	for i, volume := range volumes {
		if volume.Name == next.Name {
			volumes[i] = next
			return volumes
		}
	}
	return append(volumes, next)
}

func removeVolume(volumes []corev1.Volume, name string) []corev1.Volume {
	next := volumes[:0]
	for _, volume := range volumes {
		if volume.Name != name {
			next = append(next, volume)
		}
	}
	return next
}

func upsertVolumeMount(mounts []corev1.VolumeMount, next corev1.VolumeMount) []corev1.VolumeMount {
	for i, mount := range mounts {
		if mount.Name == next.Name {
			mounts[i] = next
			return mounts
		}
	}
	return append(mounts, next)
}

func removeVolumeMount(mounts []corev1.VolumeMount, name string) []corev1.VolumeMount {
	next := mounts[:0]
	for _, mount := range mounts {
		if mount.Name != name {
			next = append(next, mount)
		}
	}
	return next
}

func ensureIngressClass(ingress *networkingv1.Ingress, className string) {
	if className != "" {
		ingress.Spec.IngressClassName = &className
	}
}

func ensureIngressRule(ingress *networkingv1.Ingress, host string) *networkingv1.IngressRule {
	for i := range ingress.Spec.Rules {
		if ingress.Spec.Rules[i].Host == host {
			if ingress.Spec.Rules[i].HTTP == nil {
				ingress.Spec.Rules[i].HTTP = &networkingv1.HTTPIngressRuleValue{}
			}
			return &ingress.Spec.Rules[i]
		}
	}
	ingress.Spec.Rules = append(ingress.Spec.Rules, networkingv1.IngressRule{
		Host: host,
		IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{},
		},
	})
	return &ingress.Spec.Rules[len(ingress.Spec.Rules)-1]
}

func upsertIngressPath(paths []networkingv1.HTTPIngressPath, next networkingv1.HTTPIngressPath) []networkingv1.HTTPIngressPath {
	for i, path := range paths {
		if path.Path == next.Path {
			paths[i] = next
			return paths
		}
	}
	return append(paths, next)
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func ptr[T any](value T) *T {
	return &value
}
