---
name: delegate
description: Delegate bounded coding work to local Grok Build or Cursor Agent workers while the host observes and reviews. Use for Grok-first automatic selection, an explicit provider, or parallel independent read-only jobs.
---

# Delegate through Grok first

Use `delegate_start` for an independent, well-scoped task. Pass an absolute existing `workspace` and include the task's allowed paths, expected output, and verification command.

Default to `mode: "read_only"`. This starts Grok Build in plan mode without web, subagents, or memory. If Grok is absent before the process starts, Delegate Hub may start Cursor Agent in plan mode instead. Set an explicit `provider` to disable fallback or to run separate Grok and Cursor jobs in parallel.

For a scoped write task, first make the allowed files and acceptance check explicit. On native Windows use provider `grok`; Cursor writes are blocked because Cursor sandboxing is unavailable. Inspect `delegate_result`, run the named verification locally, and inspect the diff before accepting the change.

Parallel read-only jobs may share a workspace. Do not ask two jobs to write to the same workspace concurrently. Use separate worktrees for any future parallel-write support. Do not use Cursor after Grok has started: fallback exists only for an unavailable Grok executable, not for a failed, unauthenticated, out-of-credit, or cancelled Grok task.

Use `delegate_status` to poll a job, `delegate_result` once it is no longer running, and `delegate_cancel` when it must stop. Job data is stored under the current user's local cache, outside plugin caches. Never include API keys or secrets in tasks.
