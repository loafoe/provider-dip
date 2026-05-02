/*
Copyright 2025 The Crossplane Authors.

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

package devicegroup

import (
	"context"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/philips-software/go-dip-api/connect/mdm"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mdmv1alpha1 "github.com/crossplane/provider-template/apis/mdm/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-template/apis/v1alpha1"
	"github.com/crossplane/provider-template/internal/clients/dip"
	"github.com/crossplane/provider-template/internal/util"
)

const (
	errNotDeviceGroup = "managed resource is not an MDM DeviceGroup"
	errTrackPCUsage   = "cannot track ProviderConfig usage"
	errGetPC          = "cannot get ProviderConfig"
	errGetCreds       = "cannot get credentials"
	errNewClient      = "cannot create DIP client"
)

// Setup adds a controller that reconciles MDM DeviceGroup managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(mdmv1alpha1.DeviceGroupGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(mdmv1alpha1.DeviceGroupGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&mdmv1alpha1.DeviceGroup{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*mdmv1alpha1.DeviceGroup)
	if !ok {
		return nil, errors.New(errNotDeviceGroup)
	}

	if err := c.usage.Track(ctx, cr); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	m := mg.(resource.ModernManaged)
	ref := m.GetProviderConfigReference()

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: m.GetNamespace()}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	secretData, err := resource.CommonCredentialExtractor(ctx, pc.Spec.Credentials.Source, c.kube,
		xpv1.CommonCredentialSelectors{SecretRef: pc.Spec.Credentials.SecretRef})
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	cfg, err := dip.ConfigFromSecret(pc.Spec.Region, pc.Spec.Environment, secretData)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	dipClient, err := dip.NewClient(cfg)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{client: dipClient}, nil
}

type external struct {
	client *dip.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*mdmv1alpha1.DeviceGroup)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotDeviceGroup)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	if !util.IsValidUUID(externalName) {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	dg, resp, err := e.client.MDM.DeviceGroups.GetByID(externalName)
	if err != nil {
		if resp != nil && util.IsNotFoundOrInvalidID(resp.StatusCode()) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get MDM device group")
	}
	if dg == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider.ID = &dg.ID

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: e.isUpToDate(cr, dg),
	}, nil
}

func (e *external) isUpToDate(cr *mdmv1alpha1.DeviceGroup, dg *mdm.DeviceGroup) bool {
	fp := cr.Spec.ForProvider

	if fp.Name != dg.Name {
		return false
	}
	if fp.Description != nil && *fp.Description != dg.Description {
		return false
	}
	return true
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*mdmv1alpha1.DeviceGroup)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotDeviceGroup)
	}

	cr.Status.SetConditions(xpv1.Creating())

	fp := cr.Spec.ForProvider

	dg := mdm.DeviceGroup{
		ResourceType: "DeviceGroup",
		Name:         fp.Name,
		ApplicationId: mdm.Reference{
			Reference: fp.ApplicationID,
		},
	}

	if fp.Description != nil {
		dg.Description = *fp.Description
	}
	if fp.DefaultGroupGUID != nil {
		dg.DefaultGroupGuid = &mdm.Identifier{Value: *fp.DefaultGroupGUID}
	}

	created, _, err := e.client.MDM.DeviceGroups.Create(dg)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create MDM device group")
	}

	meta.SetExternalName(cr, created.ID)

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*mdmv1alpha1.DeviceGroup)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotDeviceGroup)
	}

	fp := cr.Spec.ForProvider

	dg := mdm.DeviceGroup{
		ResourceType: "DeviceGroup",
		ID:           meta.GetExternalName(cr),
		Name:         fp.Name,
		ApplicationId: mdm.Reference{
			Reference: fp.ApplicationID,
		},
	}

	if fp.Description != nil {
		dg.Description = *fp.Description
	}
	if fp.DefaultGroupGUID != nil {
		dg.DefaultGroupGuid = &mdm.Identifier{Value: *fp.DefaultGroupGUID}
	}

	_, _, err := e.client.MDM.DeviceGroups.Update(dg)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update MDM device group")
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*mdmv1alpha1.DeviceGroup)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotDeviceGroup)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	_, _, err := e.client.MDM.DeviceGroups.Delete(mdm.DeviceGroup{ID: externalName})
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete MDM device group")
	}

	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(ctx context.Context) error {
	return nil
}
