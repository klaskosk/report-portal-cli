---
name: git-workflow
description: Use when the user requests you to create a commit, branch, or pull request.
---

# report-portal-cli Git Workflow

This skill explains the Git workflow for the report-portal-cli project.

## Feature workflow

1. Create a new branch for the feature.
2. Implement the feature.
3. Ensure the linter and tests pass.
4. Create a commit with a descriptive message.
5. Push the branch to the remote repository.
6. Create a pull request.

## Revision workflow

1. Check out pull request branch.
2. Make changes to the code.
3. Ensure the linter and tests pass.
4. Amend the commit and optionally update the commit message. There should be a 1:1 correspondence between commits and pull requests.
5. Use `--force-with-lease` to update the pull request branch.
6. Optionally, update the pull request description.

## Message format

This project uses [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/). The commit message should be in the following format:

```plaintext
<type>: <description>

[body]

[trailer]
```

Best practice is that the first line should be only 50 characters long.

Pull requests should be of the same format as the commit message. Do not include a "test plan" in the pull request description.

If your task was to implement a feature from the prompts directory, mention the prompt filename in the commit message. This will add to traceability for reviews.
