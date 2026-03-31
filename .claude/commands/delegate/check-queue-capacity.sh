#!/bin/bash
# Print the number of open issues currently in each queue
for q in 0 1 2; do
  echo "queue/$q: $(gh issue list --label "queue/$q" --state open --json number | jq 'length')"
done
