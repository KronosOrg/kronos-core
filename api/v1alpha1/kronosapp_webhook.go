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

package v1alpha1

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var kronosapplog = logf.Log.WithName("kronosapp-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *KronosApp) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}


//+kubebuilder:webhook:path=/mutate-core-wecraft-tn-v1alpha1-kronosapp,mutating=true,failurePolicy=fail,sideEffects=None,groups=core.wecraft.tn,resources=kronosapps,verbs=create;update,versions=v1alpha1,name=mkronosapp.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &KronosApp{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *KronosApp) Default() {
	kronosapplog.Info("default", "name", r.Name)
	if r.Spec.WeekDays == "" {
		r.Spec.WeekDays = "*"
	}
	for _, includedObject := range r.Spec.IncludedObjects {
		if includedObject.ApiVersion == "" {
			includedObject.ApiVersion = "*"
		}
		if includedObject.Kind == "" {
			includedObject.Kind = "*"
		}
		if includedObject.Namespace == "" {
			includedObject.Namespace = "default"
		}
		if includedObject.IncludeRef == "" {
			includedObject.IncludeRef = ".*"
		}
		if includedObject.ExcludeRef == "" {
			includedObject.ExcludeRef = "^$"
		}
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-core-wecraft-tn-v1alpha1-kronosapp,mutating=false,failurePolicy=fail,sideEffects=None,groups=core.wecraft.tn,resources=kronosapps,verbs=create;update,versions=v1alpha1,name=vkronosapp.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &KronosApp{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *KronosApp) ValidateCreate() (admission.Warnings, error) {
	kronosapplog.Info("validate create", "name", r.Name)
	err := r.validateKronosApp()
	if err != nil {
		return []string{err.Error()}, err
	}
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *KronosApp) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	kronosapplog.Info("validate update", "name", r.Name)
	err := r.validateKronosApp()
	if err != nil {
		return []string{err.Error()}, err
	}
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *KronosApp) ValidateDelete() (admission.Warnings, error) {
	kronosapplog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}

func (r *KronosApp) validateScheduleStartTime() error {
	_, err := time.Parse("15:04", r.Spec.StartSleep)
	if err != nil {
		return errors.New("Start sleep time is invalid.")
	}
	return nil
}

func (r *KronosApp) validateScheduleEndTime() error {
	_, err := time.Parse("15:04", r.Spec.EndSleep)
	if err != nil {
		return errors.New("End sleep time is invalid.")
	}
	return nil
}

func (r *KronosApp) validateScheduleTimezone() error {
	_, err := time.LoadLocation(r.Spec.TimeZone)
	if err != nil {
		return errors.New("Timezone is invalid.")
	}
	return nil
}

func (r *KronosApp) validateScheduleWeekdays() error {
	pattern := `^([1-7])([-,]([1-7]))*$`
	reg := regexp.MustCompile(pattern)
	if !reg.MatchString(r.Spec.WeekDays) {
		return errors.New("Weekdays are not properly formatted.")
	}
	return nil
}

func (r *KronosApp) validateScheduleHolidays() error {
	pattern := `^\d{4}-\d{2}-\d{2}(\/\d{2})*$`
	reg := regexp.MustCompile(pattern)

	if len(r.Spec.Holidays) != 0 {
		for _, holiday := range r.Spec.Holidays {
			if !reg.MatchString(holiday.Date) {
				return errors.New(fmt.Sprintf("Date of Holiday: %s is not properly formatted.", holiday.Name))
			}
		}
	}
	return nil
}

func (r *KronosApp) validateKronosApp() error {
	err := r.validateScheduleStartTime()
	if err != nil {
		return err
	}
	err = r.validateScheduleEndTime()
	if err != nil {
		return err
	}
	err = r.validateScheduleTimezone()
	if err != nil {
		return err
	}
	err = r.validateScheduleWeekdays()
	if err != nil {
		return err
	}
	err = r.validateScheduleHolidays()
	if err != nil {
		return err
	}
	return nil
}