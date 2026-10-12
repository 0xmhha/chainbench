"""Monitoring selection, cancel, collection gaps and the shared time range on native wbft nodes; development proof only."""
from run_test_jobs import main

if __name__ == '__main__':
    main(browser_script='browser_monitoring_range.mjs', proof_name='monitoring-range', browser_timeout=900)
