# Security policy

`agent-statusline` runs on every status line refresh and edits your coding agents' settings files, so issues in either path matter.

## Reporting a vulnerability

Please report vulnerabilities privately, not in a public issue. Use GitHub's **Report a vulnerability** button on the repository's Security tab, which opens a private advisory. Include the version (`agent-statusline version`), your OS, and steps to reproduce.

You can expect an acknowledgement within a few days. Please give reasonable time for a fix before disclosing publicly.

## In scope

- A settings edit that changes or loses keys other than the status line keys it manages.
- Uninstall restoring the wrong content, or deleting a status line it did not write.
- Command injection through payload fields (custom `cmd:` segments run the command you configure, which is by design).
- Anything that makes `render` hang, crash the host tool, or leak data from the session payload.

## Supported versions

Only the latest release receives fixes until v1.0.
