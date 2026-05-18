#!/usr/bin/env bash
# Example script that demonstrates input fields and argument passing
#
# Script Arguments:
#   $1 - name: Enter your name
#   $2 - message: Enter a greeting message
#

set -e  # Exit on error

echo "=========================================="
echo "Hello, $1!"
echo "Message: $2"
echo "=========================================="
echo ""
echo "This is an example script button that:"
echo "  • Takes user inputs (name and message)"
echo "  • Passes them as positional arguments ($1, $2)"
echo "  • Runs in workspace context using <workspace> keyword"
echo ""
echo "Script executed successfully at: $(date)"
