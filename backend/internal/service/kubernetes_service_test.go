package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/npinot/vibe/backend/internal/model"
)

func TestGeneratePodName(t *testing.T) {
	projectID := uuid.New()
	podName := generatePodName(projectID)

	if len(podName) == 0 {
		t.Error("Expected non-empty pod name")
	}

	if podName[:8] != "project-" {
		t.Errorf("Expected pod name to start with 'project-', got %s", podName)
	}
}

func TestGeneratePVCName(t *testing.T) {
	projectID := uuid.New()
	pvcName := generatePVCName(projectID)

	if len(pvcName) == 0 {
		t.Error("Expected non-empty PVC name")
	}

	if pvcName[:10] != "workspace-" {
		t.Errorf("Expected PVC name to start with 'workspace-', got %s", pvcName)
	}
}

func TestBuildPVCSpec(t *testing.T) {
	projectID := uuid.New()
	pvcName := "test-pvc"
	namespace := "test-namespace"
	size := "1Gi"

	pvc := buildPVCSpec(pvcName, namespace, size, projectID)

	if pvc.Name != pvcName {
		t.Errorf("Expected PVC name %s, got %s", pvcName, pvc.Name)
	}

	if pvc.Namespace != namespace {
		t.Errorf("Expected namespace %s, got %s", namespace, pvc.Namespace)
	}

	expectedStorage := resource.MustParse(size)
	actualStorage := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if !expectedStorage.Equal(actualStorage) {
		t.Errorf("Expected storage %s, got %s", expectedStorage.String(), actualStorage.String())
	}

	if pvc.Labels["project-id"] != projectID.String() {
		t.Errorf("Expected project-id label %s, got %s", projectID.String(), pvc.Labels["project-id"])
	}

	if len(pvc.Spec.AccessModes) != 1 || pvc.Spec.AccessModes[0] != corev1.ReadWriteOnce {
		t.Error("Expected ReadWriteOnce access mode")
	}
}

func TestBuildProjectPodSpec(t *testing.T) {
	projectID := uuid.New()
	podName := "test-pod"
	namespace := "test-namespace"
	pvcName := "test-pvc"
	config := &KubernetesConfig{
		OpenCodeImage:     "opencode:latest",
		FileBrowserImage:  "file-browser:latest",
		SessionProxyImage: "session-proxy:latest",
		CPULimit:          "1000m",
		MemoryLimit:       "1Gi",
		CPURequest:        "100m",
		MemoryRequest:     "256Mi",
	}

	pod := buildProjectPodSpec(podName, namespace, pvcName, projectID, config)

	// Check pod metadata
	if pod.Name != podName {
		t.Errorf("Expected pod name %s, got %s", podName, pod.Name)
	}

	if pod.Namespace != namespace {
		t.Errorf("Expected namespace %s, got %s", namespace, pod.Namespace)
	}

	if pod.Labels["project-id"] != projectID.String() {
		t.Errorf("Expected project-id label %s, got %s", projectID.String(), pod.Labels["project-id"])
	}

	// Check containers
	if len(pod.Spec.Containers) != 3 {
		t.Fatalf("Expected 3 containers, got %d", len(pod.Spec.Containers))
	}

	// Check container names and images
	expectedContainers := map[string]string{
		"opencode-server": config.OpenCodeImage,
		"file-browser":    config.FileBrowserImage,
		"session-proxy":   config.SessionProxyImage,
	}

	for i, container := range pod.Spec.Containers {
		expectedImage, ok := expectedContainers[container.Name]
		if !ok {
			t.Errorf("Unexpected container name: %s", container.Name)
		}
		if container.Image != expectedImage {
			t.Errorf("Container %d: expected image %s, got %s", i, expectedImage, container.Image)
		}

		// Check volume mounts
		if len(container.VolumeMounts) != 1 {
			t.Errorf("Container %s: expected 1 volume mount, got %d", container.Name, len(container.VolumeMounts))
		}
		if container.VolumeMounts[0].Name != "workspace" {
			t.Errorf("Container %s: expected volume mount name 'workspace', got %s", container.Name, container.VolumeMounts[0].Name)
		}
		if container.VolumeMounts[0].MountPath != "/workspace" {
			t.Errorf("Container %s: expected mount path '/workspace', got %s", container.Name, container.VolumeMounts[0].MountPath)
		}
	}

	// Check volumes
	if len(pod.Spec.Volumes) != 1 {
		t.Fatalf("Expected 1 volume, got %d", len(pod.Spec.Volumes))
	}

	if pod.Spec.Volumes[0].Name != "workspace" {
		t.Errorf("Expected volume name 'workspace', got %s", pod.Spec.Volumes[0].Name)
	}

	if pod.Spec.Volumes[0].PersistentVolumeClaim.ClaimName != pvcName {
		t.Errorf("Expected PVC claim name %s, got %s", pvcName, pod.Spec.Volumes[0].PersistentVolumeClaim.ClaimName)
	}

	// Check restart policy
	if pod.Spec.RestartPolicy != corev1.RestartPolicyAlways {
		t.Errorf("Expected RestartPolicyAlways, got %s", pod.Spec.RestartPolicy)
	}
}

func TestCreateProjectPod(t *testing.T) {
	// Create fake clientset
	clientset := fake.NewSimpleClientset()

	config := &KubernetesConfig{
		Namespace:         "test-namespace",
		OpenCodeImage:     "opencode:latest",
		FileBrowserImage:  "file-browser:latest",
		SessionProxyImage: "session-proxy:latest",
		WorkspaceSize:     "1Gi",
		CPULimit:          "1000m",
		MemoryLimit:       "1Gi",
		CPURequest:        "100m",
		MemoryRequest:     "256Mi",
	}

	service := &kubernetesService{
		clientset: clientset,
		namespace: "test-namespace",
		config:    config,
	}

	projectID := uuid.New()
	project := &model.Project{
		ID:     projectID,
		UserID: uuid.New(),
		Name:   "test-project",
	}

	ctx := context.Background()
	err := service.CreateProjectPod(ctx, project)
	if err != nil {
		t.Fatalf("CreateProjectPod failed: %v", err)
	}

	// Verify PVC was created
	if project.WorkspacePVCName == "" {
		t.Error("Expected WorkspacePVCName to be set")
	}

	pvc, err := clientset.CoreV1().PersistentVolumeClaims("test-namespace").Get(ctx, project.WorkspacePVCName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get created PVC: %v", err)
	}

	if pvc.Labels["project-id"] != projectID.String() {
		t.Errorf("Expected PVC project-id label %s, got %s", projectID.String(), pvc.Labels["project-id"])
	}

	// Verify Pod was created
	if project.PodName == "" {
		t.Error("Expected PodName to be set")
	}

	pod, err := clientset.CoreV1().Pods("test-namespace").Get(ctx, project.PodName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get created pod: %v", err)
	}

	if pod.Labels["project-id"] != projectID.String() {
		t.Errorf("Expected pod project-id label %s, got %s", projectID.String(), pod.Labels["project-id"])
	}

	if len(pod.Spec.Containers) != 3 {
		t.Errorf("Expected 3 containers, got %d", len(pod.Spec.Containers))
	}

	// Verify project metadata was updated
	if project.PodNamespace != "test-namespace" {
		t.Errorf("Expected PodNamespace 'test-namespace', got %s", project.PodNamespace)
	}

	if project.PodCreatedAt == nil {
		t.Error("Expected PodCreatedAt to be set")
	}
}

func TestDeleteProjectPod(t *testing.T) {
	// Create fake clientset with existing pod and PVC
	projectID := uuid.New()
	podName := generatePodName(projectID)
	pvcName := generatePVCName(projectID)
	namespace := "test-namespace"

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "workspace",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
		},
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
		},
	}

	clientset := fake.NewSimpleClientset(pod, pvc)

	config := &KubernetesConfig{
		Namespace: namespace,
	}

	service := &kubernetesService{
		clientset: clientset,
		namespace: namespace,
		config:    config,
	}

	ctx := context.Background()
	err := service.DeleteProjectPod(ctx, podName, namespace)
	if err != nil {
		t.Fatalf("DeleteProjectPod failed: %v", err)
	}

	// Verify pod was deleted
	_, err = clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err == nil {
		t.Error("Expected pod to be deleted")
	}

	// Verify PVC was deleted
	_, err = clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
	if err == nil {
		t.Error("Expected PVC to be deleted")
	}
}

func TestGetPodStatus(t *testing.T) {
	projectID := uuid.New()
	podName := generatePodName(projectID)
	namespace := "test-namespace"

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	clientset := fake.NewSimpleClientset(pod)

	config := &KubernetesConfig{
		Namespace: namespace,
	}

	service := &kubernetesService{
		clientset: clientset,
		namespace: namespace,
		config:    config,
	}

	ctx := context.Background()
	status, err := service.GetPodStatus(ctx, podName, namespace)
	if err != nil {
		t.Fatalf("GetPodStatus failed: %v", err)
	}

	if status != string(corev1.PodRunning) {
		t.Errorf("Expected status 'Running', got %s", status)
	}

	// Test non-existent pod
	status, err = service.GetPodStatus(ctx, "non-existent", namespace)
	if err != nil {
		t.Fatalf("GetPodStatus for non-existent pod failed: %v", err)
	}

	if status != "NotFound" {
		t.Errorf("Expected status 'NotFound', got %s", status)
	}
}

func TestWatchPodStatus(t *testing.T) {
	projectID := uuid.New()
	podName := generatePodName(projectID)
	namespace := "test-namespace"

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}

	clientset := fake.NewSimpleClientset(pod)

	// Set up watch reactor
	watcher := watch.NewFake()
	clientset.PrependWatchReactor("pods", k8stesting.DefaultWatchReactor(watcher, nil))

	config := &KubernetesConfig{
		Namespace: namespace,
	}

	service := &kubernetesService{
		clientset: clientset,
		namespace: namespace,
		config:    config,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	statusChan, err := service.WatchPodStatus(ctx, podName, namespace)
	if err != nil {
		t.Fatalf("WatchPodStatus failed: %v", err)
	}

	// Simulate pod status change
	go func() {
		time.Sleep(100 * time.Millisecond)
		updatedPod := pod.DeepCopy()
		updatedPod.Status.Phase = corev1.PodRunning
		watcher.Modify(updatedPod)
	}()

	// Wait for status update
	select {
	case status := <-statusChan:
		if status != string(corev1.PodRunning) {
			t.Errorf("Expected status 'Running', got %s", status)
		}
	case <-ctx.Done():
		t.Fatal("Timeout waiting for status update")
	}
}
