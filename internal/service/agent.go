package service

import (
	"context"

	pb "review/api/ai/v1"
	"review/internal/biz"
)

// AgentService is the service for AI agent.
type AgentService struct {
	pb.UnimplementedAgentServiceServer

	uc *biz.AgentUsecase
}

// NewAgentService creates a new agent service.
func NewAgentService(uc *biz.AgentUsecase) *AgentService {
	return &AgentService{uc: uc}
}

// Process handles the user's natural language query.
func (s *AgentService) Process(ctx context.Context, req *pb.ProcessRequest) (*pb.ProcessResponse, error) {
	return s.uc.Process(ctx, req.SessionId, req.Query)
}

// CallTool executes a specific tool.
func (s *AgentService) CallTool(ctx context.Context, req *pb.CallToolRequest) (*pb.CallToolResponse, error) {
	result, err := s.uc.CallTool(ctx, req.ToolName, req.Arguments, req.OriginalQuery)
	if err != nil {
		return nil, err
	}
	return &pb.CallToolResponse{Result: result}, nil
}
