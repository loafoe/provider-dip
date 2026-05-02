# Native Crossplane 2.0 Provider for DIP IAM

## Overview

Native Crossplane provider for DIP IAM resources, replacing the Upjet/Terraform-based provider-hsdp with direct go-dip-api integration.

## Decisions

| Decision | Choice |
|----------|--------|
| API Group (IAM) | `iam.dip.m.crossplane.io/v1alpha1` |
| API Group (ProviderConfig) | `dip.m.crossplane.io/v1alpha1` |
| Authentication | Service Identity only (region, environment, service_id, service_private_key) |
| Config flexibility | Secret values override spec values for region/environment |
| Resource scope | Namespaced |
| Field naming | Compatible JSON names + Crossplane references (Ref/Selector) |
| Immutability | CEL validation rules |
| External name | Simple GUID |
| Deletion | Standard Crossplane requeue pattern |
| Client architecture | Single wrapper package (`internal/clients/dip`) |

## Resources

1. Organization
2. Group
3. Role
4. Application
5. Client
6. Service
7. EmailTemplate
8. PasswordPolicy
9. Proposition
10. User

## Project Structure

```
provider-dip/
├── apis/
│   ├── v1alpha1/                    # ProviderConfig
│   │   ├── types.go
│   │   ├── register.go
│   │   └── groupversion_info.go
│   ├── iam/
│   │   └── v1alpha1/                # IAM resources
│   │       ├── organization_types.go
│   │       ├── group_types.go
│   │       ├── role_types.go
│   │       ├── application_types.go
│   │       ├── client_types.go
│   │       ├── service_types.go
│   │       ├── emailtemplate_types.go
│   │       ├── passwordpolicy_types.go
│   │       ├── proposition_types.go
│   │       ├── user_types.go
│   │       ├── groupversion_info.go
│   │       └── doc.go
│   └── generate.go
├── internal/
│   ├── clients/
│   │   └── dip/
│   │       ├── client.go
│   │       └── config.go
│   └── controller/
│       ├── organization/organization.go
│       ├── group/group.go
│       ├── role/role.go
│       ├── application/application.go
│       ├── client/client.go
│       ├── service/service.go
│       ├── emailtemplate/emailtemplate.go
│       ├── passwordpolicy/passwordpolicy.go
│       ├── proposition/proposition.go
│       ├── user/user.go
│       ├── config/config.go
│       └── register.go
├── cmd/provider/main.go
└── package/crds/
```

## ProviderConfig

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

Secret format (JSON):
```json
{
  "region": "us-east",
  "environment": "client-test",
  "service_id": "your-service-uuid",
  "service_private_key": "-----BEGIN RSA PRIVATE KEY-----\n..."
}
```

Resolution: secret values override spec values.

## Type Definitions

### Organization

```go
type OrganizationParameters struct {
    Name                     string          `json:"name"`
    DisplayName              *string         `json:"displayName,omitempty"`
    Description              *string         `json:"description,omitempty"`
    ParentOrgID              *string         `json:"parentOrgId,omitempty"`
    ParentOrgRef             *xpv1.Reference `json:"parentOrgRef,omitempty"`
    ParentOrgSelector        *xpv1.Selector  `json:"parentOrgSelector,omitempty"`
    Type                     *string         `json:"type,omitempty"`
    ExternalID               *string         `json:"externalId,omitempty"`
}

type OrganizationObservation struct {
    ID     *string `json:"id,omitempty"`
    Active *bool   `json:"active,omitempty"`
}
```

Immutable: `name`

### Group

```go
type GroupParameters struct {
    Name                           string          `json:"name"`
    Description                    *string         `json:"description,omitempty"`
    ManagingOrganizationID         *string         `json:"managingOrganizationId,omitempty"`
    ManagingOrganizationRef        *xpv1.Reference `json:"managingOrganizationRef,omitempty"`
    ManagingOrganizationSelector   *xpv1.Selector  `json:"managingOrganizationSelector,omitempty"`
}

type GroupObservation struct {
    ID *string `json:"id,omitempty"`
}
```

Immutable: `name`

### Role

```go
type RoleParameters struct {
    Name                           string          `json:"name"`
    Description                    *string         `json:"description,omitempty"`
    ManagingOrganizationID         *string         `json:"managingOrganizationId,omitempty"`
    ManagingOrganizationRef        *xpv1.Reference `json:"managingOrganizationRef,omitempty"`
    ManagingOrganizationSelector   *xpv1.Selector  `json:"managingOrganizationSelector,omitempty"`
    Permissions                    []string        `json:"permissions,omitempty"`
}

type RoleObservation struct {
    ID *string `json:"id,omitempty"`
}
```

Immutable: `name`

### Application

```go
type ApplicationParameters struct {
    Name                string          `json:"name"`
    Description         *string         `json:"description,omitempty"`
    PropositionID       *string         `json:"propositionId,omitempty"`
    PropositionRef      *xpv1.Reference `json:"propositionRef,omitempty"`
    PropositionSelector *xpv1.Selector  `json:"propositionSelector,omitempty"`
    GlobalReferenceID   string          `json:"globalReferenceId"`
}

type ApplicationObservation struct {
    ID *string `json:"id,omitempty"`
}
```

Immutable: `name`, `globalReferenceId`

### Client

```go
type ClientParameters struct {
    Name                 string          `json:"name"`
    Description          *string         `json:"description,omitempty"`
    ApplicationID        *string         `json:"applicationId,omitempty"`
    ApplicationRef       *xpv1.Reference `json:"applicationRef,omitempty"`
    ApplicationSelector  *xpv1.Selector  `json:"applicationSelector,omitempty"`
    GlobalReferenceID    string          `json:"globalReferenceId"`
    RedirectionURIs      []string        `json:"redirectionURIs,omitempty"`
    ResponseTypes        []string        `json:"responseTypes,omitempty"`
    Scopes               []string        `json:"scopes,omitempty"`
    DefaultScopes        []string        `json:"defaultScopes,omitempty"`
    ConsentImplied       *bool           `json:"consentImplied,omitempty"`
    AccessTokenLifetime  *int64          `json:"accessTokenLifetime,omitempty"`
    RefreshTokenLifetime *int64          `json:"refreshTokenLifetime,omitempty"`
    IDTokenLifetime      *int64          `json:"idTokenLifetime,omitempty"`
}

type ClientObservation struct {
    ID       *string `json:"id,omitempty"`
    ClientID *string `json:"clientId,omitempty"`
    Disabled *bool   `json:"disabled,omitempty"`
}
```

Immutable: `name`, `globalReferenceId`

Connection secret: `clientId`, `clientSecret`

### Service

```go
type ServiceParameters struct {
    Name                    string                  `json:"name"`
    Description             *string                 `json:"description,omitempty"`
    ApplicationID           *string                 `json:"applicationId,omitempty"`
    ApplicationRef          *xpv1.Reference         `json:"applicationRef,omitempty"`
    ApplicationSelector     *xpv1.Selector          `json:"applicationSelector,omitempty"`
    PrivateKeySecretRef     *xpv1.SecretKeySelector `json:"privateKeySecretRef,omitempty"`
    Scopes                  []string                `json:"scopes,omitempty"`
    DefaultScopes           []string                `json:"defaultScopes,omitempty"`
    AccessTokenLifetime     *int64                  `json:"accessTokenLifetime,omitempty"`
    RefreshTokenLifetime    *int64                  `json:"refreshTokenLifetime,omitempty"`
    TokenEndpointAuthMethod *string                 `json:"tokenEndpointAuthMethod,omitempty"`
}

type ServiceObservation struct {
    ID             *string `json:"id,omitempty"`
    ServiceID      *string `json:"serviceId,omitempty"`
    OrganizationID *string `json:"organizationId,omitempty"`
    ExpiresOn      *string `json:"expiresOn,omitempty"`
}
```

Immutable: `name`

Connection secret: `serviceId`, `privateKey`

### EmailTemplate

```go
type EmailTemplateParameters struct {
    Type                           string          `json:"type"`
    ManagingOrganizationID         *string         `json:"managingOrganizationId,omitempty"`
    ManagingOrganizationRef        *xpv1.Reference `json:"managingOrganizationRef,omitempty"`
    ManagingOrganizationSelector   *xpv1.Selector  `json:"managingOrganizationSelector,omitempty"`
    Format                         *string         `json:"format,omitempty"`
    Locale                         *string         `json:"locale,omitempty"`
    Subject                        *string         `json:"subject,omitempty"`
    From                           *string         `json:"from,omitempty"`
    Message                        *string         `json:"message,omitempty"`
    Link                           *string         `json:"link,omitempty"`
}

type EmailTemplateObservation struct {
    ID *string `json:"id,omitempty"`
}
```

Immutable: `type`

### PasswordPolicy

```go
type PasswordPolicyParameters struct {
    ManagingOrganizationID         *string                     `json:"managingOrganizationId,omitempty"`
    ManagingOrganizationRef        *xpv1.Reference             `json:"managingOrganizationRef,omitempty"`
    ManagingOrganizationSelector   *xpv1.Selector              `json:"managingOrganizationSelector,omitempty"`
    ExpiryPeriodInDays             *int                        `json:"expiryPeriodInDays,omitempty"`
    HistoryCount                   *int                        `json:"historyCount,omitempty"`
    MinLength                      *int                        `json:"minLength,omitempty"`
    MaxLength                      *int                        `json:"maxLength,omitempty"`
    MinLowercase                   *int                        `json:"minLowercase,omitempty"`
    MinUppercase                   *int                        `json:"minUppercase,omitempty"`
    MinNumeric                     *int                        `json:"minNumeric,omitempty"`
    MinSpecialChars                *int                        `json:"minSpecialChars,omitempty"`
    ChallengesEnabled              *bool                       `json:"challengesEnabled,omitempty"`
    ChallengePolicy                *ChallengePolicyParameters  `json:"challengePolicy,omitempty"`
}

type ChallengePolicyParameters struct {
    DefaultQuestions     []string `json:"defaultQuestions,omitempty"`
    MinAnswerLength      *int     `json:"minAnswerLength,omitempty"`
    MaxIncorrectAttempts *int     `json:"maxIncorrectAttempts,omitempty"`
}

type PasswordPolicyObservation struct {
    ID *string `json:"id,omitempty"`
}
```

### Proposition

```go
type PropositionParameters struct {
    Name                 string          `json:"name"`
    Description          *string         `json:"description,omitempty"`
    OrganizationID       *string         `json:"organizationId,omitempty"`
    OrganizationRef      *xpv1.Reference `json:"organizationRef,omitempty"`
    OrganizationSelector *xpv1.Selector  `json:"organizationSelector,omitempty"`
    GlobalReferenceID    string          `json:"globalReferenceId"`
}

type PropositionObservation struct {
    ID *string `json:"id,omitempty"`
}
```

Immutable: `name`, `globalReferenceId`

### User

```go
type UserParameters struct {
    LoginID                       string                  `json:"loginId"`
    Email                         string                  `json:"email"`
    FirstName                     *string                 `json:"firstName,omitempty"`
    LastName                      *string                 `json:"lastName,omitempty"`
    OrganizationID                *string                 `json:"organizationId,omitempty"`
    OrganizationRef               *xpv1.Reference         `json:"organizationRef,omitempty"`
    OrganizationSelector          *xpv1.Selector          `json:"organizationSelector,omitempty"`
    PreferredLanguage             *string                 `json:"preferredLanguage,omitempty"`
    PreferredCommunicationChannel *string                 `json:"preferredCommunicationChannel,omitempty"`
    IsAgeValidated                *bool                   `json:"isAgeValidated,omitempty"`
    PasswordSecretRef             *xpv1.SecretKeySelector `json:"passwordSecretRef,omitempty"`
}

type UserObservation struct {
    ID            *string `json:"id,omitempty"`
    AccountStatus *string `json:"accountStatus,omitempty"`
    EmailVerified *bool   `json:"emailVerified,omitempty"`
}
```

Immutable: `loginId`

Connection secret: `loginId`, `password`

## Reference Resolution

Cross-resource references (Ref/Selector fields) resolve to the referenced resource's external-name annotation (the GUID). Resolution happens in the connector before CRUD operations.

Example: `parentOrgRef.name: my-parent-org` resolves to the GUID in `crossplane.io/external-name` of the referenced Organization.

Priority: Direct ID field > Ref > Selector. If multiple specified, direct ID wins.

## DIP Client Wrapper

```go
// internal/clients/dip/client.go
package dip

type Config struct {
    Region            string
    Environment       string
    ServiceID         string
    ServicePrivateKey string
}

type Client struct {
    IAM *iam.Client
}

func NewClient(cfg Config) (*Client, error)

func ConfigFromSecret(specRegion, specEnv string, secretData []byte) (Config, error)
```

## Controller Pattern

Each controller implements:
- `Setup(mgr, options)` - registers controller
- `connector.Connect()` - creates DIP client from ProviderConfig
- `external.Observe()` - gets resource by external-name GUID
- `external.Create()` - creates resource, sets external-name
- `external.Update()` - updates resource
- `external.Delete()` - deletes resource (Crossplane requeues until gone)
- `external.Disconnect()` - cleanup (no-op)

## Migration Path

1. Deploy new provider alongside existing
2. Create resources with new API group (`iam.dip.m.crossplane.io`)
3. Update manifests: change `apiVersion` and `kind`
4. Remove old provider

Field names are compatible - only API group changes.
