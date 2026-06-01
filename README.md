# podscape

A Bubble Tea TUI that draws your Kubernetes cluster as a **node-group floor
plan** — every EKS managed node group / Karpenter NodePool / GKE NodePool /
AKS agentpool is a bordered region, the nodes inside are cards arranged in a
responsive grid, pods are colour-coded boxes stacked inside each card, and
DaemonSet pods always live in a dedicated band at the bottom of every card.

Built for verifying that the workloads you ship as part of an off-the-shelf
K8s distribution land where you expect — right node pool, right priority
class, right tolerations, sane DaemonSet overhead.

Read-only. No mutating verbs are issued against the cluster.

```
┌─ ◆ NodePool: system   taints: critical=:NoSchedule ─────────────────┐
│  ╭─ ip-10-0-1-12 ──────────╮  ╭─ ⚠ ip-10-0-1-37 ────────╮           │
│  │ m5.large                │  │ m5.large                │           │
│  │ [coredns]  [metrics-srv]│  │ [coredns]  [metrics-srv]│           │
│  │ ───────────────────────  │  │ ───────────────────────  │          │
│  │ ░ kube-pr…  ░ aws-node  │  │ ░ kube-pr…  ░ aws-node  │           │
│  │ DS cpu 5%   mem 2%      │  │ DS cpu 5%   mem 2%      │           │
│  ╰─────────────────────────╯  ╰─────────────────────────╯           │
└─────────────────────────────────────────────────────────────────────┘
```

A `⚠` next to a node's name means at least one correctness finding is attached
to it. The **Findings** tab lists them.

## Features

- **Node-group floor plan** – cluster grouped by Karpenter NodePool, EKS
  managed node group, GKE NodePool, AKS agentpool, or `node-role.kubernetes.io/*`
  fallback.
- **Per-workload colour** – every Deployment / StatefulSet / Job gets a
  deterministic hue, so the same workload reads the same on every node card.
- **DaemonSet band** – DS pods always render in the bottom band of every card
  with a live `DS cpu N%  mem N%` overhead bar.
- **Three density modes** – `c` compact, `n` normal, `w` wide. Cards reflow
  into more or fewer columns automatically as the terminal resizes.
- **Scrollable, collapsible plan** – the floor plan scrolls when it outgrows the
  terminal (`pgup`/`pgdn`, `g`/`G`, or just move focus and it follows), and any
  node group can be collapsed to a one-line header with `x`/`space` to tuck away
  the clusters you aren't looking at.
- **Sortable Nodes tab** – `2` for a Bubbles table you can sort by node /
  group / DS pod count / CPU% / MEM% (`s` cycles the column).
- **Findings tab** – `3` lists every correctness issue with severity, code,
  and the affected pod/node.

### Correctness checks

| Code | Severity | What it catches |
| --- | --- | --- |
| `DS_NO_PRIORITY` | warn | DaemonSets without a `priorityClassName` (so they're not evicted under pressure). |
| `TOLERATION_UNUSED` | warn | A pod tolerates a taint that isn't on its current node — the toleration was added with intent to land on a tainted node, but the pod landed elsewhere. (DaemonSets and kubelet-injected operational tolerations are skipped.) |
| `ANTIAFFINITY_VIOLATED` | error | A pod with required hostname-level pod-anti-affinity shares a node with at least one peer matching the selector. |

More checks live in `internal/analysis/checks.go` — `RunChecks` orchestrates
them; add a new function and append its findings.

### Context selection

- `--context <name>` skips the picker.
- Otherwise, if your kubeconfig has a `current-context`, it's used.
- Otherwise, a picker UI lists all contexts.

## Quickstart

```sh
make build                       # bin/podscape
./bin/podscape                   # picker, or current-context
./bin/podscape --context prod    # skip the picker
./bin/podscape -n kube-system    # scope pod listing to a namespace
./bin/podscape --refresh 5s      # tighter refresh interval
```

## Keys

```
1/2/3   switch tab (floor plan / nodes table / findings)
tab     next tab
c/n/w   density: compact / normal / wide
arrows  move focus between node cards (the plan scrolls to follow)
x/space collapse / expand the focused node group
pgup/pgdn   scroll the floor plan   (ctrl+u / ctrl+d)
g/G     jump to top / bottom of the floor plan
enter   open detail pane (or expand a collapsed group)   esc   close
s       cycle sort column (nodes table)
r       refresh now
?       toggle full help
q       quit
```

## Project layout

```
cmd/podscape/                 flag parsing, kubeconfig, picker bootstrap
internal/k8s/                 client-go wrapper + snapshot fetch
internal/model/               internal types + k8s -> model mapping
internal/analysis/            DS overhead + correctness checks (pure functions)
internal/tui/styles/          Lipgloss theme
internal/tui/floorplan/       the main view (node groups, cards, DS bands)
internal/tui/nodestable/      secondary table view
internal/tui/findings/        findings list view
internal/tui/picker/          context picker
internal/tui/detail/          slide-over node detail panel
internal/tui/app/             root Bubble Tea model: tabs, focus, refresh
```

Each layer below `internal/tui/` is renderer-only and pure: the `Update`
functions don't touch the Kubernetes client and the renderers don't reach for
any global state, which keeps everything testable and easy to extend.

## Development

```
make help          # list targets
make build         # build into ./bin
make run           # go run (ARGS=... to pass flags)
make test          # go test
make test-race     # with the race detector
make cover         # coverage HTML at coverage.html
make fmt           # gofmt (+ goimports if installed)
make fmt-check     # CI-friendly: fail if anything needs fmt
make vet           # go vet
make lint          # golangci-lint
make staticcheck   # honnef.co/go/tools/staticcheck
make vuln          # govulncheck
make tools         # install all of the above
make check         # fmt-check + vet + lint + staticcheck + tests
make ci            # tidy + fmt-check + vet + lint + staticcheck + vuln + test-race
```

The first run of `lint` / `staticcheck` / `vuln` will `go install` the missing
tool. Lint configuration lives in `.golangci.yml`.

## Releases

Tagging a commit with `vX.Y.Z` triggers `.github/workflows/ci.yml`'s `release`
job. It builds for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`,
generates an aggregate `SHA256SUMS`, and signs that file with
[**cosign keyless**](https://docs.sigstore.dev/cosign/signing/overview/) using
GitHub's OIDC token. The signed checksum file, its `.sig`, and the issuing
certificate (`.pem`) are attached to the GitHub Release.

### Verifying a downloaded binary

```sh
# 1. download the archive, SHA256SUMS, SHA256SUMS.sig, SHA256SUMS.pem from the release.
# 2. confirm cosign signed the checksum file:
cosign verify-blob \
  --certificate SHA256SUMS.pem \
  --signature   SHA256SUMS.sig \
  --certificate-identity-regexp 'https://github.com/shearn89/podscape/.*' \
  --certificate-oidc-issuer     https://token.actions.githubusercontent.com \
  SHA256SUMS

# 3. confirm the archive matches its checksum:
sha256sum --check --ignore-missing SHA256SUMS
```

No keys to manage — the signature is provable against the public Sigstore
transparency log and the GitHub Actions workflow that produced it.

## What's deliberately out of scope (for now)

- Mutating verbs of any kind (delete, patch, evict).
- `kubectl describe / logs` passthrough — the structure is there but the
  side-pane currently only renders the model. Easy to add when wanted.
- In-cluster service-account auth — kubeconfig only.
- Anti-affinity selector `matchExpressions` — only `matchLabels` is honoured
  by the check today. Adding it is a straightforward extension.
