# Security

Report vulnerabilities privately through
[GitHub's vulnerability reporting](https://github.com/RachidChabane/shallnot/security/advisories/new),
not in a public issue. Expect an answer within a week.

Fixes are made on the latest release.

Worth knowing when you assess a report: `shallnot gate` and the agent hooks
run the `test_commands` of the project's `shallnot.yaml` through the shell.
That file is as trusted as the project's own build scripts. `shallnot check`
runs nothing.
