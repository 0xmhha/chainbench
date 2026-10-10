"""Compatibility entry point for immutable fresh WEB-05 rechecks."""
import sys
from recheck import check

if __name__ == '__main__':
    check('WEB-05', sys.argv[1])
