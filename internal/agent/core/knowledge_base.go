package core

import (
	"context"
	"review/internal/biz"
)

// KnowledgeBase handles querying the knowledge base.
type KnowledgeBase struct {
	knowledge *biz.KnowledgeUsecase
}

// NewKnowledgeBase creates a new KnowledgeBase.
func NewKnowledgeBase(uc *biz.KnowledgeUsecase) *KnowledgeBase {
	return &KnowledgeBase{
		knowledge: uc,
	}
}

// Query searches the knowledge base based on the user's intent.
func (kb *KnowledgeBase) Query(ctx context.Context, intent *Intent) (*Response, error) {
	results, err := kb.knowledge.Search(ctx, intent.Query)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return &Response{Reply: "抱歉，我暂时无法回答这个问题。"}, nil
	}

	// For simplicity, we return the first result.
	// A more advanced implementation could synthesize an answer from multiple results.
	return &Response{Reply: results[0].Answer}, nil
}