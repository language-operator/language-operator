---
description: Upgrade the language-operator gem to a target version across all Gemfiles, then push and wait for CI
---

## Inputs

- $VERSION — target version of the language-operator gem

## Directions

1. Upgrade the language-operator gem to `$VERSION` in all locations
2. Run `bundle install` in every folder containing a Gemfile
3. Commit and push to origin
4. Poll CI with `gh run list` / `gh run view` until it passes — fix failures and push again
