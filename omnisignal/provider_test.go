package omnisignal_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/plexusone/omnisignal"
	"github.com/plexusone/signal-spec/pkg/common"
	"github.com/plexusone/signal-spec/pkg/signal"
	"github.com/plexusone/signal-spec/schema"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"

	_ "github.com/grokify/aha-studio/omnisignal" // Register provider
)

func TestProviderRegistration(t *testing.T) {
	if !omnisignal.IsRegistered("aha") {
		t.Fatal("aha provider not registered")
	}
}

func TestProviderCapabilities(t *testing.T) {
	provider, err := omnisignal.New("aha", omnisignal.Config{
		APIKey: "test-key",
		Options: map[string]any{
			"subdomain": "testcompany",
		},
	})
	if err != nil {
		t.Fatalf("creating provider: %v", err)
	}
	defer provider.Close()

	caps := provider.Capabilities()

	if len(caps.SignalTypes) != 1 || caps.SignalTypes[0] != signal.TypeEnhancementRequest {
		t.Errorf("SignalTypes = %v, want [enhancement_request]", caps.SignalTypes)
	}
	if caps.MaxBatchSize != 200 {
		t.Errorf("MaxBatchSize = %d, want 200", caps.MaxBatchSize)
	}
	if caps.RateLimitPerMinute != 300 {
		t.Errorf("RateLimitPerMinute = %d, want 300", caps.RateLimitPerMinute)
	}
	if !caps.SupportsBatchFetch {
		t.Error("SupportsBatchFetch should be true")
	}
	if caps.SupportsStreaming {
		t.Error("SupportsStreaming should be false")
	}
}

func TestAhaSignalSchemaConformance(t *testing.T) {
	var raw any
	if err := json.Unmarshal(schema.SignalSchema, &raw); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("signal.schema.json", raw); err != nil {
		t.Fatalf("add schema: %v", err)
	}
	sch, err := c.Compile("signal.schema.json")
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)

	sig := signal.Signal{
		ID:     "aha-IDEA-100",
		Type:   signal.TypeEnhancementRequest,
		Status: signal.StatusNew,
		Source: common.SourceSystem{
			Type:       "product_management",
			Name:       "aha",
			ExternalID: "IDEA-100",
		},
		Domain: common.Domain{
			Name:      "product",
			Subdomain: "integrations",
		},
		Severity:    common.SeverityMedium,
		Summary:     "Add Slack integration for notifications",
		Description: "Would be great to get notifications in Slack when ideas are updated",
		ObservedAt:  now.Add(-7 * 24 * time.Hour),
		ReceivedAt:  now,
		Metadata: map[string]any{
			signal.MetaVotes:         42,
			omnisignal.MetaCurated:   true,
			"aha_reference_num":      "IDEA-100",
			"aha_score":              85,
			signal.MetaCapabilityRef: "capability:slack-integration",
		},
	}

	data, err := json.Marshal(sig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("unmarshal for validation: %v", err)
	}

	if err := sch.Validate(v); err != nil {
		t.Errorf("schema validation failed: %v", err)
	}
}
