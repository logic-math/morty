# Morty Plan Mode - Quick Reference Guide

## What is Plan Mode?

Plan Mode is an **interactive PRD refinement system** that uses Claude Code to transform rough ideas into comprehensive, actionable problem descriptions through dialogue.

## Quick Start

```bash
# 1. Create initial PRD
cat > my_idea.md << 'EOF'
# My App Idea
Brief description of what you want to build
EOF

# 2. Launch plan mode
morty plan my_idea.md

# 3. Engage in dialogue with Claude
# - Answer questions
# - Clarify requirements
# - Explore scenarios
# - Confirm decisions

# 4. Project is auto-generated
cd my-app
morty monitor
```

## How It Works

### 1. You Provide Initial PRD
Just a rough idea is fine:
```markdown
# Todo App
- Add tasks
- Mark complete
- Save to file
```

### 2. Claude Engages in Dialogue

**Phase 1: Understanding**
```
Claude: "I understand you want a todo app. Let me ask some clarifying questions:
1. Who are the primary users?
2. What's the expected scale (personal use vs. team)?
3. Are there any specific constraints?"
```

**Phase 2: Deep Dive**
```
Claude: "Let's explore the 'mark complete' feature:
- What if a user accidentally marks something complete?
- Should there be an undo function?
- Do you need a completion history?"
```

**Phase 3: Validation**
```
Claude: "Let me summarize what we've discussed:
- Personal todo app for developers
- CLI interface
- JSON file storage
- Undo/redo support
Is this correct?"
```

**Phase 4: Synthesis**
```
Claude: "I'll now generate the comprehensive problem description..."
[Creates problem_description.md with all details]
```

### 3. Project Auto-Generated

After dialogue completes, you get:
```
my-app/
├── .morty/
│   ├── PROMPT.md              # Development instructions
│   ├── fix_plan.md            # Task breakdown
│   ├── AGENT.md               # Build/test commands
│   └── specs/
│       └── problem_description.md  # Refined PRD
├── src/
├── README.md
└── .gitignore
```

## Claude Configuration

Plan mode launches Claude with these flags:

```bash
claude \
  -p "<interactive prompt with your PRD>" \
  --continue \
  --dangerously-skip-permissions \
  --allowedTools Read Write Glob Grep WebSearch WebFetch
```

### Why These Flags?

**`--continue`**
- Maintains context across the dialogue
- Remembers previous questions and answers
- Builds understanding incrementally

**`--dangerously-skip-permissions`**
- Allows full tool access for exploration
- Enables research and file operations
- Needed for comprehensive analysis

**`--allowedTools`**
- `Read/Write`: Create and edit files
- `Glob/Grep`: Search and analyze
- `WebSearch/WebFetch`: Research technologies and patterns

## System Prompt Highlights

The plan mode system prompt enables Claude to:

### Deep Exploration
- Ask probing questions
- Challenge assumptions
- Explore edge cases
- Identify hidden requirements

### Structured Thinking
- Break down complex problems
- Identify patterns
- Recognize gaps
- Map relationships

### Technical Insight
- Assess feasibility
- Suggest technologies
- Identify challenges
- Recommend architectures

### User-Centric Analysis
- Understand user personas
- Identify core features
- Prioritize by value
- Consider accessibility

## Dialogue Techniques

### 5 Whys
```
You: "Users need authentication"
Claude: "Why do users need to authenticate?"
You: "To save their data"
Claude: "Why is saving data important?"
You: "So they can access it later"
Claude: "Why is later access critical?"
[Continue to root cause...]
```

### What-If Scenarios
```
Claude: "What if the user loses internet connection?"
Claude: "What if two users modify the same data?"
Claude: "What if the API rate limit is reached?"
```

### User Journey Mapping
```
Claude: "Let's walk through a new user's first experience:
1. They open the app...
2. What do they see?
3. What's their first action?
4. What feedback do they get?"
```

## Question Patterns

Claude uses these patterns to explore deeply:

- **"What if..."** - Explore scenarios and edge cases
- **"Why..."** - Understand motivations and goals
- **"How..."** - Dive into mechanics and implementation
- **"When..."** - Clarify timing and triggers
- **"Who..."** - Identify actors and roles

## Output: problem_description.md

The final output includes:

```markdown
# Problem Description: [Project Name]

## Executive Summary
[2-3 paragraphs: What, Why, Who, Success]

## Problem Statement
[Clear articulation of the problem]

## Goals and Objectives
- Primary Goal
- Secondary Goals
- Success Metrics

## Target Users
### Primary Users
- Persona
- Needs
- Pain Points
- Goals

## Functional Requirements
### Core Features (Must Have)
1. Feature Name
   - Description
   - User Story
   - Acceptance Criteria

### Secondary Features (Should Have)
[...]

## Non-Functional Requirements
- Performance
- Security
- Usability
- Reliability

## Technical Specifications
- Technology Stack
- Architecture
- Dependencies

## User Stories
### Epic 1: [Name]
- As a [user]
- I want [goal]
- So that [benefit]

## Edge Cases
[Scenarios and expected behavior]

## Development Approach
- Phase 1: Foundation
- Phase 2: Core Features
- Phase 3: Polish

## Timeline Estimate
[Duration for each phase]

## Risks and Mitigations
[Identified risks with strategies]
```

## Tips for Effective Dialogue

### Do's ✅
- **Be specific** when answering questions
- **Think out loud** about trade-offs
- **Ask for clarification** if confused
- **Share context** about users and goals
- **Explore alternatives** suggested by Claude
- **Take your time** - good requirements need thought

### Don'ts ❌
- **Don't rush** through questions
- **Don't assume** Claude knows your domain
- **Don't skip** edge case discussions
- **Don't be vague** - specific > general
- **Don't ignore** Claude's concerns

## Example Dialogue

```
You: morty plan todo_app.md

Claude: "I've analyzed your initial PRD for a todo app. Let me ask some
clarifying questions:

1. Who are the primary users?
2. What's the expected scale?
3. What platforms should we support?"

You: "Primary users are developers. Personal use, not team. CLI only."

Claude: "Great! For developers using CLI, let's explore the interface:
- What's more important: speed or features?
- Should it integrate with other tools (git, editors)?
- Do you want keyboard shortcuts or commands?"

You: "Speed is critical. Git integration would be nice. Commands are fine."

Claude: "Let's talk about data persistence:
- Where should tasks be stored?
- Should it sync across machines?
- What if the storage file is corrupted?"

You: "Local JSON file. No sync needed. Show error and backup."

Claude: "Excellent! Let me explore edge cases:
- What if someone runs two instances simultaneously?
- Should completed tasks be archived or deleted?
- How do you handle very long task descriptions?"

[Dialogue continues...]

Claude: "Based on our discussion, I'll now generate the comprehensive
problem description..."

[Generates problem_description.md and project structure]

✅ Project 'todo-app' created successfully!
```

## Completion Signal

Claude outputs this marker when done:
```markdown
<!-- PLAN_MODE_COMPLETE -->
```

This signals Morty to:
1. Extract `problem_description.md`
2. Generate project structure
3. Create PROMPT.md, fix_plan.md, AGENT.md
4. Initialize git repository

## After Plan Mode

Once the project is generated:

```bash
cd my-app

# Review generated files
cat .morty/specs/problem_description.md
cat .morty/PROMPT.md
cat .morty/fix_plan.md

# Customize if needed
vim .morty/PROMPT.md

# Start development
morty monitor
```

## Troubleshooting

### Claude doesn't ask questions
- Check if PRD file is too detailed (plan mode works best with rough ideas)
- Ensure system prompt is loaded correctly

### Project not generated
- Verify Claude created `problem_description.md`
- Check for `<!-- PLAN_MODE_COMPLETE -->` marker in output
- Look for the file in working directory

### Dialogue feels shallow
- Provide more context in your answers
- Ask Claude to explore specific areas deeper
- Challenge Claude's assumptions

### Session lost
- Plan mode uses `--continue` for context
- If interrupted, restart: `morty plan <prd.md>`
- Previous context may be lost

## Best Practices

1. **Start rough** - Don't over-specify in initial PRD
2. **Engage deeply** - Answer questions thoughtfully
3. **Explore edge cases** - Don't skip the "what ifs"
4. **Confirm understanding** - Ask Claude to summarize
5. **Review output** - Read problem_description.md carefully
6. **Customize** - Edit PROMPT.md for your preferences

## Advanced Usage

### Custom Project Name
```bash
morty plan requirements.md my-custom-name
```

### Multiple Iterations
```bash
# First iteration
morty plan rough_idea.md

# Review problem_description.md
# Refine if needed
morty plan refined_idea.md v2
```

### Domain-Specific PRDs
```bash
# For ML projects
morty plan ml_model_requirements.md

# For web apps
morty plan webapp_requirements.md

# For CLI tools
morty plan cli_tool_requirements.md
```

## Summary

Plan Mode = **Interactive PRD Refinement** → **Comprehensive Problem Description** → **Auto-Generated Project**

**Key Benefits:**
- Transform rough ideas into detailed requirements
- Explore requirements through dialogue
- Generate comprehensive documentation
- Auto-create project structure with context
- Start development with clarity

**Remember:** The quality of the final problem description depends on the quality of the dialogue. Take your time, be thoughtful, and explore deeply!

---

For more details, see:
- `README.md` - Full documentation
- `prompts/plan_mode_system.md` - System prompt
- `morty_plan.sh` - Implementation
