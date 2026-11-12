# git-slack-bot

git-slack-bot automatically posts your GitHub pull request messages to a Slack channel of your choosing. It listens to GitHub events via a webhook and can post new PRs, react with emojis to signify merges and approvals, and post review comments under the original Slack post in a thread.

## Features

- 🔔 **Automated PR notifications** - Get notified when PRs are opened, merged, or closed
- 💬 **Threaded comments** - PR review comments appear as threaded replies in Slack
- 😀 **Emoji reactions** - Visual indicators for PR approvals, merges, and closures
- 👥 **Team member mapping** - Maps GitHub users to Slack users for proper @mentions
- 🎯 **Selective notifications** - Route team-authored or team-assigned PRs using repository filters and channel or direct-message destinations. Drafts wait until ready for review, and repeated events reuse the existing Slack post. Comments and status reactions follow each destination's thread.
- 🔒 **Secure webhooks** - Validates GitHub webhook signatures for security

## Installation

### Using Docker (Recommended)

```bash
# Pull the latest image
docker pull ghcr.io/loveholidays/git-slack-bot:latest

# Run with configuration file
docker run -d \
  --name git-slack-bot \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  ghcr.io/loveholidays/git-slack-bot:latest
```

### Using Docker Compose

```yaml
version: '3.8'
services:
  git-slack-bot:
    image: ghcr.io/loveholidays/git-slack-bot:latest
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/app/config.yaml
    environment:
      - CONFIG_PATH=/app/config.yaml
    restart: unless-stopped
```

### From GitHub Releases

1. Download the latest binary for your platform from [GitHub Releases](https://github.com/loveholidays/git-slack-bot/releases)
2. Make it executable: `chmod +x git-slack-bot`
3. Create your configuration file (see [Configuration](#configuration))
4. Run: `./git-slack-bot --config config.yaml`

### From Source

```bash
# Clone the repository
git clone https://github.com/loveholidays/git-slack-bot.git
cd git-slack-bot

# Build the application
make build

# Run with configuration
./bin/git-slack-bot --config config.yaml
```

## Quick Start

1. **Create GitHub App**: Set up a GitHub App with webhook permissions
2. **Create Slack App**: Configure the [Slack app permissions](#slack-app-setup). Direct-message rules also require `im:write` and `im:history`.
3. **Configure the bot**: Create a `config.yaml` file (see example below)
4. **Deploy**: Use Docker or binary deployment
5. **Set webhook URL**: Point your GitHub App webhook to your deployment

### Example Configuration

```yaml
github:
  token: "ghp_your_github_token_here"
  secretKey: "your_webhook_secret_here"
  org: "your-github-org"
  team: "your-team-name"
  ignoredPRUsers: ["dependabot[bot]", "renovate[bot]"]
  ignoredCommentUsers: ["github-actions[bot]"]

slack:
  token: "xoxb-your-slack-bot-token-here"
  githubEmailToSlackEmail:
    - githubEmail: "john.doe"
      slackEmail: "john.doe@company.com"
    - githubEmail: "jane.smith"
      slackEmail: "jane.smith@company.com"
  emoji:
    approve: "white_check_mark"
    merge: "merged"
    close: "x"

notifications:
  - relationship: author
    repositories: ["*"]
    excludedRepositories: ["your-org/noisy-repo"]
    destination: channel
    channelID: "C1234567890"
  - relationship: assignee
    repositories: ["your-org/deployments"]
    destination: directMessage
```

## Prerequisites

Before setting up git-slack-bot, you'll need:

### GitHub App Setup
- A GitHub App with the following permissions:
  - **Repository permissions**:
    - Pull requests: Read & Write
    - Issues: Read & Write
    - Metadata: Read
    - Contents: Read
  - **Organization permissions**:
    - Members: Read (to access team information)
  - **Subscribe to events**:
    - Pull request
    - Pull request review
    - Pull request review comment

### Slack App Setup
- A Slack App with the following OAuth scopes:
  - `chat:write` - Post messages to channels
  - `im:write` - Open direct messages (required for `destination: directMessage`)
  - `im:history` - Find existing DM notifications for deduplication and thread updates (required for `destination: directMessage`)
  - `chat:write.public` - Post to public channels without joining
  - `reactions:write` - Add emoji reactions
  - `channels:read` - List public channels (optional, for channel name resolution)

If any notification rule uses `destination: directMessage`, add **both** IM scopes under **OAuth & Permissions → Scopes → Bot Token Scopes** in [your Slack app settings](https://api.slack.com/apps). Channel-only configurations do not need these additional scopes.

For an already installed app, adding scopes does not grant them to the installed bot automatically:

1. Reinstall/reauthorize the app in the workspace after adding the scopes.
2. If Slack requires approval, use **Re-request approval** and have a workspace owner or app manager approve the new scopes, then complete reauthorization.
3. If Slack issues a different bot token, update the deployed `SLACK_TOKEN` secret and restart the bot to load it.

### Infrastructure
- A publicly accessible endpoint for webhook delivery
- Docker runtime or Go environment for deployment

## Detailed Configuration
- `github`:  
  - `token`: The security token of the github app, which will send events through a webhook
  - `secretKey`: The secret key of the github webhook to verify incoming events against
  - `org`: The github organization the team is in
  - `team`: The team which has the members to post PR for
  - `notifyAssignedFluxPRs`: Legacy optional boolean, default `false`. When `notifications` is omitted, include PRs in `loveholidays/flux` assigned to team members in the configured Slack channel.
  - `ignoredPRUsers`: Users in the github team to ignore opened PRs for. Their comments will still show up in threads.
  - `ignoredCommentUsers`: Users to ignore PR comments from. Recommended to add any automated github app account
- `slack`:
  - `token`: The security token of the slack app, which will send messages to a slack channel
  - `channelID`: Legacy channel for PR notifications when `notifications` is omitted
  - `githubEmailToSlackEmail`: Mapping between github and slack users. Needed to be able to use `@mention`s for the
correct user. Any missing users will be posted with their github user names into the slack channel
    - `githubEmail`: The github **USERNAME** of a team member
    - `slackEmail`: The slack email of the same team member
  - `emoji`:
    - `approve`: The emoji to use as a reaction when a PR is approved
    - `merge`: The emoji to use as a reaction when a PR is merged
    - `close`: The emoji to use as a reaction when a PR is closed

### Notification rules

`notifications` is a list of independent rules. Each rule uses the same fields:

| Field | Meaning |
| --- | --- |
| `relationship` | `author` or `assignee`; only members of the configured GitHub team qualify. |
| `repositories` | Full `owner/repository` names, or `"*"` for all repositories. Omitted or empty means all repositories. |
| `excludedRepositories` | Repository names to exclude from this rule. Exclusions override inclusions. |
| `destination` | `channel` or `directMessage`. |
| `channelID` | Required for `channel`; omit for `directMessage`. |

All matching rules apply. A PR that a team member both authors and is assigned can notify a channel and that member's DM. Rules targeting the same conversation share one PR notification and one thread update per event. Exclusions only affect their own rule. `github.ignoredRepos` continues to exclude repositories globally.

For DMs, the relationship determines recipients: the team author or every team assignee. Recipients must have a `slack.githubEmailToSlackEmail` mapping. Despite its name, each mapping's `githubEmail` is a GitHub login. Missing mappings and Slack failures are logged without falling back to a channel. The bot identity comes from `slack.token`.

When `notifications` is omitted, existing configuration retains its behaviour, including channel delivery for `github.notifyAssignedFluxPRs`. An explicit list replaces legacy routing; `notifications: []` disables all notifications. `slack.channelID` is optional with explicit rules.

For example, replace `github.notifyAssignedFluxPRs: true` and `slack.channelID` with:

```yaml
notifications:
  - relationship: author
    repositories: ["*"]
    destination: channel
    channelID: C0837M8C478
  - relationship: assignee
    repositories: ["loveholidays/flux"]
    destination: directMessage
```


## Usage Examples

### Basic Webhook Setup

1. **Set your webhook URL** in your GitHub App settings:
   ```
   https://your-domain.com/webhook
   ```

2. **Configure webhook events** to send:
   - Pull requests
   - Pull request reviews
   - Pull request review comments

3. **Set webhook secret** (must match `secretKey` in config)

### Environment Variables

You can also configure the bot using environment variables:

```bash
export GITHUB_TOKEN="ghp_your_token_here"
export GITHUB_SECRET_KEY="your_webhook_secret"
export SLACK_TOKEN="xoxb-your_slack_token"
export SLACK_CHANNEL_ID="C1234567890"
```

### Testing Your Setup

1. **Health Check**: Visit `http://your-deployment:8080/health` to verify the service is running
2. **Test Webhook**: Create a test PR to verify webhook delivery and Slack posting

## Troubleshooting

### Common Issues

#### Bot not receiving webhooks
- ✅ Check that your webhook URL is publicly accessible
- ✅ Verify the webhook secret matches your configuration
- ✅ Ensure your GitHub App has the correct permissions
- ✅ Check GitHub's webhook delivery logs for errors

#### Messages not appearing in Slack
- ✅ Verify your Slack bot token has the required permissions
- ✅ Check that the channel ID is correct (should start with 'C')
- ✅ Ensure the bot is invited to private channels
- ✅ Review application logs for authentication errors

#### User mentions not working
- ✅ Verify `githubEmailToSlackEmail` mapping is correct
- ✅ Use GitHub **usernames** (not emails) in the mapping
- ✅ Use actual Slack **email addresses** in the mapping
- ✅ Check that mapped users exist in both GitHub and Slack

#### Emoji reactions not working
- ✅ Ensure emoji names are valid Slack emoji codes (without colons)
- ✅ Verify the bot has `reactions:write` permission
- ✅ Check that custom emojis exist in your Slack workspace

### Debug Mode

Run with debug logging to troubleshoot issues:

```bash
# Docker
docker run -e LOG_LEVEL=debug ghcr.io/loveholidays/git-slack-bot:latest

# Binary
LOG_LEVEL=debug ./git-slack-bot --config config.yaml
```

### Getting Help

- 📖 Check our [documentation](https://github.com/loveholidays/git-slack-bot/wiki)
- 🐛 [Report bugs](https://github.com/loveholidays/git-slack-bot/issues/new?template=bug_report.yml)
- 💡 [Request features](https://github.com/loveholidays/git-slack-bot/issues/new?template=feature_request.yml)
- 💬 [Join discussions](https://github.com/loveholidays/git-slack-bot/discussions)

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and contribution guidelines.

## License

This project is licensed under the GNU Lesser General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Security

Please see [SECURITY.md](SECURITY.md) for reporting security vulnerabilities.
