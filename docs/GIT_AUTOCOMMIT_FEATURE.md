# Git Auto-Commit Feature

## Overview

Morty now automatically commits changes after each successful development loop iteration, creating restore points that enable easy rollback to any previous state.

## What Was Implemented

### 1. Core Functions (lib/common.sh)

#### `git_auto_commit(loop_count, work_summary)`
- Automatically stages and commits all changes after each loop
- Creates structured commit messages with loop metadata
- Skips commit when no changes are detected
- Returns success/failure status

**Commit message format:**
```
morty: Loop #5 - Loop iteration completed

Auto-committed by Morty development loop.

Loop: 5
Timestamp: 2024-01-15T10:30:45Z
Summary: Loop iteration completed

This commit represents the state after loop iteration 5.
You can rollback to this point using: git reset --hard HEAD~N
```

#### `git_rollback(target_loop)`
- Finds the commit for a specific loop number
- Interactive confirmation before reset
- Resets working directory to that state
- Provides clear user feedback

#### `git_loop_history()`
- Shows last 20 loop commits
- Displays with colored output (hash, date, message)
- Helps identify which loop to rollback to

### 2. Integration (morty_loop.sh)

Auto-commit is called in two places:

1. **After successful loop execution** (case 0):
   ```bash
   git_auto_commit "$loop_count" "Loop iteration completed"
   ```

2. **On project completion** (case 3):
   ```bash
   git_auto_commit "$loop_count" "Project completion"
   ```

### 3. CLI Commands (morty)

Added two new commands:

#### `morty rollback <loop-number>`
```bash
morty rollback 5    # Rollback to loop #5
```

#### `morty history`
```bash
morty history       # Show loop commit history
```

### 4. Documentation

- **README.md**: Added comprehensive "Git Auto-Commit" section
- **CHANGELOG.md**: Documented v0.2.1 with all changes
- **install.sh**: Updated help text with new commands

### 5. Testing (test_git_autocommit.sh)

Created 9 comprehensive tests:
1. Git repository initialization
2. Initial commit creation
3. Auto-commit with changes
4. Auto-commit skips when no changes
5. Multiple auto-commits
6. git_loop_history() functionality
7. git_rollback() commit detection
8. Commit message format validation
9. Working directory clean after commit

**All tests passing ✓**

## Usage Examples

### Basic Workflow

```bash
# 1. Start development loop
morty monitor

# Loop runs automatically, committing after each iteration:
# - Loop #1: morty: Loop #1 - Loop iteration completed
# - Loop #2: morty: Loop #2 - Loop iteration completed
# - Loop #3: morty: Loop #3 - Loop iteration completed

# 2. View loop history
morty history
# Output:
# a2ddd5e - 2 minutes ago - morty: Loop #3 - Loop iteration completed
# 00e341f - 3 minutes ago - morty: Loop #2 - Loop iteration completed
# 9d98c04 - 4 minutes ago - morty: Loop #1 - Loop iteration completed

# 3. Rollback to loop #2 if needed
morty rollback 2
# Prompt: This will reset your working directory. Continue? [y/N]
# Enter: y
# Output: Rolled back to Loop #2
```

### Safety Features

1. **No changes = no commit**: Prevents empty commits
2. **Interactive confirmation**: Rollback requires user confirmation
3. **Clear messaging**: Every operation provides clear feedback
4. **Metadata tracking**: Each commit includes loop number and timestamp

### Requirements

- Project must be a git repository
- Git must be installed and available
- Works with both new and existing repositories

## Benefits

### 1. Safety
Every loop creates a restore point. If Claude Code makes a mistake, you can easily rollback.

### 2. Debugging
Clear history of what happened in each loop. Identify exactly when an issue was introduced.

### 3. Experimentation
Try risky changes knowing you can always rollback. No fear of losing work.

### 4. Transparency
Complete audit trail of Morty's actions. See exactly what changed in each loop.

### 5. Collaboration
Team members can see the development history. Understand the evolution of the codebase.

## Implementation Details

### Commit Detection Logic

The `git_auto_commit()` function:
1. Checks if git is available
2. Checks if directory is a git repository
3. Stages all changes with `git add -A`
4. Checks if there are staged changes
5. Creates commit with structured message
6. Logs success/failure

### Rollback Safety

The `git_rollback()` function:
1. Verifies git is available
2. Verifies git repository exists
3. Searches for commit with loop number
4. Displays commit hash and confirmation prompt
5. Only resets on user confirmation (y/Y)
6. Provides clear success/cancellation messages

### History Display

The `git_loop_history()` function:
1. Uses `git log --grep="morty: Loop"` to filter commits
2. Displays last 20 commits with formatting
3. Shows hash (yellow), date (green), and message
4. Includes usage hint for rollback command

## Testing

Run the test suite:
```bash
cd morty
./test_git_autocommit.sh
```

Expected output:
```
✓ Test 1: Git repository initialized
✓ Test 2: Initial commit created
✓ Test 3: Auto-commit with changes works
✓ Test 4: Auto-commit skips when no changes
✓ Test 5: Multiple auto-commits work
✓ Test 6: git_loop_history() works
✓ Test 7: git_rollback() can find commits
✓ Test 8: Commit message format correct
✓ Test 9: Working directory clean after commit

All tests passed! ✨
```

## Future Enhancements

Potential improvements:
- [ ] Automatic push to remote (optional)
- [ ] Rollback by timestamp instead of loop number
- [ ] Diff view between loops
- [ ] Export loop history to markdown
- [ ] Tag important loops for easy reference
- [ ] Squash loop commits before final push

## Troubleshooting

### Issue: "Not a git repository"
**Solution**: Initialize git in your project:
```bash
cd my-project
git init
git add .
git commit -m "Initial commit"
```

### Issue: "No commit found for Loop #N"
**Solution**: That loop may not have had any changes to commit. Check history:
```bash
morty history
```

### Issue: Commits not showing in history
**Solution**: Ensure you're in the project directory where morty loop ran.

## Version

- **Added in**: v0.2.1
- **Status**: Production ready
- **Tests**: 9/9 passing

## Related Files

- `lib/common.sh` - Core functions (lines 156-261)
- `morty_loop.sh` - Integration (lines 249-280)
- `morty` - CLI commands (lines 50-87)
- `test_git_autocommit.sh` - Test suite
- `README.md` - User documentation
- `CHANGELOG.md` - Version history

---

**Summary**: Git auto-commit provides safety, transparency, and confidence in the Morty development loop by creating automatic restore points after each iteration.
