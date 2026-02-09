package resultutil

import (
	"encoding/json"
	"errors"
	"testing"
)

// Example output types (similar to what's used in the handlers)
type ExampleOutput struct {
	Message string   `json:"message"`
	Items   []string `json:"items"`
}

func TestNewSuccessResult(t *testing.T) {
	output := ExampleOutput{
		Message: "test message",
		Items:   []string{"item1", "item2"},
	}

	result := NewSuccessResult(output)

	if result.IsError() {
		t.Errorf("expected success result, got error: %v", result.Error)
	}

	if result.Data == nil {
		t.Error("expected Data to be set")
	}

	if result.JSONText == "" {
		t.Error("expected JSONText to be set")
	}

	// Verify JSON is valid and matches the data
	var decoded ExampleOutput
	if err := json.Unmarshal([]byte(result.JSONText), &decoded); err != nil {
		t.Errorf("failed to unmarshal JSONText: %v", err)
	}

	if decoded.Message != output.Message {
		t.Errorf("expected message %q, got %q", output.Message, decoded.Message)
	}
}

func TestNewErrorResult(t *testing.T) {
	errorMsg := "test error message"
	result := NewErrorResult(errors.New(errorMsg))

	if !result.IsError() {
		t.Error("expected error result")
	}

	if result.Error == nil {
		t.Error("expected Error to be set")
	}

	if result.Error.Error() != errorMsg {
		t.Errorf("expected error message %q, got %q", errorMsg, result.Error.Error())
	}

	if result.Data != nil {
		t.Error("expected Data to be nil for error result")
	}
}

func TestToMCPResult_Success(t *testing.T) {
	output := ExampleOutput{
		Message: "test",
		Items:   []string{"a", "b"},
	}

	result := NewSuccessResult(output)
	mcpResult, err := result.ToMCPResult()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if mcpResult == nil {
		t.Fatal("expected non-nil MCP result")
	}

	// The MCP result should contain the structured data
	if mcpResult.Content == nil {
		t.Error("expected MCP result content to be set")
	}
}

func TestToMCPResult_Error(t *testing.T) {
	result := NewErrorResult(errors.New("test error"))
	mcpResult, err := result.ToMCPResult()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if mcpResult == nil {
		t.Fatal("expected non-nil MCP result")
	}

	// MCP error results should have isError set to true
	if !mcpResult.IsError {
		t.Error("expected MCP result to have IsError=true")
	}
}

func TestToToolsetResult_Success(t *testing.T) {
	output := ExampleOutput{
		Message: "test",
		Items:   []string{"a", "b"},
	}

	result := NewSuccessResult(output)
	toolsetResult, err := result.ToToolsetResult()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if toolsetResult == nil {
		t.Fatal("expected non-nil Toolset result")
	}

	// The Toolset result should contain the JSON text
	if toolsetResult.Error != nil {
		t.Errorf("expected no error in result, got: %v", toolsetResult.Error)
	}

	if toolsetResult.Content == "" {
		t.Error("expected content to be set")
	}

	// Verify the content is valid JSON
	var decoded ExampleOutput
	if err := json.Unmarshal([]byte(toolsetResult.Content), &decoded); err != nil {
		t.Errorf("failed to unmarshal content: %v", err)
	}
}

func TestToToolsetResult_Error(t *testing.T) {
	errorMsg := "test error"
	result := NewErrorResult(errors.New(errorMsg))
	toolsetResult, err := result.ToToolsetResult()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if toolsetResult == nil {
		t.Fatal("expected non-nil Toolset result")
	}

	// The Toolset result should contain the error
	if toolsetResult.Error == nil {
		t.Fatal("expected error in result")
	}

	if toolsetResult.Error.Error() != errorMsg {
		t.Errorf("expected error message %q, got %q", errorMsg, toolsetResult.Error.Error())
	}
}

func TestMarshalError(t *testing.T) {
	// Create a type that can't be marshaled to JSON
	type UnmarshalableType struct {
		Channel chan int // channels can't be marshaled to JSON
	}

	result := NewSuccessResult(UnmarshalableType{Channel: make(chan int)})

	if !result.IsError() {
		t.Error("expected error result when marshaling fails")
	}

	if result.Error == nil {
		t.Error("expected Error to be set")
	}
}
