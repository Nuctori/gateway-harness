package ledger

type Ledger struct {
	Version  string    `json:"version,omitempty"`
	Projects []Project `json:"projects"`
}

type Project struct {
	ID       string    `json:"id"`
	Name     string    `json:"name,omitempty"`
	Tags     []string  `json:"tags,omitempty"`
	Sessions []Session `json:"sessions"`
}

type Session struct {
	ID        string     `json:"id"`
	Title     string     `json:"title,omitempty"`
	StartedAt string     `json:"started_at"`
	Tags      []string   `json:"tags,omitempty"`
	Events    []Event    `json:"events"`
	Artifacts []Artifact `json:"artifacts,omitempty"`
}

type Event struct {
	ID            string            `json:"id"`
	Type          string            `json:"type"`
	At            string            `json:"at"`
	RequestShape  string            `json:"request_shape,omitempty"`
	Model         string            `json:"model,omitempty"`
	Hook          string            `json:"hook,omitempty"`
	Action        string            `json:"action,omitempty"`
	PolicyVersion string            `json:"policy_version,omitempty"`
	TraceHash     string            `json:"trace_hash,omitempty"`
	ErrorCode     string            `json:"error_code,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type Artifact struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	ContentHash string            `json:"content_hash"`
	Ref         string            `json:"ref,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type Summary struct {
	Projects  int
	Sessions  int
	Events    int
	Artifacts int
}
