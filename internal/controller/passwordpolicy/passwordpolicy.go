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

package passwordpolicy

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
	errNotPasswordPolicy = "managed resource is not a PasswordPolicy"
	errTrackPCUsage      = "cannot track ProviderConfig usage"
	errGetPC             = "cannot get ProviderConfig"
	errGetCreds          = "cannot get credentials"
	errNewClient         = "cannot create DIP client"
)

// Setup adds a controller that reconciles PasswordPolicy managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(iamv1alpha1.PasswordPolicyGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(iamv1alpha1.PasswordPolicyGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&iamv1alpha1.PasswordPolicy{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*iamv1alpha1.PasswordPolicy)
	if !ok {
		return nil, errors.New(errNotPasswordPolicy)
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
	cr, ok := mg.(*iamv1alpha1.PasswordPolicy)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotPasswordPolicy)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	policy, _, err := e.client.IAM.PasswordPolicies.GetPasswordPolicyByID(externalName)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get password policy")
	}
	if policy == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider.ID = &policy.ID

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: e.isUpToDate(cr, policy),
	}, nil
}

func (e *external) isUpToDate(cr *iamv1alpha1.PasswordPolicy, policy *iam.PasswordPolicy) bool {
	fp := cr.Spec.ForProvider

	if fp.MinLength != nil && *fp.MinLength != policy.Complexity.MinLength {
		return false
	}
	if fp.MaxLength != nil && *fp.MaxLength != policy.Complexity.MaxLength {
		return false
	}
	if fp.MinLowercase != nil && *fp.MinLowercase != policy.Complexity.MinLowerCase {
		return false
	}
	if fp.MinUppercase != nil && *fp.MinUppercase != policy.Complexity.MinUpperCase {
		return false
	}
	if fp.MinNumeric != nil && *fp.MinNumeric != policy.Complexity.MinNumerics {
		return false
	}
	if fp.MinSpecialChars != nil && *fp.MinSpecialChars != policy.Complexity.MinSpecialChars {
		return false
	}
	return true
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*iamv1alpha1.PasswordPolicy)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotPasswordPolicy)
	}

	cr.Status.SetConditions(xpv1.Creating())

	fp := cr.Spec.ForProvider

	policy := iam.PasswordPolicy{}

	if fp.ManagingOrganizationID != nil {
		policy.ManagingOrganization = *fp.ManagingOrganizationID
	}
	if fp.ExpiryPeriodInDays != nil {
		policy.ExpiryPeriodInDays = *fp.ExpiryPeriodInDays
	}
	if fp.HistoryCount != nil {
		policy.HistoryCount = *fp.HistoryCount
	}
	if fp.MinLength != nil {
		policy.Complexity.MinLength = *fp.MinLength
	}
	if fp.MaxLength != nil {
		policy.Complexity.MaxLength = *fp.MaxLength
	}
	if fp.MinLowercase != nil {
		policy.Complexity.MinLowerCase = *fp.MinLowercase
	}
	if fp.MinUppercase != nil {
		policy.Complexity.MinUpperCase = *fp.MinUppercase
	}
	if fp.MinNumeric != nil {
		policy.Complexity.MinNumerics = *fp.MinNumeric
	}
	if fp.MinSpecialChars != nil {
		policy.Complexity.MinSpecialChars = *fp.MinSpecialChars
	}
	if fp.ChallengesEnabled != nil {
		policy.ChallengesEnabled = *fp.ChallengesEnabled
	}
	if fp.ChallengePolicy != nil {
		policy.ChallengePolicy = &iam.ChallengePolicy{}
		if fp.ChallengePolicy.DefaultQuestions != nil {
			policy.ChallengePolicy.DefaultQuestions = fp.ChallengePolicy.DefaultQuestions
		}
		if fp.ChallengePolicy.MaxIncorrectAttempts != nil {
			policy.ChallengePolicy.MaxIncorrectAttempts = *fp.ChallengePolicy.MaxIncorrectAttempts
		}
	}

	created, _, err := e.client.IAM.PasswordPolicies.CreatePasswordPolicy(policy)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create password policy")
	}

	meta.SetExternalName(cr, created.ID)

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*iamv1alpha1.PasswordPolicy)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotPasswordPolicy)
	}

	fp := cr.Spec.ForProvider

	policy := iam.PasswordPolicy{
		ID: meta.GetExternalName(cr),
	}

	if fp.ManagingOrganizationID != nil {
		policy.ManagingOrganization = *fp.ManagingOrganizationID
	}
	if fp.ExpiryPeriodInDays != nil {
		policy.ExpiryPeriodInDays = *fp.ExpiryPeriodInDays
	}
	if fp.HistoryCount != nil {
		policy.HistoryCount = *fp.HistoryCount
	}
	if fp.MinLength != nil {
		policy.Complexity.MinLength = *fp.MinLength
	}
	if fp.MaxLength != nil {
		policy.Complexity.MaxLength = *fp.MaxLength
	}
	if fp.MinLowercase != nil {
		policy.Complexity.MinLowerCase = *fp.MinLowercase
	}
	if fp.MinUppercase != nil {
		policy.Complexity.MinUpperCase = *fp.MinUppercase
	}
	if fp.MinNumeric != nil {
		policy.Complexity.MinNumerics = *fp.MinNumeric
	}
	if fp.MinSpecialChars != nil {
		policy.Complexity.MinSpecialChars = *fp.MinSpecialChars
	}
	if fp.ChallengesEnabled != nil {
		policy.ChallengesEnabled = *fp.ChallengesEnabled
	}
	if fp.ChallengePolicy != nil {
		policy.ChallengePolicy = &iam.ChallengePolicy{}
		if fp.ChallengePolicy.DefaultQuestions != nil {
			policy.ChallengePolicy.DefaultQuestions = fp.ChallengePolicy.DefaultQuestions
		}
		if fp.ChallengePolicy.MaxIncorrectAttempts != nil {
			policy.ChallengePolicy.MaxIncorrectAttempts = *fp.ChallengePolicy.MaxIncorrectAttempts
		}
	}

	_, _, err := e.client.IAM.PasswordPolicies.UpdatePasswordPolicy(policy)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update password policy")
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*iamv1alpha1.PasswordPolicy)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotPasswordPolicy)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	_, _, err := e.client.IAM.PasswordPolicies.DeletePasswordPolicy(iam.PasswordPolicy{ID: externalName})
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete password policy")
	}

	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(ctx context.Context) error {
	return nil
}
