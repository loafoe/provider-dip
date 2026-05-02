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
	"github.com/philips-software/go-nih-signer"
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

	// Create signer for service authentication
	signer, err := signer.New(cfg.ServiceID, cfg.ServicePrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	iamClient, err := iam.NewClient(nil, &iam.Config{
		Region:      cfg.Region,
		Environment: cfg.Environment,
		SharedKey:   cfg.ServiceID,
		SecretKey:   cfg.ServicePrivateKey,
		Signer:      signer,
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
