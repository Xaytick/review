package core

import (
	"context"
	"fmt"
	"review/internal/biz"
)

// Response represents the output from the agent's processing.
type Response struct {
	Reply string      // The textual content to be shown to the user.
	Tools []*biz.Tool // A list of tools that can be used.
	Error error       // Any error that occurred during processing.
}

// Tool is an interface for any action the agent can perform.
type Tool interface {
	Name() string
	Call(ctx context.Context, args map[string]interface{}) (string, error)
}

// ToolManager manages the available tools and their execution.
type ToolManager struct {
	tools map[string]Tool
}

// NewToolManager creates a new ToolManager.
func NewToolManager() *ToolManager {
	return &ToolManager{
		tools: make(map[string]Tool),
	}
}

// RegisterTool adds a new tool to the manager.
func (tm *ToolManager) RegisterTool(tool Tool) {
	tm.tools[tool.Name()] = tool
}

// Execute runs the appropriate tool based on the intent.
func (tm *ToolManager) Execute(ctx context.Context, intent *Intent) (*Response, error) {
	tool, ok := tm.tools[intent.ToolName]
	if !ok {
		return nil, fmt.Errorf("tool '%s' not found", intent.ToolName)
	}

	result, err := tool.Call(ctx, intent.Arguments)
	if err != nil {
		return nil, fmt.Errorf("error executing tool '%s': %w", intent.ToolName, err)
	}

	return &Response{Reply: result}, nil
}