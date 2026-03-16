/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"net/url"

	"github.com/volcengine/volcengine-go-sdk/service/vefaas"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

// SandboxManagerConfig is the configuration for the veFaaS sandbox lifecycle manager.
type SandboxManagerConfig struct {
	// AccessKey is the Volcengine AK. Required.
	AccessKey string
	// SecretKey is the Volcengine SK. Required.
	SecretKey string
	// Region is the Volcengine region. Default: "cn-beijing".
	Region string
	// FunctionID is the veFaaS function ID. Required.
	FunctionID string
}

// SandboxManager manages sandbox lifecycle via the Volcengine veFaaS control plane API.
type SandboxManager struct {
	client     *vefaas.VEFAAS
	functionID string
}

// NewSandboxManager creates a new sandbox lifecycle manager.
func NewSandboxManager(config *SandboxManagerConfig) (*SandboxManager, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if config.AccessKey == "" || config.SecretKey == "" {
		return nil, fmt.Errorf("AccessKey and SecretKey are required")
	}
	if config.FunctionID == "" {
		return nil, fmt.Errorf("FunctionID is required")
	}

	region := config.Region
	if region == "" {
		region = "cn-beijing"
	}

	sess, err := session.NewSession(volcengine.NewConfig().
		WithCredentials(credentials.NewStaticCredentials(config.AccessKey, config.SecretKey, "")).
		WithRegion(region).
		WithMaxRetries(0))
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &SandboxManager{
		client:     vefaas.New(sess),
		functionID: config.FunctionID,
	}, nil
}

// CreateSandbox creates a new sandbox instance and returns its ID.
func (m *SandboxManager) CreateSandbox() (string, error) {
	resp, err := m.client.CreateSandbox(&vefaas.CreateSandboxInput{
		FunctionId: volcengine.String(m.functionID),
	})
	if err != nil {
		return "", fmt.Errorf("create sandbox failed: %w", err)
	}
	if resp.SandboxId == nil {
		return "", fmt.Errorf("create sandbox returned empty ID")
	}
	return *resp.SandboxId, nil
}

// DescribeSandbox returns the status of the specified sandbox.
func (m *SandboxManager) DescribeSandbox(sandboxID string) (*vefaas.DescribeSandboxOutput, error) {
	resp, err := m.client.DescribeSandbox(&vefaas.DescribeSandboxInput{
		FunctionId: volcengine.String(m.functionID),
		SandboxId:  volcengine.String(sandboxID),
	})
	if err != nil {
		return nil, fmt.Errorf("describe sandbox failed: %w", err)
	}
	return resp, nil
}

// KillSandbox terminates the specified sandbox.
func (m *SandboxManager) KillSandbox(sandboxID string) error {
	_, err := m.client.KillSandbox(&vefaas.KillSandboxInput{
		FunctionId: volcengine.String(m.functionID),
		SandboxId:  volcengine.String(sandboxID),
	})
	if err != nil {
		return fmt.Errorf("kill sandbox failed: %w", err)
	}
	return nil
}

// DataPlaneBaseURL returns the data plane base URL for the given sandbox,
// which can be passed to NewAIOSandboxBackend as BaseURL.
func (m *SandboxManager) DataPlaneBaseURL(gatewayURL, sandboxID string) string {
	u, err := url.Parse(gatewayURL)
	if err != nil {
		return fmt.Sprintf("%s?faasInstanceName=%s", gatewayURL, url.QueryEscape(sandboxID))
	}
	q := u.Query()
	q.Set("faasInstanceName", sandboxID)
	u.RawQuery = q.Encode()
	return u.String()
}
