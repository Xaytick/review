package core

import (
	"context"

	"review/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type Engine struct {
	knowledge *biz.KnowledgeUsecase
	log       *log.Helper
	NLU       *NLUProcessor
	Tools     *ToolManager
}

func NewEngine(uc *biz.KnowledgeUsecase, logger log.Logger) *Engine {
	return &Engine{
		knowledge: uc,
		log:       log.NewHelper(logger),
		NLU:       NewNLUProcessor(),      // Initialize NLU processor
		Tools:     NewToolManager(),       // Initialize Tool manager
	}
}

func (e *Engine) Process(ctx context.Context, query string) (*biz.AgentResponse, error) {
	// 1. 自然语言理解
	intent := e.NLU.DetectIntent(query)

	// 2. 工具调用或知识检索
	if intent.NeedsTool {
		response, err := e.Tools.Execute(ctx, intent)
		if err != nil {
			return nil, err
		}
		return &biz.AgentResponse{Answer: response.Reply, Tools: response.Tools}, nil
	} else {
		kb := NewKnowledgeBase(e.knowledge)
		response, err := kb.Query(ctx, intent)
		if err != nil {
			return nil, err
		}
		return &biz.AgentResponse{Answer: response.Reply, Tools: response.Tools}, nil
	}
}

// CallTool is a placeholder for now.
func (e *Engine) CallTool(ctx context.Context, req *biz.ToolRequest) (*biz.ToolResponse, error) {
	e.log.WithContext(ctx).Infof("CallTool: %v", req)
	// In the future, this can be used for more complex tool-use scenarios.
	return &biz.ToolResponse{Result: "Tool executed successfully"}, nil
}
