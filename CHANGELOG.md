# Morty Changelog

## v0.2.1 - Git Auto-Commit (2026-02-14)

### Added
- **Git Auto-Commit**: Automatic commit after each successful loop
  - `git_auto_commit()` function in `lib/common.sh`
  - Auto-commits changes with loop metadata
  - Creates restore points for easy rollback

- **Rollback Command**: `morty rollback <loop-number>`
  - Rollback to any previous loop iteration
  - Interactive confirmation before reset
  - Finds commits by loop number

- **History Command**: `morty history`
  - Shows last 20 loop commits
  - Displays loop numbers and timestamps
  - Helps identify rollback targets

### Changed
- `morty_loop.sh`: Integrated git_auto_commit() calls
  - Commits after successful loops (case 0)
  - Commits on completion (case 3)
- `morty`: Added rollback and history command routing
- `install.sh`: Updated help text with new commands
- `README.md`: Added Git Auto-Commit section

### Benefits
- **Safety**: Every loop creates a restore point
- **Debugging**: Track when issues were introduced
- **Experimentation**: Try changes with confidence
- **Transparency**: Clear history of loop actions

---

## v0.2.0 - Plan Mode Edition (2026-02-14)

### Major Changes

#### Added
- **Plan Mode**: Interactive PRD refinement with Claude Code
  - `morty_plan.sh`: Complete plan mode implementation
  - `prompts/plan_mode_system.md`: 3000+ line system prompt
  - 4-phase dialogue framework (Understanding → Deep Dive → Validation → Synthesis)
  - Auto-generates comprehensive `problem_description.md`
  - Auto-creates project structure with context-rich files
  
- **Documentation**
  - `PLAN_MODE_GUIDE.md`: Comprehensive quick reference guide
  - `README.md`: Complete rewrite with plan mode focus
  
- **Testing**
  - `test_plan_mode.sh`: 10 comprehensive tests for plan mode

#### Removed
- `morty_init.sh`: Replaced by plan mode
- `morty_import.sh`: Replaced by plan mode
- `test_morty.sh`: Old tests for removed features
- `DEMO.md`: Old demo documentation
- `PROJECT_SUMMARY.md`: Old project summary
- `QUICKSTART.md`: Old quickstart guide

#### Changed
- `morty`: Updated command routing (removed init/import, added plan)
- `install.sh`: Updated to install plan mode components

### Features

**Plan Mode Capabilities**:
- Interactive dialogue with Claude Code
- Deep exploration using 5 Whys, What-If scenarios
- User journey mapping
- Constraint exploration
- Comprehensive output template
- Project type detection (Python, Node.js, Rust, Go)
- Auto-generated PROMPT.md, fix_plan.md, AGENT.md

**Claude Configuration**:
- `--continue`: Context preservation
- `--dangerously-skip-permissions`: Full tool access
- `--allowedTools`: Read, Write, Glob, Grep, WebSearch, WebFetch

### Testing
- All 10 plan mode tests passing
- Verified command routing
- Verified system prompt content
- Verified project generation

### Documentation
- Complete README rewrite (450+ lines)
- Plan Mode Quick Reference Guide (420+ lines)
- System prompt documentation (3000+ lines)

---

## v0.1.0 - Initial Release (2026-02-14)

### Added
- `morty_init.sh`: Create new projects from scratch
- `morty_import.sh`: Import from PRD documents
- `morty_enable.sh`: Enable Morty in existing projects
- `morty_loop.sh`: Main development loop
- `morty_monitor.sh`: tmux monitoring
- `lib/common.sh`: Shared utilities
- Basic documentation and tests

### Features
- Project initialization
- PRD import with regex parsing
- Development loop with lifecycle management
- Exit hooks for PROMPT.md updates
- tmux 3-pane monitoring
- Project type detection

### Testing
- 5 tests for init, import, enable
- All tests passing

---

## Migration Guide: v0.1.0 → v0.2.0

### Breaking Changes

**Removed Commands**:
- `morty init <project>` → Use `morty plan <prd.md>`
- `morty import <prd.md>` → Use `morty plan <prd.md>`

**New Workflow**:
```bash
# Old way (v0.1.0)
morty init my-project
# or
morty import requirements.md

# New way (v0.2.0)
morty plan requirements.md
# This launches interactive dialogue, then generates project
```

### What's Different?

**v0.1.0**: Static PRD parsing
- Read PRD file
- Extract tasks with regex
- Generate basic structure

**v0.2.0**: Interactive refinement
- Read initial PRD
- Launch Claude Code dialogue
- Refine through conversation
- Generate comprehensive problem description
- Create context-rich project

### Benefits of v0.2.0

1. **Better Requirements**: Dialogue uncovers hidden needs
2. **Deeper Understanding**: Exploration techniques (5 Whys, What-If)
3. **Richer Context**: Generated files reference each other
4. **Complete Documentation**: problem_description.md is comprehensive
5. **User Involvement**: Collaborative refinement process

---

## Roadmap

### Future Enhancements
- [ ] Plan mode templates for different domains (ML, web, CLI)
- [ ] Save and resume plan mode sessions
- [ ] Export problem_description.md to other formats
- [ ] Integration with issue trackers
- [ ] Metrics and analytics for development loop
- [ ] Log rotation
- [ ] Dry-run mode

### Feedback Welcome
Please report issues or suggestions at the project repository.

