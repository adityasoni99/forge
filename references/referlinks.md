# Referlinks audit (processed)

Raw URL inbox for research. **March–April 2026 audit:** entries below were reviewed and
either merged into [references.md](references.md), reflected in [docs/design.md](../docs/design.md),
or explicitly skipped. Prefer **references.md** as the canonical curated list.
The former inbox file `references/newlinks.md` was folded into this audit and removed.

---

## Merged into references.md

| Link | Notes |
|------|--------|
| https://github.com/VoltAgent/awesome-claude-code-subagents | § Multi-agent orchestration |
| https://github.com/VoltAgent/awesome-agent-skills | § Skills, evals, and tuning |
| https://github.com/coleam00/ralph-loop-quickstart | § Harness patterns and long-running agents |
| https://www.youtube.com/watch?v=uegyRTOrXSU | § Workflows (Cole Medin) |
| https://www.youtube.com/watch?v=ttdWPDmBN_4 | § Workflows (Cole Medin) |
| https://the-pocket.github.io/PocketFlow/design_pattern/ | § Design patterns and optimization (+ PocketFlow line in Agent frameworks) |
| https://github.com/usestrix/strix | § Agent frameworks and runtimes |
| https://hboon.com/using-the-skill-creator-skill-to-improve-your-existing-skills/ | § Skills, evals, and tuning |
| https://github.com/NousResearch/hermes-agent | § Multi-agent orchestration |
| https://github.com/luongnv89/claude-howto/tree/main | § Harness patterns |
| https://github.com/Yeachan-Heo/oh-my-claudecode | § Multi-agent orchestration |
| https://github.com/shanraisshan/claude-code-best-practice | § Harness patterns |
| https://github.com/microsoft/agent-lightning | § Design patterns and optimization |
| https://github.com/OpenBMB/ChatDev | § Multi-agent orchestration |
| https://claude.com/blog/improving-skill-creator-test-measure-and-refine-agent-skills | § Skills, evals (skill-creator evals, benchmarks, A/B, triggers) |
| https://github.com/coleam00/archon | § Agent frameworks (YAML workflow engine, closest peer) + design.md §2, §7 |
| https://github.com/addyosmani/agent-skills | § Skills, evals (20 production SDLC skills, quality gates) |
| https://github.com/open-gitagent/gitagent | § Multi-agent orchestration (git-native agent spec, SkillsFlow YAML) + Layer 2 plan |
| https://github.com/HKUDS/OpenSpace | § Skills, evals (self-evolving skills, MCP host) + Layer 2 plan |
| https://github.com/multica-ai/multica | § Agent frameworks (managed agents, multi-CLI, Go) + Layer 3 plan |
| https://github.com/vxcontrol/pentagi | § Multi-agent orchestration (Docker sandbox, Go backend, knowledge graph) |
| https://github.com/tirth8205/code-review-graph | § Harness patterns (AST graph, 22 MCP tools, impact analysis) |
| https://github.com/safishamsi/graphify | § Harness patterns (knowledge graph skill, MCP server) |
| https://github.com/InsForge/InsForge | § Isolation, messaging (MCP-driven BaaS for agents) |
| https://github.com/coleam00/excalidraw-diagram-skill | § Skills, evals (exemplar skill with Playwright verification loop) |
| https://github.com/rtk-ai/rtk | § Design patterns (shell output compression for context efficiency) |
| https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f | § Memory, context (LLM Wiki: Ingest / Query / Lint) |
| https://claude.com/blog/seeing-like-an-agent | § Harness engineering theory (tool design, progressive disclosure) |
| https://github.com/SuperagenticAI/metaharness | § Harness engineering theory (optimize executable harness + ledgers) |
| https://github.com/garrytan/gstack | § Multi-agent orchestration (skill-pack software factory) |
| https://github.com/eugeniughelbur/obsidian-second-brain | § Memory, context (Karpathy-style vault skills) |

**PDF (companion to this list):** [The-Complete-Guide-to-Building-Skill-for-Claude_260330_185835.pdf](The-Complete-Guide-to-Building-Skill-for-Claude_260330_185835.pdf) — merged under § Skills, evals, and tuning; cited in design.md §5.5.

---

## Reflected in design.md (not duplicated as primary links)

- Release hygiene (no embedded source in shipped maps) — general lesson from npm/sourcemap incidents; **no** VentureBeat / kuber.studio URLs in references.md (IP-sensitive secondary coverage).
- ChatDev positioning — §2 research table + §7 differentiators.
- PocketFlow design patterns — §2, §4.1, §7.
- Skill specification PDF — §5.5.
- Archon positioning — §2 research synthesis + §7 differentiators (closest open-source peer).
- gitagent SkillsFlow YAML — noted in Layer 2 plan post-MVP prior art.
- OpenSpace skill evolution — noted in Layer 2 plan post-MVP prior art.
- Multica managed-agents — noted in Layer 3 plan post-MVP prior art.
- Karpathy LLM Wiki gist — §5.6 memory / git-native artifacts; ingest-query-lint as blueprint-shaped operations.
- Metaharness — optional note in harness eval / factory evidence trails (outer optimization loop).
- GRC agent article (Ethan Troy) — tool economics, deferred loading, structured outputs: fold into Layer 2 plan as patterns; cite Anthropic docs for canonical tool APIs.

---

## Skipped (by audit decision)

| Link | Reason |
|------|--------|
| https://github.com/SakanaAI/AI-Scientist-v2 | Niche tree-search / research orchestration; defer unless Forge adds branching search policies. |
| https://github.com/fathyb/carbonyl | Terminal browser only; low overlap with blueprint/harness/factory. |
| https://github.com/slavingia/skills | Small vertical skill pack; low priority vs ecosystem lists. |
| https://venturebeat.com/technology/claude-codes-source-code-appears-to-have-leaked-heres-what-we-know | Leak coverage; lesson folded into design.md §11 release hygiene without citing. |
| https://github.com/instructkr/claw-code | Interesting harness taxonomy; murky IP provenance—omit from formal references. |
| https://github.com/anthropics/claude-code/pull/41518 | Ephemeral / disputed PR tied to sensitive incident—not a durable reference. |
| https://kuber.studio/blog/AI/Claude-Code's-Entire-Source-Code-Got-Leaked-via-a-Sourcemap-in-npm,-Let's-Talk-About-it | Leak-derived architecture tour; not cited; release hygiene only in design.md. |
| https://github.com/Kuberwastaken/claude-code | Not recommended (leak-adjacent mirror narrative); use public specs and open repos instead. |
| local: `/CursorProjects/claudeCode` | Leak-derived Claude Code source mirror; architectural patterns extracted abstractly into design.md (§5.3, §5.7, §5.8, §6.5, §11) and Layer 2/3 plans. Not cited as a formal reference due to IP provenance. |
| https://www.youtube.com/watch?v=qMnClynCAmM | Cole Medin Archon demo; repo link (coleam00/archon) merged instead. |
| https://www.youtube.com/watch?v=KjEFy5wjFQg | Top 10 skills roundup video; individual repos cited where relevant. |
| https://www.linkedin.com/posts/rakshit-gupta-8487a816a_i-let-openai-review-code-that-claude-wrote-share-7446980916461240320-7gXM | Social post: Codex reviews Claude Code output; pattern noted (multi-model review) but LinkedIn not a durable reference. |
| https://www.linkedin.com/posts/paoloperrone_ive-replaced-4000month-in-llm-infrastructure-share-7445938696857493505-V7Ad | Social post: open-source inference stack; tangential to orchestration. |
| https://www.linkedin.com/posts/sabahudin-murtic_i-have-100x-browser-tabs-open-about-claude-share-7442530632351596544-PsWv | Social post: curated repo list; individual repos evaluated separately. |
| https://supabase.com/docs/guides/getting-started/ai-skills | Lightweight getting-started page for skills CLI; pattern noted but not architectural. |
| https://supabase.com/docs/guides/local-development/cli/getting-started | General CLI docs; Docker local-dev pattern is already covered by Forge's factory design. |
| https://vercel.com/docs/cli | General CLI docs; MCP project config interesting but tangential. |
| https://cli.github.com/ | Already implicit in Forge plans (`gh pr create`); well-known, no need to cite. |
| https://github.com/microsoft/playwright-cli | Already referenced in design.md (Playwright QA); well-known tool. |
| https://github.com/teng-lin/notebooklm-py | Niche NotebookLM wrapper; low overlap with blueprint/harness/factory. |
| https://github.com/rowboatlabs/rowboat | Personal knowledge agent (email, calendar); not code orchestration. |
| https://github.com/JuliusBrussee/caveman | Token compression trick + eval harness; interesting but not architectural. |
| https://github.com/SuperagenticAI/turboagents | KV-cache / RAG compression library; below the orchestration layer. |
| https://github.com/phuryn/claude-usage | Usage dashboard for Claude Code; narrow observability utility. |
| https://github.com/firecrawl/cli | Web scraping CLI + skill; tangential to core orchestration. |
| https://github.com/NousResearch/hermes-agent/tree/main/skills/creative/manim-video | Niche creative skill under repo already cited; no new orchestration pattern. |
| https://github.com/tomascortereal/claude-code-setup | Large composition setup; overlaps plugins already cited individually; not merged as standalone. |
| https://github.com/lokeshmavale/CopilotSessionBrowser | Windows session browser for Copilot CLI; narrow; not architectural. |
| https://github.com/mvanhorn/last30days-skill | Research skill; Map-reduce pattern; skill-tier not engine prior art. |
| Multiple LinkedIn URLs (newlinks inbox) | Social posts; not durable references; patterns traced to repos/blogs where applicable. |
| https://tylerfolkman.substack.com/p/i-read-the-claude-code-source-leak | Leak-framed setup advice; do not cite as primary; lessons overlap hooks vs long CLAUDE.md (see design hygiene). |
| https://medium.com/@amitmahajan.cloud/your-team-is-using-claude-wrong-heres-how-to-fix-it-c77b89ba3195 | Popularizer; redundant with OpenAI/Anthropic harness refs. |
| https://github.com/GitFrog1111/badclaude | Gag interrupt utility; no Forge design value. |
| https://github.com/forrestchang/andrej-karpathy-skills | Single-file Karpathy-style defaults; thin vs references methodology section. |
| https://ethantroy.dev/posts/grc-agent-claude-sdk/ | GRC vertical; useful practitioner patterns—prefer official Anthropic SDK/MCP docs for canonical cites. |
| https://www.ccunpacked.dev/ | Third-party Claude Code explainer; optional literacy; verify live before relying. |
| https://www.youtube.com/watch?app=desktop&v=EsTrWCV0Ph4 | Generic “agentic AI” course (Nick Saraev); not Forge-specific prior art. |

---

## Maintenance

Add new raw URLs below the table if needed; periodically fold into **references.md**
and update this audit section.
