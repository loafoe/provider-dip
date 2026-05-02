# Native DIP Provider Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a native Crossplane 2.0 provider for DIP IAM resources using go-dip-api directly.

**Architecture:** Single client wrapper package wraps go-dip-api. Each IAM resource has its own type definition and controller. ProviderConfig handles service identity authentication with secret override support.

**Tech Stack:** Go 1.24+, Crossplane Runtime v2, go-dip-api, controller-runtime

---

## File Structure

```
apis/
  v1alpha1/
    types.go              # ProviderConfig (modify existing)
    register.go           # Update Group constant (modify existing)
  iam/
    v1alpha1/
      doc.go              # Package doc
      groupversion_info.go # Group/version registration
      organization_types.go
      group_types.go
      role_types.go
      application_types.go
      client_types.go
      service_types.go
      emailtemplate_types.go
      passwordpolicy_types.go
      proposition_types.go
      user_types.go
      zz_generated.deepcopy.go  # Generated
      zz_generated.managed.go   # Generated
      zz_generated.managedlist.go # Generated
    iam.go                # Group registration
  dip.go                  # Top-level scheme registration (modify existing)
internal/
  clients/
    dip/
      client.go           # DIP client wrapper
  controller/
    config/config.go      # ProviderConfig controller (modify existing)
    organization/organization.go
    group/group.go
    role/role.go
    application/application.go
    client/client.go
    service/service.go
    emailtemplate/emailtemplate.go
    passwordpolicy/passwordpolicy.go
    proposition/proposition.go
    user/user.go
    register.go           # Controller registration (modify existing)
go.mod                    # Add go-dip-api dependency
```

---

## Task 1: Update go.mod and API Group Constants

**Files:**
- Modify: `go.mod`
- Modify: `apis/v1alpha1/register.go`

- [ ] **Step 1: Add go-dip-api dependency**

```bash
go get github.com/philips-software/go-dip-api@latest
```

- [ ] **Step 2: Update ProviderConfig API group**

In `apis/v1alpha1/register.go`, change:

```go
const (
	Group   = "dip.m.crossplane.io"
	Version = "v1alpha1"
)
```

- [ ] **Step 3: Run go mod tidy**

```bash
go mod tidy
```

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum apis/v1alpha1/register.go
git commit -m "feat: add go-dip-api dependency and update API group"
```

---

## Task 2: Update ProviderConfig Types

**Files:**
- Modify: `apis/v1alpha1/types.go`

- [ ] **Step 1: Update ProviderConfigSpec with region/environment**

Replace the contents of `apis/v1alpha1/types.go`:

```go
package v1alpha1

import (
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProviderConfigStatus defines the status of a Provider.
type ProviderConfigStatus struct {
	xpv1.ProviderConfigStatus `json:",inline"`
}

// ProviderCredentials required to authenticate.
type ProviderCredentials struct {
	// Source of the provider credentials.
	// +kubebuilder:validation:Enum=Secret
	// +kubebuilder:default=Secret
	Source xpv1.CredentialsSource `json:"source"`

	// SecretRef references a secret containing service_id and service_private_key.
	// The secret may also contain region and environment to override spec values.
	// +optional
	SecretRef *xpv1.SecretKeySelector `json:"secretRef,omitempty"`
}

// ProviderConfigSpec defines the configuration for connecting to DIP.
type ProviderConfigSpec struct {
	// Region is the DIP IAM region. Can be overridden by secret.
	// +optional
	Region string `json:"region,omitempty"`

	// Environment is the DIP IAM environment. Can be overridden by secret.
	// +optional
	Environment string `json:"environment,omitempty"`

	// Credentials for service identity authentication.
	// +kubebuilder:validation:Required
	Credentials ProviderCredentials `json:"credentials"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="REGION",type="string",JSONPath=".spec.region"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,provider,dip}

// ProviderConfig configures the DIP provider.
type ProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProviderConfigSpec   `json:"spec"`
	Status ProviderConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProviderConfigList contains a list of ProviderConfig.
type ProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfig `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="CONFIG-NAME",type="string",JSONPath=".providerConfigRef.name"
// +kubebuilder:printcolumn:name="RESOURCE-KIND",type="string",JSONPath=".resourceRef.kind"
// +kubebuilder:printcolumn:name="RESOURCE-NAME",type="string",JSONPath=".resourceRef.name"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,provider,dip}

// ProviderConfigUsage indicates that a resource is using a ProviderConfig.
type ProviderConfigUsage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	xpv2.TypedProviderConfigUsage `json:",inline"`
}

// +kubebuilder:object:root=true

// ProviderConfigUsageList contains a list of ProviderConfigUsage.
type ProviderConfigUsageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfigUsage `json:"items"`
}
```

- [ ] **Step 2: Update register.go to remove ClusterProviderConfig**

Update `apis/v1alpha1/register.go` init function:

```go
func init() {
	SchemeBuilder.Register(&ProviderConfig{}, &ProviderConfigList{})
	SchemeBuilder.Register(&ProviderConfigUsage{}, &ProviderConfigUsageList{})
}
```

Also remove the ClusterProviderConfig metadata vars, keeping only:

```go
// ProviderConfig type metadata.
var (
	ProviderConfigKind             = reflect.TypeOf(ProviderConfig{}).Name()
	ProviderConfigGroupKind        = schema.GroupKind{Group: Group, Kind: ProviderConfigKind}.String()
	ProviderConfigGroupVersionKind = SchemeGroupVersion.WithKind(ProviderConfigKind)
)

// ProviderConfigUsage type metadata.
var (
	ProviderConfigUsageKind             = reflect.TypeOf(ProviderConfigUsage{}).Name()
	ProviderConfigUsageGroupVersionKind = SchemeGroupVersion.WithKind(ProviderConfigUsageKind)

	ProviderConfigUsageListKind             = reflect.TypeOf(ProviderConfigUsageList{}).Name()
	ProviderConfigUsageListGroupVersionKind = SchemeGroupVersion.WithKind(ProviderConfigUsageListKind)
)
```

- [ ] **Step 3: Commit**

```bash
git add apis/v1alpha1/
git commit -m "feat: update ProviderConfig with region/environment fields"
```

---

## Task 3: Create DIP Client Wrapper

**Files:**
- Create: `internal/clients/dip/client.go`

- [ ] **Step 1: Create client wrapper**

```go
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

package dip

import (
	"encoding/json"
	"fmt"

	"github.com/philips-software/go-dip-api/iam"
)

// Config holds DIP client configuration.
type Config struct {
	Region            string
	Environment       string
	ServiceID         string
	ServicePrivateKey string
}

// Client wraps go-dip-api IAM client.
type Client struct {
	IAM *iam.Client
}

// NewClient creates a new DIP client from config.
func NewClient(cfg Config) (*Client, error) {
	if cfg.Region == "" {
		return nil, fmt.Errorf("region is required")
	}
	if cfg.Environment == "" {
		return nil, fmt.Errorf("environment is required")
	}
	if cfg.ServiceID == "" {
		return nil, fmt.Errorf("service_id is required")
	}
	if cfg.ServicePrivateKey == "" {
		return nil, fmt.Errorf("service_private_key is required")
	}

	iamClient, err := iam.NewClient(nil, &iam.Config{
		Region:         cfg.Region,
		Environment:    cfg.Environment,
		ServiceID:      cfg.ServiceID,
		ServicePrivateKey: cfg.ServicePrivateKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create IAM client: %w", err)
	}

	return &Client{IAM: iamClient}, nil
}

// ConfigFromSecret parses config from ProviderConfig spec and secret JSON.
// Secret values override spec values for region and environment.
func ConfigFromSecret(specRegion, specEnv string, secretData []byte) (Config, error) {
	var creds map[string]string
	if err := json.Unmarshal(secretData, &creds); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	cfg := Config{
		Region:            specRegion,
		Environment:       specEnv,
		ServiceID:         creds["service_id"],
		ServicePrivateKey: creds["service_private_key"],
	}

	// Secret values override spec values
	if v, ok := creds["region"]; ok && v != "" {
		cfg.Region = v
	}
	if v, ok := creds["environment"]; ok && v != "" {
		cfg.Environment = v
	}

	return cfg, nil
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/clients/dip/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/clients/dip/
git commit -m "feat: add DIP client wrapper"
```

---

## Task 4: Create IAM API Group Structure

**Files:**
- Create: `apis/iam/v1alpha1/doc.go`
- Create: `apis/iam/v1alpha1/groupversion_info.go`
- Create: `apis/iam/iam.go`

- [ ] **Step 1: Create doc.go**

```go
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

// Package v1alpha1 contains the v1alpha1 group IAM resources of the DIP provider.
// +kubebuilder:object:generate=true
// +groupName=iam.dip.m.crossplane.io
// +versionName=v1alpha1
package v1alpha1
```

- [ ] **Step 2: Create groupversion_info.go**

```go
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

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	Group   = "iam.dip.m.crossplane.io"
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)
```

- [ ] **Step 3: Create iam.go**

```go
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

// Package iam contains IAM API versions.
package iam

import (
	"k8s.io/apimachinery/pkg/runtime"

	iamv1alpha1 "github.com/crossplane/provider-template/apis/iam/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, iamv1alpha1.SchemeBuilder.AddToScheme)
}

// AddToSchemes may be used to add all resources defined in the project to a Scheme.
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme.
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
```

- [ ] **Step 4: Commit**

```bash
git add apis/iam/
git commit -m "feat: add IAM API group structure"
```

---

## Task 5: Create Organization Type

**Files:**
- Create: `apis/iam/v1alpha1/organization_types.go`

- [ ] **Step 1: Create organization_types.go**

```go
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

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// OrganizationParameters are the configurable fields of an Organization.
type OrganizationParameters struct {
	// Name of the organization. Immutable after creation.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="name is immutable"
	Name string `json:"name"`

	// DisplayName for the organization.
	// +optional
	DisplayName *string `json:"displayName,omitempty"`

	// Description of the organization.
	// +optional
	Description *string `json:"description,omitempty"`

	// ParentOrgID is the parent organization GUID.
	// +optional
	ParentOrgID *string `json:"parentOrgId,omitempty"`

	// ParentOrgRef references an Organization to populate parentOrgId.
	// +optional
	ParentOrgRef *xpv1.Reference `json:"parentOrgRef,omitempty"`

	// ParentOrgSelector selects an Organization to populate parentOrgId.
	// +optional
	ParentOrgSelector *xpv1.Selector `json:"parentOrgSelector,omitempty"`

	// Type of the organization (e.g., Hospital).
	// +optional
	Type *string `json:"type,omitempty"`

	// ExternalID is a client-defined identifier.
	// +optional
	ExternalID *string `json:"externalId,omitempty"`
}

// OrganizationObservation are the observable fields of an Organization.
type OrganizationObservation struct {
	// ID is the GUID of the organization.
	ID *string `json:"id,omitempty"`

	// Active indicates if the organization is active.
	Active *bool `json:"active,omitempty"`
}

// OrganizationSpec defines the desired state of an Organization.
type OrganizationSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              OrganizationParameters `json:"forProvider"`
}

// OrganizationStatus represents the observed state of an Organization.
type OrganizationStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          OrganizationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// Organization is the Schema for the Organization API.
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,dip}
type Organization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              OrganizationSpec   `json:"spec"`
	Status            OrganizationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OrganizationList contains a list of Organization.
type OrganizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Organization `json:"items"`
}

// Organization type metadata.
var (
	OrganizationKind             = reflect.TypeOf(Organization{}).Name()
	OrganizationGroupKind        = schema.GroupKind{Group: Group, Kind: OrganizationKind}.String()
	OrganizationKindAPIVersion   = OrganizationKind + "." + SchemeGroupVersion.String()
	OrganizationGroupVersionKind = SchemeGroupVersion.WithKind(OrganizationKind)
)

func init() {
	SchemeBuilder.Register(&Organization{}, &OrganizationList{})
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./apis/iam/...
```

- [ ] **Step 3: Commit**

```bash
git add apis/iam/v1alpha1/organization_types.go
git commit -m "feat: add Organization type"
```

---

## Task 6: Create Group Type

**Files:**
- Create: `apis/iam/v1alpha1/group_types.go`

- [ ] **Step 1: Create group_types.go**

```go
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

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// GroupParameters are the configurable fields of a Group.
type GroupParameters struct {
	// Name of the group. Immutable after creation.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="name is immutable"
	Name string `json:"name"`

	// Description of the group.
	// +optional
	Description *string `json:"description,omitempty"`

	// ManagingOrganizationID is the organization GUID that manages this group.
	// +optional
	ManagingOrganizationID *string `json:"managingOrganizationId,omitempty"`

	// ManagingOrganizationRef references an Organization.
	// +optional
	ManagingOrganizationRef *xpv1.Reference `json:"managingOrganizationRef,omitempty"`

	// ManagingOrganizationSelector selects an Organization.
	// +optional
	ManagingOrganizationSelector *xpv1.Selector `json:"managingOrganizationSelector,omitempty"`
}

// GroupObservation are the observable fields of a Group.
type GroupObservation struct {
	// ID is the GUID of the group.
	ID *string `json:"id,omitempty"`
}

// GroupSpec defines the desired state of a Group.
type GroupSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              GroupParameters `json:"forProvider"`
}

// GroupStatus represents the observed state of a Group.
type GroupStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          GroupObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// Group is the Schema for the Group API.
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,dip}
type Group struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GroupSpec   `json:"spec"`
	Status            GroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GroupList contains a list of Group.
type GroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Group `json:"items"`
}

// Group type metadata.
var (
	GroupKind             = reflect.TypeOf(Group{}).Name()
	GroupGroupKind        = schema.GroupKind{Group: Group, Kind: GroupKind}.String()
	GroupKindAPIVersion   = GroupKind + "." + SchemeGroupVersion.String()
	GroupGroupVersionKind = SchemeGroupVersion.WithKind(GroupKind)
)

func init() {
	SchemeBuilder.Register(&Group{}, &GroupList{})
}
```

- [ ] **Step 2: Commit**

```bash
git add apis/iam/v1alpha1/group_types.go
git commit -m "feat: add Group type"
```

---

## Task 7: Create Role Type

**Files:**
- Create: `apis/iam/v1alpha1/role_types.go`

- [ ] **Step 1: Create role_types.go**

```go
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

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// RoleParameters are the configurable fields of a Role.
type RoleParameters struct {
	// Name of the role. Immutable after creation.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="name is immutable"
	Name string `json:"name"`

	// Description of the role.
	// +optional
	Description *string `json:"description,omitempty"`

	// ManagingOrganizationID is the organization GUID that manages this role.
	// +optional
	ManagingOrganizationID *string `json:"managingOrganizationId,omitempty"`

	// ManagingOrganizationRef references an Organization.
	// +optional
	ManagingOrganizationRef *xpv1.Reference `json:"managingOrganizationRef,omitempty"`

	// ManagingOrganizationSelector selects an Organization.
	// +optional
	ManagingOrganizationSelector *xpv1.Selector `json:"managingOrganizationSelector,omitempty"`

	// Permissions is the list of permission names assigned to this role.
	// +optional
	Permissions []string `json:"permissions,omitempty"`
}

// RoleObservation are the observable fields of a Role.
type RoleObservation struct {
	// ID is the GUID of the role.
	ID *string `json:"id,omitempty"`
}

// RoleSpec defines the desired state of a Role.
type RoleSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              RoleParameters `json:"forProvider"`
}

// RoleStatus represents the observed state of a Role.
type RoleStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          RoleObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// Role is the Schema for the Role API.
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,dip}
type Role struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RoleSpec   `json:"spec"`
	Status            RoleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RoleList contains a list of Role.
type RoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Role `json:"items"`
}

// Role type metadata.
var (
	RoleKind             = reflect.TypeOf(Role{}).Name()
	RoleGroupKind        = schema.GroupKind{Group: Group, Kind: RoleKind}.String()
	RoleKindAPIVersion   = RoleKind + "." + SchemeGroupVersion.String()
	RoleGroupVersionKind = SchemeGroupVersion.WithKind(RoleKind)
)

func init() {
	SchemeBuilder.Register(&Role{}, &RoleList{})
}
```

- [ ] **Step 2: Commit**

```bash
git add apis/iam/v1alpha1/role_types.go
git commit -m "feat: add Role type"
```

---

## Task 8: Create Proposition Type

**Files:**
- Create: `apis/iam/v1alpha1/proposition_types.go`

- [ ] **Step 1: Create proposition_types.go**

```go
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

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// PropositionParameters are the configurable fields of a Proposition.
type PropositionParameters struct {
	// Name of the proposition. Immutable after creation.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="name is immutable"
	Name string `json:"name"`

	// Description of the proposition.
	// +optional
	Description *string `json:"description,omitempty"`

	// OrganizationID is the organization GUID this proposition belongs to.
	// +optional
	OrganizationID *string `json:"organizationId,omitempty"`

	// OrganizationRef references an Organization.
	// +optional
	OrganizationRef *xpv1.Reference `json:"organizationRef,omitempty"`

	// OrganizationSelector selects an Organization.
	// +optional
	OrganizationSelector *xpv1.Selector `json:"organizationSelector,omitempty"`

	// GlobalReferenceID is a global unique identifier. Immutable after creation.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="globalReferenceId is immutable"
	GlobalReferenceID string `json:"globalReferenceId"`
}

// PropositionObservation are the observable fields of a Proposition.
type PropositionObservation struct {
	// ID is the GUID of the proposition.
	ID *string `json:"id,omitempty"`
}

// PropositionSpec defines the desired state of a Proposition.
type PropositionSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              PropositionParameters `json:"forProvider"`
}

// PropositionStatus represents the observed state of a Proposition.
type PropositionStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          PropositionObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// Proposition is the Schema for the Proposition API.
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,dip}
type Proposition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PropositionSpec   `json:"spec"`
	Status            PropositionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PropositionList contains a list of Proposition.
type PropositionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Proposition `json:"items"`
}

// Proposition type metadata.
var (
	PropositionKind             = reflect.TypeOf(Proposition{}).Name()
	PropositionGroupKind        = schema.GroupKind{Group: Group, Kind: PropositionKind}.String()
	PropositionKindAPIVersion   = PropositionKind + "." + SchemeGroupVersion.String()
	PropositionGroupVersionKind = SchemeGroupVersion.WithKind(PropositionKind)
)

func init() {
	SchemeBuilder.Register(&Proposition{}, &PropositionList{})
}
```

- [ ] **Step 2: Commit**

```bash
git add apis/iam/v1alpha1/proposition_types.go
git commit -m "feat: add Proposition type"
```

---

## Task 9: Create Application Type

**Files:**
- Create: `apis/iam/v1alpha1/application_types.go`

- [ ] **Step 1: Create application_types.go**

```go
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

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// ApplicationParameters are the configurable fields of an Application.
type ApplicationParameters struct {
	// Name of the application. Immutable after creation.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="name is immutable"
	Name string `json:"name"`

	// Description of the application.
	// +optional
	Description *string `json:"description,omitempty"`

	// PropositionID is the proposition GUID this application belongs to.
	// +optional
	PropositionID *string `json:"propositionId,omitempty"`

	// PropositionRef references a Proposition.
	// +optional
	PropositionRef *xpv1.Reference `json:"propositionRef,omitempty"`

	// PropositionSelector selects a Proposition.
	// +optional
	PropositionSelector *xpv1.Selector `json:"propositionSelector,omitempty"`

	// GlobalReferenceID is a global unique identifier. Immutable after creation.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="globalReferenceId is immutable"
	GlobalReferenceID string `json:"globalReferenceId"`
}

// ApplicationObservation are the observable fields of an Application.
type ApplicationObservation struct {
	// ID is the GUID of the application.
	ID *string `json:"id,omitempty"`
}

// ApplicationSpec defines the desired state of an Application.
type ApplicationSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ApplicationParameters `json:"forProvider"`
}

// ApplicationStatus represents the observed state of an Application.
type ApplicationStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ApplicationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// Application is the Schema for the Application API.
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,dip}
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ApplicationSpec   `json:"spec"`
	Status            ApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationList contains a list of Application.
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

// Application type metadata.
var (
	ApplicationKind             = reflect.TypeOf(Application{}).Name()
	ApplicationGroupKind        = schema.GroupKind{Group: Group, Kind: ApplicationKind}.String()
	ApplicationKindAPIVersion   = ApplicationKind + "." + SchemeGroupVersion.String()
	ApplicationGroupVersionKind = SchemeGroupVersion.WithKind(ApplicationKind)
)

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}
```

- [ ] **Step 2: Commit**

```bash
git add apis/iam/v1alpha1/application_types.go
git commit -m "feat: add Application type"
```

---

## Task 10: Create Client Type

**Files:**
- Create: `apis/iam/v1alpha1/client_types.go`

- [ ] **Step 1: Create client_types.go**

```go
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

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// ClientParameters are the configurable fields of a Client.
type ClientParameters struct {
	// Name of the client. Immutable after creation.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="name is immutable"
	Name string `json:"name"`

	// Description of the client.
	// +optional
	Description *string `json:"description,omitempty"`

	// ApplicationID is the application GUID this client belongs to.
	// +optional
	ApplicationID *string `json:"applicationId,omitempty"`

	// ApplicationRef references an Application.
	// +optional
	ApplicationRef *xpv1.Reference `json:"applicationRef,omitempty"`

	// ApplicationSelector selects an Application.
	// +optional
	ApplicationSelector *xpv1.Selector `json:"applicationSelector,omitempty"`

	// GlobalReferenceID is a global unique identifier. Immutable after creation.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="globalReferenceId is immutable"
	GlobalReferenceID string `json:"globalReferenceId"`

	// RedirectionURIs for OAuth2 redirects.
	// +optional
	RedirectionURIs []string `json:"redirectionURIs,omitempty"`

	// ResponseTypes (e.g., code, token).
	// +optional
	ResponseTypes []string `json:"responseTypes,omitempty"`

	// Scopes available to this client.
	// +optional
	Scopes []string `json:"scopes,omitempty"`

	// DefaultScopes applied when none requested.
	// +optional
	DefaultScopes []string `json:"defaultScopes,omitempty"`

	// ConsentImplied skips user consent screen.
	// +optional
	ConsentImplied *bool `json:"consentImplied,omitempty"`

	// AccessTokenLifetime in seconds.
	// +optional
	AccessTokenLifetime *int64 `json:"accessTokenLifetime,omitempty"`

	// RefreshTokenLifetime in seconds.
	// +optional
	RefreshTokenLifetime *int64 `json:"refreshTokenLifetime,omitempty"`

	// IDTokenLifetime in seconds.
	// +optional
	IDTokenLifetime *int64 `json:"idTokenLifetime,omitempty"`
}

// ClientObservation are the observable fields of a Client.
type ClientObservation struct {
	// ID is the GUID of the client.
	ID *string `json:"id,omitempty"`

	// ClientID is the OAuth2 client_id.
	ClientID *string `json:"clientId,omitempty"`

	// Disabled indicates if the client is disabled.
	Disabled *bool `json:"disabled,omitempty"`
}

// ClientSpec defines the desired state of a Client.
type ClientSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ClientParameters `json:"forProvider"`
}

// ClientStatus represents the observed state of a Client.
type ClientStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ClientObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// Client is the Schema for the Client API.
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,dip}
type Client struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ClientSpec   `json:"spec"`
	Status            ClientStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClientList contains a list of Client.
type ClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Client `json:"items"`
}

// Client type metadata.
var (
	ClientKind             = reflect.TypeOf(Client{}).Name()
	ClientGroupKind        = schema.GroupKind{Group: Group, Kind: ClientKind}.String()
	ClientKindAPIVersion   = ClientKind + "." + SchemeGroupVersion.String()
	ClientGroupVersionKind = SchemeGroupVersion.WithKind(ClientKind)
)

func init() {
	SchemeBuilder.Register(&Client{}, &ClientList{})
}
```

- [ ] **Step 2: Commit**

```bash
git add apis/iam/v1alpha1/client_types.go
git commit -m "feat: add Client type"
```

---

## Task 11: Create Service Type

**Files:**
- Create: `apis/iam/v1alpha1/service_types.go`

- [ ] **Step 1: Create service_types.go**

```go
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

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// ServiceParameters are the configurable fields of a Service.
type ServiceParameters struct {
	// Name of the service. Immutable after creation.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="name is immutable"
	Name string `json:"name"`

	// Description of the service.
	// +optional
	Description *string `json:"description,omitempty"`

	// ApplicationID is the application GUID this service belongs to.
	// +optional
	ApplicationID *string `json:"applicationId,omitempty"`

	// ApplicationRef references an Application.
	// +optional
	ApplicationRef *xpv1.Reference `json:"applicationRef,omitempty"`

	// ApplicationSelector selects an Application.
	// +optional
	ApplicationSelector *xpv1.Selector `json:"applicationSelector,omitempty"`

	// PrivateKeySecretRef optionally references a secret containing a private key.
	// If not provided, a key pair will be generated.
	// +optional
	PrivateKeySecretRef *xpv1.SecretKeySelector `json:"privateKeySecretRef,omitempty"`

	// Scopes available to this service.
	// +optional
	Scopes []string `json:"scopes,omitempty"`

	// DefaultScopes applied when none requested.
	// +optional
	DefaultScopes []string `json:"defaultScopes,omitempty"`

	// AccessTokenLifetime in seconds.
	// +optional
	AccessTokenLifetime *int64 `json:"accessTokenLifetime,omitempty"`

	// RefreshTokenLifetime in seconds.
	// +optional
	RefreshTokenLifetime *int64 `json:"refreshTokenLifetime,omitempty"`

	// TokenEndpointAuthMethod (e.g., private_key_jwt).
	// +optional
	TokenEndpointAuthMethod *string `json:"tokenEndpointAuthMethod,omitempty"`
}

// ServiceObservation are the observable fields of a Service.
type ServiceObservation struct {
	// ID is the GUID of the service.
	ID *string `json:"id,omitempty"`

	// ServiceID is the service identifier used for authentication.
	ServiceID *string `json:"serviceId,omitempty"`

	// OrganizationID is the organization this service belongs to.
	OrganizationID *string `json:"organizationId,omitempty"`

	// ExpiresOn is when the service credentials expire.
	ExpiresOn *string `json:"expiresOn,omitempty"`
}

// ServiceSpec defines the desired state of a Service.
type ServiceSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ServiceParameters `json:"forProvider"`
}

// ServiceStatus represents the observed state of a Service.
type ServiceStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ServiceObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// Service is the Schema for the Service API.
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,dip}
type Service struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ServiceSpec   `json:"spec"`
	Status            ServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceList contains a list of Service.
type ServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Service `json:"items"`
}

// Service type metadata.
var (
	ServiceKind             = reflect.TypeOf(Service{}).Name()
	ServiceGroupKind        = schema.GroupKind{Group: Group, Kind: ServiceKind}.String()
	ServiceKindAPIVersion   = ServiceKind + "." + SchemeGroupVersion.String()
	ServiceGroupVersionKind = SchemeGroupVersion.WithKind(ServiceKind)
)

func init() {
	SchemeBuilder.Register(&Service{}, &ServiceList{})
}
```

- [ ] **Step 2: Commit**

```bash
git add apis/iam/v1alpha1/service_types.go
git commit -m "feat: add Service type"
```

---

## Task 12: Create User Type

**Files:**
- Create: `apis/iam/v1alpha1/user_types.go`

- [ ] **Step 1: Create user_types.go**

```go
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

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// UserParameters are the configurable fields of a User.
type UserParameters struct {
	// LoginID is the user's login identifier. Immutable after creation.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="loginId is immutable"
	LoginID string `json:"loginId"`

	// Email address of the user.
	// +kubebuilder:validation:Required
	Email string `json:"email"`

	// FirstName of the user.
	// +optional
	FirstName *string `json:"firstName,omitempty"`

	// LastName of the user.
	// +optional
	LastName *string `json:"lastName,omitempty"`

	// OrganizationID is the organization GUID this user belongs to.
	// +optional
	OrganizationID *string `json:"organizationId,omitempty"`

	// OrganizationRef references an Organization.
	// +optional
	OrganizationRef *xpv1.Reference `json:"organizationRef,omitempty"`

	// OrganizationSelector selects an Organization.
	// +optional
	OrganizationSelector *xpv1.Selector `json:"organizationSelector,omitempty"`

	// PreferredLanguage (e.g., en-US).
	// +optional
	PreferredLanguage *string `json:"preferredLanguage,omitempty"`

	// PreferredCommunicationChannel (e.g., email, sms).
	// +optional
	PreferredCommunicationChannel *string `json:"preferredCommunicationChannel,omitempty"`

	// IsAgeValidated indicates if user's age has been validated.
	// +optional
	IsAgeValidated *bool `json:"isAgeValidated,omitempty"`

	// PasswordSecretRef references a secret containing the initial password.
	// +optional
	PasswordSecretRef *xpv1.SecretKeySelector `json:"passwordSecretRef,omitempty"`
}

// UserObservation are the observable fields of a User.
type UserObservation struct {
	// ID is the GUID of the user.
	ID *string `json:"id,omitempty"`

	// AccountStatus (e.g., ACTIVE, INACTIVE).
	AccountStatus *string `json:"accountStatus,omitempty"`

	// EmailVerified indicates if email has been verified.
	EmailVerified *bool `json:"emailVerified,omitempty"`
}

// UserSpec defines the desired state of a User.
type UserSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              UserParameters `json:"forProvider"`
}

// UserStatus represents the observed state of a User.
type UserStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          UserObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// User is the Schema for the User API.
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,dip}
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              UserSpec   `json:"spec"`
	Status            UserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UserList contains a list of User.
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

// User type metadata.
var (
	UserKind             = reflect.TypeOf(User{}).Name()
	UserGroupKind        = schema.GroupKind{Group: Group, Kind: UserKind}.String()
	UserKindAPIVersion   = UserKind + "." + SchemeGroupVersion.String()
	UserGroupVersionKind = SchemeGroupVersion.WithKind(UserKind)
)

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
```

- [ ] **Step 2: Commit**

```bash
git add apis/iam/v1alpha1/user_types.go
git commit -m "feat: add User type"
```

---

## Task 13: Create EmailTemplate Type

**Files:**
- Create: `apis/iam/v1alpha1/emailtemplate_types.go`

- [ ] **Step 1: Create emailtemplate_types.go**

```go
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

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// EmailTemplateParameters are the configurable fields of an EmailTemplate.
type EmailTemplateParameters struct {
	// Type of email template (e.g., ACCOUNT_VERIFICATION, PASSWORD_RESET). Immutable.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="type is immutable"
	Type string `json:"type"`

	// ManagingOrganizationID is the organization GUID that manages this template.
	// +optional
	ManagingOrganizationID *string `json:"managingOrganizationId,omitempty"`

	// ManagingOrganizationRef references an Organization.
	// +optional
	ManagingOrganizationRef *xpv1.Reference `json:"managingOrganizationRef,omitempty"`

	// ManagingOrganizationSelector selects an Organization.
	// +optional
	ManagingOrganizationSelector *xpv1.Selector `json:"managingOrganizationSelector,omitempty"`

	// Format of the email (HTML or TEXT).
	// +kubebuilder:validation:Enum=HTML;TEXT
	// +optional
	Format *string `json:"format,omitempty"`

	// Locale for the template (e.g., en-US).
	// +optional
	Locale *string `json:"locale,omitempty"`

	// Subject line of the email.
	// +optional
	Subject *string `json:"subject,omitempty"`

	// From address for the email.
	// +optional
	From *string `json:"from,omitempty"`

	// Message body content.
	// +optional
	Message *string `json:"message,omitempty"`

	// Link to include in the email.
	// +optional
	Link *string `json:"link,omitempty"`
}

// EmailTemplateObservation are the observable fields of an EmailTemplate.
type EmailTemplateObservation struct {
	// ID is the GUID of the email template.
	ID *string `json:"id,omitempty"`
}

// EmailTemplateSpec defines the desired state of an EmailTemplate.
type EmailTemplateSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              EmailTemplateParameters `json:"forProvider"`
}

// EmailTemplateStatus represents the observed state of an EmailTemplate.
type EmailTemplateStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          EmailTemplateObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// EmailTemplate is the Schema for the EmailTemplate API.
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,dip}
type EmailTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              EmailTemplateSpec   `json:"spec"`
	Status            EmailTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EmailTemplateList contains a list of EmailTemplate.
type EmailTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EmailTemplate `json:"items"`
}

// EmailTemplate type metadata.
var (
	EmailTemplateKind             = reflect.TypeOf(EmailTemplate{}).Name()
	EmailTemplateGroupKind        = schema.GroupKind{Group: Group, Kind: EmailTemplateKind}.String()
	EmailTemplateKindAPIVersion   = EmailTemplateKind + "." + SchemeGroupVersion.String()
	EmailTemplateGroupVersionKind = SchemeGroupVersion.WithKind(EmailTemplateKind)
)

func init() {
	SchemeBuilder.Register(&EmailTemplate{}, &EmailTemplateList{})
}
```

- [ ] **Step 2: Commit**

```bash
git add apis/iam/v1alpha1/emailtemplate_types.go
git commit -m "feat: add EmailTemplate type"
```

---

## Task 14: Create PasswordPolicy Type

**Files:**
- Create: `apis/iam/v1alpha1/passwordpolicy_types.go`

- [ ] **Step 1: Create passwordpolicy_types.go**

```go
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

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// ChallengePolicyParameters configures security challenge questions.
type ChallengePolicyParameters struct {
	// DefaultQuestions available for users.
	// +optional
	DefaultQuestions []string `json:"defaultQuestions,omitempty"`

	// MinAnswerLength for challenge answers.
	// +optional
	MinAnswerLength *int `json:"minAnswerLength,omitempty"`

	// MaxIncorrectAttempts before lockout.
	// +optional
	MaxIncorrectAttempts *int `json:"maxIncorrectAttempts,omitempty"`
}

// PasswordPolicyParameters are the configurable fields of a PasswordPolicy.
type PasswordPolicyParameters struct {
	// ManagingOrganizationID is the organization GUID that manages this policy.
	// +optional
	ManagingOrganizationID *string `json:"managingOrganizationId,omitempty"`

	// ManagingOrganizationRef references an Organization.
	// +optional
	ManagingOrganizationRef *xpv1.Reference `json:"managingOrganizationRef,omitempty"`

	// ManagingOrganizationSelector selects an Organization.
	// +optional
	ManagingOrganizationSelector *xpv1.Selector `json:"managingOrganizationSelector,omitempty"`

	// ExpiryPeriodInDays before password must be changed.
	// +optional
	ExpiryPeriodInDays *int `json:"expiryPeriodInDays,omitempty"`

	// HistoryCount of previous passwords that cannot be reused.
	// +optional
	HistoryCount *int `json:"historyCount,omitempty"`

	// MinLength of password.
	// +optional
	MinLength *int `json:"minLength,omitempty"`

	// MaxLength of password.
	// +optional
	MaxLength *int `json:"maxLength,omitempty"`

	// MinLowercase characters required.
	// +optional
	MinLowercase *int `json:"minLowercase,omitempty"`

	// MinUppercase characters required.
	// +optional
	MinUppercase *int `json:"minUppercase,omitempty"`

	// MinNumeric characters required.
	// +optional
	MinNumeric *int `json:"minNumeric,omitempty"`

	// MinSpecialChars required.
	// +optional
	MinSpecialChars *int `json:"minSpecialChars,omitempty"`

	// ChallengesEnabled enables security challenge questions.
	// +optional
	ChallengesEnabled *bool `json:"challengesEnabled,omitempty"`

	// ChallengePolicy configures security challenge questions.
	// +optional
	ChallengePolicy *ChallengePolicyParameters `json:"challengePolicy,omitempty"`
}

// PasswordPolicyObservation are the observable fields of a PasswordPolicy.
type PasswordPolicyObservation struct {
	// ID is the GUID of the password policy.
	ID *string `json:"id,omitempty"`
}

// PasswordPolicySpec defines the desired state of a PasswordPolicy.
type PasswordPolicySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              PasswordPolicyParameters `json:"forProvider"`
}

// PasswordPolicyStatus represents the observed state of a PasswordPolicy.
type PasswordPolicyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          PasswordPolicyObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// PasswordPolicy is the Schema for the PasswordPolicy API.
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,dip}
type PasswordPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PasswordPolicySpec   `json:"spec"`
	Status            PasswordPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PasswordPolicyList contains a list of PasswordPolicy.
type PasswordPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PasswordPolicy `json:"items"`
}

// PasswordPolicy type metadata.
var (
	PasswordPolicyKind             = reflect.TypeOf(PasswordPolicy{}).Name()
	PasswordPolicyGroupKind        = schema.GroupKind{Group: Group, Kind: PasswordPolicyKind}.String()
	PasswordPolicyKindAPIVersion   = PasswordPolicyKind + "." + SchemeGroupVersion.String()
	PasswordPolicyGroupVersionKind = SchemeGroupVersion.WithKind(PasswordPolicyKind)
)

func init() {
	SchemeBuilder.Register(&PasswordPolicy{}, &PasswordPolicyList{})
}
```

- [ ] **Step 2: Commit**

```bash
git add apis/iam/v1alpha1/passwordpolicy_types.go
git commit -m "feat: add PasswordPolicy type"
```

---

## Task 15: Update Scheme Registration and Generate Code

**Files:**
- Modify: `apis/dip.go` (was `apis/template.go`)
- Run: code generation

- [ ] **Step 1: Rename and update apis/template.go to apis/dip.go**

```go
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

// Package apis contains Kubernetes API for the DIP provider.
package apis

import (
	"k8s.io/apimachinery/pkg/runtime"

	iamv1alpha1 "github.com/crossplane/provider-template/apis/iam/v1alpha1"
	dipv1alpha1 "github.com/crossplane/provider-template/apis/v1alpha1"
)

func init() {
	AddToSchemes = append(AddToSchemes,
		dipv1alpha1.SchemeBuilder.AddToScheme,
		iamv1alpha1.SchemeBuilder.AddToScheme,
	)
}

// AddToSchemes may be used to add all resources defined in the project to a Scheme.
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme.
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
```

- [ ] **Step 2: Run code generation**

```bash
make generate
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add apis/
git commit -m "feat: update scheme registration and generate code"
```

---

## Task 16: Create Organization Controller

**Files:**
- Create: `internal/controller/organization/organization.go`

- [ ] **Step 1: Create organization controller**

```go
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

package organization

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
	errNotOrganization = "managed resource is not an Organization"
	errTrackPCUsage    = "cannot track ProviderConfig usage"
	errGetPC           = "cannot get ProviderConfig"
	errGetCreds        = "cannot get credentials"
	errNewClient       = "cannot create DIP client"
)

// Setup adds a controller that reconciles Organization managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(iamv1alpha1.OrganizationGroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(iamv1alpha1.OrganizationGroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&iamv1alpha1.Organization{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*iamv1alpha1.Organization)
	if !ok {
		return nil, errors.New(errNotOrganization)
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
	cr, ok := mg.(*iamv1alpha1.Organization)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOrganization)
	}

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	org, _, err := e.client.IAM.Organizations.GetOrganizationByID(externalName)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get organization")
	}
	if org == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider.ID = &org.ID
	cr.Status.AtProvider.Active = &org.Active

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: e.isUpToDate(cr, org),
	}, nil
}

func (e *external) isUpToDate(cr *iamv1alpha1.Organization, org *iam.Organization) bool {
	fp := cr.Spec.ForProvider

	if fp.Name != org.Name {
		return false
	}
	if fp.Description != nil && *fp.Description != org.Description {
		return false
	}
	if fp.DisplayName != nil && *fp.DisplayName != org.DisplayName {
		return false
	}
	return true
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*iamv1alpha1.Organization)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOrganization)
	}

	cr.Status.SetConditions(xpv1.Creating())

	fp := cr.Spec.ForProvider

	org := iam.Organization{
		Name: fp.Name,
	}

	if fp.Description != nil {
		org.Description = *fp.Description
	}
	if fp.DisplayName != nil {
		org.DisplayName = *fp.DisplayName
	}
	if fp.ParentOrgID != nil {
		org.Parent.Value = *fp.ParentOrgID
	}
	if fp.Type != nil {
		org.Type = *fp.Type
	}
	if fp.ExternalID != nil {
		org.ExternalID = *fp.ExternalID
	}

	created, _, err := e.client.IAM.Organizations.CreateOrganization(org)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create organization")
	}

	meta.SetExternalName(cr, created.ID)

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*iamv1alpha1.Organization)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOrganization)
	}

	fp := cr.Spec.ForProvider

	org := iam.Organization{
		ID:   meta.GetExternalName(cr),
		Name: fp.Name,
	}

	if fp.Description != nil {
		org.Description = *fp.Description
	}
	if fp.DisplayName != nil {
		org.DisplayName = *fp.DisplayName
	}

	_, _, err := e.client.IAM.Organizations.UpdateOrganization(org)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update organization")
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*iamv1alpha1.Organization)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotOrganization)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		return managed.ExternalDelete{}, nil
	}

	_, _, err := e.client.IAM.Organizations.DeleteOrganization(iam.Organization{ID: externalName})
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete organization")
	}

	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(ctx context.Context) error {
	return nil
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/controller/organization/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/controller/organization/
git commit -m "feat: add Organization controller"
```

---

## Task 17: Create Remaining IAM Controllers

**Files:**
- Create: `internal/controller/group/group.go`
- Create: `internal/controller/role/role.go`
- Create: `internal/controller/proposition/proposition.go`
- Create: `internal/controller/application/application.go`
- Create: `internal/controller/client/client.go`
- Create: `internal/controller/service/service.go`
- Create: `internal/controller/user/user.go`
- Create: `internal/controller/emailtemplate/emailtemplate.go`
- Create: `internal/controller/passwordpolicy/passwordpolicy.go`

Each controller follows the same pattern as Organization (Task 16). Key differences per resource:

| Resource | IAM Service Method | Unique Fields |
|----------|-------------------|---------------|
| Group | Groups.CreateGroup, GetGroupByID, etc. | ManagingOrganizationID |
| Role | Roles.CreateRole, GetRoleByID, etc. | Permissions, ManagingOrganizationID |
| Proposition | Propositions.CreateProposition, etc. | GlobalReferenceID, OrganizationID |
| Application | Applications.CreateApplication, etc. | GlobalReferenceID, PropositionID |
| Client | Clients.CreateClient, etc. | OAuth2 fields, connection secret |
| Service | Services.CreateService, etc. | PrivateKey, connection secret |
| User | Users.CreateUser, etc. | LoginID, Email, password handling |
| EmailTemplate | EmailTemplates methods | Type, Format, Locale |
| PasswordPolicy | PasswordPolicies methods | Complexity rules, ChallengePolicy |

- [ ] **Step 1: Create group controller**

Create `internal/controller/group/group.go` following the Organization pattern with `e.client.IAM.Groups` methods.

- [ ] **Step 2: Create role controller**

Create `internal/controller/role/role.go` with permissions handling.

- [ ] **Step 3: Create proposition controller**

Create `internal/controller/proposition/proposition.go` with GlobalReferenceID.

- [ ] **Step 4: Create application controller**

Create `internal/controller/application/application.go` with PropositionID reference.

- [ ] **Step 5: Create client controller**

Create `internal/controller/client/client.go` with connection secret for clientId/clientSecret.

- [ ] **Step 6: Create service controller**

Create `internal/controller/service/service.go` with connection secret for serviceId/privateKey.

- [ ] **Step 7: Create user controller**

Create `internal/controller/user/user.go` with password secret handling.

- [ ] **Step 8: Create emailtemplate controller**

Create `internal/controller/emailtemplate/emailtemplate.go`.

- [ ] **Step 9: Create passwordpolicy controller**

Create `internal/controller/passwordpolicy/passwordpolicy.go` with nested ChallengePolicy.

- [ ] **Step 10: Verify compilation**

```bash
go build ./internal/controller/...
```

- [ ] **Step 11: Commit**

```bash
git add internal/controller/
git commit -m "feat: add remaining IAM controllers"
```

---

## Task 18: Update Controller Registration

**Files:**
- Modify: `internal/controller/register.go`

- [ ] **Step 1: Update register.go**

```go
/*
Copyright 2020 The Crossplane Authors.

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

package controller

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/provider-template/internal/controller/application"
	"github.com/crossplane/provider-template/internal/controller/client"
	"github.com/crossplane/provider-template/internal/controller/config"
	"github.com/crossplane/provider-template/internal/controller/emailtemplate"
	"github.com/crossplane/provider-template/internal/controller/group"
	"github.com/crossplane/provider-template/internal/controller/organization"
	"github.com/crossplane/provider-template/internal/controller/passwordpolicy"
	"github.com/crossplane/provider-template/internal/controller/proposition"
	"github.com/crossplane/provider-template/internal/controller/role"
	"github.com/crossplane/provider-template/internal/controller/service"
	"github.com/crossplane/provider-template/internal/controller/user"
)

// Setup creates all DIP controllers and adds them to the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		config.Setup,
		organization.Setup,
		group.Setup,
		role.Setup,
		proposition.Setup,
		application.Setup,
		client.Setup,
		service.Setup,
		user.Setup,
		emailtemplate.Setup,
		passwordpolicy.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/controller/register.go
git commit -m "feat: register all IAM controllers"
```

---

## Task 19: Generate CRDs and Build

**Files:**
- Run: make generate, make manifests

- [ ] **Step 1: Generate all code**

```bash
make generate
```

- [ ] **Step 2: Generate CRDs**

```bash
make manifests
```

- [ ] **Step 3: Build provider**

```bash
make build
```

- [ ] **Step 4: Commit generated files**

```bash
git add .
git commit -m "feat: generate CRDs and build provider"
```

---

## Task 20: Add Example Manifests

**Files:**
- Create: `examples/providerconfig/providerconfig.yaml`
- Create: `examples/iam/organization.yaml`

- [ ] **Step 1: Create ProviderConfig example**

```yaml
apiVersion: dip.m.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
  namespace: crossplane-system
spec:
  region: us-east
  environment: client-test
  credentials:
    source: Secret
    secretRef:
      name: dip-credentials
      namespace: crossplane-system
      key: credentials
```

- [ ] **Step 2: Create Organization example**

```yaml
apiVersion: iam.dip.m.crossplane.io/v1alpha1
kind: Organization
metadata:
  name: example-org
  namespace: crossplane-system
spec:
  providerConfigRef:
    name: default
  forProvider:
    name: example-organization
    displayName: "Example Organization"
    description: "Managed by Crossplane"
    parentOrgId: "your-parent-org-guid"
```

- [ ] **Step 3: Commit**

```bash
git add examples/
git commit -m "docs: add example manifests"
```

---

## Summary

| Task | Description |
|------|-------------|
| 1 | Update go.mod and API group |
| 2 | Update ProviderConfig types |
| 3 | Create DIP client wrapper |
| 4 | Create IAM API group structure |
| 5-14 | Create type definitions (10 resources) |
| 15 | Update scheme registration |
| 16 | Create Organization controller |
| 17 | Create remaining controllers |
| 18 | Update controller registration |
| 19 | Generate CRDs and build |
| 20 | Add example manifests |
