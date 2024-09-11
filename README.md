# github-slack-bot
A bot for posting automatic github pull request messages to slack

## Configuration
- `github`:  
  - `token`: The security token of the github app, which will send events through a webhook
  - `org`: The github organization the team is in
  - `team`: The team which has the members to post PR for
  - `ignoredPRUsers`: Users in the github team to ignore opened PRs for. Their comments will still show up in threads.
  - `ignoredCommentUsers`: Users to ignore PR comments from. Recommended to add any automated github app account
  - `secretKey`: The secret key of the github webhook to verify incoming events against
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