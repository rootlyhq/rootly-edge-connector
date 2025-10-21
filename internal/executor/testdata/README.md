# Test Fixtures

This directory contains test fixtures for script execution tests.

## Fixtures

- `fixtures/test.py` - Python script for testing Python interpreter detection
- `fixtures/test.js` - Node.js script for testing Node interpreter detection
- `fixtures/test.rb` - Ruby script for testing Ruby interpreter detection
- `fixtures/test.go` - Go script for testing Go interpreter detection
- `fixtures/test.bash` - Bash script for testing Bash interpreter detection
- `fixtures/test.sh` - Shell script for testing sh interpreter detection
- `fixtures/test_shebang` - Script with no extension to test shebang execution

All scripts read the `REC_PARAM_MESSAGE` environment variable and print it prefixed with the language name.
