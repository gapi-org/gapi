# Security Policy

Gapi is an alpha project, but security issues are still taken seriously.

## Supported Versions

Until v1, the latest public release or the `main` branch is the primary supported version.

## Reporting A Vulnerability

Please do not open a public issue for a suspected vulnerability.

Report privately by emailing:

```text
kushagra1122@users.noreply.github.com
```

Include:

- Description of the vulnerability
- Steps to reproduce
- Impact
- Affected version or commit
- Any suggested fix, if available

## Response Expectations

The project maintainer will try to:

- acknowledge the report within 7 days
- confirm impact and affected versions
- prepare a fix when appropriate
- credit the reporter if they want public credit

## Scope

Security-sensitive areas include:

- request binding and body limits
- middleware behavior
- authentication helpers
- dependency injection
- response serialization
- OpenAPI output that may expose unintended metadata

## Responsible Disclosure

Please give the project time to investigate and release a fix before publishing details.
