"""Node detail, plan review and binary surface on native wbft nodes; development proof only."""
from run_test_jobs import main

if __name__ == '__main__':
    main(browser_script='browser_node_detail.mjs', proof_name='node-detail', browser_timeout=900)
