package plugins

import "github.com/alltomatos/watinkdev/business/pkg/sdk"

// AssistantPlugin — "Assistentes de IA". CRUD only here; runtime dispatch
// (synthetic Flow, Agent Runtime reuse, Pipeline events) lands in later
// issues. See docs/agents/assistants.md and ADR 0027.
type AssistantPlugin struct{}

func (ap *AssistantPlugin) GetManifest() sdk.PluginManifest {
	return sdk.PluginManifest{
		Slug:        "assistant",
		Name:        "Assistentes de IA",
		Version:     "1.0.0",
		Description: "Automação de conversas por IA — pipeline, flow, persona ou roteador",
		Type:        "pro",
	}
}

func (ap *AssistantPlugin) OnInstall(core sdk.WatinkCore) error {
	return nil
}

func (ap *AssistantPlugin) OnActivate(core sdk.WatinkCore) error {
	ac := NewAssistantController()
	rc := NewAssistantRouterController()
	gc := NewAiGatewayController()
	agc := NewAssistantGroupController()

	core.RegisterRoute("GET", "/assistants", withPermission("assistants", "read", ac.List))
	core.RegisterRoute("POST", "/assistants", withPermission("assistants", "manage", ac.Create))
	core.RegisterRoute("GET", "/assistants/:id", withPermission("assistants", "read", ac.Get))
	core.RegisterRoute("PUT", "/assistants/:id", withPermission("assistants", "manage", ac.Update))
	core.RegisterRoute("DELETE", "/assistants/:id", withPermission("assistants", "manage", ac.Delete))
	core.RegisterRoute("POST", "/assistants/:id/duplicate", withPermission("assistants", "manage", ac.Duplicate))
	core.RegisterRoute("POST", "/assistants/:id/test", withPermission("assistants", "manage", ac.Test))

	core.RegisterRoute("GET", "/assistants/:id/groups", withPermission("assistants", "read", agc.List))
	core.RegisterRoute("PUT", "/assistants/:id/groups", withPermission("assistants", "manage", agc.Toggle))

	core.RegisterRoute("GET", "/assistants/:id/router-options", withPermission("assistants", "read", rc.List))
	core.RegisterRoute("POST", "/assistants/:id/router-options", withPermission("assistants", "manage", rc.Create))
	core.RegisterRoute("PUT", "/assistants/:id/router-options/:optionId", withPermission("assistants", "manage", rc.Update))
	core.RegisterRoute("DELETE", "/assistants/:id/router-options/:optionId", withPermission("assistants", "manage", rc.Delete))

	core.RegisterRoute("GET", "/ai-gateways", withPermission("assistants", "read", gc.List))
	core.RegisterRoute("POST", "/ai-gateways", withPermission("assistants", "manage", gc.Create))
	core.RegisterRoute("GET", "/ai-gateways/:id", withPermission("assistants", "read", gc.Get))
	core.RegisterRoute("PUT", "/ai-gateways/:id", withPermission("assistants", "manage", gc.Update))
	core.RegisterRoute("DELETE", "/ai-gateways/:id", withPermission("assistants", "manage", gc.Delete))
	core.RegisterRoute("POST", "/ai-gateways/:id/test", withPermission("assistants", "manage", gc.Test))
	core.RegisterRoute("GET", "/ai-gateways/:id/models", withPermission("assistants", "read", gc.ListModels))

	registerAssistantPipelineEvents(core, core.GetDB())
	registerAssistantIdleSweep(core, core.GetDB())

	return nil
}

func (ap *AssistantPlugin) OnDeactivate(core sdk.WatinkCore) error {
	return nil
}
