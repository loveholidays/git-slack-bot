import io
import json
import os
import unittest
from unittest.mock import mock_open, patch
from urllib.error import URLError

import remind


def response(payload):
    return io.BytesIO(json.dumps(payload).encode())


def vector(rows):
    return {'status': 'success', 'data': {'resultType': 'vector', 'result': rows}}


def image(app='payment-service', namespace='web', spec='service:feature'):
    return {'metric': {'namespace': namespace, 'label_app': app, 'container': app, 'image_spec': spec}, 'value': [0, '1']}


class ReminderTest(unittest.TestCase):
    @patch.dict(os.environ, {'PROMETHEUS_URL': 'http://prometheus:9090', 'SLACK_CHANNEL_ID': 'C0BQSACN6EB', 'SLACK_TOKEN': 'test-token'})
    def test_posts_grouped_deduplicated_images_to_staging_channel(self):
        rows = [image(), image(), image('flight-club-radar', 'packages', 'sha256:abc')]
        with patch('remind.urlopen', side_effect=[response(vector(rows)), response({'ok': True})]) as request:
            remind.main()
        self.assertEqual(request.call_count, 2)
        post = request.call_args_list[1].args[0]
        body = json.loads(post.data)
        self.assertEqual(post.full_url, 'https://slack.com/api/chat.postMessage')
        self.assertEqual(body['channel'], 'C0BQSACN6EB')
        self.assertEqual(body['text'].count('web/payment-service'), 1)
        self.assertIn('sha256:abc', body['text'])
        self.assertIn('restore a `prod-*` image', body['text'])

    @patch.dict(os.environ, {'PROMETHEUS_URL': 'http://prometheus:9090'})
    def test_no_matches_checks_inventory_and_sends_nothing(self):
        with patch('remind.urlopen', side_effect=[response(vector([])), response(vector([{'value': [0, '1']}]))]) as request:
            remind.main()
        self.assertEqual(request.call_count, 2)
        self.assertTrue(all(isinstance(call.args[0], str) for call in request.call_args_list))

    @patch.dict(os.environ, {'PROMETHEUS_URL': 'http://prometheus:9090'})
    def test_missing_inventory_fails_instead_of_reporting_clear(self):
        with patch('remind.urlopen', side_effect=[response(vector([])), response(vector([]))]):
            with self.assertRaisesRegex(RuntimeError, 'inventory is unavailable'):
                remind.main()

    @patch.dict(os.environ, {'PROMETHEUS_URL': 'http://prometheus:9090'})
    def test_prometheus_error_fails(self):
        with patch('remind.urlopen', return_value=response({'status': 'error', 'error': 'query_failed'})):
            with self.assertRaisesRegex(RuntimeError, 'Prometheus query failed'):
                remind.main()

    @patch.dict(os.environ, {'PROMETHEUS_URL': 'http://prometheus:9090', 'SLACK_CHANNEL_ID': 'C0BQSACN6EB', 'SLACK_TOKEN': 'test-token'})
    def test_slack_permission_error_fails(self):
        with patch('remind.urlopen', side_effect=[response(vector([image()])), response({'ok': False, 'error': 'not_in_channel'})]):
            with self.assertRaisesRegex(RuntimeError, 'not_in_channel'):
                remind.main()

    def test_large_reports_are_split_without_losing_images(self):
        rows = [image(f'service-{i}', spec=f'registry/service-{i}:feature') for i in range(100)]
        messages = list(remind.messages(rows))
        self.assertGreater(len(messages), 1)
        self.assertTrue(all(len(message) <= 3500 for message in messages))
        self.assertEqual(sum(message.count('\n• ') for message in messages), 100)

    def test_image_names_cannot_create_slack_mentions(self):
        text = next(remind.messages([image(spec='service:<@U123>')]))
        self.assertNotIn('<@U123>', text)
        self.assertIn('&lt;@U123&gt;', text)


class AuthorsTest(unittest.TestCase):
    def setUp(self):
        self.env = patch.dict(os.environ, {'GITHUB_TOKEN': 'github-test', 'SLACK_TOKEN': 'slack-test'})
        self.env.start()
        self.addCleanup(self.env.stop)
        self.authors = remind.Authors({'slack': {'githubEmailToSlackEmail': [
            {'githubEmail': 'apastena-lh', 'slackEmail': 'antonello.pastena@loveholidays.com'},
        ]}})
        self.pr = {'number': 1440, 'user': {'login': 'apastena-lh'},
                   'head': {'ref': 'fix/shared-http-retry-classification',
                            'repo': {'full_name': 'loveholidays/flight-club'}}}
        self.spec = 'eu.gcr.io/loveholidays-ci-cd/flight-club-ski-jump:fix_shared-http-retry-classification@sha256:abc'

    def test_branch_tag_maps_both_apps_to_one_repo_and_mentions_pr_author(self):
        with patch('remind.urlopen', side_effect=[response([self.pr]), response({'ok': True, 'user': {'id': 'UJMJ26A48'}})]) as request:
            messages = list(remind.messages([image(spec=self.spec), image('flight-club-radar', spec=self.spec.replace('ski-jump', 'radar'))], self.authors))
        self.assertEqual(request.call_count, 2)  # Cache the repository PRs and Slack user across images.
        self.assertIn('/repos/loveholidays/flight-club/pulls?', request.call_args_list[0].args[0].full_url)
        self.assertEqual(request.call_args_list[0].args[0].get_header('Authorization'), 'Bearer github-test')
        self.assertEqual(request.call_args_list[1].args[0].get_header('Authorization'), 'Bearer slack-test')
        self.assertEqual(messages[0].count('<@UJMJ26A48>'), 2)
        self.assertIn('https://github.com/loveholidays/flight-club/pull/1440|PR #1440', messages[0])

    def test_ambiguous_branch_tags_do_not_mention_anyone(self):
        other = {**self.pr, 'number': 1441, 'head': {**self.pr['head'], 'ref': 'fix_shared-http-retry-classification'}}
        with patch('remind.urlopen', return_value=response([self.pr, other])):
            self.assertEqual(self.authors.describe(self.spec), '')

    def test_fork_pr_does_not_match_internal_image(self):
        fork = {**self.pr, 'head': {**self.pr['head'], 'repo': {'full_name': 'someone/flight-club'}}}
        with patch('remind.urlopen', return_value=response([fork])):
            self.assertEqual(self.authors.describe(self.spec), '')

    def test_missing_mapping_or_slack_permission_keeps_author_and_pr_link(self):
        for emails in ({}, self.authors.emails.copy()):
            with self.subTest(emails=emails):
                authors = remind.Authors({'slack': {'githubEmailToSlackEmail': []}})
                authors.emails = emails
                with patch('remind.urlopen', side_effect=[response([self.pr]), response({'ok': False, 'error': 'missing_scope'})]):
                    text = authors.describe(self.spec)
                self.assertIn('apastena-lh', text)
                self.assertIn('PR #1440', text)
                self.assertNotIn('<@', text)

    def test_lookup_failure_does_not_drop_the_image_report(self):
        failure = URLError('GitHub unavailable')
        with patch('remind.urlopen', side_effect=failure):
            text = next(remind.messages([image(spec=self.spec)], self.authors))
        self.assertIn(self.spec, text)
        self.assertNotIn('<@', text)

    def test_slack_lookup_failure_preserves_github_attribution(self):
        with patch('remind.urlopen', side_effect=[response([self.pr]), TimeoutError()]):
            self.assertIn('apastena-lh', self.authors.describe(self.spec))

    def test_digest_only_and_external_images_do_not_make_api_calls(self):
        with patch('remind.urlopen') as request:
            for spec in ['eu.gcr.io/loveholidays-ci-cd/service@sha256:abc', 'other.registry/service:feature', 'sha256:abc']:
                self.assertEqual(self.authors.describe(spec), '')
        request.assert_not_called()

    def test_commit_tag_resolves_associated_pr(self):
        with patch('remind.urlopen', side_effect=[response([self.pr]), response({'ok': True, 'user': {'id': 'UJMJ26A48'}})]) as request:
            text = self.authors.describe('eu.gcr.io/loveholidays-ci-cd/flight-club-ski-jump:1f7fd52')
        self.assertIn('/commits/1f7fd52/pulls', request.call_args_list[0].args[0].full_url)
        self.assertIn('<@UJMJ26A48>', text)

    def test_pr_lookup_paginates_and_rejects_incomplete_lists(self):
        unrelated = {**self.pr, 'head': {**self.pr['head'], 'ref': 'unrelated'}}
        with patch('remind.urlopen', side_effect=[response([unrelated] * 100), response([self.pr]), response({'ok': True, 'user': {'id': 'UJMJ26A48'}})]) as request:
            self.assertIn('<@UJMJ26A48>', self.authors.describe(self.spec))
        self.assertIn('page=2', request.call_args_list[1].args[0].full_url)
        authors = remind.Authors({'slack': {'githubEmailToSlackEmail': []}})
        with patch('remind.urlopen', side_effect=[response([self.pr] * 100) for _ in range(3)]):
            self.assertEqual(authors.describe(self.spec), '')

    def test_author_lookup_budget_does_not_block_report(self):
        self.authors.deadline = 0
        with patch('remind.urlopen') as request:
            text = next(remind.messages([image(spec=self.spec)], self.authors))
        request.assert_not_called()
        self.assertIn(self.spec, text)

    @patch.dict(os.environ, {'PROMETHEUS_URL': 'http://prometheus:9090', 'BOT_CONFIG_PATH': '/config/bot-config.yaml', 'SLACK_CHANNEL_ID': 'C0BQSACN6EB'})
    def test_main_posts_author_mention_using_shared_bot_config(self):
        config = 'slack:\n  githubEmailToSlackEmail:\n    - githubEmail: apastena-lh\n      slackEmail: antonello.pastena@loveholidays.com\n'
        with patch('builtins.open', mock_open(read_data=config)), patch('remind.urlopen', side_effect=[
                response(vector([image(spec=self.spec)])), response([self.pr]),
                response({'ok': True, 'user': {'id': 'UJMJ26A48'}}), response({'ok': True})]) as request:
            remind.main()
        post = request.call_args_list[-1].args[0]
        self.assertEqual(post.full_url, 'https://slack.com/api/chat.postMessage')
        self.assertIn('<@UJMJ26A48>', json.loads(post.data)['text'])

    @patch.dict(os.environ, {'PROMETHEUS_URL': 'http://prometheus:9090', 'BOT_CONFIG_PATH': '/config/bot-config.yaml', 'SLACK_CHANNEL_ID': 'C0BQSACN6EB'})
    def test_main_still_posts_when_github_is_unavailable(self):
        with patch('builtins.open', mock_open(read_data='slack:\n  githubEmailToSlackEmail: []\n')), patch('remind.urlopen', side_effect=[
                response(vector([image(spec=self.spec)])), URLError('GitHub unavailable'), response({'ok': True})]) as request:
            remind.main()
        self.assertIn(self.spec, json.loads(request.call_args_list[-1].args[0].data)['text'])


if __name__ == '__main__':
    unittest.main()
