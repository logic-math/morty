# Morty Examples

Example configurations and scripts for using Morty in different scenarios.

## Enterprise CLI Wrapper

### `ai_cli_wrapper.sh`

Example enterprise Claude Code CLI wrapper that demonstrates:

- **Authentication**: SSO, token, and certificate-based auth
- **Configuration**: Environment-specific configs (dev, staging, prod)
- **Proxy Support**: Corporate proxy configuration
- **Team Tracking**: Billing and usage tracking
- **Metrics Logging**: Usage metrics for monitoring

**Features:**
- Multiple authentication methods (SSO, token, cert)
- Environment-based configuration
- HTTP proxy support
- Team identification for billing
- Usage metrics logging
- Debug mode
- Error handling

**Usage:**

```bash
# Copy and customize for your organization
cp ai_cli_wrapper.sh /usr/local/bin/ai_cli
chmod +x /usr/local/bin/ai_cli

# Edit configuration
vim /opt/company/config/claude_prod.conf

# Use with Morty
export CLAUDE_CODE_CLI="ai_cli --config prod --auth sso"
morty plan requirements.md
```

**Configuration File Example:**

```bash
# /opt/company/config/claude_prod.conf

# API Configuration
ENTERPRISE_API_ENDPOINT="https://api.company.com/claude"
ENTERPRISE_PROXY="http://proxy.company.com:8080"

# Rate Limiting
MAX_REQUESTS_PER_HOUR=100

# Logging
LOG_LEVEL="info"
LOG_FILE="/var/log/company/claude.log"
```

**Authentication Methods:**

1. **SSO (Single Sign-On)**
   ```bash
   ai_cli --auth sso
   ```
   Requires: `company-sso-auth` command

2. **Token**
   ```bash
   export ENTERPRISE_API_KEY="your-api-key"
   ai_cli --auth token
   ```

3. **Certificate**
   ```bash
   ai_cli --auth cert
   ```
   Requires: Certificate files in config directory

**With Morty:**

```bash
# Set environment
export CLAUDE_CODE_CLI="ai_cli --config prod --auth sso --team data-science"

# Use normally
morty plan requirements.md
morty monitor
```

## Common Use Cases

### 1. Enterprise Setup with SSO

```bash
#!/bin/bash
# setup-morty-enterprise.sh

# Configure enterprise CLI
export CLAUDE_CODE_CLI="ai_cli --config prod --auth sso"
export MAX_LOOPS=100
export LOOP_DELAY=10

# Add to PATH
export PATH="$HOME/.local/bin:$PATH"

# Verify setup
echo "Testing enterprise CLI..."
$CLAUDE_CODE_CLI --version

echo "Enterprise setup complete!"
echo "Usage: morty plan requirements.md"
```

### 2. Multi-Environment Configuration

```bash
#!/bin/bash
# switch-env.sh

ENV=${1:-dev}

case $ENV in
    dev)
        export CLAUDE_CODE_CLI="ai_cli --config dev --auth token"
        export MAX_LOOPS=50
        export LOOP_DELAY=5
        ;;
    staging)
        export CLAUDE_CODE_CLI="ai_cli --config staging --auth sso"
        export MAX_LOOPS=100
        export LOOP_DELAY=10
        ;;
    prod)
        export CLAUDE_CODE_CLI="ai_cli --config prod --auth sso --team production"
        export MAX_LOOPS=200
        export LOOP_DELAY=30
        ;;
    *)
        echo "Unknown environment: $ENV"
        exit 1
        ;;
esac

echo "Switched to $ENV environment"
echo "CLAUDE_CODE_CLI: $CLAUDE_CODE_CLI"
```

Usage:
```bash
source switch-env.sh staging
morty monitor
```

### 3. Team-Specific Configuration

```bash
#!/bin/bash
# team-config.sh

TEAM=${1:-default}

# Base configuration
export CLAUDE_CODE_CLI="ai_cli --config prod --auth sso"

# Team-specific settings
case $TEAM in
    data-science)
        export CLAUDE_CODE_CLI="$CLAUDE_CODE_CLI --team data-science"
        export MAX_LOOPS=200
        ;;
    backend)
        export CLAUDE_CODE_CLI="$CLAUDE_CODE_CLI --team backend"
        export MAX_LOOPS=100
        ;;
    frontend)
        export CLAUDE_CODE_CLI="$CLAUDE_CODE_CLI --team frontend"
        export MAX_LOOPS=150
        ;;
esac

echo "Team: $TEAM"
echo "CLI: $CLAUDE_CODE_CLI"
```

### 4. Debug Mode

```bash
#!/bin/bash
# debug-morty.sh

# Enable debug mode
export CLAUDE_CODE_CLI="ai_cli --debug --auth token"
export MAX_LOOPS=5
export LOOP_DELAY=0

# Run with verbose output
set -x
morty start
```

### 5. Rate-Limited Setup

```bash
#!/bin/bash
# rate-limited-morty.sh

# Slow down to avoid rate limits
export CLAUDE_CODE_CLI="ai_cli --config prod --auth sso"
export MAX_LOOPS=50
export LOOP_DELAY=60  # 1 minute between loops

echo "Running with rate limiting..."
echo "Delay: ${LOOP_DELAY}s between loops"

morty monitor
```

## Customization Guide

### Creating Your Own Wrapper

1. **Copy the example:**
   ```bash
   cp ai_cli_wrapper.sh my_cli.sh
   ```

2. **Customize authentication:**
   ```bash
   # Edit the auth section
   case $AUTH_METHOD in
       your-method)
           # Your authentication logic
           ;;
   esac
   ```

3. **Add your configuration:**
   ```bash
   # Load your config files
   source /your/config/path
   ```

4. **Test the wrapper:**
   ```bash
   ./my_cli.sh --version
   ./my_cli.sh -p "Hello, Claude"
   ```

5. **Install and use:**
   ```bash
   sudo cp my_cli.sh /usr/local/bin/my_cli
   chmod +x /usr/local/bin/my_cli
   export CLAUDE_CODE_CLI="my_cli"
   morty plan requirements.md
   ```

### Wrapper Requirements

Your CLI wrapper must:
1. Accept all Claude Code arguments and pass them through
2. Exit with the same exit code as Claude Code
3. Output Claude Code's stdout/stderr without modification
4. Handle authentication before calling Claude Code

**Minimal Example:**

```bash
#!/bin/bash
# minimal_wrapper.sh

# Your setup
export API_KEY="your-key"
export API_ENDPOINT="https://your-endpoint.com"

# Call Claude Code with all arguments
exec claude --api-key "$API_KEY" --endpoint "$API_ENDPOINT" "$@"
```

## Testing Your Configuration

### Test CLI Wrapper

```bash
# Test basic execution
$CLAUDE_CODE_CLI --version

# Test with simple prompt
$CLAUDE_CODE_CLI -p "Say hello"

# Test with Morty
echo "# Test" > test.md
morty plan test.md test-project
```

### Verify Environment

```bash
#!/bin/bash
# verify-setup.sh

echo "=== Morty Configuration ==="
echo "CLAUDE_CODE_CLI: ${CLAUDE_CODE_CLI:-claude}"
echo "MAX_LOOPS: ${MAX_LOOPS:-50}"
echo "LOOP_DELAY: ${LOOP_DELAY:-5}"
echo ""

echo "=== Testing CLI ==="
$CLAUDE_CODE_CLI --version || echo "CLI test failed"
echo ""

echo "=== Morty Installation ==="
which morty || echo "Morty not in PATH"
morty --version
echo ""

echo "Setup verification complete!"
```

## Troubleshooting

### Wrapper Not Working

1. Check permissions: `chmod +x /path/to/wrapper`
2. Test directly: `./wrapper --version`
3. Check PATH: `which wrapper`
4. Enable debug: `wrapper --debug`

### Authentication Fails

1. Verify credentials are set
2. Check network/proxy settings
3. Test auth command separately
4. Check certificate validity (if using certs)

### Morty Can't Find CLI

1. Verify environment variable: `echo $CLAUDE_CODE_CLI`
2. Test CLI directly: `$CLAUDE_CODE_CLI --version`
3. Check if CLI is in PATH: `which $CLAUDE_CODE_CLI`
4. Use absolute path: `export CLAUDE_CODE_CLI="/full/path/to/cli"`

## Additional Resources

- [Configuration Guide](../CONFIGURATION.md) - Complete configuration reference
- [Main README](../../README.md) - Morty documentation
- [Plan Mode Guide](../PLAN_MODE_GUIDE.md) - Plan mode usage

---

**Last Updated**: 2026-02-14
**Examples Version**: 0.2.1
