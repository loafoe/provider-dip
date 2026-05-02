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

package emailtemplate

import (
	"context"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/philips-software/go-dip-api/iam"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	iamv1alpha1 "github.com/crossplane/provider-template/apis/iam/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-template/apis/v1alpha1"
	"github.com/crossplane/provider-template/internal/clients/dip"
)

const (
	errNotEmailTemplate = "managed resource is not an EmailTemplate"
	errTrackPCUsage     = "cannot track ProviderConfig usage"
	errGetPC            = "cannot get ProviderConfig"
	errGetCreds         = "cannot get credentials"
	errNewClient        = "cannot create DIP client"
)

// Setup adds a controller that reconciles EmailTemplate managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(iamv1alpha1.EmailTemplateGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(iamv1alpha1.EmailTemplateGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&iamv1alpha1.EmailTemplate{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*iamv1alpha1.EmailTemplate)
	if !ok {
		return nil, errors.New(errNotEmailTemplate)
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
	cr, ok := mg.(*iamv1alpha1.EmailTemplate)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotEmailTemplate)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	template, _, err := e.client.IAM.EmailTemplates.GetTemplateByID(externalName)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get email template")
	}
	if template == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider.ID = &template.ID

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: e.isUpToDate(cr, template),
	}, nil
}

func (e *external) isUpToDate(cr *iamv1alpha1.EmailTemplate, template *iam.EmailTemplate) bool {
	fp := cr.Spec.ForProvider

	if fp.Type != template.Type {
		return false
	}
	if fp.Subject != nil && *fp.Subject != template.Subject {
		return false
	}
	if fp.Message != nil && *fp.Message != template.Message {
		return false
	}
	return true
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*iamv1alpha1.EmailTemplate)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotEmailTemplate)
	}

	cr.Status.SetConditions(xpv1.Creating())

	fp := cr.Spec.ForProvider

	template := iam.EmailTemplate{
		Type: fp.Type,
	}

	if fp.ManagingOrganizationID != nil {
		template.ManagingOrganization = *fp.ManagingOrganizationID
	}
	if fp.Format != nil {
		template.Format = *fp.Format
	}
	if fp.Locale != nil {
		template.Locale = *fp.Locale
	}
	if fp.Subject != nil {
		template.Subject = *fp.Subject
	}
	if fp.From != nil {
		template.From = *fp.From
	}
	if fp.Message != nil {
		template.Message = *fp.Message
	}
	if fp.Link != nil {
		template.Link = *fp.Link
	}

	created, _, err := e.client.IAM.EmailTemplates.CreateTemplate(template)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create email template")
	}

	meta.SetExternalName(cr, created.ID)

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*iamv1alpha1.EmailTemplate)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotEmailTemplate)
	}

	// Note: The DIP API doesn't support updating email templates
	// This is a limitation of the IAM API
	_ = cr

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*iamv1alpha1.EmailTemplate)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotEmailTemplate)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	_, _, err := e.client.IAM.EmailTemplates.DeleteTemplate(iam.EmailTemplate{ID: externalName})
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete email template")
	}

	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(ctx context.Context) error {
	return nil
}
