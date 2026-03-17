package cliproxy

import (
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/registry"
	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/config"
)

func TestRegisterModelsForAuth_CodexFreeLatestModelsDisabledByDefault(t *testing.T) {
	service := &Service{cfg: &config.Config{}}
	auth := &coreauth.Auth{
		ID:       "codex-free-default",
		Provider: "codex",
		Status:   coreauth.StatusActive,
		Attributes: map[string]string{
			"plan_type": "free",
		},
	}

	reg := registry.GetGlobalRegistry()
	reg.UnregisterClient(auth.ID)
	t.Cleanup(func() {
		reg.UnregisterClient(auth.ID)
	})

	service.registerModelsForAuth(auth)
	models := reg.GetModelsForClient(auth.ID)
	if !hasModelID(models, "gpt-5.2-codex") {
		t.Fatal("expected free plan to keep registering gpt-5.2-codex")
	}
	if hasModelID(models, "gpt-5.3-codex") {
		t.Fatal("expected gpt-5.3-codex to stay disabled for free plan by default")
	}
	if hasModelID(models, "gpt-5.4") {
		t.Fatal("expected gpt-5.4 to stay disabled for free plan by default")
	}
}

func TestRegisterModelsForAuth_CodexFreeLatestModelsEnabled(t *testing.T) {
	service := &Service{
		cfg: &config.Config{
			CodexFreeLatestModels: true,
		},
	}
	auth := &coreauth.Auth{
		ID:       "codex-free-latest",
		Provider: "codex",
		Status:   coreauth.StatusActive,
		Attributes: map[string]string{
			"plan_type": "free",
		},
	}

	reg := registry.GetGlobalRegistry()
	reg.UnregisterClient(auth.ID)
	t.Cleanup(func() {
		reg.UnregisterClient(auth.ID)
	})

	service.registerModelsForAuth(auth)
	models := reg.GetModelsForClient(auth.ID)
	if !hasModelID(models, "gpt-5.3-codex") {
		t.Fatal("expected gpt-5.3-codex to be registered when codex-free-latest-models is enabled")
	}
	if !hasModelID(models, "gpt-5.4") {
		t.Fatal("expected gpt-5.4 to be registered when codex-free-latest-models is enabled")
	}
}

func hasModelID(models []*ModelInfo, want string) bool {
	for _, model := range models {
		if model != nil && model.ID == want {
			return true
		}
	}
	return false
}
