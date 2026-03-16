# Kubernetes Deployment Review — Field-Level Checklist

## Gate 1: Security

### Secrets Management

| Check | YAML Field Path | Severity |
|-------|----------------|----------|
| No plaintext secret in env value | `spec.template.spec.containers[*].env[*].value` | CRITICAL |
| Secret ref uses secretKeyRef | `spec.template.spec.containers[*].env[*].valueFrom.secretKeyRef` | CRITICAL |
| No secrets in command/args | `spec.template.spec.containers[*].command`, `args` | CRITICAL |
| imagePullSecrets uses named Secret | `spec.template.spec.imagePullSecrets[*].name` | HIGH |

### Image Hygiene

| Check | YAML Field Path | Severity |
|-------|----------------|----------|
| Image tag is not `latest` | `spec.template.spec.containers[*].image` | CRITICAL |
| Image pulled from trusted registry | `spec.template.spec.containers[*].image` | HIGH |
| imagePullPolicy is `Always` if using mutable tag | `spec.template.spec.containers[*].imagePullPolicy` | MEDIUM |

### RBAC & Identity

| Check | YAML Field Path | Severity |
|-------|----------------|----------|
| serviceAccountName is not `default` (unless intentional) | `spec.template.spec.serviceAccountName` | HIGH |
| automountServiceAccountToken disabled if API not needed | `spec.template.spec.automountServiceAccountToken` | MEDIUM |
| Associated ServiceAccount has least-privilege RBAC | External: `ClusterRole` / `Role` + `Binding` | HIGH |

### Pod Security (securityContext)

| Check | YAML Field Path | Severity |
|-------|----------------|----------|
| runAsNonRoot: true | `spec.template.spec.securityContext.runAsNonRoot` | HIGH |
| runAsUser is not 0 | `spec.template.spec.securityContext.runAsUser` | HIGH |
| readOnlyRootFilesystem: true | `spec.template.spec.containers[*].securityContext.readOnlyRootFilesystem` | MEDIUM |
| allowPrivilegeEscalation: false | `spec.template.spec.containers[*].securityContext.allowPrivilegeEscalation` | HIGH |
| capabilities.drop: ["ALL"] | `spec.template.spec.containers[*].securityContext.capabilities.drop` | HIGH |
| seccompProfile: RuntimeDefault (k8s 1.19+) | `spec.template.spec.securityContext.seccompProfile.type` | MEDIUM |

### Network Exposure

| Check | YAML Field Path | Severity |
|-------|----------------|----------|
| hostNetwork is absent or false | `spec.template.spec.hostNetwork` | HIGH |
| hostPort is absent | `spec.template.spec.containers[*].ports[*].hostPort` | HIGH |
| NetworkPolicy exists in namespace | External resource | HIGH |

---

## Gate 2: Reliability

### Resource Management

| Check | YAML Field Path | Severity |
|-------|----------------|----------|
| CPU request set | `spec.template.spec.containers[*].resources.requests.cpu` | HIGH |
| Memory request set | `spec.template.spec.containers[*].resources.requests.memory` | HIGH |
| CPU limit set | `spec.template.spec.containers[*].resources.limits.cpu` | MEDIUM |
| Memory limit set | `spec.template.spec.containers[*].resources.limits.memory` | HIGH |
| Limits ≤ 4× requests | `resources.limits` / `resources.requests` ratio | MEDIUM |

### Availability

| Check | YAML Field Path | Severity |
|-------|----------------|----------|
| replicas ≥ 2 for user-facing services | `spec.replicas` | HIGH |
| PodDisruptionBudget exists for HA workloads | External: `PodDisruptionBudget` resource | HIGH |
| podAntiAffinity or topologySpreadConstraints present | `spec.template.spec.affinity.podAntiAffinity` or `spec.template.spec.topologySpreadConstraints` | MEDIUM |

### Health Probes

| Check | YAML Field Path | Severity |
|-------|----------------|----------|
| livenessProbe defined | `spec.template.spec.containers[*].livenessProbe` | HIGH |
| readinessProbe defined | `spec.template.spec.containers[*].readinessProbe` | HIGH |
| startupProbe for slow starters | `spec.template.spec.containers[*].startupProbe` | MEDIUM |
| Liveness probe uses internal health, not external dep | `spec.template.spec.containers[*].livenessProbe.httpGet.path` | MEDIUM |
| initialDelaySeconds set on probes | `*.livenessProbe.initialDelaySeconds`, `*.readinessProbe.initialDelaySeconds` | MEDIUM |

### Deployment Strategy

| Check | YAML Field Path | Severity |
|-------|----------------|----------|
| strategy.type is set explicitly | `spec.strategy.type` | LOW |
| maxUnavailable set | `spec.strategy.rollingUpdate.maxUnavailable` | MEDIUM |
| maxSurge set | `spec.strategy.rollingUpdate.maxSurge` | LOW |
| minReadySeconds set | `spec.minReadySeconds` | LOW |

### Graceful Shutdown

| Check | YAML Field Path | Severity |
|-------|----------------|----------|
| terminationGracePeriodSeconds set | `spec.template.spec.terminationGracePeriodSeconds` | MEDIUM |
| preStop lifecycle hook present for request-serving containers | `spec.template.spec.containers[*].lifecycle.preStop` | MEDIUM |

---

## Gate 3: Observability

### Labels

| Check | YAML Field Path | Severity |
|-------|----------------|----------|
| app.kubernetes.io/name present | `metadata.labels["app.kubernetes.io/name"]` | MEDIUM |
| app.kubernetes.io/version present | `metadata.labels["app.kubernetes.io/version"]` | LOW |
| app.kubernetes.io/component present | `metadata.labels["app.kubernetes.io/component"]` | LOW |
| app.kubernetes.io/part-of present | `metadata.labels["app.kubernetes.io/part-of"]` | LOW |
| app.kubernetes.io/managed-by present | `metadata.labels["app.kubernetes.io/managed-by"]` | LOW |
| Owner/team annotation present | `metadata.annotations["owner"]` or similar | MEDIUM |

### Metrics

| Check | YAML Field Path | Severity |
|-------|----------------|----------|
| Metrics port declared as named port | `spec.template.spec.containers[*].ports[*].name: metrics` | MEDIUM |
| Prometheus scrape annotations present | `metadata.annotations["prometheus.io/scrape"]` | MEDIUM |
| prometheus.io/port annotation matches metrics port | `metadata.annotations["prometheus.io/port"]` | LOW |

### Logging

| Check | YAML Field Path | Severity |
|-------|----------------|----------|
| Container logs to stdout/stderr (not file) | Application-level — ask or infer from image | MEDIUM |
| Log level configurable via env var | `spec.template.spec.containers[*].env[*].name: LOG_LEVEL` | LOW |

---

## Gate 4: Summary Template

```
## Summary

**Verdict:** [NO-GO / CONDITIONAL APPROVAL / APPROVED]

**Critical Findings (must fix before deploy):**
- [CRITICAL] <field>: <issue> → <fix>

**High Findings (fix in current sprint):**
- [HIGH] <field>: <issue> → <fix>

**Medium Findings (track as follow-up):**
- [MEDIUM] <field>: <issue> → <fix>

**Low Findings (informational):**
- [LOW] <field>: <issue> → <fix>
```

### Verdict Criteria

| Verdict | Condition |
|---------|-----------|
| `NO-GO` | Any CRITICAL finding present |
| `CONDITIONAL APPROVAL` | No CRITICAL; one or more HIGH findings tracked as issues |
| `APPROVED` | No CRITICAL, no HIGH; MEDIUM and LOW findings documented |
