# SonarQube Cloud + Gitar — rule checks and AI review on every PR

**GiCoffee · Gitea · CSCI 435 New-Tech Showcase 2**

- **Tutorial video:** [watch on YouTube](https://youtu.be/E_GnFsQX44c)
- **SonarQube Cloud:** <https://sonarcloud.io> · Docs: <https://docs.sonarsource.com/sonarqube-cloud/>
- **Gitar:** <https://gitar.ai>
- **Version:** both are hosted services with no version number. Our CI runs `SonarSource/sonarqube-scan-action` v8.3.0.
- **Brochure version:** [handout.pdf](handout.pdf)

## The problem

PRs, especially AI-written ones, arrive faster than people can review them, and linters only catch what they're set up to catch. Bugs, security holes and untested code get merged on green checks.

## What they found on our repo

[PR #152](https://github.com/CSCI-435-SE/gitea/pull/152) has four planted problems. `go vet`, ESLint and our unit tests all passed it.

- **SonarQube Cloud:** Quality Gate failed. Coverage on new code was 20% (needs ≥ 80%), the security rating was E and the reliability rating was D (both need A).
- **Gitar:** changes requested, with 4 findings: an open redirect, `.sort()` sorting numbers as strings, a test with no assertion, and a new Go handler with no test. It also marked the failing CI job (a data race in `modules/timeutil`) as unrelated to the PR.

## Why these tools

| | AI chat | SonarQube | Gitar |
|---|---|---|---|
| Runs on every PR by itself | ✗ | ✓ | ✓ |
| Same result every run | ✗ | ✓ | ✗ |
| Pass/fail gate | ✗ | ✓ | ✗ |
| Tracks coverage | ✗ | ✓ | ✗ |
| Knows your own helpers | if pasted | ✗ | ✓ |
| Fixes failing CI | if pasted | ✗ | ✓ |

SonarQube sets a measurable bar every PR must meet. Gitar covers what needs judgment and writes the fix.

## Setup

### SonarQube Cloud, with coverage (what we did)

1. Sign in at <https://sonarcloud.io> with GitHub and import the repo. An org owner has to approve it, and the repo needs a LICENSE file.
2. Turn off Automatic Analysis (Administration → Analysis Method).
3. Add a `SONAR_TOKEN` repository secret.
4. Add `sonar-project.properties` with the project key, organization and coverage report paths.
5. Add a CI job: tests with coverage, a full-history checkout (`fetch-depth: 0`), then `sonarqube-scan-action`. Run it on pushes to main too, as the baseline.

**First result:** open a PR. The quality gate shows up as a check and a comment.

Our full config: [`sonar-project.properties`](https://github.com/CSCI-435-SE/gitea/blob/main/sonar-project.properties) and [`pull-db-tests.yml`](https://github.com/CSCI-435-SE/gitea/blob/main/.github/workflows/pull-db-tests.yml).

*No coverage needed?* Skip steps 2–5: Automatic Analysis scans the repo as soon as you import it, just without coverage.

### Gitar (about 5 minutes)

1. Sign in at <https://app.gitar.ai> with GitHub.
2. Install the Gitar GitHub App on the repo. On an organization, an owner may have to approve it.
3. Open a PR. gitar-bot reviews it and explains CI failures. Tick the box under a suggestion to commit it.

| On the PR | What happens |
|---|---|
| Comment `Gitar fix CI` | Fixes the failing CI |
| Comment `gitar auto-apply:on` | Commits fixes without asking (Pro) |
| Add the `gitar-skip` label | Skips this PR |

## When to use them, and when not

**SonarQube Cloud**

- ✅ A bar every PR must meet, with coverage and ratings tracked over time.
- ⚠️ Free for public projects, and for private ones up to 50k lines of code. The Team plan starts at $34 a month.
- ⚠️ Rules don't know your project's code: it flags Gitea's own redirect check as unsafe.

**Gitar**

- ✅ Plain-language findings, fixes you can commit, and CI failures sorted into real and unrelated.
- ⚠️ Free for qualifying open-source projects. Otherwise a 14-day Pro trial, then $20 (Core) or $40 (Pro) per user a month, billed annually. Auto-apply needs Pro.
- ⚠️ It's AI, and its fixes can clash with SonarQube: its open-redirect fix uses that same Gitea check.

**Both** send your code to a third-party service.

<sub>Prices checked November 9, 2026. They change often.</sub>
