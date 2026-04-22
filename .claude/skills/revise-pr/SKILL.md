---
name: revise-pr
description: Revise a pull request.
disable-model-invocation: true
---

# Revise Pull Request

This skill is used to revise a pull request based on pull request review comments.

## Workflow

1. Get the pull request review comments from the `gh` CLI. (`gh pr view --comments` should be enough).
2. Review the comments and decide what to do.
3. Make changes as necessary and/or post a rebuttal to a review comment.
4. Launch a subagent with the `code-review` skill to review the changes locally.
5. If the subagent from 4 finds any issues, repeat steps 2-4. That includes launching a new `code-review` subagent, you must get approval before proceeding.
6. If the subagent from 4 approves the changes, commit the changes.
7. Follow your `git-workflow` skill to update the pull request with the changes.
8. Wait until the checks pass for the pull request.
9. If the checks fail, repeat steps 2-8. Otherwise, you are done.

## Notes

- You are not required to follow any suggested fix, as long as your solution addresses the issue.
- You may pause and ask the user for help if a tricky issue arises.
- You may offer a rebuttal to a review comment. Use the `gh` CLI to add a comment to the pull request with your rebuttal.
