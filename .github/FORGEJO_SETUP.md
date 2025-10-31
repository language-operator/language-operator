# Forgejo CI/CD Setup Guide

This guide walks through setting up the CI/CD pipelines on Forgejo.

## Prerequisites

- Forgejo instance with Actions enabled
- Forgejo Actions runners configured
- Docker registry credentials for `git.theryans.io`

## Step 1: Enable Forgejo Actions

In your Forgejo `app.ini`:

```ini
[actions]
ENABLED = true
DEFAULT_ACTIONS_URL = https://github.com
```

Restart Forgejo after configuration changes.

## Step 2: Register Actions Runners

On your runner machine(s):

```bash
# Download forgejo-runner
wget https://dl.forgejo.org/runner/latest/forgejo-runner-linux-amd64
chmod +x forgejo-runner-linux-amd64
sudo mv forgejo-runner-linux-amd64 /usr/local/bin/forgejo-runner

# Register runner
forgejo-runner register \
  --instance https://git.theryans.io \
  --token YOUR_RUNNER_TOKEN \
  --name "runner-1" \
  --labels "ubuntu-latest:docker://node:16-bullseye"

# Start runner
forgejo-runner daemon
```

### Runner Token

Get the runner registration token from Forgejo:
1. Go to Repository → Settings → Actions → Runners
2. Click "Add Runner"
3. Copy the registration token

### Runner Labels

Supported labels for GitHub Actions compatibility:
- `ubuntu-latest` - Ubuntu 22.04 runner
- `ubuntu-20.04` - Ubuntu 20.04 runner
- `debian-latest` - Debian runner

## Step 3: Configure Repository Secrets

Navigate to: **Repository → Settings → Secrets**

Add the following secrets:

### Required Secrets

| Secret Name | Description | Example Value |
|-------------|-------------|---------------|
| `REGISTRY_USERNAME` | Registry username | `your-username` |
| `REGISTRY_PASSWORD` | Registry password/token | `your-token-here` |

### Optional Secrets

| Secret Name | Description | When Needed |
|-------------|-------------|-------------|
| `CODECOV_TOKEN` | Codecov upload token | For coverage reports |
| `SLACK_WEBHOOK` | Slack webhook URL | For notifications |

## Step 4: Verify Workflows

Check that workflows are detected:

1. Go to **Repository → Actions**
2. You should see:
   - Build and Push Images
   - Test
   - PR Checks
   - Release Helm Chart

## Step 5: Test the Pipeline

### Trigger a Build

**Option 1: Push to main**
```bash
git checkout main
git commit --allow-empty -m "chore: trigger ci"
git push origin main
```

**Option 2: Create a PR**
```bash
git checkout -b test-ci
echo "test" > test.txt
git add test.txt
git commit -m "test: ci pipeline"
git push origin test-ci
```

Then create a PR in Forgejo UI.

**Option 3: Manual workflow dispatch**
1. Go to Actions → Build and Push Images
2. Click "Run workflow"
3. Select branch
4. Click "Run"

### Check Build Status

1. Go to **Repository → Actions**
2. Click on the workflow run
3. View logs for each job

## Troubleshooting

### Workflows Not Appearing

**Check**: Is Actions enabled?
```bash
# In Forgejo app.ini
[actions]
ENABLED = true
```

**Check**: Are runners registered?
```bash
# On Forgejo: Settings → Actions → Runners
# Should show at least one runner with "idle" status
```

### Builds Failing

**Problem**: "No runners available"
- **Solution**: Register more runners or ensure existing runners are online

**Problem**: "docker: command not found"
- **Solution**: Install Docker on runner machine

**Problem**: "Permission denied pushing to registry"
- **Solution**: Verify REGISTRY_USERNAME and REGISTRY_PASSWORD secrets are correct

### Images Not Pushing

**Problem**: Authentication error
```bash
# Test registry access manually:
docker login git.theryans.io
# Enter credentials from secrets
```

**Problem**: Rate limited
- **Solution**: Use registry mirror or wait for rate limit reset

### Slow Builds

**Problem**: No cache hits
- Forgejo Actions supports GitHub Actions cache
- Ensure `cache-from` and `cache-to` are properly configured in workflows

**Problem**: Downloading dependencies every time
- Consider setting up a local package mirror
- Use volume mounts for Go/npm cache on runners

## Advanced Configuration

### Custom Runner with Docker-in-Docker

For building Docker images, use DinD runner:

```yaml
# runner-config.yaml
labels:
  - "ubuntu-latest:docker://catthehacker/ubuntu:act-latest"
  - "ubuntu-dind:docker://docker:dind"
```

### Multiple Runners for Parallelization

Register multiple runners to speed up matrix builds:

```bash
# Runner 1
forgejo-runner register --name "runner-1" --labels "ubuntu-latest:docker://node:16-bullseye"

# Runner 2
forgejo-runner register --name "runner-2" --labels "ubuntu-latest:docker://node:16-bullseye"

# Runner 3
forgejo-runner register --name "runner-3" --labels "ubuntu-latest:docker://node:16-bullseye"
```

With 3 runners, matrix jobs will run in parallel.

### Runner as Systemd Service

Create `/etc/systemd/system/forgejo-runner.service`:

```ini
[Unit]
Description=Forgejo Actions Runner
After=network.target

[Service]
Type=simple
User=forgejo-runner
WorkingDirectory=/var/lib/forgejo-runner
ExecStart=/usr/local/bin/forgejo-runner daemon
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable forgejo-runner
sudo systemctl start forgejo-runner
sudo systemctl status forgejo-runner
```

### Monitoring Runners

Check runner health:

```bash
# View runner logs
journalctl -u forgejo-runner -f

# Check runner status
systemctl status forgejo-runner

# Test runner connectivity
curl -I https://git.theryans.io
```

## Differences from GitHub Actions

Forgejo Actions is mostly compatible but has some differences:

### Supported

✅ Most GitHub Actions from marketplace
✅ Docker-based actions
✅ JavaScript actions
✅ Matrix builds
✅ Secrets
✅ Caching
✅ Artifacts

### Not Supported / Different

❌ Some GitHub-specific actions (github/codeql-action)
⚠️ Artifact retention (may differ from GitHub)
⚠️ Cache size limits (may differ)
⚠️ Runner labels (must be configured explicitly)

### Workarounds

For GitHub-specific actions, use alternatives:

| GitHub Action | Forgejo Alternative |
|---------------|---------------------|
| `actions/cache` | Supported directly |
| `github/codeql-action` | Use SonarQube or Semgrep |
| `github/super-linter` | Run linters individually |

## Best Practices

1. **Tag Runners Appropriately**: Use labels that match GitHub Actions
2. **Keep Runners Updated**: Regularly update forgejo-runner binary
3. **Monitor Disk Space**: Runners can fill up with Docker images/cache
4. **Use Secrets**: Never hardcode credentials in workflows
5. **Test Locally**: Use `act` to test workflows before pushing

## Support

For issues with Forgejo Actions:
- [Forgejo Actions Documentation](https://forgejo.org/docs/latest/user/actions/)
- [Forgejo Issue Tracker](https://codeberg.org/forgejo/forgejo/issues)

For issues with this project's workflows:
- Open an issue in this repository
- Check [.github/workflows/README.md](.github/workflows/README.md) for workflow docs
