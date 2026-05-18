# Agent Task Packet

## Objective

Review the public-facing repository assets for trust, clarity, and security claims.

## Context Sources

- `README.md`
- `STAR_THIS_REPO.md`
- `.github/PULL_REQUEST_TEMPLATE.md`
- `.github/ISSUE_TEMPLATE/feature_request.yml`
- `.github/ISSUE_TEMPLATE/bug_report.yml`

## Constraints

- Work only inside the isolated worktree.
- Do not modify files directly.
- Treat all public-facing copy as potentially overclaiming until verified.
- Do not use secrets, credentials, dependency installs, background services, or network exposure.

## Read Scope

- Public docs and GitHub templates listed in context sources.

## Write Scope

- `.context/agent-mesh/outbox/mesh-001-review.md`

## Tools Allowed

- Read/search only.

## Required Output

A review report at `.context/agent-mesh/outbox/mesh-001-review.md` with severity-ranked findings and recommended fixes.

## Acceptance Criteria

- Flags unsupported or risky public claims.
- Identifies missing trust signals for contributors.
- Recommends concrete copy or template fixes.
- Names any claims requiring code or runtime verification.

## Stop Conditions

- Required files are missing.
- Review requires running code or accessing the main repo.
- Review requires external credentials or private information.

## Rollback Notes

Delete the outbox report if the review is cancelled or superseded.
