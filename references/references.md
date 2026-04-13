# Forge — Reference Library

Curated links used during Forge’s design. Each entry notes **why it matters** for
harness engineering, agent factories, or this repo’s architecture.

For the canonical system design, see [docs/design.md](../docs/design.md).

---

## Stripe Minions and scale

- [ByteByteGo — How Stripe’s Minions ship 1,300+ PRs](https://blog.bytebytego.com/p/how-stripes-minions-ship-1300-prs) — High-level narrative of unattended agents, Devboxes, and factory-style throughput.
- [Stripe Dev Blog — Minions: one-shot end-to-end coding agents (Part 1)](https://stripe.dev/blog/minions-stripes-one-shot-end-to-end-coding-agents) — Official description of motivation, harness, and workflow.
- [Stripe Dev Blog — Minions (Part 2)](https://stripe.dev/blog/minions-stripes-one-shot-end-to-end-coding-agents-part-2) — Deeper operational and engineering detail on the same system.
- [InfoQ — Stripe autonomous coding agents (news)](https://www.infoq.com/news/2026/03/stripe-autonomous-coding-agents/) — Industry write-up; useful for citations and external framing.
- [Lenny’s Newsletter — How Stripe built Minions / AI coding](https://www.lennysnewsletter.com/p/how-stripe-built-minionsai-coding) — Product/engineering storytelling; good for non-reader-friendly summaries.

**Forge takeaway:** Blueprints + isolation + curated tools + bounded CI retry are the
industrial pattern to emulate in open source.

---

## Harness engineering theory

- [OpenAI — Harness engineering](https://openai.com/index/harness-engineering/) — Layered harness thinking, AGENTS.md-style structure, operational discipline around agents.
- [Anthropic — Harness design for long-running application development](https://www.anthropic.com/engineering/harness-design-long-running-apps) — Planner/generator/evaluator, sprint contracts, Playwright QA, context resets vs compaction, when to simplify the harness.
- [Anthropic — Seeing like an agent: tool design in Claude Code](https://claude.com/blog/seeing-like-an-agent) — Shape tools to model abilities; iterate designs empirically; structured output via tools vs brittle formatting; progressive disclosure and subagents vs prompt bloat; audit tool sets as models improve.
- [SuperagenticAI/metaharness](https://github.com/SuperagenticAI/metaharness) — Open-source outer loop that optimizes the **executable harness** (instructions, scripts, validation): filesystem-backed runs, candidate ledgers, write-scope limits, eval matrices; complements “prompt-only” tuning.
- [Ignorance.ai — The emerging discipline of harness engineering](https://www.ignorance.ai/p/the-emerging-harness-engineering) — Conceptual framing of harness engineering as a field.
- [Medium — The CI/CD of Code Itself (DebaA)](https://medium.com/@DebaA/the-ci-cd-of-code-itself-2c63ce65013e) — Treating generated code with delivery discipline (tests, gates, pipelines).
- [dev.to — Skill Creator v2 in VS Code (Debs Obrien)](https://dev.to/debs_obrien/i-used-skill-creator-v2-to-improve-one-of-my-agent-skills-in-vs-code-fhd) — Practical skill iteration loop in tooling.

**Forge takeaway:** Separate evaluation from generation; use contracts and
deterministic gates; revisit harness complexity as models improve.

---

## Agent frameworks and runtimes

- [block/goose — Goose](https://github.com/block/goose) — Agent harness Stripe forked for Minions; baseline for “agent + tools” ergonomics.
- [langchain-ai/deepagents](https://github.com/langchain-ai/deepagents) — Batteries-included harness: planning, filesystem tools, sub-agents, summarization; CLI and LangGraph integration.
- [The-Pocket/PocketFlow](https://github.com/The-Pocket/PocketFlow) — Minimal graph-first framework; reinforces “workflow = graph” mental model. See also the [PocketFlow design patterns](https://the-pocket.github.io/PocketFlow/design_pattern/) catalog (Agent, Workflow, RAG, Map Reduce, Multi-Agents).
- [usestrix/strix](https://github.com/usestrix/strix) — Docker-isolated autonomous security agents, multi-agent graph workflows, CI/headless runs; validates sandbox + graph + pipeline pattern in a vertical harness.
- [ruvnet/ruflo](https://github.com/ruvnet/ruflo) — Multi-agent orchestration / swarm-style patterns (inspiration for parallel factory runs).
- [bradygaster/squad](https://github.com/bradygaster/squad) — Persistent agent teams, repo-local state, coordinator routing; ideas for memory + team roles.

- [coleam00/archon — Archon](https://github.com/coleam00/archon) — YAML workflow engine for AI coding: deterministic + agent nodes, loop/gate nodes, validation, human approval, PR delivery; CLI/web/Slack/GitHub surfaces. Closest open-source peer to Forge's blueprint-first model.
- [multica-ai/multica](https://github.com/multica-ai/multica) — Managed-agents platform (Go + Postgres + Next.js): assign issues to agents like teammates, multi-CLI adapters (Claude Code, Codex, OpenClaw), daemon-isolated runs, WebSocket progress streaming.

**Forge takeaway:** Forge wraps external agents and focuses on **blueprint +
gates + factory** rather than replacing Goose/DeepAgents. Archon validates the
YAML-orchestrated harness pattern; Multica validates the managed-agents task
routing model.

---

## Multi-agent orchestration and virtual teams

- [OpenBMB/ChatDev](https://github.com/OpenBMB/ChatDev) — YAML-driven multi-agent platform (DevAll 2.0): validated workflows, Docker, FastAPI, PyPI SDK; closest open-source comparable to Forge’s blueprint-first orchestration.
- [VoltAgent/awesome-claude-code-subagents](https://github.com/VoltAgent/awesome-claude-code-subagents) — Curated Claude Code subagents with YAML frontmatter (tools, model); templates for role naming and least-privilege tool sets.
- [Yeachan-Heo/oh-my-claudecode](https://github.com/Yeachan-Heo/oh-my-claudecode) — Staged pipelines (plan → PRD → exec → verify → fix), tmux-isolated workers, skill triggers; pipeline UX parallel to blueprints; lightweight isolation without Docker.
- [NousResearch/hermes-agent](https://github.com/NousResearch/hermes-agent) — Multiple execution backends (local, Docker, SSH, etc.), subagents/RPC, skill hub; prior art for adapter diversity and skill improvement loops.

- [open-gitagent/gitagent](https://github.com/open-gitagent/gitagent) — Git-native portable agent spec (manifest + identity + rules + skills + workflows + tools); SkillsFlow YAML with `depends_on` and mixed skill/agent/tool steps; adapters for Claude Code, Cursor, CrewAI, etc.
- [vxcontrol/pentagi — PentAGI](https://github.com/vxcontrol/pentagi) — Self-hosted autonomous agent system (Go backend, React frontend): Docker-sandboxed execution, 20+ tools, multi-agent delegation, knowledge graph (Graphiti + Neo4j), observability (Grafana, Langfuse); security domain but strong patterns for sandboxing and long-horizon orchestration.
- [garrytan/gstack](https://github.com/garrytan/gstack) — Large skill-pack “software factory”: sprint-style pipelines, specialist roles, cross-agent tooling, safety guardrails; prior art for packaging many harness behaviors as skills (vs a standalone engine).

**Forge takeaway:** YAML graphs + staged roles are widespread; Forge differentiates via **layered engine/harness/factory** and **deterministic gates** as first-class nodes.

---

## Methodology, skills, and agent practice

- [BMAD Method — Docs](https://docs.bmad-method.org/) — Structured AI-driven dev methodology (phases, agents).
- [bmad-code-org/BMAD-METHOD](https://github.com/bmad-code-org/BMAD-METHOD) — Source for BMAD assets and workflows.
- [gsd-build/get-shit-done](https://github.com/gsd-build/get-shit-done) — GSD meta-prompting and execution discipline.
- [obra/superpowers](https://github.com/obra/superpowers) — Skills + workflows (including brainstorming → implementation); quality bar patterns.
- [affaan-m/everything-claude-code](https://github.com/affaan-m/everything-claude-code) — Large harness optimization system: skills, hooks, eval loops, security, cross-tool parity.

**Forge takeaway:** Methodology repos inform **defaults**; Forge supplies the
**runtime** (engine + optional factory).

---

## Skills, evals, and tuning

- [Anthropic — Complete Guide to Building Skills for Claude (PDF)](The-Complete-Guide-to-Building-Skill-for-Claude_260330_185835.pdf) — Official skill layout (SKILL.md, scripts/, references/, assets/), progressive disclosure, trigger/functional/performance testing, MCP pairing, distribution.
- [VoltAgent/awesome-agent-skills](https://github.com/VoltAgent/awesome-agent-skills) — Large index of Agent Skills across hosts (Claude Code, Codex, Cursor, Copilot); single pointer to the wider skill ecosystem.
- [hboon.com — Using skill-creator to improve existing skills](https://hboon.com/using-the-skill-creator-skill-to-improve-your-existing-skills/) — Eval-driven iteration: gap analysis, asserted test prompts, parallel old-vs-new runs, grading.
- [Tessl — Anthropic brings evals to Skill Creator](https://tessl.io/blog/anthropic-brings-evals-to-skill-creator-heres-why-thats-a-big-deal/) — Why eval-driven skill development matters.
- [Medium — Claude Code: build, evaluate, tune skills (Richard Hightower)](https://medium.com/@richardhightower/claude-code-how-to-build-evaluate-and-tune-ai-agent-skills-34afa808d1c9) — Practical skill tuning narrative.
- [Medium — Framework showdown: Superpowers vs BMAD vs SpecKit vs GSD](https://medium.com/ai-in-plain-english/the-great-framework-showdown-superpowers-vs-bmad-vs-speckit-vs-gsd-360983101c10) — Comparative landscape for positioning Forge.

- [Anthropic — Improving Skill Creator: test, measure, and refine](https://claude.com/blog/improving-skill-creator-test-measure-and-refine-agent-skills) — Evals as regression tests for skills, benchmark mode (pass rate / time / tokens), multi-agent eval isolation, A/B comparator, trigger-description tuning, model-obsolescence signal.
- [addyosmani/agent-skills](https://github.com/addyosmani/agent-skills) — 20 production-grade MIT skills organized around a full SDLC lifecycle (define → plan → build → verify → review → ship); quality gates, context engineering, TDD, and delivery as named skills with verification sections.
- [HKUDS/OpenSpace](https://github.com/HKUDS/OpenSpace) — Self-evolving skills layer delivered as MCP: skills that auto-fix/auto-improve from usage, shared evolution across agents, community skill marketplace, daily quality evaluation.
- [coleam00/excalidraw-diagram-skill](https://github.com/coleam00/excalidraw-diagram-skill) — Exemplar high-quality skill with Playwright-based render-inspect-fix verification loop; demonstrates SKILL.md + references/ bundle layout with closed-loop quality.

**Forge takeaway:** Treat skills as **testable artifacts** with lifecycle, not
static markdown only.

---

## Harness patterns and long-running agents

- [coleam00/ralph-loop-quickstart](https://github.com/coleam00/ralph-loop-quickstart) — Outer loop over Claude Code: fresh context per iteration, machine-readable PRD tasks, activity log, optional browser verification; parallels RunState + bounded retries.
- [shanraisshan/claude-code-best-practice](https://github.com/shanraisshan/claude-code-best-practice) — Command / agent / skill orchestration vocabulary, sandbox permissions, worktrees, hooks; governance patterns for harness policy.
- [luongnv89/claude-howto](https://github.com/luongnv89/claude-howto) — Structured tutorials on skills-as-folders, hooks, subagents; operator mental models for composable workflows.

- [safishamsi/graphify](https://github.com/safishamsi/graphify) — Claude Code skill that builds a queryable knowledge graph (NetworkX, tree-sitter) from code/docs/images; MCP server, watch mode, git hooks; large token reduction vs raw file reads.
- [tirth8205/code-review-graph](https://github.com/tirth8205/code-review-graph) — AST graph (tree-sitter) with blast-radius / impact analysis, ~22 MCP tools + prompts for review/debug/onboard; validates MCP-first code intelligence for quality gates.

**Forge takeaway:** Long-running reliability comes from **explicit outer loops**, **scoped context**, and **verification hooks**—encode as blueprint nodes and factory policies.

---

## Memory, context, and distribution

- [Karpathy — LLM Wiki (gist)](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f) — Persistent, LLM-maintained markdown wiki between raw sources and schema (`CLAUDE.md` / `AGENTS.md`); **Ingest / Query / Lint** operations; `index.md` + `log.md` navigation; compounding knowledge vs stateless RAG-only flows.
- [eugeniughelbur/obsidian-second-brain](https://github.com/eugeniughelbur/obsidian-second-brain) — Claude Code skills that implement Karpathy-style vault maintenance: scheduled agents, “thinking tools,” bi-temporal facts; concrete skill bundle for durable repo knowledge.
- [thedotmack/claude-mem](https://github.com/thedotmack/claude-mem) — Session capture, compression, reinjection patterns.
- [context7.com](https://context7.com/) — Library documentation retrieval for agents (pattern: reduce hallucination via targeted docs).
- [Claude Code — Channels reference](https://code.claude.com/docs/en/channels-reference) — How Anthropic frames multi-surface/agent routing (reference for triggers and integrations).

**Forge takeaway:** Progressive disclosure + durable repo artifacts beat dumping
whole codebases into context.

---

## Isolation, messaging, and integration

- [bbrowning/paude](https://github.com/bbrowning/paude) — Run coding agents in containers with network filtering and git workflows.
- [chenhg5/cc-connect](https://github.com/chenhg5/cc-connect) — Go bridge from agents to chat platforms (Slack, Discord, Telegram, etc.).

- [InsForge/InsForge](https://github.com/InsForge/InsForge) — MCP-driven backend-as-a-service for agent work: semantic layer over auth, Postgres, storage, edge functions, model gateway; agents interact via InsForge MCP tools.

**Forge takeaway:** Factory layer should **compose** isolation + triggers rather
than reinvent them.

---

## Design patterns and optimization

- [PocketFlow — Design patterns](https://the-pocket.github.io/PocketFlow/design_pattern/) — Named patterns (Agent, Workflow, RAG, Map Reduce, Structured Output, Multi-Agents); taxonomy to compare against blueprint node types.
- [microsoft/agent-lightning](https://github.com/microsoft/agent-lightning) — Traces, LightningStore, RL/prompt optimization around existing agents; inspiration for structured run telemetry and offline improvement (v0.3+ direction).

- [rtk-ai/rtk](https://github.com/rtk-ai/rtk) — Rust CLI proxy that filters/compresses shell command output before it reaches the model; editor hooks for Claude Code, Cursor, Codex; validates context-engineering-at-the-tool-boundary pattern.

**Forge takeaway:** Align blueprint vocabulary with established patterns; separate **observability/training** from core orchestration until the learning story is in scope.

---

## Workflows, creators, and community

- [meleantonio/ChernyCode](https://github.com/meleantonio/ChernyCode) — Curated Claude Code workflow resources (plans, verification, patterns).
- [Readwise shared — Claude Code workflow notes (example)](https://readwise.io/reader/shared/01kgcamtex6zews0fvz94a8qg4/) — Short-form practitioner notes (URL may require Readwise).
- [Readwise shared — Additional workflow notes (example)](https://readwise.io/reader/shared/01kgb6njjekq2hpxc0ycymbrcg/) — Complementary practitioner notes.
- [YouTube — GSD / agentic development (example)](https://www.youtube.com/watch?v=o5Mi5SYSDnY) — Video explainer referenced in research (verify relevance if the exact talk shifts).
- [YouTube — Cole Medin: You’re Hardly Using What Claude Code Has to Offer](https://www.youtube.com/watch?v=uegyRTOrXSU) — Multi-agent teams, worktrees, `/batch`, hooks, context hygiene (vendor-specific; useful for product patterns).
- [YouTube — Cole Medin: The 5 Techniques Separating Top Agentic Engineers](https://www.youtube.com/watch?v=ttdWPDmBN_4) — PRDs as contracts, priming, modular rules, commandification (methodology aligned with blueprints + scoped rules).

**Forge takeaway:** Encode proven habits (plan, verify, parallelize) as **blueprint
nodes and policies**, not one-off chat prompts.

---

## Community discussion

- [Reddit r/ExperiencedDevs — Companies implementing coding agents?](https://www.reddit.com/r/ExperiencedDevs/comments/1rknwd8/anybodys_companies_successfully_implement/?rdt=41330) — Practitioner skepticism and adoption patterns; useful for risk sections in PRDs.

**Forge takeaway:** Adoption requires **trust, cost control, and review UX**—not
raw model strength.

---

## IBM-style analysis (planning citation)

- *Note:* Forge planning cited **IBM research** on error detection: LLM-only vs
  LLM + deterministic analysis (e.g. ~45% vs ~94% in the cited narrative). If you
  need the exact paper, search IBM publications on LLM static analysis pairing;
  the **design lesson** is encoded in Forge either way: **combine** model judgment
  with linters/tests.

---

## Maintenance

When adding a new reference:

1. Place it under the best-fitting category (or add a category).
2. Add **one or two sentences** on what Forge learns from it.
3. Prefer primary sources (official blogs, repos) over aggregators when possible.
