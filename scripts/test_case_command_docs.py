"""Keep the generated public command guide complete as cases are added."""
from collections import Counter
import json
from pathlib import Path
import re
import subprocess
import unittest

REPO = Path(__file__).resolve().parents[1]
BEGIN = '<!-- BEGIN generated: scripts/gen-case-commands.py -->'
END = '<!-- END generated -->'


class CaseCommandDocsTest(unittest.TestCase):
    def setUp(self):
        document = (REPO / 'tests/tc/HOW-TO-USE.md').read_text()
        self.assertEqual(document.count(BEGIN), 1)
        self.assertEqual(document.count(END), 1)
        self.generated = document.split(BEGIN, 1)[1].split(END, 1)[0].strip()
        self.cases = {p.relative_to(REPO).as_posix() for p in (REPO / 'tests/tc').rglob('*.json')}

    def test_every_owned_case_has_exactly_one_cli_command(self):
        commands = Counter(re.findall(r'bin/chainbench run (tests/tc/[^\s]+\.json)', self.generated))
        problems = {
            'missing': sorted(self.cases - commands.keys()),
            'orphaned': sorted(commands.keys() - self.cases),
            'duplicates': {path: count for path, count in commands.items() if count != 1},
        }
        self.assertFalse(any(problems.values()), json.dumps(problems, indent=2))

    def test_announced_count_matches_the_current_case_tree(self):
        count = re.search(r'케이스 \*\*(\d+)건\*\*', self.generated)
        self.assertIsNotNone(count)
        self.assertEqual(int(count[1]), len(self.cases))

    def test_committed_fragment_matches_the_actual_generator(self):
        result = subprocess.run(['python3', '-B', str(REPO / 'scripts/gen-case-commands.py')],
                                cwd=REPO, text=True, capture_output=True, check=True)
        self.assertEqual(self.generated, result.stdout.strip(), 'Regenerate with scripts/gen-case-commands.py --write')

    def test_ci_runs_the_command_guide_guard(self):
        workflow = (REPO / '.github/workflows/ci.yml').read_text()
        self.assertRegex(workflow, r'(?m)^\s*run:\s*make case-docs\s*$')


if __name__ == '__main__':
    unittest.main()
