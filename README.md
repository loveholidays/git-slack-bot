# git-slack-bot

git-slack-bot automatically posts your GitHub pull request messages to a Slack channel of your choosing. It listens to GitHub events via a webhook and can post new PRs, react with emojis to signify merges and approvals, and post review comments under the original Slack post in a thread.

## Features

- 🔔 **Automated PR notifications** - Get notified when PRs are opened, merged, or closed
- 💬 **Threaded comments** - PR review comments appear as threaded replies in Slack
- 😀 **Emoji reactions** - Visual indicators for PR approvals, merges, and closures
- 👥 **Team member mapping** - Maps GitHub users to Slack users for proper @mentions
- 🎯 **Selective notifications** - Configure which users and events to track
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
2. **Create Slack App**: Set up a Slack App with chat:write and reactions:write permissions
3. **Configure the bot**: Create a `config.yaml` file (see example below)
4. **Deploy**: Use Docker or binary deployment
5. **Set webhook URL**: Point your GitHub App webhook to your deployment

### Data Science code-review setup

Use [config.example.yaml](config.example.yaml) as the starting point. It is preconfigured for the Data Science team and its three selected repositories. Keep `teamMemberRole: "member"` to exclude Data Science team maintainers. A non-draft PR is posted when opened; a draft PR is posted only after GitHub emits `ready_for_review`.

When installing the GitHub App, select **Only select repositories** and choose those same repositories. The application allowlist is a second safeguard in case the App receives webhooks from another repository.

### Example Configuration

```yaml
github:
  token: "ghp_your_github_token_here"
  secretKey: "your_webhook_secret_here"
  org: "your-github-org"
  team: "Data Science"
  # "member" excludes GitHub team maintainers; "all" is the backwards-compatible default.
  teamMemberRole: "member"
  # Full GitHub repository names. Webhooks for any other repository are ignored.
  allowedRepos:
    - "loveholidays/insights-auto-optimisation"
    - "loveholidays/koios"
    - "loveholidays/data-science-dashboards"
  ignoredPRUsers: ["dependabot[bot]", "renovate[bot]"]
  ignoredCommentUsers: ["github-actions[bot]"]

slack:
  token: "xoxb-your-slack-bot-token-here"
  # #data-science-code-review
  channelID: "C09NJQKR1H7"
  githubEmailToSlackEmail:
    - githubEmail: "john.doe"
      slackEmail: "john.doe@company.com"
    - githubEmail: "jane.smith"
      slackEmail: "jane.smith@company.com"
  emoji:
    approve: "white_check_mark"
    merge: "merged"
    close: "x"
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
  - `chat:write.public` - Post to public channels without joining
  - `reactions:write` - Add emoji reactions
  - `channels:read` - List public channels (optional, for channel name resolution)

### Infrastructure
- A publicly accessible endpoint for webhook delivery
- Docker runtime or Go environment for deployment

## Detailed Configuration
- `github`:  
  - `token`: The security token of the github app, which will send events through a webhook
  - `secretKey`: The secret key of the github webhook to verify incoming events against
  - `org`: The github organization the team is in
  - `team`: The team which has the members to post PR for
  - `teamMemberRole`: GitHub team role to notify for. Set to `member` to exclude team maintainers. Valid values are `all` (default), `member`, and `maintainer`.
  - `allowedRepos`: Optional allowlist of full `owner/repository` names. Set this to one repository to ensure the bot processes webhooks only for that repository.
  - `ignoredPRUsers`: Users in the github team to ignore opened PRs for. Their comments will still show up in threads.
  - `ignoredCommentUsers`: Users to ignore PR comments from. Recommended to add any automated github app account
- `slack`:
  - `token`: The security token of the slack app, which will send messages to a slack channel
  - `channelID`: The slack channel id to post the PR messages to
  - `githubEmailToSlackEmail`: Mapping between github and slack users. Needed to be able to use `@mention`s for the
correct user. Any missing users will be posted with their github user names into the slack channel
    - `githubEmail`: The github **USERNAME** of a team member
    - `slackEmail`: The slack email of the same team member
  - `emoji`:
    - `approve`: The emoji to use as a reaction when a PR is approved
    - `merge`: The emoji to use as a reaction when a PR is merged
    - `close`: The emoji to use as a reaction when a PR is closed

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
