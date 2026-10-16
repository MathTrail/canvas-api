# AGENTS.md

The working rules of this repository are in [CLAUDE.md](CLAUDE.md), and they bind every coding agent, not only Claude Code. Read it in full before any work and follow it as written: what it asks of Claude, it asks of you. It stays the one copy of the rules; this file only says what stands in for the parts that are Claude Code's own:

- **The cloud-session exception to "Devcontainer only"** is for a cloud session of Claude Code. A Codex session that does not see `MATHTRAIL_DEVCONTAINER=1` stops before its first edit, as that rule says.
- **The review before a change is called done.** `/code-review high` is a Claude Code command, and you cannot run it; Codex's `/review` stands in for it, started by the author. Everything else in that rule holds as written: prepare the clean copy and run the checks in it, then stop and ask the author to run `/review` there; verify and handle its findings as the rule says, with at most the one second round it allows. Until a review has run, the report says that it did not.
- **A live run's narrow permission.** Claude Code gets it in `.claude/settings.local.json`. Under Codex the author approves each of the run's commands once, as it comes, never for the rest of the session, so nothing approved outlives the run; do not ask for a looser sandbox or approval mode for it.
- **A live run's report** names its roles as they were: what the author did and what Codex did.
- **Memory.** Claude Code's memory is not shared with you. What has to outlive a session goes into the repository's documents, as CLAUDE.md says.
