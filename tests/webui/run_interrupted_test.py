"""Restart a real dashboard during an owned native test, without auto recovery."""
from run_test_jobs import main

if __name__ == "__main__":
    main(browser_script="browser_interrupted_test.mjs", proof_name="interrupted-test", restart=True)
