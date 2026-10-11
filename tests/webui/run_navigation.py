"""Shared selection, top bar, dashboard and job detail on a native wbft network; development proof only."""
from run_test_jobs import main

if __name__ == '__main__':
    main(browser_script='browser_navigation.mjs', proof_name='navigation', browser_timeout=600)
