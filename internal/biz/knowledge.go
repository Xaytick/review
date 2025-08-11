package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

// Knowledge represents a single entry in the knowledge base.
type Knowledge struct {
	ID       int64
	Question string
	Answer   string
	Category string
}

// KnowledgeRepo is the repository interface for the knowledge base.
type KnowledgeRepo interface {
	Search(ctx context.Context, query string) ([]*Knowledge, error)
}

// KnowledgeUsecase is a Knowledge usecase.
type KnowledgeUsecase struct {
	repo KnowledgeRepo
	log  *log.Helper
}

// NewKnowledgeUsecase creates a new KnowledgeUsecase.
func NewKnowledgeUsecase(repo KnowledgeRepo, logger log.Logger) *KnowledgeUsecase {
	return &KnowledgeUsecase{repo: repo, log: log.NewHelper(logger)}
}

// Search searches the knowledge base.
func (uc *KnowledgeUsecase) Search(ctx context.Context, query string) ([]*Knowledge, error) {
	return uc.repo.Search(ctx, query)
}

// ToolRequest represents a request to call a tool.
type ToolRequest struct {
	ToolName string `json:"tool_name"`
	Query    string `json:"query"`
}

// ToolResponse represents the response from a tool call.
type ToolResponse struct {
	Result string `json:"result"`
}

// AgentResponse represents the response from the agent.	
type AgentResponse struct {
	Answer string  `json:"answer"`
	Tools  []*Tool `json:"tools"`
}

// Tool represents a tool that the agent can use.
type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Endpoint    string `json:"endpoint"`
}