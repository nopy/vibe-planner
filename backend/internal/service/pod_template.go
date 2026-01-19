package service

import (
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// buildProjectPodSpec creates a pod specification with 3 containers and shared PVC
func buildProjectPodSpec(podName, namespace, pvcName string, projectID uuid.UUID, config *KubernetesConfig) *corev1.Pod {
	// Common volume mount
	volumeMount := corev1.VolumeMount{
		Name:      "workspace",
		MountPath: "/workspace",
	}

	// Resource requirements
	resources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(config.CPULimit),
			corev1.ResourceMemory: resource.MustParse(config.MemoryLimit),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(config.CPURequest),
			corev1.ResourceMemory: resource.MustParse(config.MemoryRequest),
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":        "opencode-project",
				"project-id": projectID.String(),
			},
			Annotations: map[string]string{
				"project-id": projectID.String(),
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyAlways,
			Containers: []corev1.Container{
				// Container 1: OpenCode Server
				{
					Name:  "opencode-server",
					Image: config.OpenCodeImage,
					Ports: []corev1.ContainerPort{
						{
							Name:          "http",
							ContainerPort: 3003,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: []corev1.VolumeMount{volumeMount},
					Resources:    resources,
					Env: []corev1.EnvVar{
						{
							Name:  "WORKSPACE_DIR",
							Value: "/workspace",
						},
						{
							Name:  "PROJECT_ID",
							Value: projectID.String(),
						},
					},
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/health",
								Port: intstr.FromInt(3003),
							},
						},
						InitialDelaySeconds: 30,
						PeriodSeconds:       10,
						TimeoutSeconds:      5,
						FailureThreshold:    3,
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/ready",
								Port: intstr.FromInt(3003),
							},
						},
						InitialDelaySeconds: 10,
						PeriodSeconds:       5,
						TimeoutSeconds:      3,
						FailureThreshold:    3,
					},
				},
				// Container 2: File Browser Sidecar
				{
					Name:  "file-browser",
					Image: config.FileBrowserImage,
					Ports: []corev1.ContainerPort{
						{
							Name:          "http",
							ContainerPort: 3001,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: []corev1.VolumeMount{volumeMount},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("50m"),
							corev1.ResourceMemory: resource.MustParse("50Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("100Mi"),
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "WORKSPACE_DIR",
							Value: "/workspace",
						},
						{
							Name:  "PORT",
							Value: "3001",
						},
					},
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/healthz",
								Port: intstr.FromInt(3001),
							},
						},
						InitialDelaySeconds: 5,
						PeriodSeconds:       10,
						TimeoutSeconds:      3,
						FailureThreshold:    3,
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/healthz",
								Port: intstr.FromInt(3001),
							},
						},
						InitialDelaySeconds: 3,
						PeriodSeconds:       5,
						TimeoutSeconds:      3,
						FailureThreshold:    3,
					},
				},
				// Container 3: Session Proxy Sidecar
				{
					Name:  "session-proxy",
					Image: config.SessionProxyImage,
					Ports: []corev1.ContainerPort{
						{
							Name:          "http",
							ContainerPort: 3002,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: []corev1.VolumeMount{volumeMount},
					Resources:    resources,
					Env: []corev1.EnvVar{
						{
							Name:  "WORKSPACE_DIR",
							Value: "/workspace",
						},
						{
							Name:  "PORT",
							Value: "3002",
						},
						{
							Name:  "OPENCODE_URL",
							Value: "http://localhost:3003",
						},
					},
				},
			},
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

	return pod
}

// buildPVCSpec creates a PersistentVolumeClaim specification
func buildPVCSpec(pvcName, namespace, size string, projectID uuid.UUID) *corev1.PersistentVolumeClaim {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":        "opencode-project",
				"project-id": projectID.String(),
			},
			Annotations: map[string]string{
				"project-id": projectID.String(),
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(size),
				},
			},
		},
	}

	return pvc
}
