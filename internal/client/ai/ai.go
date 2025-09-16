package ai

import (
	"context"
	"review/internal/conf"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type AIClient struct {
	llm *openai.LLM
}

func NewAIClient(c *conf.AI) (*AIClient, error) {
	// 使用OpenAI兼容的客户端连接Qwen
	// 注意：Qwen使用Authorization: Bearer <api_key>的方式
	llm, err := openai.New(
		openai.WithToken(c.ApiKey),
		openai.WithModel(c.Model),
		openai.WithBaseURL(c.BaseUrl), // 使用Qwen的API端点
	)
	if err != nil {
		return nil, err
	}
	return &AIClient{llm: llm}, nil
}

// GetLLM 获取LLM实例
func (c *AIClient) GetLLM() *openai.LLM {
	return c.llm
}

// ModerateText 使用LLM审核文本内容
// 返回值: is_approved, reason
func (c *AIClient) ModerateText(ctx context.Context, text string) (bool, string, error) {
	prompt := `你是一个严格的内容审核员。你的任务是判断给定的评论是否包含不当内容。

不当内容主要分为以下几类：
- 辱骂：包含人身攻击、侮辱性言论或粗俗语言。
- 广告：推广产品、服务或网站，包含链接或联系方式。
- 垃圾信息：无意义的字符、重复文本或与主题无关的内容。
- 色情：涉及露骨的性描述或性暗示。
- 暴力：宣扬、描述或鼓励暴力行为。
- 其他：包含不当内容，如政治敏感话题、宗教敏感话题、种族歧视、性别歧视、地域歧视等。

你的输出必须严格遵循以下格式：
- 如果评论内容得当，只回答“是”。
- 如果评论内容不当，回答“否”，然后紧跟一个冒号“：”，并用一句话简要说明理由。

示例 1:
[评论内容]: "这个产品真是太棒了，强烈推荐！"
你的回答: 是

示例 2:
[评论内容]: "想赚钱吗？快来加我VX: 123456"
你的回答: 否：包含广告和联系方式。

示例 3:
[评论内容]: "方却无法前期亲子课女郎尾气污染"
你的回答: 否：包含垃圾信息。

现在，请审核以下评论：
[评论内容]: "` + text + `"`

	completion, err := llms.GenerateFromSinglePrompt(ctx, c.llm, prompt)
	if err != nil {
		// 如果AI服务出错，默认通过审核，记录错误
		return true, "AI service error, content approved by default: " + err.Error(), nil
	}

	// 这里需要添加解析AI返回结果的逻辑
	// 为简化起见，我们假设如果返回以"是"开头则为通过
	if strings.HasPrefix(completion, "是") {
		return true, "Content approved by AI.", nil
	}
	// 如果以"否"开头，提取并返回后面的理由
	if strings.HasPrefix(completion, "否") {
		reason := strings.TrimSpace(strings.TrimPrefix(completion, "否"))
		// 移除可能的前缀，如冒号或逗号
		reason = strings.TrimPrefix(reason, "，")
		reason = strings.TrimPrefix(reason, ",")
		reason = strings.TrimPrefix(reason, "：")
		reason = strings.TrimPrefix(reason, ":")
		// 如果理由为空，提供一个默认理由
		if reason == "" {
			reason = "内容不当，但未提供具体理由。"
		}
		return false, reason, nil
	}
	// 如果AI返回的格式不符合预期，默认通过
	return true, "AI response format unexpected, content approved by default", nil
}
