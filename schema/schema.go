package schema

import _ "embed"

//go:embed gateway-harness.policy.schema.json
var PolicyJSON string

//go:embed gateway-harness.adapter.schema.json
var AdapterJSON string

//go:embed gateway-harness.conformance.schema.json
var ConformanceJSON string

//go:embed gateway-harness.ledger.schema.json
var LedgerJSON string

//go:embed gateway-harness.ledger-record.schema.json
var LedgerRecordJSON string

//go:embed gateway-harness.steward.schema.json
var StewardJSON string

//go:embed gateway-harness.steward-proposal.schema.json
var StewardProposalJSON string
