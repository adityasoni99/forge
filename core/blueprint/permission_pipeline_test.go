package blueprint

import (
	"context"
	"testing"
)

func TestPermissionPipelineAllowByRule(t *testing.T) {
	rules := []PermissionRule{
		{Pattern: "build*", Decision: PermissionAllow},
	}
	pp := NewPermissionPipeline(rules, nil, true)
	node := &DeterministicNode{id: "build_step"}

	decision, err := pp.Check(context.Background(), node, NewRunState("test", "run-1"))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if decision != PermissionAllow {
		t.Errorf("decision = %v, want Allow", decision)
	}
}

func TestPermissionPipelineDenyByRule(t *testing.T) {
	rules := []PermissionRule{
		{Pattern: "deploy*", Decision: PermissionDeny},
	}
	pp := NewPermissionPipeline(rules, nil, true)
	node := &DeterministicNode{id: "deploy_prod"}

	decision, err := pp.Check(context.Background(), node, NewRunState("test", "run-1"))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if decision != PermissionDeny {
		t.Errorf("decision = %v, want Deny", decision)
	}
}

func TestPermissionPipelineAskBecomeDenyInHeadless(t *testing.T) {
	rules := []PermissionRule{
		{Pattern: "deploy*", Decision: PermissionAsk},
	}
	pp := NewPermissionPipeline(rules, nil, true)
	node := &DeterministicNode{id: "deploy_staging"}

	decision, err := pp.Check(context.Background(), node, NewRunState("test", "run-1"))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if decision != PermissionDeny {
		t.Errorf("decision = %v, want Deny (headless Ask->Deny)", decision)
	}
}

func TestPermissionPipelineAskDelegatesToHandler(t *testing.T) {
	rules := []PermissionRule{
		{Pattern: "deploy*", Decision: PermissionAsk},
	}
	handler := &mockApprovalHandler{approved: true, response: "go ahead"}
	pp := NewPermissionPipeline(rules, handler, false)
	node := &DeterministicNode{id: "deploy_staging"}

	decision, err := pp.Check(context.Background(), node, NewRunState("test", "run-1"))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if decision != PermissionAllow {
		t.Errorf("decision = %v, want Allow (handler approved)", decision)
	}
}

func TestPermissionPipelineNoMatchAllows(t *testing.T) {
	rules := []PermissionRule{
		{Pattern: "deploy*", Decision: PermissionDeny},
	}
	pp := NewPermissionPipeline(rules, nil, true)
	node := &DeterministicNode{id: "test_unit"}

	decision, err := pp.Check(context.Background(), node, NewRunState("test", "run-1"))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if decision != PermissionAllow {
		t.Errorf("decision = %v, want Allow (no matching rule)", decision)
	}
}
