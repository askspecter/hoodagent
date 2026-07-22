package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/askspecter/holt/internal/agent"
	"github.com/askspecter/holt/internal/config"
	"github.com/askspecter/holt/internal/holtruntime"
	"github.com/askspecter/holt/internal/mcp"
	"github.com/askspecter/holt/internal/modelregistry"
	"github.com/askspecter/holt/internal/providerhealth"
	"github.com/askspecter/holt/internal/providermodeldiscovery"
	"github.com/askspecter/holt/internal/sandbox"
	"github.com/askspecter/holt/internal/sessions"
	"github.com/askspecter/holt/internal/tools"
	"github.com/askspecter/holt/internal/usage"
)

// Options configures the reusable Holt terminal UI shell.
type Options struct {
	Cwd                         string
	UserConfigPath              string
	DoctorUserConfigPath        string
	ProjectConfigPath           string
	ProviderName                string
	ModelName                   string
	ProviderProfile             config.ProviderProfile
	SavedProviders              []config.ProviderProfile // all configured providers, for the /model multi-provider list
	FavoriteModels              []string
	RecapsEnabled               bool
	Provider                    holtruntime.Provider
	NewProvider                 func(config.ProviderProfile) (holtruntime.Provider, error)
	ProbeProviderHealth         func(context.Context, providerhealth.Options) providerhealth.Result
	DiscoverProviderModels      func(context.Context, config.ProviderProfile) ([]providermodeldiscovery.Model, error)
	DiscoverOllamaContextWindow func(ctx context.Context, baseURL string, model string) (int, error)
	RuntimeMessageSink          func(tea.Msg)
	Registry                    *tools.Registry
	SessionStore                *sessions.Store
	SandboxStore                *sandbox.GrantStore
	MCPConfig                   config.MCPConfig
	MCPPermissionStore          *mcp.PermissionStore
	MCPTokenStore               *mcp.TokenStore
	MCPCommand                  func(context.Context, []string) MCPCommandResult
	SandboxSetupCommand         func(context.Context) SandboxSetupCommandResult
	UsageTracker                *usage.Tracker
	SessionCompactor            SessionCompactor
	PrService                   *PrService

	AgentOptions    agent.Options
	PermissionMode  agent.PermissionMode
	ReasoningEffort modelregistry.ReasoningEffort
	ResponseStyle   string
	// Theme is the operator's palette preference: "auto" (default), a built-in
	// ("dark"/"light"), or a registered color theme. Set from the --theme flag;
	// falls back to HOLT_THEME, then the persisted SavedTheme, then auto.
	Theme string
	// SavedTheme is the theme persisted in user config (Preferences.Theme). Applied
	// at startup below --theme and HOLT_THEME, so a /theme choice survives restart.
	SavedTheme string
	UserAgent  string

	// Notify configures completion / awaiting-input notifications.
	Notify config.NotifyConfig

	// AltScreen tells the model it is running inside Bubble Tea's alternate
	// screen. Run sets this for the interactive app; tests can leave it false
	// to exercise the native scrollback renderer.
	AltScreen bool

	// Setup configures the first-run/setup takeover. It is shown before the
	// normal chat surface when Visible is true.
	Setup SetupOptions
}

type MCPCommandResult struct {
	Config   config.MCPConfig
	Output   string
	Error    string
	ExitCode int
}

type SandboxSetupCommandResult struct {
	Output   string
	Error    string
	ExitCode int
}

// SetupOptions configures the guided first-run provider setup takeover.
type SetupOptions struct {
	Visible    bool
	Required   bool
	ConfigPath string
	Providers  []SetupProviderOption
	Save       func(SetupSelection) (SetupResult, error)
}

// SetupProviderOption is one provider choice offered by the setup takeover.
type SetupProviderOption struct {
	ID           string
	Name         string
	DefaultModel string
	EnvVar       string
	RequiresAuth bool
	Local        bool
	Recommended  bool
}

// SetupSelection is the user's setup choice.
type SetupSelection struct {
	CatalogID string
	Name      string
	BaseURL   string
	Model     string
	APIKey    string
}

// SetupResult describes a completed setup write.
type SetupResult struct {
	ConfigPath string
	Provider   config.ProviderProfile
}
