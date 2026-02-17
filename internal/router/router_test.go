package router

import (
	"context"
	"testing"
)

func TestClassify_Keywords_Cost(t *testing.T) {
	r := New(nil, false)
	tests := []string{
		"how much does this cost?",
		"estimate the pricing for this deployment",
		"what is the budget impact?",
		"is this expensive?",
	}
	for _, msg := range tests {
		got := r.Classify(context.Background(), "", msg)
		if got != IntentCost {
			t.Errorf("Classify(%q) = %v, want cost", msg, got)
		}
	}
}

func TestClassify_Keywords_Ops(t *testing.T) {
	r := New(nil, false)
	tests := []string{
		"deploy this to staging",
		"promote to production",
		"check for drift",
		"send a notification",
		"rollback the release",
	}
	for _, msg := range tests {
		got := r.Classify(context.Background(), "", msg)
		if got != IntentOps {
			t.Errorf("Classify(%q) = %v, want ops", msg, got)
		}
	}
}

func TestClassify_Keywords_Status(t *testing.T) {
	r := New(nil, false)
	tests := []string{"what is the status?", "health check"}
	for _, msg := range tests {
		got := r.Classify(context.Background(), "", msg)
		if got != IntentStatus {
			t.Errorf("Classify(%q) = %v, want status", msg, got)
		}
	}
}

func TestClassify_Keywords_Help(t *testing.T) {
	r := New(nil, false)
	tests := []string{"help me", "how to use this", "what can you do?", "show me the documentation"}
	for _, msg := range tests {
		got := r.Classify(context.Background(), "", msg)
		if got != IntentHelp {
			t.Errorf("Classify(%q) = %v, want help", msg, got)
		}
	}
}

func TestClassify_Keywords_Analyze(t *testing.T) {
	r := New(nil, false)
	tests := []string{
		"scan this terraform code",
		"check my bicep template",
		"audit the compliance",
		"review this for security issues",
		"analyze my infrastructure",
	}
	for _, msg := range tests {
		got := r.Classify(context.Background(), "", msg)
		if got != IntentAnalyze {
			t.Errorf("Classify(%q) = %v, want analyze", msg, got)
		}
	}
}

func TestClassify_Keywords_UnknownFallsToHelp(t *testing.T) {
	r := New(nil, false)
	msg := "random question about nothing"
	got := r.Classify(context.Background(), "", msg)
	if got != IntentHelp {
		t.Errorf("Classify(%q) = %v, want help (fallback)", msg, got)
	}
}

func TestClassify_NoLLM_FallsBackToKeywords(t *testing.T) {
	r := New(nil, true)
	got := r.Classify(context.Background(), "", "scan this code")
	if got != IntentAnalyze {
		t.Errorf("Classify with nil LLM client = %v, want analyze (keyword fallback)", got)
	}
}

func TestIntentConstants(t *testing.T) {
	tests := []struct {
		intent Intent
		want   string
	}{
		{IntentAnalyze, "analyze"},
		{IntentCost, "cost"},
		{IntentOps, "ops"},
		{IntentStatus, "status"},
		{IntentHelp, "help"},
	}
	for _, tt := range tests {
		if string(tt.intent) != tt.want {
			t.Errorf("Intent %v = %q, want %q", tt.intent, string(tt.intent), tt.want)
		}
	}
}
