# Issue tracker: GitHub

Issues and specs live as GitHub issues in `mokevnin/schemalens`. Use the `gh` CLI for all operations.

- **Create**: `gh issue create --title "..." --body-file <file>`
- **Read**: `gh issue view <number> --json number,title,body,labels,comments`
- **List**: `gh issue list --state open --label ready-for-agent`
- **Comment**: `gh issue comment <number> --body "..."`
- **Label**: `gh issue edit <number> --add-label "..." / --remove-label "..."`
- **Close**: `gh issue close <number> --comment "..."`
- **Sub-issue of a parent**: `gh issue create --parent <parent> ...` (or `Part of #<parent>` at the top of the body).

`gh` infers the repo from the clone's remote. Pull requests are not a request surface.
