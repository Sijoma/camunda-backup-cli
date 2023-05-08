package restore

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"c8backup/pkg/backup-client/elastic"
	"c8backup/pkg/backup-client/webapps"
	zeebeBackup "c8backup/pkg/backup-client/zeebe"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	autov1 "k8s.io/client-go/applyconfigurations/autoscaling/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

var statefulsets *apps.StatefulSetList
var deployments *apps.DeploymentList
var pvcs *v1.PersistentVolumeClaimList

func Restore(namespace string, backupID int64, elasticUrl, operateUrl, tasklistUrl, optimizeUrl, zeebeUrl, snapshotRepository string) {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	fmt.Println(backupID)

	ctx := context.Background()
	deployments, statefulsets = getRelatedApps(ctx, kubeClient, namespace)
	if err != nil {
		log.Fatalln(err)
	}

	// We gather all the snapshot names
	elasticClient := elastic.NewElasticClient(elasticUrl, snapshotRepository)
	optimizeClient, _ := webapps.NewBackupClient("optimize", optimizeUrl)
	operateClient, _ := webapps.NewBackupClient("operate", operateUrl)
	tasklistClient, _ := webapps.NewBackupClient("tasklist", tasklistUrl)
	zeebeClient := zeebeBackup.NewZeebeClient(zeebeUrl)

	snapshotNames := gatherSnapshotNames(ctx, backupID, elasticClient, []BackupGetter{optimizeClient, tasklistClient, operateClient})
	fmt.Println(snapshotNames)
	if !(len(snapshotNames) > 0) {
		log.Fatalln("not enough snapshots")
	}

	// Get also zeebe snapshot names
	backupResponse, err := zeebeClient.GetBackup(ctx, backupID)
	if err != nil {
		log.Fatalln(err)
	}
	for _, zeebeBkp := range backupResponse.Details {
		snapshotNames = append(snapshotNames, zeebeBkp.SnapshotId)
	}

	// We shut down related apps
	err = shutdownApps(ctx, kubeClient, namespace)
	if err != nil {
		log.Fatalln(err)
	}

	// Delete everything in elasticsearch
	err = elasticClient.DeleteAllIndices(ctx)
	if err != nil {
		log.Fatalln(err)
		return
	}

	err = deleteZeebeData(ctx, kubeClient, namespace, false)
	if err != nil {
		log.Fatalln(err)
	}

	// Restore the snapshots of the backups
	fmt.Printf("restoring %v\n", snapshotNames)
	err = elasticClient.RestoreSnapshots(ctx, snapshotNames)
	if err != nil {
		fmt.Println("error on snapshot restore", err)
		return
	}

	fmt.Println("restoring zeebe")
	err = restoreZeebe(ctx, kubeClient, namespace, backupID, false)
	if err != nil {
		log.Fatalln(err)
	}

	// Give it some time before scaling up
	fmt.Println("sleeping 10 seconds")
	time.Sleep(time.Second * 10)

	// We reset the apps
	errorList := resetApps(ctx, kubeClient)
	for _, err := range errorList {
		fmt.Println(err)
	}
	if len(errorList) > 0 {
		log.Fatalln("there were errors")
	}
}

func restoreZeebe(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string, backupID int64, alreadyStarted bool) error {
	zeebe := statefulsets.Items[0].DeepCopy()
	if !alreadyStarted {
		for _, item := range pvcs.Items {
			restoreJob := NewRestoreJob(item, zeebe, backupID)
			create, err := kubeClient.BatchV1().Jobs(namespace).Create(ctx, restoreJob, metav1.CreateOptions{FieldManager: "c8-backup"})
			if err != nil {
				return err
			}
			fmt.Println("Created restore job", create.Name)
		}
	}

	// "WATCH" jobs and delete them once they finish recursively
	time.Sleep(time.Second)
	// Create delete job per PVC
	jobs, err := kubeClient.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "job=restore-zeebe",
	})
	if err != nil {
		return err
	}

	runningJobs := 0
	for _, item := range jobs.Items {
		if item.Status.CompletionTime != nil {
			err := kubeClient.BatchV1().Jobs(namespace).Delete(ctx, item.Name, metav1.DeleteOptions{})
			if err != nil {
				fmt.Println("unable to delete job", err)
			}
		} else {
			runningJobs += 1
		}
	}
	if runningJobs > 0 {
		time.Sleep(time.Second)
		return restoreZeebe(ctx, kubeClient, namespace, backupID, true)
	} else {
		return nil
	}

}

func deleteZeebeData(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string, alreadyStarted bool) error {
	var err error
	// Get Zeebe PVCS
	if !alreadyStarted {

		pvcs, err = kubeClient.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{LabelSelector: "app.kubernetes.io/app=zeebe"})
		if err != nil {
			return err
		}

		for _, pvc := range pvcs.Items {
			pvcName := pvc.Name
			job := NewDeletionJob(pvcName, namespace)
			create, err := kubeClient.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{FieldManager: "c8-backup"})
			if err != nil {
				return err
			}
			fmt.Println("Created delete job", create.Name)
		}
	}

	// "WATCH" jobs and delete them once they finish recursively
	time.Sleep(time.Second)
	// Create delete job per PVC
	jobs, err := kubeClient.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "job=delete-zeebe",
	})
	if err != nil {
		return err
	}

	runningJobs := 0
	for _, item := range jobs.Items {
		if item.Status.CompletionTime != nil {
			err := kubeClient.BatchV1().Jobs(namespace).Delete(ctx, item.Name, metav1.DeleteOptions{})
			if err != nil {
				fmt.Println("unable to delete job", err)
			}
		} else {
			runningJobs += 1
		}
	}
	if runningJobs > 0 {
		time.Sleep(time.Second)
		return deleteZeebeData(ctx, kubeClient, namespace, true)
	} else {
		return nil
	}
}

type BackupGetter interface {
	GetBackup(context.Context, int64) (*webapps.BackupResponse, error)
}

func gatherSnapshotNames(ctx context.Context, backupID int64, elasticClient *elastic.Client, clients []BackupGetter) []string {
	var snapshotNames []string
	for _, client := range clients {
		backupResp, err := client.GetBackup(ctx, backupID)
		if err != nil {
			return nil
		}

		for _, detail := range backupResp.Details {
			snapshotNames = append(snapshotNames, detail.SnapshotName)
		}
	}

	backupResp, err := elasticClient.GetBackup(ctx, backupID)
	if err != nil {
		return nil
	}
	for _, backup := range backupResp.Snapshots {
		snapshotNames = append(snapshotNames, backup.Snapshot)
	}
	return snapshotNames
}

func getRelatedApps(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string) (*apps.DeploymentList, *apps.StatefulSetList) {
	var err error
	deployments, err = kubeClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/app=operate",
	})
	if err != nil {
		log.Println(err)
		return nil, nil
	}
	statefulsets, err = kubeClient.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("metadata.name", "zeebe").String(),
	})
	if err != nil {
		log.Println(err)
		return nil, nil
	}

	return deployments, statefulsets
}

func shutdownApps(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string) error {
	for _, deployment := range deployments.Items {
		fmt.Println(deployment.Name)
		scaleConfig := autov1.Scale()
		scaleConfig.Spec = autov1.ScaleSpec()
		scaleConfig.Spec.WithReplicas(0)
		scale, err := kubeClient.AppsV1().Deployments(namespace).ApplyScale(ctx, deployment.Name, scaleConfig, metav1.ApplyOptions{
			FieldManager: "c8-backup",
			Force:        true,
		})
		if err != nil {
			fmt.Println(err)
			return err
		}
		fmt.Println("SCALED DEPLOYMENT", scale.String())
	}

	for _, sts := range statefulsets.Items {
		scaleConfig := autov1.Scale()
		scaleConfig.Spec = autov1.ScaleSpec()
		scaleConfig.Spec.WithReplicas(0)
		scale, err := kubeClient.AppsV1().StatefulSets(namespace).ApplyScale(ctx, sts.Name, scaleConfig, metav1.ApplyOptions{
			FieldManager: "c8-backup",
			Force:        true,
		})
		if err != nil {
			fmt.Println("ERR STS", err)
			return err
		}
		fmt.Println("STS SCALE", scale.String())
	}

	return nil
}

func resetApps(ctx context.Context, kubeClient *kubernetes.Clientset) []error {
	var errors []error
	for _, deployment := range deployments.Items {
		scaleConfig := autov1.Scale()
		scaleConfig.Spec = autov1.ScaleSpec()
		scaleConfig.Spec.WithReplicas(*deployment.Spec.Replicas)
		scale, err := kubeClient.AppsV1().Deployments(deployment.Namespace).ApplyScale(ctx, deployment.Name, scaleConfig, metav1.ApplyOptions{
			FieldManager: "c8-backup",
			Force:        true,
		})
		if err != nil {
			errors = append(errors, err)
		}
		fmt.Println("SCALED DEPLOYMENT", scale.String())
	}

	for _, sts := range statefulsets.Items {
		scaleConfig := autov1.Scale()
		scaleConfig.Spec = autov1.ScaleSpec()
		scaleConfig.Spec.WithReplicas(*sts.Spec.Replicas)
		scale, err := kubeClient.AppsV1().StatefulSets(sts.Namespace).ApplyScale(ctx, sts.Name, scaleConfig, metav1.ApplyOptions{
			FieldManager: "c8-backup",
			Force:        true,
		})
		if err != nil {
			fmt.Println("ERR STS", err)
			errors = append(errors, err)
		}
		fmt.Println("STS SCALE", scale.String())
	}

	return errors
}
