"""Settings: user management, audit, credential usage and revocation on native stablenet nodes; development proof only."""
from run_test_jobs import main

if __name__ == '__main__':
    main(browser_script='browser_settings.mjs', proof_name='settings', browser_timeout=900)
