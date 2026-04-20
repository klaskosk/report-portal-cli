---
name: merge-prs
description: Merge pull requests.
disable-model-invocation: true
---

# Merge Pull Requests

This skill is used to merge all open pull requests in the repository. Note that "merge" means squash merging for this repository.

## Workflow

1. Get all open pull requests in the repository using the `gh` CLI.
2. Start with the lowest number PR and merge it (only if checks pass).
3. Check out the next PR and rebase against the new main.
4. Wait until the checks pass for the next PR.
5. Merge the next PR.
6. Repeat until all PRs are merged.

## Notes

- Simple issues with rebasing should be resolved in this process.
- You may pause and ask the user for help if a tricky issue arises. This is preferable to breaking something with merge conflicts.
- If you encounter an auth issue, consider if that is due to a sandbox rather than a real auth issue. Retry with elevated privileges if possible.

Thank you for your help!
