Guide: Collaborating with AI on Architecture Decisions

## Core Principle

AI should be a **sparring partner**, not an oracle. Its value is in:

1. Synthesizing production experience from thousands of systems
2. Challenging your assumptions
3. Suggesting alternatives you haven't considered
4. Explaining tradeoffs with concrete examples

You remain the **decision maker** because:

1. You own the constraints (budget, timeline, scale)
2. You bear the consequences of failure
3. You're building intuition, not just a system

---

## How to Get Value from AI

### ❌ Ineffective Prompts

**Too vague**:

> "Should I use Go or Rust for my API?"
> 

**Problem**: No context, forces AI to guess or give generic answer.

**AI is a code monkey**:

> "Write me a rate limiter in Go"
> 

**Problem**: You learn nothing about why, just copy-paste code.

**Accepting first answer**:

> AI: "Use Redis for distributed rate limiting"
You: "Okay!" moves on
> 

**Problem**: No exploration of tradeoffs or alternatives.

---

### ✅ Effective Prompts

**Provide context**:

> "I'm building an API gateway that needs to rate limit requests per tenant. Constraints: must run on 512MB instance, handle 10k req/sec, p99 latency <10ms. Should I use in-memory state, Redis, or SQLite? Walk me through the tradeoffs."
> 

**Why this works**: Specific constraints force specific reasoning.

**Challenge AI**:

> AI: "Use Redis for distributed rate limiting"
You: "What's the latency overhead of Redis calls? If each rate limit check is ~1ms, that's 10% of my latency budget. Are there alternatives?"
> 

**Why this works**: Forces deeper analysis, surfaces hidden costs.

**Ask for experience**:

> "What do companies like Stripe, GitHub, and Cloudflare do for rate limiting at scale? What problems did they encounter that I should anticipate?"
> 

**Why this works**: Learn from others' production mistakes.

**Request architectural alternatives**:

> "I'm planning to use token bucket algorithm for rate limiting. Show me 3 alternative approaches, when each would be better, and what the implementation complexity difference is."
> 

**Why this works**: Expands solution space, teaches pattern recognition.

**Demand justification**:

> "You suggested Elixir for the workflow engine. Convince me why Elixir's actor model is worth learning a new language versus using Go with explicit state machines."
> 

**Why this works**: Forces cost-benefit analysis.

---

## The Architecture Decision Loop

### Step 1: Frame the Problem (You Lead)

**Template**:

```
Problem: [What are we building?]
Constraints: [Non-negotiable requirements]
Scale: [Expected load, growth]
Unknowns: [What don't we know yet?]

```

**Example**:

```
Problem: Multi-tenant rate limiting for API gateway
Constraints:
- 512MB memory
- <10ms p99 latency
- 10k req/sec sustained
- Hot-reload tenant configs
Scale: Start with 100 tenants, plan for 10k
Unknowns: How to handle burst traffic fairly

```

### Step 2: Generate Options (AI Helps)

**Prompt**:

> "Given the above constraints, propose 3 different architectural approaches. For each, explain: (1) core idea, (2) strengths, (3) weaknesses, (4) implementation complexity."
> 

**What you're looking for**:

- Diversity of approaches (not just variations of the same idea)
- Honest about weaknesses (red flags if everything is perfect)
- Concrete tradeoffs (not vague "it depends")

### Step 3: Challenge Assumptions (You Push Back)

**Prompt**:

> "You suggested [option]. What happens when [failure scenario]? What's the fallback?"
> 

**Example**:

> "You suggested Redis for state. What happens if Redis goes down? What's the recovery time? What if network latency spikes?"
> 

**What you're looking for**:

- Failure mode analysis
- Graceful degradation strategies
- Operational complexity

### Step 4: Make Decision (You Decide)

**Template**:

```
Decision: [What we're doing]
Rationale: [Why, with specific reasoning]
Rejected alternatives: [What we're not doing and why]
Risks: [What could go wrong]
Success criteria: [How we'll know it works]

```

**Example**:

```
Decision: In-memory token bucket with periodic snapshots to SQLite

Rationale:
- Meets latency requirement (<1μs vs Redis ~1ms)
- Fits in memory (100 bytes/tenant × 10k = 1MB)
- SQLite snapshots every 1s for durability
- Can scale to 100k tenants before hitting memory limits

Rejected alternatives:
- Redis: Network latency too high for our budget
- SQLite per-request: Disk I/O would kill performance

Risks:
- Snapshot window = up to 1s of data loss on crash
- Vertical scaling only (can't distribute)

Success criteria:
- p99 latency <10ms (we're targeting <1ms for rate limiter)
- 10k sustained req/sec on single core
- 0 false negatives (never allow over-limit requests)

```

### Step 5: Document & Validate (You + AI)

**You write ADR** (Architecture Decision Record):

# ADR 001: In-Memory Rate Limiting with SQLite Snapshots

## Status

Proposed

## Context

[Problem statement from Step 1]

## Decision

[From Step 4]

## Consequences

- Positive: [Benefits]
- Negative: [Costs/limitations]
- Neutral: [Side effects]

## Validation Plan

- [ ]  Benchmark: Achieve <1ms p99 latency
- [ ]  Load test: Sustain 10k req/sec
- [ ]  Chaos test: Kill process, verify recovery from snapshot
- [ ]  Memory test: Profile with 100k tenants

```

**AI helps validate**:
> "Review this ADR. What am I missing? What could go wrong that I haven't considered?"

---

## Asking AI to Explain Tradeoffs

### Language Tradeoffs

**Prompt Template**:
> "I'm deciding between [Language A] and [Language B] for [specific component]. The constraints are [list constraints]. Explain:
> 1. What feature of each language is most relevant here?
> 2. Show me a concrete example where one would fail and the other wouldn't
> 3. What's the operational cost difference (deployment, debugging, monitoring)?"

**Example**:
> "I'm deciding between Go and Rust for the API gateway hot path. Constraints: <10ms p99 latency, 512MB memory, must handle 10k req/sec. Explain:
> 1. How does Go's GC vs Rust's ownership affect latency?
> 2. Show me a scenario where GC pauses would violate our latency SLA
> 3. What's the debugging experience difference when we have a production latency spike?"

### System Design Tradeoffs

**Prompt Template**:
> "For [problem], I'm choosing between [approach A] and [approach B]. Explain:
> 1. At what scale does each approach break?
> 2. What's the operational complexity difference?
> 3. How does each handle [specific failure scenario]?"

**Example**:
> "For distributed rate limiting, I'm choosing between: (A) centralized Redis with optimistic locking, (B) local state with gossip protocol. Explain:
> 1. At what request rate does each approach break?
> 2. What happens when network partitions occur?
> 3. What's the monitoring and debugging complexity for each?"

### Pattern Tradeoffs

**Prompt Template**:
> "For [specific use case], when should I use [Pattern A] vs [Pattern B]? Give me:
> 1. A concrete example where Pattern A clearly wins
> 2. A concrete example where Pattern B clearly wins
> 3. Warning signs that I've chosen the wrong one"

**Example**:
> "For workflow state management, when should I use event sourcing vs traditional CRUD? Give me:
> 1. A scenario where event sourcing's replay capability is essential
> 2. A scenario where event sourcing is overkill
> 3. Warning signs that my event-sourced system is becoming too complex"

---

## Red Flags: When to Distrust AI

### 🚩 Vague Answers
**AI says**: "It depends on your use case"
**You say**: "Be specific. Given these exact constraints, which is better and why?"

### 🚩 No Downsides
**AI says**: "This approach is perfect for your needs"
**You say**: "What are the downsides? Every decision has tradeoffs."

### 🚩 Buzzword Bingo
**AI says**: "Use microservices with event-driven architecture and CQRS"
**You say**: "Why? We're a single-node system with 100 users. Justify this complexity."

### 🚩 No Concrete Numbers
**AI says**: "This will scale well"
**You say**: "Define 'well'. Give me specific numbers: req/sec, latency, memory usage."

### 🚩 Ignoring Constraints
**AI suggests**: "Use Kubernetes for deployment"
**Your constraint**: "Must run on a $5/month VPS"
**You say**: "This violates my constraint. Suggest alternatives that fit the budget."

### 🚩 Copying Without Understanding
**AI provides**: 200 lines of code
**You say**: "Explain the core algorithm in 3 sentences. What are the edge cases?"

---

## AI-Assisted Learning Prompts

### Understanding a Technology

**Template**:
> "Explain [technology] like I'm an experienced engineer who's never used it. Focus on:
> 1. What problem it solves that alternatives don't
> 2. When NOT to use it
> 3. A simple example that shows its core value"

**Example**:
> "Explain Elixir's actor model like I'm an experienced Go engineer. Focus on:
> 1. What concurrent problem it solves that goroutines don't
> 2. When goroutines would be simpler/better
> 3. A simple example showing supervisors automatically recovering from crashes"

### Evaluating Design Patterns

**Template**:
> "I want to understand [pattern] deeply. Explain:
> 1. What code smell does this pattern eliminate?
> 2. Show me 'before' and 'after' code for a real scenario
> 3. What new problems does this pattern introduce?"

**Example**:
> "I want to understand the Repository pattern deeply. Explain:
> 1. What coupling problem does it solve?
> 2. Show me a test that's easier with Repository vs direct DB calls
> 3. When does Repository become over-abstraction?"

### Challenging Common Wisdom

**Template**:
> "Everyone says [common advice], but I'm skeptical. Steelman the opposite position:
> 1. When is [common advice] actually wrong?
> 2. What context makes the opposite approach better?
> 3. How do I know which applies to my situation?"

**Example**:
> "Everyone says 'use microservices for scalability', but I'm skeptical. Steelman monoliths:
> 1. When do monoliths actually scale better?
> 2. What problems do microservices introduce that aren't worth it?
> 3. How do I know when to split vs keep together?"

---

## Example Conversation Flow

### You (Framing):
> "I'm building an event-driven workflow engine. I need to decide on storage for workflow state. Requirements:
> - Exactly-once execution (can't duplicate tasks)
> - Survive crashes (durable state)
> - Fast state transitions (<100ms p99)
> - Support 1000 concurrent workflows
>
> I'm considering: (A) Postgres with event sourcing, (B) Redis with periodic snapshots to Postgres. Walk me through the tradeoffs."

### AI (Exploring):
> [Detailed comparison of both approaches, covering consistency, durability, performance, complexity]

### You (Challenging):
> "You suggest Postgres for ACID guarantees. But won't that add latency? Each state transition is a write. Show me the math: what's the expected latency per transition?"

### AI (Justifying):
> [Calculates: Postgres write ~5ms, plus serialization ~1ms, total ~6ms. Suggests: batch transitions, use prepared statements to optimize]

### You (Exploring Alternatives):
> "6ms is acceptable but not great. What if we did: in-memory state, append to WAL (write-ahead log), async Postgres snapshot every 1s? What do we lose?"

### AI (Analyzing):
> [Explains: Faster (sub-ms state transitions), but recovery is complex. Up to 1s of work lost on crash. Need WAL replay logic.]

### You (Deciding):
> "I'll use in-memory + WAL for now. Complexity is worth it for latency. But I'll design the interface so I can swap storage later if recovery logic becomes too complex. Write me an interface specification for a WorkflowStateStore."

### AI (Implementing):
> [Provides interface with methods: saveState, loadState, appendEvent, etc.]

---

## Your Responsibilities

AI is a tool, not a replacement for thinking. You must:

1. **Define success criteria**: AI can't guess what "good" means for your project
2. **Set constraints**: Budget, time, scale, expertise level
3. **Push for specifics**: Reject vague answers, demand numbers
4. **Explore failure modes**: AI often optimizes for happy path
5. **Make final calls**: You own the decision and its consequences
6. **Document reasoning**: Future you will forget why you chose X over Y
7. **Validate with reality**: Deploy, measure, learn

## AI's Responsibilities (What to Demand)

1. **Concrete tradeoffs**: Not "it depends", but "if X then A wins, if Y then B wins"
2. **Experience synthesis**: "Companies at scale do X because of Y problem"
3. **Honest limitations**: "I don't know" is better than hallucination
4. **Alternative perspectives**: Challenge your ideas, suggest different approaches
5. **Explain the 'why'**: Don't just give solutions, teach the reasoning

---

## Quick Reference Card

**When stuck on a decision**:
1. State the problem with concrete constraints
2. Ask AI for 3 different approaches
3. Challenge each approach with failure scenarios
4. Pick one, document reasoning in ADR
5. Build it, measure it, learn from reality

**When AI gives advice**:
1. Ask: "What are the downsides?"
2. Ask: "At what scale does this break?"
3. Ask: "Show me a real-world example"
4. Ask: "What would you do differently?"

**When learning a technology**:
1. Ask: "What problem does this solve that alternatives don't?"
2. Ask: "When should I NOT use this?"
3. Build something small with it
4. Break it, see what fails

```