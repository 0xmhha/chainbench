"""A review must reject hidden gaps, stale inputs and hopeful completion flags."""
import copy
import json
from pathlib import Path
import unittest
from tc_sufficiency import REVIEW, go_symbols, validate_review

ROOT = Path(__file__).resolve().parents[2]


class CriterionReviewTests(unittest.TestCase):
    def setUp(self):
        self.review = json.loads((ROOT / REVIEW).read_text())

    def test_current_review_has_all_ids_and_preserves_open_conditions(self):
        self.assertEqual(validate_review(ROOT, self.review), [])
        self.assertFalse(self.review['allCriteriaAccepted'])
        self.assertEqual(len(self.review['records']), 87)

    def test_unverified_claims_stale_sources_and_hidden_conditions_fail(self):
        def self_merge(review):
            row = next(r for r in review['records'] if r['status'] == 'merged')
            row['mergedInto'] = row['id']

        mutations = {
            'missing criterion': lambda x: x['records'].pop(),
            'duplicate criterion': lambda x: x['records'][0].update(id=x['records'][1]['id']),
            'unknown criterion': lambda x: x['records'][0].update(id='NODE-999'),
            'false overall completion': lambda x: x.update(allCriteriaAccepted=True),
            'false partial completion': lambda x: x['records'][0].update(fullCriterionAccepted=True),
            'hidden remaining condition': lambda x: x['records'][0].update(remainingConditions=[]),
            'stale original source': lambda x: x['criterionSource'].update(sha256='0' * 64),
            'stale criterion section': lambda x: x['records'][0].update(criterionSha256='0' * 64),
            'missing target': lambda x: x['records'][0]['targets'][0].update(path='missing.go'),
            'stale target': lambda x: x['records'][0]['targets'][0].update(sha256='0' * 64),
            'false scoped proof': lambda x: x['records'][1].update(verifiedEvidence=[]),
            'stale evidence': lambda x: x['records'][1]['verifiedEvidence'][0].update(sha256='0' * 64),
            'wrong merge target': lambda x: next(r for r in x['records'] if r['status']=='merged').update(mergedInto='WBFT-999'),
            'self merge': self_merge,
            'missing function': lambda x: next(t for r in x['records'] for t in r['targets'] if 'function' in t).update(function='UnknownFunction'),
            'wrong status counts': lambda x: x['statusCounts'].update(partial=0),
        }
        for name, change in mutations.items():
            with self.subTest(name=name):
                review = copy.deepcopy(self.review)
                change(review)
                self.assertTrue(validate_review(ROOT, review), name)

    def test_changed_assertion_and_escaping_paths_fail(self):
        for mode in ['statement', 'path']:
            review = copy.deepcopy(self.review)
            anchor = next(o for r in review['records'] for o in r['observations'] if 'pointer' in o)
            if mode == 'statement':
                anchor['statement'] = {'expect': 'anything', 'is': 'passed'}
            else:
                anchor['path'] = '../outside.json'
            self.assertTrue(validate_review(ROOT, review), mode)

    def test_comment_and_string_are_not_go_function_declarations(self):
        sources = {'fixture.go': 'package fixture\n// func HopefulPass() {}\nvar text = "func HopefulPass() {}"\nfunc Actual() {}\ntype Receiver struct{}\nfunc (*Receiver) Check() {}\n'}
        self.assertEqual(go_symbols(ROOT, sources), {'fixture.go': {'Actual', 'Receiver.Check'}})

    def test_malformed_review_shapes_return_errors(self):
        for field, value in [('records', 'hopeful success'), ('criterionSource', []),
                             ('records', [{'id': ['TX-001']}])]:
            review = copy.deepcopy(self.review); review[field] = value
            self.assertTrue(validate_review(ROOT, review), field)


if __name__ == '__main__':
    unittest.main()
