"""Owned native resource retention and cleanup development proof."""
from run_test_jobs import main

if __name__ == "__main__":
    main(browser_script="browser_resource_holds.mjs", proof_name="resource-holds")
