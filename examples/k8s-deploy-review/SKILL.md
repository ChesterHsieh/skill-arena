---
name: k8s-deploy-review
description: "Kubernetes deployment manifest review skill. Trigger when user mentions: Kubernetes, K8s, kubectl, deployment, manifest, YAML, helm chart, pod, service, configmap, secret, ingress, deploy review, StatefulSet, DaemonSet, CronJob, resource limits, liveness probe, readiness probe, network policy, RBAC, pod security."
---

## When to Activate

Activate this skill whenever the user's request involves any of the following scenarios:

- Pasting a Kubernetes YAML manifest (Deployment, StatefulSet, DaemonSet, Job, CronJob) and asking for a review
- Asking "is this deployment production-ready?" or similar quality questions
- Requesting a security audit of a K8s workload
- Reviewing a Helm chart values file or rendered template
- Asking about resource limits, liveness/readiness probes, or replica strategy
- Asking whether a deployment is correctly configured for high availability
- Requesting a go/no-go decision before promoting a workload to production
- Asking about network policies, RBAC, or pod security standards for a given manifest
- Mid-review continuation: user provides partially reviewed manifest and asks to complete the analysis

---

## Review Workflow

The review proceeds through **4 gates in fixed order**: Security → Reliability → Observability → Summary.
Never skip a gate. Never reorder gates. If the user says some gates are already done, start from where they left off.

---

### Gate 1 — Security

Check each item. Tag each finding with `[CRITICAL]`, `[HIGH]`, `[MEDIUM]`, or `[LOW]`.

**Secrets Management**
- [ ] No plaintext secrets in `env[].value` or `envFrom` referencing ConfigMaps — secrets must use `secretKeyRef` or a secrets manager (Vault, AWS Secrets Manager)
- [ ] No sensitive data in `args` or `command` fields
- [ ] `imagePullSecrets` uses a named Secret, not hardcoded credentials

**Image Hygiene**
- [ ] Image tag is pinned to a specific digest or immutable tag — never `latest`
- [ ] Image is pulled from a trusted, private registry where possible
- [ ] `imagePullPolicy: Always` when using a mutable tag (if tag cannot be changed)

**RBAC & Identity**
- [ ] Pod does not use `serviceAccountName: default` unless intentional
- [ ] Service account has least-privilege permissions (no `ClusterAdmin` unless justified)
- [ ] `automountServiceAccountToken: false` if the workload does not call the Kubernetes API

**Pod Security**
- [ ] `securityContext.runAsNonRoot: true` — container does not run as UID 0
- [ ] `securityContext.readOnlyRootFilesystem: true` where possible
- [ ] `securityContext.allowPrivilegeEscalation: false`
- [ ] `capabilities.drop: ["ALL"]` — drop all Linux capabilities, add back only what is needed

**Network Exposure**
- [ ] `hostNetwork: true` is absent or explicitly justified
- [ ] `hostPort` is absent unless required (prefer Service objects)
- [ ] A `NetworkPolicy` exists or is planned for this workload (note absence as `[HIGH]` for production namespaces)

---

### Gate 2 — Reliability

**Resource Management**
- [ ] `resources.requests.cpu` and `resources.requests.memory` are set for every container
- [ ] `resources.limits.cpu` and `resources.limits.memory` are set for every container
- [ ] Limits are not set excessively high (> 4x requests is a smell)

**Availability**
- [ ] `replicas >= 2` for any user-facing or critical background service
- [ ] A `PodDisruptionBudget` (PDB) is present or noted as missing for stateful or HA workloads
- [ ] `topologySpreadConstraints` or `podAntiAffinity` rules prevent all pods landing on one node

**Health Probes**
- [ ] `livenessProbe` is defined — action (httpGet, exec, tcpSocket), `initialDelaySeconds`, `periodSeconds`
- [ ] `readinessProbe` is defined — action, `initialDelaySeconds`, `failureThreshold`
- [ ] `startupProbe` is present for slow-starting containers (avoids premature liveness kills)
- [ ] Liveness probe does not call an endpoint that can fail due to external dependencies (would cause cascade restarts)

**Deployment Strategy**
- [ ] `strategy.type` is set (`RollingUpdate` preferred; `Recreate` only if stateful and cannot run two versions simultaneously)
- [ ] `strategy.rollingUpdate.maxUnavailable` and `maxSurge` are configured explicitly
- [ ] `minReadySeconds` is set to allow the rolling update to stabilize before proceeding

**Graceful Shutdown**
- [ ] `terminationGracePeriodSeconds` is set and long enough for in-flight requests to drain
- [ ] `preStop` lifecycle hook or `SIGTERM` handler is present if the container needs time to drain

---

### Gate 3 — Observability

**Labels and Annotations**
- [ ] Standard labels are present: `app.kubernetes.io/name`, `app.kubernetes.io/version`, `app.kubernetes.io/component`, `app.kubernetes.io/part-of`
- [ ] `app.kubernetes.io/managed-by` is set (e.g., `helm`, `kustomize`, `kubectl`)
- [ ] Team/owner annotation is present (e.g., `owner: platform-team`)

**Metrics**
- [ ] Metrics port is declared as a named container port (e.g., `name: metrics, containerPort: 9090`)
- [ ] Prometheus scrape annotations are present (`prometheus.io/scrape`, `prometheus.io/port`, `prometheus.io/path`) or a ServiceMonitor CRD is expected
- [ ] Metrics endpoint does not require authentication that would block scraping

**Logging**
- [ ] Container logs to stdout/stderr (not to a file inside the container)
- [ ] Structured JSON logging is used or noted as a recommendation
- [ ] Log level is configurable via environment variable (not hardcoded)

**Tracing**
- [ ] If the service makes outbound calls, trace context propagation headers are expected (W3C TraceContext / OpenTelemetry)

---

### Gate 4 — Summary

After completing all three gates:

1. **Aggregate findings** — list all findings from all gates with their severity tags, grouped by gate.
2. **Final Verdict** — render one of:
   - `APPROVED — ready for production deployment`
   - `CONDITIONAL APPROVAL — deploy with the following HIGH/MEDIUM items tracked as follow-up`
   - `NO-GO — address CRITICAL findings before deploying`
3. **Prioritized fix list** — ordered by severity (CRITICAL first), show the specific field path to change and the recommended value.

---

## Output Format

Structure your response as four clearly headed sections:

```
## Security Review
[CRITICAL] <field>: <finding>
[HIGH] <field>: <finding>
...

## Reliability Review
[HIGH] <field>: <finding>
...

## Observability Review
[MEDIUM] <field>: <finding>
...

## Summary
**Verdict:** NO-GO / CONDITIONAL APPROVAL / APPROVED

**Prioritized Fix List:**
1. [CRITICAL] spec.template.spec.containers[0].env[0].value — move DB_PASSWORD to a Secret and reference with secretKeyRef
2. [CRITICAL] spec.template.spec.containers[0].image — pin image tag to a digest or immutable version tag
3. [HIGH] spec.replicas — increase to at least 2 for high availability
...
```

- Each finding must include the full YAML field path where possible.
- The verdict must be stated explicitly — do not leave it ambiguous.
- For a `CONDITIONAL APPROVAL`, list exactly which items must be tracked as follow-up issues before the next release cycle.

---

## Notes

### Kubernetes Version Considerations (1.24+)

- **PodSecurityPolicy** was removed in 1.25. Use **Pod Security Admission** (labels on namespace: `pod-security.kubernetes.io/enforce`) instead.
- **`securityContext.seccompProfile`** (`RuntimeDefault` or `Localhost`) is available since 1.19 and recommended for 1.24+.
- **`topologySpreadConstraints`** is GA since 1.19 — prefer it over `podAntiAffinity` for spread constraints.
- **Ephemeral containers** (for debugging) are GA since 1.25 — mention as a debugging option when relevant.

### StatefulSet vs Deployment

| Concern | Deployment | StatefulSet |
|---------|-----------|------------|
| Pod identity | Ephemeral, interchangeable | Stable hostname (`pod-0`, `pod-1`) |
| Storage | Shared PVC or no PVC | Per-pod PVC via `volumeClaimTemplates` |
| Rolling update | Standard rolling | Ordered pod-by-pod update |
| Readiness probe | Strongly recommended | Critical — gates ordered rollout |
| PodDisruptionBudget | Recommended for HA | Required for quorum-sensitive systems |
| `replicas: 1` | Acceptable for non-critical workers | Often correct for primary-only DBs, note the risk |

When reviewing a StatefulSet, additionally check:
- `volumeClaimTemplates` storage class and access mode
- `podManagementPolicy` (`OrderedReady` vs `Parallel`)
- Whether the application handles pod restarts and ordinal identity correctly

### References

See `references/checklist.md` for the complete field-level checklist with specific YAML paths for each gate.
