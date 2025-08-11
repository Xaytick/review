package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"review/internal/client/ai"
	"strconv"
	"strings"
	"sync"

	pb "review/api/ai/v1"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tmc/langchaingo/llms"
)

// AgentUsecase is the usecase for AI agent.
type AgentUsecase struct {
	log      *log.Helper
	aiClient *ai.AIClient
	reviewUC *ReviewUsecase // Dependency on ReviewUsecase
	// simple in-memory memory store: sessionID -> messages
	memMu  sync.RWMutex
	memory map[string][]message
}

// NewAgentUsecase creates a new agent usecase.
func NewAgentUsecase(logger log.Logger, aiClient *ai.AIClient, reviewUC *ReviewUsecase) *AgentUsecase {
	return &AgentUsecase{
		log:      log.NewHelper(logger),
		aiClient: aiClient,
		reviewUC: reviewUC,
		memory:   make(map[string][]message),
	}
}

type message struct {
	Role string `json:"role"` // user | assistant
	Text string `json:"text"`
}

// Process handles the core logic of the agent by calling an LLM with conversation memory.
func (uc *AgentUsecase) Process(ctx context.Context, sessionID, query string) (*pb.ProcessResponse, error) {
	uc.log.WithContext(ctx).Infof("Processing query with LLM: %s", query)

	// Get user from context to personalize tools
	user, err := userFromContext(ctx)
	var tools string
	if err == nil { // If user is logged in
		tools = getToolsForRole(user.Role)
	} else { // Fallback for unauthenticated users or errors
		uc.log.WithContext(ctx).Warnf("Could not get user from context, falling back to public. Error: %v", err)
		tools = getToolsForRole("public")
	}

	history := uc.getHistory(sessionID)
	prompt := buildSystemPromptWithMemory(tools, history, query)

	llmResponse, err := llms.GenerateFromSinglePrompt(
		ctx,
		uc.aiClient.GetLLM(),
		prompt,
	)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("LLM generation failed: %v", err)
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}
	uc.log.WithContext(ctx).Infof("LLM raw response: %s", llmResponse)

	// parse model output
	resp, err := parseLLMResponse(llmResponse)
	if err == nil {
		// persist memory
		uc.appendHistory(sessionID, message{Role: "user", Text: query})
		if resp.FinalAnswer != "" {
			uc.appendHistory(sessionID, message{Role: "assistant", Text: resp.FinalAnswer})
		}
	}
	return resp, err
}

// CallTool executes the tool with RBAC checks.
func (uc *AgentUsecase) CallTool(ctx context.Context, toolName, arguments, originalQuery string) (string, error) {
	uc.log.WithContext(ctx).Infof("Calling tool: %s with args: %s for query: %s", toolName, arguments, originalQuery)

	user, err := userFromContext(ctx)
	if user.Role != "customer" && user.Role != "merchant" && user.Role != "reviewer" {
		return "", errors.Forbidden("FORBIDDEN", "invalid role")
	}

	var rawResult any

	switch toolName {
	case "GetReview":
		// Allowed for all logged-in users
		var args struct {
			ReviewID string `json:"reviewID"`
		}
		if err = json.Unmarshal([]byte(arguments), &args); err != nil {
			return "", errors.BadRequest("INVALID_ARGUMENTS", "无法解析GetReview的参数")
		}
		reviewID, err := strconv.ParseInt(args.ReviewID, 10, 64)
		if err != nil {
			return "", errors.BadRequest("INVALID_ARGUMENTS", "reviewID必须是有效的数字")
		}
		rawResult, err = uc.reviewUC.GetReview(ctx, reviewID)

	case "ListReviewByStoreID":
		var args struct {
			StoreID string `json:"storeID"`
		}
		if err = json.Unmarshal([]byte(arguments), &args); err != nil {
			return "", errors.BadRequest("INVALID_ARGUMENTS", "无法解析ListReviewByStoreID的参数")
		}
		storeID, err := strconv.ParseInt(args.StoreID, 10, 64)
		if err != nil {
			return "", errors.BadRequest("INVALID_ARGUMENTS", "storeID必须是有效的数字")
		}

		// Enhanced RBAC check: Merchants can only list reviews for their own store.
		if user.Role == "merchant" && user.StoreID != storeID {
			return "", errors.Forbidden("FORBIDDEN", "商家只能查询自己店铺的评论")
		}

		rawResult, err = uc.reviewUC.ListReviewByStoreID(ctx, storeID, 1, 10)

	case "ListMyReviews":
		// RBAC: This tool is implicitly for the logged-in user, role check in getToolsForRole
		if user.Role != "customer" {
			return "", errors.Forbidden("FORBIDDEN", "只有顾客才能查询自己的评论")
		}
		rawResult, err = uc.reviewUC.ListReviewByUserID(ctx, user.UserID, 1, 10)

	default:
		return "", errors.NotFound("TOOL_NOT_FOUND", fmt.Sprintf("未找到名为 '%s' 的工具", toolName))
	}

	if err != nil {
		return "", err
	}

	return uc.summarizeResult(ctx, originalQuery, rawResult)
}

// summarizeResult sends the tool's output and original query to the LLM for a context-aware summary.
func (uc *AgentUsecase) summarizeResult(ctx context.Context, originalQuery string, result any) (string, error) {
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal tool result: %w", err)
	}

	summaryPrompt := fmt.Sprintf(`
你是一个乐于助人的AI助手Cortex。一个工具已经运行完毕，并返回了以下的JSON数据。
你的任务是根据用户的“原始问题”，从这些JSON数据中提取用户最关心的信息，并组织成一段清晰、友好、易于理解的自然语言回复。
不要杜撰JSON中不存在的信息。直接呈现核心信息即可,优先使用分点作答的格式。

用户的原始问题: "%s"

工具返回的JSON数据:
%s

请根据用户的原始问题，生成你的自然语言回复。
`, originalQuery, string(resultBytes))

	summary, err := llms.GenerateFromSinglePrompt(ctx, uc.aiClient.GetLLM(), summaryPrompt)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("LLM summarization failed: %v", err)
		return string(resultBytes), nil
	}

	uc.log.WithContext(ctx).Infof("LLM summary: %s", summary)
	return summary, nil
}

// func buildSystemPrompt(tools, query string) string {
// 	return fmt.Sprintf(`
// 你是一个强大的人工智能助手，你的名字叫 Cortex。你的任务是帮助用户与评论系统进行交互。
// 你必须遵循以下规则：
// 1. 分析用户的查询，判断是应该直接回答，还是应该使用工具来获取信息。
// 2. 如果你需要使用工具，你必须在思考(thought)后，从下面提供的可用工具列表中选择一个，并生成一个符合该工具参数格式的JSON对象。
// 3. 你的输出必须是一个单一的、可被解析的JSON对象，不得包含任何JSON以外的额外文本、解释或注释。
// 4. 如果用户的意图不明确或缺少必要信息，你应该直接回答，向用户提问以获取更多信息。
// 5. 如果用户的查询与评论系统无关，你应该直接回答。

// 可用工具列表:
// %s

// 用户的查询: "%s"

// 请严格按照以下格式输出JSON：
// {
//   "thought": "这里是你的思考过程...",
//   "tool_call": { "tool_name": "...", "arguments": "{...}" }
// }
// 或者
// {
//   "thought": "这里是你的思考过程...",
//   "final_answer": "你的直接回答内容。"
// }

// 现在，请处理用户的查询。
// `, tools, query)
// }

// buildSystemPromptWithMemory builds a prompt that includes short conversation history.
func buildSystemPromptWithMemory(tools string, history []message, query string) string {
	var historyLines []string
	// keep last up to 6 turns (12 messages)
	start := 0
	if len(history) > 12 {
		start = len(history) - 12
	}
	for _, m := range history[start:] {
		prefix := "[用户]"
		if m.Role == "assistant" {
			prefix = "[Cortex]"
		}
		historyLines = append(historyLines, fmt.Sprintf("%s %s", prefix, m.Text))
	}
	joinedHistory := strings.Join(historyLines, "\n")
	if joinedHistory == "" {
		joinedHistory = "(无历史对话)"
	}
	return fmt.Sprintf(`
你是一个强大的人工智能助手，你的名字叫 Cortex。你的任务是帮助用户与评论系统进行交互。
你必须遵循以下规则：
1. 结合对话上下文回答问题；若需要数据请调用工具。
2. 如果你需要使用工具，你必须在思考(thought)后，从下面提供的可用工具列表中选择一个，并生成一个符合该工具参数格式的JSON对象。
3. 你的输出必须是一个单一的、可被解析的JSON对象，不得包含任何JSON以外的额外文本、解释或注释。
4. 如果用户的意图不明确或缺少必要信息，你应该直接回答，向用户提问以获取更多信息。
5. 如果用户的查询与评论系统无关，你应该直接回答。

对话历史：
%s

可用工具列表:
%s

用户的查询: "%s"

请严格按照以下格式输出JSON：
{
  "thought": "这里是你的思考过程...",
  "tool_call": { "tool_name": "...", "arguments": "{...}" }
}
或者
{
  "thought": "这里是你的思考过程...",
  "final_answer": "你的直接回答内容。"
}

现在，请处理用户的查询。
`, joinedHistory, tools, query)
}

func (uc *AgentUsecase) getHistory(sessionID string) []message {
	if sessionID == "" {
		return nil
	}
	uc.memMu.RLock()
	defer uc.memMu.RUnlock()
	return append([]message(nil), uc.memory[sessionID]...)
}

func (uc *AgentUsecase) appendHistory(sessionID string, msg message) {
	if sessionID == "" {
		return
	}
	uc.memMu.Lock()
	defer uc.memMu.Unlock()
	uc.memory[sessionID] = append(uc.memory[sessionID], msg)
	// cap at 100 messages to prevent unbounded growth
	if len(uc.memory[sessionID]) > 100 {
		uc.memory[sessionID] = uc.memory[sessionID][len(uc.memory[sessionID])-100:]
	}
}

func getToolsForRole(role string) string {
	// Base tools available to any logged-in user with specific roles
	baseTools := []string{
		`{
			"name": "GetReview",
			"description": "根据评论ID获取单条评论的详细信息。",
			"parameters": { "type": "object", "properties": { "reviewID": { "type": "string", "description": "评论的唯一ID" } }, "required": ["reviewID"] }
		}`,
		`{
			"name": "ListReviewByStoreID",
			"description": "根据店铺ID查询该店铺的评论列表。商家只能查询自己店铺的评论。",
			"parameters": { "type": "object", "properties": { "storeID": { "type": "string", "description": "店铺的唯一ID" } }, "required": ["storeID"] }
		}`,
	}

	// Customer-specific tools
	customerTools := []string{
		`{
			"name": "ListMyReviews",
			"description": "查询我（当前登录用户）自己发布过的所有评论列表，不需要提供任何参数。",
			"parameters": { "type": "object", "properties": {} }
		}`,
	}

	log.Infof("Getting tools for role: '%s'", role)

	var tools []string
	switch role {
	case "customer":
		tools = append(baseTools, customerTools...)
	case "merchant", "reviewer":
		tools = baseTools
	default:
		// For "public" or any other role, no tools are available.
		return "[]"
	}

	return fmt.Sprintf("[%s]", strings.Join(tools, ","))
}

type llmOutput struct {
	Thought  string `json:"thought"`
	ToolCall *struct {
		ToolName  string `json:"tool_name"`
		Arguments string `json:"arguments"`
	} `json:"tool_call,omitempty"`
	FinalAnswer string `json:"final_answer,omitempty"`
}

func parseLLMResponse(response string) (*pb.ProcessResponse, error) {
	sanitizedResponse := strings.Trim(response, " \n\r\t`")
	sanitizedResponse = strings.TrimPrefix(sanitizedResponse, "json")
	sanitizedResponse = strings.Trim(sanitizedResponse, " \n\r\t")

	var output llmOutput
	err := json.Unmarshal([]byte(sanitizedResponse), &output)
	if err != nil {
		return &pb.ProcessResponse{
			Thought:     "LLM returned a non-JSON response, treating as a final answer.",
			FinalAnswer: response,
		}, nil
	}

	resp := &pb.ProcessResponse{
		Thought:     output.Thought,
		FinalAnswer: output.FinalAnswer,
	}

	if output.ToolCall != nil {
		resp.ToolCall = &pb.ToolCall{
			ToolName:  output.ToolCall.ToolName,
			Arguments: output.ToolCall.Arguments,
		}
	}

	return resp, nil
}
