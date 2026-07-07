package event

type Type string

const (
	TypeRequestPrePrompt  Type = "request.pre_prompt"
	TypeUpstreamError     Type = "upstream.error"
	TypeRequestPreContext Type = "request.pre_context"
)

type Event struct {
	Type      Type              `json:"type" yaml:"type"`
	TraceID   string            `json:"trace_id,omitempty" yaml:"trace_id,omitempty"`
	RequestID string            `json:"request_id,omitempty" yaml:"request_id,omitempty"`
	Model     string            `json:"model,omitempty" yaml:"model,omitempty"`
	TenantID  string            `json:"tenant_id,omitempty" yaml:"tenant_id,omitempty"`
	UserID    string            `json:"user_id,omitempty" yaml:"user_id,omitempty"`
	Tags      []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Error     *ErrorInfo        `json:"error,omitempty" yaml:"error,omitempty"`
	Attempt   *AttemptInfo      `json:"attempt,omitempty" yaml:"attempt,omitempty"`
	Context   *Context          `json:"context,omitempty" yaml:"context,omitempty"`
}

type ErrorInfo struct {
	Status  int    `json:"status,omitempty" yaml:"status,omitempty"`
	Code    string `json:"code,omitempty" yaml:"code,omitempty"`
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
}

type AttemptInfo struct {
	Index int `json:"index,omitempty" yaml:"index,omitempty"`
	Max   int `json:"max,omitempty" yaml:"max,omitempty"`
}

type Context struct {
	MaxTokens       int64             `json:"max_tokens,omitempty" yaml:"max_tokens,omitempty"`
	EstimatedTokens int64             `json:"estimated_tokens,omitempty" yaml:"estimated_tokens,omitempty"`
	TokenEstimator  string            `json:"token_estimator,omitempty" yaml:"token_estimator,omitempty"`
	Messages        []Message         `json:"messages,omitempty" yaml:"messages,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type Message struct {
	ID              string            `json:"id,omitempty" yaml:"id,omitempty"`
	Role            string            `json:"role,omitempty" yaml:"role,omitempty"`
	Content         string            `json:"content,omitempty" yaml:"content,omitempty"`
	EstimatedTokens int64             `json:"estimated_tokens,omitempty" yaml:"estimated_tokens,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

func (e Event) HasTag(tag string) bool {
	for _, existing := range e.Tags {
		if existing == tag {
			return true
		}
	}
	return false
}
