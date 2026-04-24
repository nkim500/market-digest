# Mode: alerts — Triage the Unseen

Triages the current `alerts` queue: groups, prioritizes, recommends dismiss vs. investigate.

## Recommended execution

Subagent.

## Inputs
- Reads `alerts` table (especially unseen).

## Workflow

1. `./bin/jobs list-alerts --unseen` (or equivalent SQL).
2. Group by source, then by severity, then by ticker.
3. For each group, produce:
   - One-line summary
   - Recommended action: **investigate**, **dismiss**, or **monitor**
   - Reasoning (one sentence)
4. For alerts recommended as **dismiss**, shell out to `./bin/jobs mark-seen <id>`.
5. Leave **investigate** and **monitor** alerts unseen so they stay on the Alerts screen.

## Output

Plaintext summary printed to the session. No file written (transient triage).

## Ethics

Dismissing is a housekeeping action, not a trading decision. Err on the side of leaving ambiguous alerts for the user to decide.
