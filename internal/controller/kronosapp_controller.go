/*
Copyright 2024 IsmailAbdelkefi.

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

package kronosapp

import (
	"context"
	// "fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.infra.wecraft.tn/wecraft/automation/ifra/kronos/api/v1alpha1"
)

// KronosAppReconciler reconciles a KronosApp object
type KronosAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core.wecraft.tn,resources=kronosapps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.wecraft.tn,resources=kronosapps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.wecraft.tn,resources=kronosapps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KronosApp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *KronosAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.Log
	kronosApp, err := r.getKronosApp(ctx, req)
	if err != nil {
		l.Error(err, "Unable to fetch KronosApp")
		if apierrors.IsNotFound(err) {
			scheduleInfo.Delete(prometheus.Labels{
				"name":      req.Name,
				"namespace": req.Namespace,
			})
		}
		return ctrl.Result{}, err
	}
	scheduleInfo.With(prometheus.Labels{
		"name":      req.Name,
		"namespace": req.Namespace,
	}).Set(0)
	secretName := getSecretName(req.Name)
	secret, err := r.getSecret(ctx, secretName, req.Namespace)
	if err != nil {
		err := checkIfSecretWasCreatedPreviously(kronosApp, req.Name)
		if err != nil {
			l.Error(err, "Fetching Secret Records")
		}
		err = r.createSecret(ctx, secretName, req.Namespace)
		if err != nil {
			l.Error(err, "Creating Secret")
		}
		l.Info("secret created", "secret name", secretName, "namespace", req.Namespace)
		err = r.registerSecret(ctx, secretName, kronosApp)
		if err != nil {
			l.Error(err, "Updating Created Secrets")
		}
		return ctrl.Result{
			Requeue: true,
		}, nil
	}
	l.Info("secret found", "secret", secret.Name)
	schedule, err := NewSleepSchedule(kronosApp.Spec.StartSleep, kronosApp.Spec.EndSleep, kronosApp.Spec.WeekDays, kronosApp.Spec.TimeZone, kronosApp.Spec.Holidays)
	if err != nil {
		l.Error(err, "Creating Schedule")
		return ctrl.Result{}, err
	}
	ok, additionRequeueDuration, err := IsTimeToSleep(*schedule, kronosApp)
	if err != nil {
		return ctrl.Result{}, err
	}
	var requeueTime time.Duration
	if additionRequeueDuration != 0 {
		requeueTime = additionRequeueDuration
		l.Info("System is in Holiday", "requeue time", formatDuration(requeueTime))
	} else {
		requeueTime = getRequeueTime(*schedule)
		l.Info("Getting Requeue Time", "requeue time", formatDuration(requeueTime))
	}

	l.Info("isTimeToSleep", "execute", ok, "error", err)
	if ok {
		inclusive, err := ValidateIncludedObjects(kronosApp.Spec.IncludedObjects)
		if err != nil {
			l.Error(err, "Validating Included Objects")
			return ctrl.Result{}, err
		}
		includedObjects, err := FetchIncludedObjects(ctx, r.Client, kronosApp.Spec.IncludedObjects, inclusive)
		if err != nil {
			l.Error(err, "Fetching Included Objects")
			return ctrl.Result{}, err
		}
		l.Info("Fetching Included Resources", "Total Resources", includedObjects.GetObjectsTotalCount(), "Included Resources", includedObjects.GetObjectsCount())
		failedObjects, err := putIncludedObjectsToSleep(ctx, r.Client, secret, includedObjects)
		if err != nil {
			l.Error(err, "Putting Included Objects To Sleep")
			return ctrl.Result{}, err
		}
		if len(failedObjects) != 0 {
			logFailedObjects(failedObjects, l)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{
			RequeueAfter: requeueTime,
		}, nil
	} else {
		err := CheckIfSecretContainsData(secret)
		if err != nil {
			l.Error(err, "Restoring Replicas")
		} else {
			err = WakeUpResources(ctx, r.Client, secret)
			if err != nil {
				l.Error(err, "Waking Up Resources")
				return ctrl.Result{}, err
			}
			err = purgeSecretData(ctx, r.Client, secret)
			if err != nil {
				l.Error(err, "Purging Secret's Data")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{
			RequeueAfter: requeueTime,
		}, nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *KronosAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.KronosApp{}).
		Complete(r)
}

func (r *KronosAppReconciler) getKronosApp(ctx context.Context, req ctrl.Request) (*v1alpha1.KronosApp, error) {
	kronosApp := &v1alpha1.KronosApp{}
	err := r.Get(ctx, req.NamespacedName, kronosApp)
	if err != nil {
		return nil, err
	}
	return kronosApp, nil
}

func logFailedObjects(failedObjects map[string][]string, l logr.Logger) {
	l.Info("Logging Failed Sleep", "deployments", failedObjects["Deployment"], "StatefulSets", failedObjects["StatefulSets"], "CronJobs", failedObjects["CronJobs"])
}
