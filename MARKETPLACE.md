# Marketplace metadata

This file records the intended public listing metadata. It is not a submission
and does not grant permission to publish the repository or create a marketplace
issue.

Submission requirements last reviewed 2026-08-26 against
[`HANCORE-linux/omarchy-plugin-marketplace/SUBMISSION.md`](https://github.com/HANCORE-linux/omarchy-plugin-marketplace/blob/main/SUBMISSION.md).

- Repository: `https://github.com/SurreptitiousFabric/omarchy-frame`
- Plugin ID: `io.github.surreptitiousfabric.omarchy-frame`
- Name: `Samsung Frame`
- Category: `Hardware`
- Tags: `quickshell`, `system`, `power-management`
- Preview: omitted
- License: MIT
- External runtime dependencies: Omarchy/Quickshell and optional `avahi-browse`
  discovery fallback; users do not need Go

The marketplace's current static baseline engine was run locally against exact
candidate `0e14e31f4f3786ad1b5f9e066ecf1f66743edcf3`. It returned
`review-required` with no findings and no approval block. Its only capability
was `bundled-executable-binary`, which is intentional: static ARM64 and AMD64
backends let users install the plugin without Go. Marketplace publication will
therefore require a maintainer to review and attest that exact capability.

Before submission, the owner must confirm the exact repository visibility,
ownership/permission statement, checklist, title, and issue body required by
the current Omarchy marketplace submission guide. An agent must show the final
issue to the owner and receive explicit approval before creating it.
