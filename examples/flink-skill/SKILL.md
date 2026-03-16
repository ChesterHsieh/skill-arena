---
name: flink-skill
description: "Apache Flink expert skill. Trigger when user mentions: Flink, DataStream, Table API, Kafka source, watermark, backpressure, checkpoint, state backend, exactly-once, CEP, tumbling window, sliding window, session window, stream processing, stateful computation, Flink job, JobManager, TaskManager, savepoint."
---

## When to Activate

Activate this skill whenever the user's request involves any of the following scenarios:

- Writing a new Flink DataStream API job (Java or Python)
- Using the Table API / SQL for stream or batch queries
- Connecting to Kafka as a source or sink (KafkaSource, KafkaSink, FlinkKafkaConsumer)
- Defining watermark strategies for event-time processing
- Choosing or configuring window types: tumbling, sliding, session, global
- Diagnosing or tuning checkpoint failures, timeouts, or performance issues
- Selecting and configuring a state backend (HashMapStateBackend, RocksDBStateBackend)
- Implementing exactly-once or at-least-once semantics end-to-end
- Using Complex Event Processing (CEP) with Flink's PatternStream API
- Investigating backpressure using the Flink Web UI or metrics
- Configuring savepoints, incremental checkpointing, or state migration
- Asking about Flink cluster topology, parallelism, or slot allocation
- Migrating jobs between Flink versions (1.14 → 1.17 → 1.19)

---

## Workflow

### Path A — Code Generation

Use when the user asks to write a new Flink job or add a feature to an existing one.

1. **Clarify inputs and outputs.** Identify the source (Kafka topic, file, socket), the sink (Kafka, JDBC, filesystem), event schema, and whether the job should use event time or processing time.
2. **Identify the window strategy.** Ask (or infer) whether a tumbling, sliding, or session window is needed. Confirm the key field and aggregate function.
3. **Choose the state backend.** Default to `HashMapStateBackend` for small state; recommend `RocksDBStateBackend` with incremental checkpointing when state may exceed available heap.
4. **Generate the complete, executable code.** Include all imports, a `main` method or entry point, proper watermark assignment, the full operator chain, and a working sink.
5. **Add a watermark strategy.** For event-time jobs always emit watermarks — prefer `WatermarkStrategy.forBoundedOutOfOrderness` with a documented tolerance value.
6. **Annotate checkpointing.** Show `env.enableCheckpointing(intervalMs)` with a comment explaining the chosen interval and storage backend.
7. **Verify the code compiles in your head.** Walk through the type chain (KeyedStream, WindowedStream, etc.) to catch common type-erasure and serialization mistakes before outputting.

### Path B — Architecture Design

Use when the user is designing a new streaming system or evaluating Flink for a use case.

1. **Understand scale and SLAs.** Ask about event throughput, acceptable latency, state size, and retention window.
2. **Recommend parallelism and deployment.** Map the workload to suggested parallelism, number of TaskManagers, memory config, and whether Flink-on-Kubernetes or a standalone cluster is appropriate.
3. **Select the state backend and checkpoint strategy.** Justify the choice (in-memory vs. RocksDB, periodic vs. incremental, checkpoint interval vs. latency trade-off).
4. **Design the operator graph.** Draw the logical DAG with sources → transformations → sinks, noting where `keyBy` boundaries occur and where state lives.
5. **Address fault tolerance.** Explain the delivery guarantee (exactly-once requires Kafka transactions + two-phase commit sink), and show how savepoints enable job upgrades.
6. **Identify risks.** Flag potential hotspots (skewed keys, large state per key), serialization pitfalls (Avro vs. Kryo), and backpressure propagation paths.

### Path C — Error Diagnosis

Use when the user pastes an error message, slow-job symptoms, or checkpoint failure logs.

1. **Identify the error class.** Is it a checkpoint timeout, out-of-memory, serialization error, network buffer exhaustion, or operator exception?
2. **Ask for context if missing.** Request parallelism, state backend, checkpoint interval, approximate event rate, and relevant TaskManager/JobManager logs.
3. **Map the error to root causes.** For each identified class, list the two or three most common root causes with a one-sentence explanation.
4. **Provide concrete diagnostic steps.** Point the user to specific Flink Web UI panels (Backpressure tab, Checkpoints tab) or metrics (`numRecordsInPerSecond`, `lastCheckpointDuration`, `numBytesInLocal`).
5. **Give actionable fixes.** For each root cause, provide the configuration knob or code change that resolves it, with the specific property name (e.g., `execution.checkpointing.timeout`, `state.backend.rocksdb.block.cache-size`).
6. **Suggest a verification step.** Tell the user how to confirm the fix worked (metric to watch, log line to look for).

---

## Output Format

1. **Code block first.** Always open with a fenced code block tagged with the language (`java` or `python`). The code must be complete and executable — never use placeholder comments like `// TODO: implement`.

2. **200-word explanation after the code.** Immediately following the code block, write a prose explanation (targeting ~200 words) covering:
   - What the job does end-to-end
   - Key design decisions made (window type, state backend, watermark tolerance)
   - How the job achieves the stated delivery guarantee

3. **Caveats section.** End every response with a short `### Caveats` subsection listing:
   - Any assumptions made about the user's environment (Flink version, Kafka version)
   - Known limitations of the generated approach (e.g., "this uses processing time; switch to event time if your source emits timestamps")
   - Any configuration values that must be tuned for production (e.g., checkpoint interval, parallelism, state TTL)

---

## Notes

### Version Compatibility

| Feature | Flink 1.14 | Flink 1.17 | Flink 1.19 |
|---------|-----------|-----------|-----------|
| `KafkaSource` (new source API) | Yes | Yes | Yes |
| Unaligned checkpoints (stable) | Preview | Stable | Stable |
| Incremental RocksDB checkpoints | Yes | Yes | Yes |
| Generic incremental checkpointing | No | No | Yes |
| State processor API | Yes | Yes | Yes (improved) |
| Flink SQL: changelog sources | Limited | Stable | Stable |

- Always ask the user which Flink version they are on before generating code that uses APIs introduced in 1.17+.
- Flink 1.14 requires `FlinkKafkaConsumer` (legacy); 1.15+ can use `KafkaSource` builder API.
- `WatermarkStrategy.forBoundedOutOfOrderness` is the preferred API since 1.11; do not use the deprecated `AssignerWithPeriodicWatermarks` interface.

### Java vs Python API Differences

| Concern | Java DataStream API | PyFlink DataStream API |
|---------|--------------------|-----------------------|
| Type safety | Generics enforced at compile time | Runtime duck-typing |
| Serialization | Flink TypeInformation (Kryo fallback) | Pickle (slower) or Arrow |
| Custom operators | `RichFlatMapFunction`, `ProcessFunction` | `FlatMapFunction`, `KeyedProcessFunction` |
| CEP | Full support (`PatternStream`) | Limited (use Table API workarounds) |
| Performance | ~10-20% faster for CPU-bound ops | Acceptable for I/O-bound pipelines |

- Prefer Java for production, latency-sensitive, or CEP-heavy jobs.
- PyFlink is appropriate for data-science teams comfortable with Python tooling.
- See `references/datastream-api.md` for quick-reference snippets on KafkaSource setup, watermark strategies, and window types.
