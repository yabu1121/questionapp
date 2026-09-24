# AGENTS.md

## Editing policy

- By default, treat this repository as read-only.
- Do not create, edit, move, format, or delete files unless the user explicitly asks for implementation or file changes.
- Requests for feedback, review, explanation, diagnosis, examples, or suggested code do not authorize file changes.
- When the user asks for feedback or an example, respond in chat only.
- Do not infer permission to edit from earlier requests. Permission applies only to the files and changes explicitly requested in the current request.
- Before making changes, state which files will be changed and why.
- Preserve all user-written code, including incomplete or non-compiling code, unless the user explicitly asks to replace or fix it.
- Read-only inspection commands are allowed when needed to answer the user.
- Do not run commands that mutate the database, containers, dependencies, generated files, or external state unless the user explicitly requests that action.

## Communication

- Prefer concise feedback in Japanese.
- Clearly separate required fixes from optional improvements.
- If the user's intent to modify files is ambiguous, do not modify anything; ask first.
