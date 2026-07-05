package conformance

import (
	"encoding/json"

	"github.com/Nuctori/gateway-harness/adapter"
	"github.com/Nuctori/gateway-harness/policy"
)

type Fixture struct {
	Name           string           `json:"name"`
	RequestShape   string           `json:"request_shape"`
	RequiredGuards []string         `json:"required_guards,omitempty"`
	Adapter        adapter.Manifest `json:"adapter"`
	Policy         policy.Policy    `json:"policy"`
	Request        json.RawMessage  `json:"request,omitempty"`
}

type Summary struct {
	Name           string
	Adapter        string
	RequestShape   string
	Programs       int
	Steps          int
	Actions        int
	RequiredGuards int
}
