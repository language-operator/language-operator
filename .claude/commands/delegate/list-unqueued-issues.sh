#!/bin/bash
# List open GitHub issues not yet assigned to any queue or in-progress
gh issue list --state open --json number,title,labels \
  | jq '[.[] | select(
      (.labels | map(.name) | map(startswith("queue/") or . == "in-progress") | any) | not
    )]'
