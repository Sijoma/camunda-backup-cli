package restore

import (
	"fmt"
	"strings"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewDeletionJob(pvcName, pvcNamespace string) *v1.Job {
	return &v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "delete-" + pvcName,
			Namespace: pvcNamespace,
			Labels: map[string]string{
				"job": "delete-zeebe",
			},
		},
		Spec: v1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "delete-" + pvcName,
							Image: "busybox:latest",
							Command: []string{
								"/bin/sh",
								"-c",
								"rm -rf /usr/local/zeebe/data/*",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/usr/local/zeebe/data",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvcName,
								},
							},
						},
					},
				},
			},
		},
	}
}

func NewRestoreJob(pvc corev1.PersistentVolumeClaim, sts *apps.StatefulSet, backupID int64) *v1.Job {
	container := sts.Spec.Template.Spec.Containers[0].DeepCopy()
	zeebeImage := container.Image
	env := container.Env
	nodeID := strings.Split(pvc.Name, "-")[2]
	env = append(env, corev1.EnvVar{
		Name:  "ZEEBE_BROKER_CLUSTER_NODEID",
		Value: nodeID,
	})

	return &v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "restore-" + pvc.Name,
			Namespace: pvc.Namespace,
			Labels: map[string]string{
				"job": "restore-zeebe",
			},
		},
		Spec: v1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "restore-" + pvc.Name,
							Image: zeebeImage,
							Command: []string{
								"/usr/local/zeebe/bin/restore",
								fmt.Sprintf("--backupId=%d", backupID),
							},
							Env: env,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/usr/local/zeebe/data",
								},
								{
									Name:      "gcs-backup-key",
									MountPath: "usr/local/share/gcs-key",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvc.Name,
								},
							},
						},
						{
							Name: "gcs-backup-key",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "gcs-backup-secret",
								},
							},
						},
					},
				},
			},
		},
	}
}
