#!/bin/sh
# Smoke test: verify node, the WebSocket terminal server deps, and the claude
# binary are present.
set -e

node --version >/dev/null 2>&1 || { echo "FAIL: node not found"; exit 1; }
echo "PASS: node present"

cd /app
node -e "
require('node-pty');
require('ws');
require.resolve('@xterm/xterm/lib/xterm.js');
require.resolve('@xterm/xterm/css/xterm.css');
require.resolve('@xterm/addon-fit/lib/addon-fit.js');
" >/dev/null 2>&1 || { echo "FAIL: terminal server deps not resolvable"; exit 1; }
echo "PASS: terminal server deps present"

claude --version >/dev/null 2>&1 || { echo "FAIL: claude not found"; exit 1; }
echo "PASS: claude present"

tmux -V >/dev/null 2>&1 || { echo "FAIL: tmux not found"; exit 1; }
echo "PASS: tmux present"

echo "All tests passed"
