package llm

import (
	"context"
	"testing"
)

type stubClient struct {
	generateResult string
	embedResult    []float32
	reachable      bool
	status         StatusInfo
}

func (s *stubClient) Generate(_ context.Context, _ string) (string, error) {
	return s.generateResult, nil
}

func (s *stubClient) Embed(_ context.Context, _ string) ([]float32, error) {
	return s.embedResult, nil
}

func (s *stubClient) Reachable(_ context.Context) (bool, error) {
	return s.reachable, nil
}

func (s *stubClient) Status(_ context.Context) StatusInfo {
	return s.status
}

func TestSwappableClient_DelegatesToInner(t *testing.T) {
	inner := &stubClient{
		generateResult: "hello",
		embedResult:    []float32{1.0, 2.0},
		reachable:      true,
		status:         StatusInfo{Provider: "ollama", Reachable: true},
	}
	sc := NewSwappable(inner)
	ctx := context.Background()

	got, err := sc.Generate(ctx, "prompt")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got != "hello" {
		t.Errorf("Generate = %q, want %q", got, "hello")
	}

	embed, err := sc.Embed(ctx, "text")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(embed) != 2 {
		t.Errorf("Embed len = %d, want 2", len(embed))
	}

	reachable, err := sc.Reachable(ctx)
	if err != nil {
		t.Fatalf("Reachable: %v", err)
	}
	if !reachable {
		t.Error("Reachable = false, want true")
	}

	status := sc.Status(ctx)
	if status.Provider != "ollama" {
		t.Errorf("Status.Provider = %q, want %q", status.Provider, "ollama")
	}
}

func TestSwappableClient_SwapChangesInner(t *testing.T) {
	first := &stubClient{generateResult: "first"}
	second := &stubClient{generateResult: "second"}

	sc := NewSwappable(first)
	ctx := context.Background()

	got, _ := sc.Generate(ctx, "")
	if got != "first" {
		t.Errorf("before swap: Generate = %q, want %q", got, "first")
	}

	sc.Swap(second)

	got, _ = sc.Generate(ctx, "")
	if got != "second" {
		t.Errorf("after swap: Generate = %q, want %q", got, "second")
	}
}

func TestSwappableClient_ImplementsLLMClient(t *testing.T) {
	inner := &stubClient{}
	sc := NewSwappable(inner)

	// Compile-time check: *SwappableClient satisfies LLMClient.
	var _ LLMClient = sc
}
