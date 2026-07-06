# Goal Gate Acceptance Matrix

This document maps the current Goal Gate / AI-in-loop implementation to the stated acceptance
criteria.

It is intentionally evidence-first: each item points at the current code, tests, or demo surface
that proves the behavior.

## Acceptance Checklist

1. Default config must not trigger any goal/steward/harness behavior on ordinary traffic.
   - Evidence:
     - `adapter.ExecuteGoalGate` returns `goal_gate_disabled` when config is not enabled.
     - `TestExecuteGoalGateDisabledSkipsWithoutRunnerCall` in `adapter/goal_gate_test.go`.
     - `TestDemoHostExecuteSkipsWhenConfigDisabled` in `examples/goal-gate-host-http/main_test.go`.
   - Status: covered.

2. After configuring `goal.before_complete`, a simulated complete event must call the runner.
   - Evidence:
     - `steward.ReviewGoalCompletion` and `steward.ExecuteGoalGateSidecar`.
     - `TestReviewGoalCompletionInvokesExternalAgent` in `steward/goal_review_test.go`.
     - `TestExecuteGoalGateEnabledInvokesRunnerAndReturnsAppendRecord` in `adapter/goal_gate_test.go`.
     - `TestDemoHostExecuteRunsConfiguredRunner` in `examples/goal-gate-host-http/main_test.go`.
   - Status: covered.

3. When the runner returns `goal.approve_complete`, Gate must allow complete.
   - Evidence:
     - `ApplyGoalReviewResult` maps approval to `allow_complete=true`.
     - `TestEvaluateGoalProposalApprovesCompletion` in `steward/goal_review_test.go`.
     - `TestExecuteGoalGateCLI` in `cmd/gateway-harness/main_test.go`.
     - `TestDemoHostExecuteRunsConfiguredRunner` in `examples/goal-gate-host-http/main_test.go`.
   - Status: covered.

4. When the runner returns `goal.reject_complete + goal.request_continue`, Gate must block complete and surface a continue instruction.
   - Evidence:
     - `ApplyGoalReviewResult` maps rejection+continue to `continue_work=true`.
     - `TestEvaluateGoalProposalRejectsAndRequestsContinuation` in `steward/goal_review_test.go`.
     - `TestExecuteGoalGateCLIRejectContinuePath` in `cmd/gateway-harness/main_test.go`.
     - `TestDemoHostExecuteRejectContinuePath` in `examples/goal-gate-host-http/main_test.go`.
   - Status: covered.

5. Proposals that use undeclared actions must be rejected.
   - Evidence:
     - `steward.ValidateProposal` rejects actions outside the spec.
     - `RunExternalAgentInDir` validates the returned proposal before it can be applied.
     - `TestExecuteGoalGateInvalidProposalReturnsStructuredProposalFailure` in `adapter/goal_gate_test.go`.
     - `TestDemoHostExecuteRejectsInvalidProposal` in `examples/goal-gate-host-http/main_test.go`.
   - Status: covered.

6. Events that use undeclared inputs must be rejected.
   - Evidence:
     - `ValidateEvent` rejects event inputs not declared by the steward spec.
     - `adapter.applyGoalGateDefaults` rejects event inputs not enabled by runtime config.
     - `TestRunExternalAgentRejectsUndeclaredInput` in `steward/run_test.go`.
     - `TestExecuteGoalGateRejectsEventInputOutsideRuntimeConfig` in `adapter/goal_gate_test.go`.
   - Status: covered.

7. Runner failure must not silently pass; it must return an auditable error.
   - Evidence:
     - `adapter.failGoalGate` returns structured `GoalGateResult.failure` and, when possible, `append_record`.
     - `TestReviewGoalCompletionReturnsRunnerFailure` in `steward/goal_review_test.go`.
     - `TestExecuteGoalGateRunnerFailureReturnsAuditableErrorResult` in `adapter/goal_gate_test.go`.
     - `TestExecuteGoalGateCLIRunnerFailurePrintsStructuredResult` in `cmd/gateway-harness/main_test.go`.
   - Status: covered.

8. `max_continue_attempts` must prevent infinite loops.
   - Evidence:
     - `EvaluateGoalProposal` enforces max attempts, duplicate reason dedupe, and cooldown.
     - `TestEvaluateGoalProposalBlocksRetryAtMaxAttempts` in `steward/goal_review_test.go`.
     - `TestEvaluateGoalProposalBlocksDuplicateReason` in `steward/goal_review_test.go`.
     - `TestEvaluateGoalProposalBlocksCooldown` in `steward/goal_review_test.go`.
   - Status: covered.

9. WebUI must be able to configure enabled state, hook, runner command, allowed inputs, allowed actions, and max continue attempts.
   - Evidence:
      - `goal-gate-config-schema`, `goal-gate-form-schema`, `goal-gate-form-model`, and `goal-gate-host-bundle`.
      - Chat-first + manual settings demo in `examples/goal-gate-host/ui-demo.html`.
      - Live HTTP demo host in `examples/goal-gate-host-http/`.
      - Default review scope is now the minimal summary set; broader inputs such as `changed_files`, `verification_summary`, and `user_goal` require explicit opt-in through the UI or config assistant.
   - Status: covered by demo surface and schema contract.

10. Documentation must explain transparency defaults and how to enable / verify Goal Gate.
    - Evidence:
      - Overview and CLI usage in `README.md` and `README.zh-CN.md`.
      - Static/chat-first demo guide in `examples/goal-gate-host/README.md`.
      - Live HTTP demo guide in `examples/goal-gate-host-http/README.md`.
      - This acceptance matrix.
    - Status: covered.

## Current boundary

The current implementation intentionally stops at the contract / host-integration layer:

- Gateway Harness does not embed a model runtime.
- The external agent is still an explicit runner process.
- Goal Gate does not silently intercept anything unless the host enables it.
- `context.inject` remains an explicit continuation patch suggestion, not an automatic mutation.

That boundary is part of the intended design, not an incomplete stub.
