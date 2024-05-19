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

	"github.com/KronosOrg/kronos-core/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// KronosAppReconciler reconciles a KronosApp object
type KronosAppReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Metrics Metrics
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
			r.Metrics.ScheduleInfo.Delete(prometheus.Labels{
				"name":      req.Name,
				"namespace": req.Namespace,
			})
		}
		return ctrl.Result{}, err
	}
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
	isHoliday, ok, additionRequeueDuration, err := IsTimeToSleep(*schedule, kronosApp)
	if err != nil {
		return ctrl.Result{}, err
	}
	var requeueTime time.Duration
	if isHoliday {
		requeueTime = additionRequeueDuration
		l.Info("System is in Holiday", "requeue time", formatDuration(requeueTime))
	} else {
		requeueTime = getRequeueTime(*schedule)
		l.Info("Getting Requeue Time", "requeue time", formatDuration(requeueTime))
	}

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
	currentStatus := kronosApp.Status
	newStatus := kronosApp.GetNewKronosAppStatus(ok, isHoliday, schedule.now.Add(requeueTime), includedObjects.GetObjectsTotalCount())
	err = kronosApp.SetNewKronosAppStatus(ctx, r.Client, newStatus)
	if err != nil {
		l.Error(err, "Updating KronosApp Status")
		return ctrl.Result{}, err
	}
	r.deleteOldMetrics(req, currentStatus)
	r.exportAdditionalMetrics(req, newStatus, ok)
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
		if len(failedObjects) != 0 {
			logFailedObjects(failedObjects, l)
			return ctrl.Result{}, nil
		}
		if err != nil {
			l.Error(err, "Putting Included Objects To Sleep")
			return ctrl.Result{}, err
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
	pred := predicate.GenerationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.KronosApp{}).
		WithEventFilter(pred).
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

func (r *KronosAppReconciler) exportAdditionalMetrics(req ctrl.Request, newStatus v1alpha1.KronosAppStatus, isTimeToSleep bool) {
	var value float64
	if isTimeToSleep {
		value = 0
	} else {
		value = 1
	}
	r.Metrics.InDepthScheduleInfo.With(prometheus.Labels{
		"name":              req.Name,
		"namespace":         req.Namespace,
		"status":            newStatus.Status,
		"reason":            newStatus.Reason,
		"handled_resources": newStatus.HandledResources,
		"next_operation":    newStatus.NextOperation,
	}).Set(value)
	r.Metrics.ScheduleInfo.With(prometheus.Labels{
		"name":      req.Name,
		"namespace": req.Namespace,
	}).Set(value)
}

func (r *KronosAppReconciler) deleteOldMetrics(req ctrl.Request, oldStatus v1alpha1.KronosAppStatus) {
	r.Metrics.InDepthScheduleInfo.Delete(prometheus.Labels{
		"name":              req.Name,
		"namespace":         req.Namespace,
		"status":            oldStatus.Status,
		"reason":            oldStatus.Reason,
		"handled_resources": oldStatus.HandledResources,
		"next_operation":    oldStatus.NextOperation,
	})
}

func logFailedObjects(failedObjects map[string][]string, l logr.Logger) {
	l.Info("Logging Failed Sleep", "deployments", failedObjects["Deployment"], "StatefulSets", failedObjects["StatefulSets"], "CronJobs", failedObjects["CronJobs"])
}
