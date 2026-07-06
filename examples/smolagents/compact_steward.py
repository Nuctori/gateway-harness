#!/usr/bin/env python3
import json
import os
import re
import sys

from smolagents import CodeAgent, LiteLLMModel


def main() -> int:
    event = json.load(sys.stdin)
    if event.get("redacted") is not True:
        raise SystemExit("steward event must be redacted")

    model_id = os.environ.get("GATEWAY_HARNESS_STEWARD_MODEL")
    if not model_id:
        raise SystemExit("GATEWAY_HARNESS_STEWARD_MODEL is required, for example openai/kimi-for-coding")

    model_kwargs = {}
    api_base = os.environ.get("OPENAI_API_BASE")
    if api_base:
        model_kwargs["api_base"] = api_base

    agent = CodeAgent(
        tools=[],
        model=LiteLLMModel(model_id=model_id, **model_kwargs),
        max_steps=3,
    )
    result = agent.run(
        "You are a Gateway Harness steward. Read this redacted event and return only JSON. "
        "The JSON must be a steward proposal with fields: version, id, steward, hook, outputs. "
        "Allowed output actions are context.inject, ledger.artifact.create, diagnosis.note.create, "
        "and session.tags.update. Do not output context.truncate, policy.patch.propose, markdown, "
        "or explanatory prose.\n\n"
        + json.dumps(event, ensure_ascii=False, indent=2)
    )
    proposal = extract_json(str(result))
    print(json.dumps(proposal, ensure_ascii=False, indent=2))
    return 0


def extract_json(text: str) -> dict:
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        match = re.search(r"\{.*\}", text, re.S)
        if not match:
            raise SystemExit("agent did not return JSON")
        return json.loads(match.group(0))


if __name__ == "__main__":
    raise SystemExit(main())
