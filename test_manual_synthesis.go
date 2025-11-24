package main

import (
	"context"
	"fmt"
	"log"

	"github.com/language-operator/language-operator/pkg/synthesis"
	"github.com/go-logr/logr"
)

func main() {
	// Mock chat model that just returns the fallback
	mockModel := &FallbackTestModel{}
	synthesizer := synthesis.NewSynthesizer(mockModel, logr.Discard())

	req := synthesis.AgentSynthesisRequest{
		Instructions: "review my spreadsheet daily at 4pm",
		Tools:        []string{"google-sheets", "email"},
		AgentName:    "test-agent",
		Namespace:    "default",
	}

	resp, err := synthesizer.SynthesizeAgent(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Generated DSL Code:")
	fmt.Println("==================")
	fmt.Println(resp.DSLCode)
}

// FallbackTestModel simulates template loading failure to test fallback
type FallbackTestModel struct{}

func (f *FallbackTestModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	// This will trigger template parse error, forcing fallback usage
	return nil, fmt.Errorf("mock template failure to test fallback")
}
