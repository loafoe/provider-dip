# provider-dip

A native [Crossplane](https://crossplane.io/) 2.0 Provider for the Digital Innovation Platform (DIP), enabling declarative management of IAM, MDM, and Provisioning resources.

## Overview

This provider uses the [go-dip-api](https://github.com/philips-software/go-dip-api) library directly to interact with DIP services, providing full lifecycle management of resources through Kubernetes custom resources.

## Supported Resources

### IAM (Identity and Access Management)

| Resource | API Group | Kind |
|----------|-----------|------|
| Organization | iam.dip.m.crossplane.io | Organization |
| Proposition | iam.dip.m.crossplane.io | Proposition |
| Application | iam.dip.m.crossplane.io | Application |
| Group | iam.dip.m.crossplane.io | Group |
| Role | iam.dip.m.crossplane.io | Role |
| Service | iam.dip.m.crossplane.io | Service |
| Client | iam.dip.m.crossplane.io | Client |
| User | iam.dip.m.crossplane.io | User |
| EmailTemplate | iam.dip.m.crossplane.io | EmailTemplate |
| PasswordPolicy | iam.dip.m.crossplane.io | PasswordPolicy |

### MDM (Master Data Management)

| Resource | API Group | Kind |
|----------|-----------|------|
| Proposition | mdm.dip.m.crossplane.io | Proposition |
| Application | mdm.dip.m.crossplane.io | Application |
| StandardService | mdm.dip.m.crossplane.io | StandardService |
| DeviceGroup | mdm.dip.m.crossplane.io | DeviceGroup |
| DeviceType | mdm.dip.m.crossplane.io | DeviceType |
| AuthenticationMethod | mdm.dip.m.crossplane.io | AuthenticationMethod |

### Provisioning

| Resource | API Group | Kind |
|----------|-----------|------|
| OrgConfiguration | provisioning.dip.m.crossplane.io | OrgConfiguration |

## Installation

Install the provider using crossplane CLI or a DeploymentRuntimeConfig:

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-dip
spec:
  package: ghcr.io/loafoe/provider-dip:v0.3.0
```

## Configuration

### ProviderConfig

Create a ProviderConfig that references your DIP credentials:

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

### Credentials Secret

The credentials secret should contain a JSON object with your service identity:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: dip-credentials
  namespace: crossplane-system
type: Opaque
stringData:
  credentials: |
    {
      "service_id": "your-service-id",
      "service_private_key": "-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----"
    }
```

The secret can optionally override `region` and `environment` from the ProviderConfig spec.

## Example Resources

### Organization

```yaml
apiVersion: iam.dip.m.crossplane.io/v1alpha1
kind: Organization
metadata:
  name: my-org
  namespace: crossplane-system
spec:
  forProvider:
    name: my-organization
    description: My organization
    parentOrganizationId: <parent-org-uuid>
  providerConfigRef:
    name: default
```

### Application

```yaml
apiVersion: iam.dip.m.crossplane.io/v1alpha1
kind: Application
metadata:
  name: my-app
  namespace: crossplane-system
spec:
  forProvider:
    name: my-application
    description: My application
    propositionId: <proposition-uuid>
    globalReferenceId: my-app-ref
  providerConfigRef:
    name: default
```

### Client

```yaml
apiVersion: iam.dip.m.crossplane.io/v1alpha1
kind: Client
metadata:
  name: my-client
  namespace: crossplane-system
spec:
  forProvider:
    name: my-client
    description: My OAuth2 client
    applicationId: <application-uuid>
    clientId: my-client-id
    type: Public
    globalReferenceId: my-client-ref
    redirectionURIs:
      - https://example.com/callback
    responseTypes:
      - code
    passwordSecretRef:
      name: client-password
      namespace: crossplane-system
      key: password
  providerConfigRef:
    name: default
```

## Development

### Prerequisites

- Go 1.23+
- Docker
- kubectl
- A Kubernetes cluster with Crossplane installed

### Build

```shell
make build
```

### Run locally

```shell
make run
```

### Build and push image

```shell
make docker-build docker-push IMG=ghcr.io/loafoe/provider-dip:v0.3.0
```

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.
