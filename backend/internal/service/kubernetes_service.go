package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/npinot/vibe/backend/internal/model"
)

// KubernetesService defines operations for managing Kubernetes resources
type KubernetesService interface {
	// CreateProjectPod creates a new pod with 3 containers and a PVC for the project
	CreateProjectPod(ctx context.Context, project *model.Project) error

	// DeleteProjectPod deletes the pod and PVC associated with the project
	DeleteProjectPod(ctx context.Context, podName, namespace string) error

	// GetPodStatus retrieves the current status of a pod
	GetPodStatus(ctx context.Context, podName, namespace string) (string, error)

	// WatchPodStatus watches for status changes of a pod
	WatchPodStatus(ctx context.Context, podName, namespace string) (<-chan string, error)

	// GetPodIP retrieves the IP address of a pod
	GetPodIP(ctx context.Context, podName, namespace string) (string, error)
}

// kubernetesService implements the KubernetesService interface
type kubernetesService struct {
	clientset kubernetes.Interface
	namespace string
	config    *KubernetesConfig
}

// KubernetesConfig holds configuration for Kubernetes operations
type KubernetesConfig struct {
	Namespace           string
	OpenCodeImage       string
	OpenCodeServerImage string
	FileBrowserImage    string
	SessionProxyImage   string
	WorkspaceSize       string
	CPULimit            string
	MemoryLimit         string
	CPURequest          string
	MemoryRequest       string
}

// NewKubernetesService creates a new Kubernetes service
func NewKubernetesService(kubeconfig, namespace string, config *KubernetesConfig) (KubernetesService, error) {
	clientset, err := initKubernetesClient(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client: %w", err)
	}

	if config == nil {
		config = &KubernetesConfig{
			Namespace:           namespace,
			OpenCodeImage:       "registry.legal-suite.com/opencode/app:latest",
			OpenCodeServerImage: "registry.legal-suite.com/opencode/opencode-server:latest",
			FileBrowserImage:    "registry.legal-suite.com/opencode/file-browser-sidecar:latest",
			SessionProxyImage:   "registry.legal-suite.com/opencode/session-proxy-sidecar:latest",
			WorkspaceSize:       "1Gi",
			CPULimit:            "1000m",
			MemoryLimit:         "1Gi",
			CPURequest:          "100m",
			MemoryRequest:       "256Mi",
		}
	}

	if config.Namespace == "" {
		config.Namespace = namespace
	}

	return &kubernetesService{
		clientset: clientset,
		namespace: namespace,
		config:    config,
	}, nil
}

// initKubernetesClient initializes a Kubernetes client
// Tries in-cluster config first, falls back to kubeconfig
func initKubernetesClient(kubeconfig string) (kubernetes.Interface, error) {
	var config *rest.Config
	var err error

	// Try in-cluster config first
	config, err = rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		if kubeconfig == "" {
			return nil, fmt.Errorf("not running in cluster and no kubeconfig provided: %w", err)
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return clientset, nil
}

// CreateProjectPod creates a new pod with 3 containers and a PVC for the project
func (k *kubernetesService) CreateProjectPod(ctx context.Context, project *model.Project) error {
	// Generate unique names
	podName := generatePodName(project.ID)
	pvcName := generatePVCName(project.ID)

	// Create PVC first
	pvc := buildPVCSpec(pvcName, k.config.Namespace, k.config.WorkspaceSize, project.ID)
	createdPVC, err := k.clientset.CoreV1().PersistentVolumeClaims(k.config.Namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create PVC: %w", err)
	}

	// Create Pod
	pod := buildProjectPodSpec(podName, k.config.Namespace, pvcName, project.ID, k.config)
	createdPod, err := k.clientset.CoreV1().Pods(k.config.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		// Cleanup PVC if pod creation fails
		_ = k.clientset.CoreV1().PersistentVolumeClaims(k.config.Namespace).Delete(ctx, createdPVC.Name, metav1.DeleteOptions{})
		return fmt.Errorf("failed to create pod: %w", err)
	}

	// Update project with pod metadata
	project.PodName = createdPod.Name
	project.PodNamespace = createdPod.Namespace
	project.WorkspacePVCName = createdPVC.Name
	project.PodStatus = string(createdPod.Status.Phase)
	now := time.Now()
	project.PodCreatedAt = &now

	return nil
}

// DeleteProjectPod deletes the pod and PVC associated with the project
func (k *kubernetesService) DeleteProjectPod(ctx context.Context, podName, namespace string) error {
	// Get pod to find associated PVC
	pod, err := k.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get pod: %w", err)
	}

	// Extract PVC name from pod volumes
	var pvcName string
	if pod != nil {
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil {
				pvcName = volume.PersistentVolumeClaim.ClaimName
				break
			}
		}
	}

	// Delete pod
	if pod != nil {
		err = k.clientset.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete pod: %w", err)
		}
	}

	// Delete PVC if found
	if pvcName != "" {
		err = k.clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, pvcName, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete PVC: %w", err)
		}
	}

	return nil
}

// GetPodStatus retrieves the current status of a pod
func (k *kubernetesService) GetPodStatus(ctx context.Context, podName, namespace string) (string, error) {
	pod, err := k.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return "NotFound", nil
		}
		return "", fmt.Errorf("failed to get pod: %w", err)
	}

	return string(pod.Status.Phase), nil
}

// WatchPodStatus watches for status changes of a pod
func (k *kubernetesService) WatchPodStatus(ctx context.Context, podName, namespace string) (<-chan string, error) {
	// Create a watch for the specific pod
	watcher, err := k.clientset.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", podName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pod watcher: %w", err)
	}

	statusChan := make(chan string, 10)

	// Start watching in a goroutine
	go func() {
		defer close(statusChan)
		defer watcher.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.ResultChan():
				if !ok {
					return
				}

				switch event.Type {
				case watch.Added, watch.Modified:
					if pod, ok := event.Object.(*corev1.Pod); ok {
						status := string(pod.Status.Phase)
						select {
						case statusChan <- status:
						case <-ctx.Done():
							return
						}
					}
				case watch.Deleted:
					select {
					case statusChan <- "Deleted":
					case <-ctx.Done():
						return
					}
					return
				case watch.Error:
					return
				}
			}
		}
	}()

	return statusChan, nil
}

// generatePodName generates a unique pod name for a project
func generatePodName(projectID uuid.UUID) string {
	// Kubernetes names must be lowercase alphanumeric + hyphens
	// Max length is 253 characters
	shortID := projectID.String()[:8]
	return fmt.Sprintf("project-%s", shortID)
}

// generatePVCName generates a unique PVC name for a project
func generatePVCName(projectID uuid.UUID) string {
	shortID := projectID.String()[:8]
	return fmt.Sprintf("workspace-%s", shortID)
}

// GetPodIP retrieves the IP address of a pod
func (k *kubernetesService) GetPodIP(ctx context.Context, podName, namespace string) (string, error) {
	pod, err := k.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return "", fmt.Errorf("pod not found")
		}
		return "", fmt.Errorf("failed to get pod: %w", err)
	}

	if pod.Status.PodIP == "" {
		return "", fmt.Errorf("pod IP not yet assigned (pod may not be running)")
	}

	return pod.Status.PodIP, nil
}
