package integration

import (
	"context"
	"fmt"
	"sync"
)

type IntegrationType string

const (
	IntegrationNative  IntegrationType = "native"
	IntegrationZapier  IntegrationType = "zapier"
	IntegrationWebhook IntegrationType = "webhook"
	IntegrationMCP     IntegrationType = "mcp"
)

type Integration interface {
	Type() IntegrationType
	Name() string
	Connect(ctx context.Context, config map[string]string) error
	Disconnect(ctx context.Context) error
	Execute(ctx context.Context, action string, params map[string]interface{}) (map[string]interface{}, error)
	IsConnected() bool
}

type Config struct {
	ID             string            `json:"id"`
	OrgID          string            `json:"org_id"`
	Type           IntegrationType   `json:"type"`
	Name           string            `json:"name"`
	Credentials    map[string]string `json:"credentials"`
	Settings       map[string]string `json:"settings"`
	Enabled        bool              `json:"enabled"`
}

type Manager struct {
	mu           sync.RWMutex
	integrations map[string]Integration
	configs      map[string]*Config
}

func NewManager() *Manager {
	return &Manager{
		integrations: make(map[string]Integration),
		configs:      make(map[string]*Config),
	}
}

func (m *Manager) Register(name string, integration Integration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.integrations[name] = integration
}

func (m *Manager) Connect(ctx context.Context, orgID string, name string, credentials map[string]string) error {
	m.mu.RLock()
	integration, ok := m.integrations[name]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("integration %q not registered", name)
	}

	if err := integration.Connect(ctx, credentials); err != nil {
		return fmt.Errorf("connect %s: %w", name, err)
	}

	m.mu.Lock()
	m.configs[orgID+":"+name] = &Config{
		OrgID:       orgID,
		Type:        integration.Type(),
		Name:        name,
		Credentials: credentials,
		Enabled:     true,
	}
	m.mu.Unlock()

	return nil
}

func (m *Manager) Execute(ctx context.Context, orgID string, name string, action string, params map[string]interface{}) (map[string]interface{}, error) {
	m.mu.RLock()
	integration, ok := m.integrations[name]
	config := m.configs[orgID+":"+name]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("integration %q not registered", name)
	}
	if config == nil || !config.Enabled {
		return nil, fmt.Errorf("integration %q not configured for org %s", name, orgID)
	}

	return integration.Execute(ctx, action, params)
}

func (m *Manager) List(orgID string) []*Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var configs []*Config
	for key, cfg := range m.configs {
		if cfg.OrgID == orgID || key == orgID+":"+cfg.Name {
			configs = append(configs, cfg)
		}
	}
	return configs
}

func (m *Manager) ListAvailable() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.integrations))
	for name := range m.integrations {
		names = append(names, name)
	}
	return names
}
