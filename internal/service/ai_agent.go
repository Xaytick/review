package service

import (
	"context"
	pb "review/api/ai/v1"
	"review/internal/client/ai"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
)

type AIAgentService struct {
	pb.UnimplementedAIAgentServer
	aiClient *ai.AIClient
	log      *log.Helper
}

func NewAIAgentService(aiClient *ai.AIClient, logger log.Logger) *AIAgentService {
	return &AIAgentService{aiClient: aiClient, log: log.NewHelper(logger)}
}

func (s *AIAgentService) Chat(ctx context.Context, req *pb.ChatRequest) (*pb.ChatReply, error) {
	// 构建提示：结合上下文和用户输入，并指定语言
	fullPrompt := "你是一个乐于助人的评论服务助手，请优先使用中文回答。之前的对话内容:\n" + strings.Join(req.Context, "\n") + "\n\n用户提问: " + req.Prompt

	response, err := s.aiClient.GetLLM().Call(ctx, fullPrompt)
	if err != nil {
		s.log.Errorf("AI generation failed: %v", err)
		return nil, err
	}

	return &pb.ChatReply{Response: response}, nil
}
