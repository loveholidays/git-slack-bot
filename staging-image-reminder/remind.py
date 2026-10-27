import html
import json
import os
import re
import time
from functools import cache
from urllib.error import URLError
from urllib.parse import urlencode
from urllib.request import Request, urlopen

import yaml


# Exclude the reminder's own infrastructure container as well as existing sidecars.
QUERY = '''
max by (lh_cluster, namespace, label_app, container, image_spec) (
  kube_pod_container_info{
    container!~"metrics|nginx|cloud-sql-proxy|ship-heap-dump|redis-node",
    image_spec!~"eu[.]gcr[.]io/loveholidays-ci-cd/shopping-staging-image-reminder(:|@).*",
    image_spec!="",
    image_spec!~".*:prod-.*"
  }
  * on (lh_cluster, namespace, pod, uid)
    group_left(label_app)
  max by (lh_cluster, namespace, pod, uid, label_app) (
    kube_pod_labels{label_team=%s}
  )
)
'''


def query(expression):
    url = os.environ['PROMETHEUS_URL'].rstrip('/') + '/api/v1/query?' + urlencode({'query': expression})
    with urlopen(url, timeout=30) as response:
        payload = json.load(response)
    if payload['status'] != 'success' or payload['data']['resultType'] != 'vector':
        raise RuntimeError('Prometheus query failed or returned an unexpected result type')
    return payload['data']['result']


def image_reference(image):
    # Only our own image names map to repositories in the loveholidays org.
    prefix = 'eu.gcr.io/loveholidays-ci-cd/'
    if not image.startswith(prefix):
        return None
    name, separator, tag = image[len(prefix):].split('@', 1)[0].partition(':')
    if not separator or not re.fullmatch('[A-Za-z0-9_.-]+', name):
        return None
    repository = {'flight-club-radar': 'flight-club', 'flight-club-ski-jump': 'flight-club'}.get(name, name)
    return repository, tag


def branch_matches(pr, repository, tag):
    head = pr['head']
    return (head['repo'] and head['repo']['full_name'] == f'loveholidays/{repository}'
            and head['ref'].replace('/', '_') == tag)


class Authors:
    def __init__(self, config):
        self.emails = {user['githubEmail']: user['slackEmail'] for user in config['slack']['githubEmailToSlackEmail']}
        self.deadline = time.monotonic() + 30

    def api(self, url, token):
        remaining = self.deadline - time.monotonic()
        if remaining <= 0:
            raise TimeoutError('Author lookup budget exhausted')
        request = Request(url, headers={'Authorization': 'Bearer ' + token, 'Accept': 'application/json'})
        with urlopen(request, timeout=min(5, remaining)) as response:
            return json.load(response)

    @cache
    def pulls(self, repository):
        pulls = []
        for page in range(1, 4):
            batch = self.api(f'https://api.github.com/repos/loveholidays/{repository}/pulls?state=open&per_page=100&page={page}', os.environ['GITHUB_TOKEN'])
            pulls.extend(batch)
            if len(batch) < 100:
                return pulls
        raise ValueError('Incomplete PR list; cannot safely attribute image')

    @cache
    def mention(self, login):
        fallback = html.escape(login, quote=False)
        email = self.emails.get(login)
        if not email:
            return fallback
        try:
            result = self.api('https://slack.com/api/users.lookupByEmail?' + urlencode({'email': email}), os.environ['SLACK_TOKEN'])
        except (URLError, TimeoutError, ValueError):
            return fallback
        user_id = result.get('user', {}).get('id', '')
        if result.get('ok') and re.fullmatch('[UW][A-Z0-9]+', user_id):
            return f'<@{user_id}>'
        return fallback

    def matching_pulls(self, repository, tag):
        if re.fullmatch('[0-9a-f]{7,40}', tag):
            return self.api(f'https://api.github.com/repos/loveholidays/{repository}/commits/{tag}/pulls', os.environ['GITHUB_TOKEN'])
        return [pr for pr in self.pulls(repository) if branch_matches(pr, repository, tag)]

    @cache
    def describe(self, image):
        reference = image_reference(image)
        if not reference:
            return ''
        repository, tag = reference
        try:
            pulls = self.matching_pulls(repository, tag)
            if len(pulls) != 1:
                return ''
            pr = pulls[0]
            login = pr['user']['login']
            number = int(pr['number'])
            link = f'<https://github.com/loveholidays/{repository}/pull/{number}|PR #{number}>'
            author = self.mention(login)
            return f' — {author} ({link})'
        except (URLError, TimeoutError, ValueError, KeyError):
            print(f'Author lookup unavailable for {repository}; reporting image without attribution.')
            return ''


def image_lines(results, authors):
    images = sorted({tuple(row['metric'][key] for key in ('namespace', 'label_app', 'container', 'image_spec')) for row in results})
    for namespace, app, container, image in images:
        line = html.escape(f'\n• {namespace}/{app} ({container}): {image}', quote=False)
        if authors:
            line += authors.describe(image)
        yield line


def messages(results, team, authors=None):
    team_name = html.escape(team.capitalize(), quote=False)
    heading = f'*{team_name} staging: non-standard images*\nEngineers testing these builds: please restore a `prod-*` image when finished.\n'
    text = heading
    for line in image_lines(results, authors):
        if len(text) + len(line) > 3500:
            yield text
            text = heading
        text += line
    if text != heading:
        yield text


def load_authors():
    if not os.environ.get('GITHUB_TOKEN') or not os.environ.get('BOT_CONFIG_PATH'):
        return None
    with open(os.environ['BOT_CONFIG_PATH'], encoding='utf-8') as config_file:
        return Authors(yaml.safe_load(config_file))


def post_message(text):
    body = json.dumps({'channel': os.environ['SLACK_CHANNEL_ID'], 'text': text, 'unfurl_links': False, 'unfurl_media': False}).encode()
    request = Request('https://slack.com/api/chat.postMessage', data=body, headers={
        'Authorization': 'Bearer ' + os.environ['SLACK_TOKEN'],
        'Content-Type': 'application/json; charset=utf-8',
    })
    with urlopen(request, timeout=30) as response:
        payload = json.load(response)
    if not payload.get('ok'):
        raise RuntimeError('Slack rejected the reminder: ' + payload.get('error', 'unknown_error'))


def main():
    team = os.environ['TEAM']
    if not team:
        raise ValueError('TEAM must not be empty')
    team_label = json.dumps(team, ensure_ascii=False)
    results = query(QUERY % team_label)
    if not results:
        # Empty results are safe only when both inventory metrics are available.
        inventory = query('count(kube_pod_container_info{image_spec!=""}) > 0 and on() count(kube_pod_labels{label_team=%s}) > 0' % team_label)
        if not inventory:
            raise RuntimeError(f'{team} image inventory is unavailable; cannot confirm standard tags')
        print(f'No non-standard {team} staging images.')
        return
    authors = load_authors()
    for text in messages(results, team, authors):
        post_message(text)
    print(f'Reported {len(results)} non-standard image entries.')


if __name__ == '__main__':
    main()
