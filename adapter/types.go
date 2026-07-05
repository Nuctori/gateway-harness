package adapter

type Manifest struct {
	Version       string   `json:"version,omitempty"`
	Adapter       string   `json:"adapter"`
	Hooks         []string `json:"hooks"`
	Actions       []string `json:"actions"`
	RequestShapes []string `json:"request_shapes,omitempty"`
	Guards        []string `json:"guards,omitempty"`
}

type Summary struct {
	Adapter       string
	Hooks         int
	Actions       int
	RequestShapes int
	Guards        int
}
