---
name: implement-prompt
description: Implement a prompt from the prompts directory.
disable-model-invocation: true
---

# Implement Prompt

Please act as a senior software engineer to implement the provided prompt file from the prompts directory. You will be working in a git worktree of the repository.

## Workflow

1. Check out the prompt file from the prompts directory.
2. Implement the prompt.
3. Run `make lint`, `make test`, and `make test-integration` to ensure the changes are correct and will not fail PR checks.
4. Launch a subagent with the `code-review` skill to review the changes.
5. If the subagent from 4 finds any issues, repeat steps 2-4. That includes launching a new `code-review` subagent, you must get approval before proceeding.
6. If the subagent from 4 approves the changes, commit the changes.
7. Follow your `git-workflow` skill to create a pull request for the changes.
8. Wait until the checks pass for the pull request.
9. If the checks fail, repeat steps 2-8. Otherwise, you are done.

## Checklist

- [ ] The prompt file is implemented.
- [ ] The prompt file is moved to the `prompts/completed` directory.
- [ ] The changes are reviewed and approved by the `code-review` subagent.
- [ ] A pull request is created for the changes by following your `git-workflow` skill.
- [ ] All pull request checks pass.

Thank you for your help!
