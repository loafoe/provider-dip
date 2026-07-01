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
	"regexp"
	"strings"

	"github.com/philips-software/go-dip-api/connect/mdm"
	"github.com/philips-software/go-dip-api/connect/provisioning"
	"github.com/philips-software/go-dip-api/iam"
)

// Config holds DIP client configuration.
type Config struct {
	Region            string
	Environment       string
	ServiceID         string
	ServicePrivateKey string
	// TokenAudience optionally overrides the JWT "aud" claim used for service
	// login. Required for Keycloak-backed IAM realms (e.g. the us-east preview
	// environment) that validate against the realm issuer rather than the
	// access token endpoint.
	TokenAudience string
}

// Client wraps go-dip-api IAM, MDM, and Provisioning clients.
type Client struct {
	IAM          *iam.Client
	MDM          *mdm.Client
	Provisioning *provisioning.Client
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

	// Create IAM client
	iamClient, err := iam.NewClient(nil, &iam.Config{
		Region:        cfg.Region,
		Environment:   cfg.Environment,
		TokenAudience: cfg.TokenAudience,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create IAM client: %w", err)
	}

	// Authenticate using service identity (JWT-based OAuth2)
	err = iamClient.ServiceLogin(iam.Service{
		ServiceID:  cfg.ServiceID,
		PrivateKey: cfg.ServicePrivateKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate service identity '%s': %w", cfg.ServiceID, err)
	}

	// Create MDM client using the authenticated IAM client. MDM is not
	// available in every region/environment (e.g. us-east preview only exposes
	// IAM/IDM), so treat an unresolvable endpoint as "MDM not configured"
	// rather than a fatal error. MDM controllers surface a clear error if a
	// user tries to manage MDM resources without it.
	mdmClient, err := mdm.NewClient(iamClient, &mdm.Config{
		Region:      cfg.Region,
		Environment: cfg.Environment,
	})
	if err != nil {
		mdmClient = nil
	}

	// Create Provisioning client using the authenticated IAM client. Same
	// availability caveat as MDM above.
	provisioningClient, err := provisioning.NewClient(iamClient, &provisioning.Config{
		Region:      cfg.Region,
		Environment: cfg.Environment,
	})
	if err != nil {
		provisioningClient = nil
	}

	return &Client{IAM: iamClient, MDM: mdmClient, Provisioning: provisioningClient}, nil
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
		TokenAudience:     creds["token_audience"],
	}

	// Secret values override spec values
	if v, ok := creds["region"]; ok && v != "" {
		cfg.Region = v
	}
	if v, ok := creds["environment"]; ok && v != "" {
		cfg.Environment = v
	}

	// Fix private key format if needed (add newlines every 64 chars)
	cfg.ServicePrivateKey = formatPrivateKey(cfg.ServicePrivateKey)

	// Validate required fields
	if cfg.Region == "" {
		return Config{}, fmt.Errorf("region is required: set in ProviderConfig spec or credentials secret")
	}
	if cfg.Environment == "" {
		return Config{}, fmt.Errorf("environment is required: set in ProviderConfig spec or credentials secret")
	}
	if cfg.ServiceID == "" {
		return Config{}, fmt.Errorf("service_id is required in credentials secret")
	}
	if cfg.ServicePrivateKey == "" {
		return Config{}, fmt.Errorf("service_private_key is required in credentials secret")
	}

	return cfg, nil
}

// formatPrivateKey ensures the private key has proper PEM formatting with newlines.
func formatPrivateKey(key string) string {
	// If already properly formatted, return as-is
	if strings.Contains(key, "\n") {
		return key
	}

	// Extract header, body, and footer
	re := regexp.MustCompile(`(-----BEGIN [A-Z ]+-----)(.+)(-----END [A-Z ]+-----)`)
	matches := re.FindStringSubmatch(key)
	if len(matches) != 4 {
		return key
	}

	header := matches[1]
	body := matches[2]
	footer := matches[3]

	// Split body into 64-character lines
	var lines []string
	for i := 0; i < len(body); i += 64 {
		end := i + 64
		if end > len(body) {
			end = len(body)
		}
		lines = append(lines, body[i:end])
	}

	return header + "\n" + strings.Join(lines, "\n") + "\n" + footer
}
