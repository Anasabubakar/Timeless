package agent

import (
	"fmt"

	"github.com/timeless/backend/internal/ai/provider"
)

// ToolAllowlist gates which named tools a model is permitted to invoke.
// No agent in this codebase currently populates CompletionRequest.Tools
// or acts on CompletionResponse.ToolCalls — the provider-level Tool/
// ToolCall types exist but nothing wires them up yet. This exists so
// that whenever tool-calling is added, whatever executes a tool call
// validates it against an explicit allowlist by construction, rather
// than trusting whatever name/arguments the model returned — an LLM's
// tool-call output is model-generated content, not a trusted
// instruction, and should be checked the same way any other untrusted
// input at a system boundary would be.
type ToolAllowlist struct {
	allowed map[string]bool
}

func NewToolAllowlist(names ...string) *ToolAllowlist {
	allowed := make(map[string]bool, len(names))
	for _, n := range names {
		allowed[n] = true
	}
	return &ToolAllowlist{allowed: allowed}
}

// IsAllowed reports whether name is a tool this allowlist permits.
func (a *ToolAllowlist) IsAllowed(name string) bool {
	return a.allowed[name]
}

// ValidateCall checks a model-returned tool call against the allowlist
// before anything executes it. A tool call for a name not on the list
// is rejected outright — the model doesn't get to introduce a new
// capability just by naming one in its response.
func (a *ToolAllowlist) ValidateCall(call provider.ToolCall) error {
	if call.Name == "" {
		return fmt.Errorf("tool call has no name")
	}
	if !a.IsAllowed(call.Name) {
		return fmt.Errorf("tool %q is not on the allowlist", call.Name)
	}
	return nil
}
