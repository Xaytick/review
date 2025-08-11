package core

// Intent represents the user's intent derived from the query.
type Intent struct {
	Name      string            // The name of the intent (e.g., "search_reviews", "create_appeal")
	NeedsTool bool              // Whether this intent requires a tool to be executed
	ToolName  string            // The name of the tool to execute
	Arguments map[string]interface{} // Arguments for the tool
	Query     string            // Original user query
}

// NLUProcessor is responsible for Natural Language Understanding.
// It detects the intent from a user's query.
type NLUProcessor struct {
	// In a real implementation, this would hold dependencies for a remote NLU service or a local model.
}

// NewNLUProcessor creates a new NLUProcessor.
func NewNLUProcessor() *NLUProcessor {
	return &NLUProcessor{}
}

// DetectIntent analyzes the query and returns the user's intent.
// This is a simplified placeholder implementation.
func (p *NLUProcessor) DetectIntent(query string) *Intent {
	// TODO: Implement actual NLU logic.
	// This could involve keyword matching, regex, or calling a third-party NLU service (like Dialogflow, LUIS, or a self-hosted model).

	// For now, we'll use a very simple keyword-based detection.
	if query == "最新评论" {
		return &Intent{
			Name:      "search_reviews",
			NeedsTool: true,
			ToolName:  "ListReviews",
			Arguments: map[string]interface{}{"page_size": 10, "page_num": 1},
			Query:     query,
		}
	}

	// Default intent is to query the knowledge base.
	return &Intent{
		Name:      "knowledge_query",
		NeedsTool: false,
		Query:     query,
	}
}