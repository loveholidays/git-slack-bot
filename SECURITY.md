# Security Policy

## Reporting Security Vulnerabilities

We take the security of git-slack-bot seriously. If you discover a security vulnerability, please report it responsibly.

### How to Report

Please report security vulnerabilities by creating a GitHub issue at:

🔒 **https://github.com/loveholidays/git-slack-bot/issues**

Please label your issue with the `security` tag and provide detailed information about the vulnerability.

Include as much of the following information as possible:

- Type of issue (e.g., buffer overflow, SQL injection, cross-site scripting, etc.)
- Full paths of source file(s) related to the manifestation of the issue
- The location of the affected source code (tag/branch/commit or direct URL)
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit the issue

### Response Timeline

We will acknowledge receipt of your vulnerability report within **48 hours** and will send you regular updates about our progress.

We aim to:
- Provide an initial response within 48 hours
- Provide a detailed response within 7 days
- Release a fix within 30 days (depending on complexity)

### Disclosure Policy

- We will coordinate the timing of any public disclosure with you
- We prefer to fully investigate and patch vulnerabilities before any public disclosure
- We will credit you in our security advisory unless you prefer to remain anonymous

## Supported Versions

We provide security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| Latest  | ✅ Yes             |
| < 1.0   | ❌ No              |

## Security Scanning

> **Note**: Automated security scanning features (CodeQL, SARIF uploads, dependency review) require GitHub Advanced Security, which is not enabled on this private repository. These features will be automatically enabled once the repository is made public.
>
> For now, manual security scanning can be performed locally using:
> - `go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...`
> - `go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest && gosec ./...`

## Security Best Practices

When deploying git-slack-bot:

### 🔐 Secrets Management
- Never commit tokens, secrets, or credentials to version control
- Use environment variables or secure secret management systems
- Rotate tokens regularly
- Use least-privilege principle for token permissions

### 🌐 Network Security
- Deploy behind a reverse proxy with TLS termination
- Use HTTPS for all webhook endpoints
- Implement rate limiting to prevent abuse
- Validate all incoming webhook signatures

### 🏗️ Infrastructure Security
- Keep your deployment environment updated
- Use container scanning for Docker deployments
- Monitor logs for suspicious activity
- Implement proper access controls

### ⚙️ Configuration Security
- Validate all configuration inputs
- Use strong webhook secrets (minimum 32 characters)
- Regularly audit user mappings and permissions
- Monitor for unauthorized configuration changes

## Known Security Considerations

### Webhook Security
- git-slack-bot validates GitHub webhook signatures using HMAC-SHA256
- Always configure a strong webhook secret
- Monitor webhook delivery logs for anomalies

### Slack Token Security
- Use bot tokens (starting with `xoxb-`) not user tokens
- Limit bot permissions to minimum required scopes
- Monitor Slack app audit logs for unusual activity

### GitHub Token Security
- Use GitHub App tokens with minimal required permissions
- Prefer GitHub App installation tokens over personal access tokens
- Monitor GitHub App activity in your organization

## Security Contact

For general security questions or concerns, please create a GitHub issue at:
- 🔒 https://github.com/loveholidays/git-slack-bot/issues
- For urgent security issues, please label your issue with both `security` and `urgent` tags

Thank you for helping keep git-slack-bot and our users safe!