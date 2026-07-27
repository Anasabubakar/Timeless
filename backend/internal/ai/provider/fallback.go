package provider

import (
	"context"
	"errors"
	"fmt"
)

// FallbackChain tries each provider in order and returns the first
// successful completion. If every provider fails, it returns a combined
// error so the caller can see what actually went wrong at each hop.
type FallbackChain struct {
	providers []Provider
}

func NewFallbackChain(providers ...Provider) *FallbackChain {
	return &FallbackChain{providers: providers}
}

func (f *FallbackChain) Name() string { return "auto" }

func (f *FallbackChain) Models() []string {
	var models []string
	for _, p := range f.providers {
		models = append(models, p.Models()...)
	}
	return models
}

func (f *FallbackChain) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	var errs []error
	for _, p := range f.providers {
		resp, err := p.Complete(ctx, req)
		if err == nil {
			return resp, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", p.Name(), err))
	}
	return nil, fmt.Errorf("all providers failed: %w", errors.Join(errs...))
}

func (f *FallbackChain) Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error) {
	var errs []error
	for _, p := range f.providers {
		ch, err := p.Stream(ctx, req)
		if err == nil {
			return ch, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", p.Name(), err))
	}
	return nil, fmt.Errorf("all providers failed: %w", errors.Join(errs...))
}
