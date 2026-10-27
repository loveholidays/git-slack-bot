# Shopping staging image reminder

This one-shot job queries Prometheus for Shopping containers running non-`prod-*`
images and posts a grouped report to Slack. It was moved from Flux so application
code is built and tested here; Flux owns its CronJob and configuration.

Set `PROMETHEUS_URL`, `SLACK_CHANNEL_ID`, and `SLACK_TOKEN`. Optional author
attribution uses `GITHUB_TOKEN` with pull-request read access and `BOT_CONFIG_PATH`
pointing to Shopping Bot's existing YAML configuration. The reminder reads only
`slack.githubEmailToSlackEmail` from that file. Slack's `users:read.email` scope
resolves those email addresses to mentions; missing mappings or lookup permissions
fall back to the GitHub login.

Internal image names map to repositories of the same name, except
`flight-club-radar` and `flight-club-ski-jump`, which both map to `flight-club`.
Branch tags match a unique open PR whose source branch has `/` replaced by `_`.
Fork PRs are excluded from branch matching. Bare Git SHA tags use associated PRs.
Digest-only images, unknown tags and ambiguous PR matches remain in the report
without attribution. PR authors are identified, not the person who deployed them.
An image pinned as `branch-tag@sha256:...` still uses the branch tag for attribution;
this does not verify that the digest is the PR's current head.

GitHub and Slack author lookups are cached within the run and have a combined
30-second budget. Lookup failures never suppress the image report. Prometheus
inventory and Slack posting failures still fail the job.

Run tests with:

```sh
pip install -r staging-image-reminder/requirements.txt
python -m unittest discover -s staging-image-reminder -p 'test_*.py'
```

Build the image from the `staging-image-reminder/` directory. Its Dockerfile runs
the same tests before producing the image. No credentials are needed at build time.
