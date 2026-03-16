# Flink DataStream API — Quick Reference

## KafkaSource Setup (Flink 1.15+)

```java
KafkaSource<String> source = KafkaSource.<String>builder()
    .setBootstrapServers("broker1:9092,broker2:9092")
    .setTopics("user-events")
    .setGroupId("my-flink-group")
    .setStartingOffsets(OffsetsInitializer.earliest())
    .setValueOnlyDeserializer(new SimpleStringSchema())
    .build();

DataStream<String> stream = env.fromSource(
    source,
    WatermarkStrategy.forBoundedOutOfOrderness(Duration.ofSeconds(10)),
    "Kafka Source"
);
```

For Flink 1.14 and earlier, use the legacy `FlinkKafkaConsumer`:

```java
FlinkKafkaConsumer<String> consumer = new FlinkKafkaConsumer<>(
    "user-events",
    new SimpleStringSchema(),
    kafkaProps
);
consumer.setStartFromEarliest();
DataStream<String> stream = env.addSource(consumer);
```

---

## Watermark Strategies

### Bounded Out-of-Orderness (most common)

```java
WatermarkStrategy.<MyEvent>forBoundedOutOfOrderness(Duration.ofSeconds(5))
    .withTimestampAssigner((event, recordTimestamp) -> event.getEventTime());
```

Use when events may arrive slightly out of order but within a known bound.

### Monotonically Increasing Timestamps

```java
WatermarkStrategy.<MyEvent>forMonotonousTimestamps()
    .withTimestampAssigner((event, ts) -> event.getTimestamp());
```

Use for strictly ordered sources (e.g., a single Kafka partition read in order).

### No Watermarks (processing time only)

```java
WatermarkStrategy.noWatermarks()
```

Use only when you explicitly choose `ProcessingTime` windows and do not need event-time semantics.

### Custom Watermark Generator

```java
WatermarkStrategy.<MyEvent>forGenerator(
    ctx -> new BoundedOutOfOrdernessWatermarks<>(Duration.ofSeconds(30))
).withTimestampAssigner((e, ts) -> e.getEventTimeMs());
```

---

## Window Types and When to Use Each

### Tumbling Window

```java
stream
    .keyBy(Event::getUserId)
    .window(TumblingEventTimeWindows.of(Time.minutes(5)))
    .aggregate(new SumAggregator());
```

**Use when:** Each event belongs to exactly one fixed-size, non-overlapping time bucket.
**Example:** Total purchases per user per 5-minute period.

### Sliding Window

```java
stream
    .keyBy(Event::getUserId)
    .window(SlidingEventTimeWindows.of(Time.minutes(5), Time.seconds(30)))
    .aggregate(new MovingAverageAggregator());
```

**Use when:** You need overlapping windows (e.g., a 5-minute window that advances every 30 seconds).
**Warning:** Each event belongs to `window_size / slide_size` windows — high overlap increases state size.

### Session Window

```java
stream
    .keyBy(Event::getSessionId)
    .window(EventTimeSessionWindows.withGap(Time.minutes(30)))
    .process(new SessionProcessor());
```

**Use when:** Activity groups naturally by idle gaps (e.g., user sessions, click streams).
**Note:** Session windows are dynamic — their boundaries are determined at runtime based on event gaps.

### Global Window (manual trigger)

```java
stream
    .keyBy(Event::getKey)
    .window(GlobalWindows.create())
    .trigger(CountTrigger.of(1000))
    .process(new BatchProcessor());
```

**Use when:** You need custom triggering logic (count-based, external signal). Requires an explicit trigger — defaults to never firing.

---

## Late Event Handling

```java
OutputTag<Event> lateTag = new OutputTag<Event>("late-events") {};

SingleOutputStreamOperator<Result> main = stream
    .keyBy(Event::getUserId)
    .window(TumblingEventTimeWindows.of(Time.minutes(5)))
    .allowedLateness(Time.seconds(30))
    .sideOutputLateData(lateTag)
    .aggregate(new SumAggregator());

DataStream<Event> lateStream = main.getSideOutput(lateTag);
lateStream.addSink(new LateEventSink());
```

---

## State Backends

### HashMapStateBackend (default, in-memory)

```java
env.setStateBackend(new HashMapStateBackend());
env.getCheckpointConfig().setCheckpointStorage("s3://my-bucket/checkpoints");
```

**When to use:** Total state fits comfortably in TaskManager heap (< 1 GB per slot).
**Pros:** Low latency access, no serialization overhead during processing.
**Cons:** State lost if TaskManager dies between checkpoints; large state causes GC pressure.

### RocksDBStateBackend (disk-spilling)

```java
env.setStateBackend(new EmbeddedRocksDBStateBackend(true)); // true = incremental
env.getCheckpointConfig().setCheckpointStorage("s3://my-bucket/checkpoints");
```

**When to use:** State exceeds available heap, or job retains state over long windows (hours/days).
**Pros:** Handles very large state; incremental checkpoints drastically reduce checkpoint size.
**Cons:** ~2-5x slower state access than in-memory due to serialization + disk I/O.

---

## Checkpoint Configuration Best Practices

```java
CheckpointConfig cc = env.getCheckpointConfig();

// Interval: balance between recovery time and throughput impact
cc.setCheckpointInterval(60_000);           // every 60 seconds

// Timeout: must be long enough for all operators to acknowledge
cc.setCheckpointTimeout(120_000);           // 2 minutes

// Concurrency: 1 is safe; increase only if checkpoints are slow
cc.setMaxConcurrentCheckpoints(1);

// Minimum pause between checkpoints (prevents back-to-back checkpoints)
cc.setMinPauseBetweenCheckpoints(30_000);   // 30 seconds

// Retain checkpoints on cancellation (useful for debugging)
cc.setExternalizedCheckpointCleanup(
    ExternalizedCheckpointCleanup.RETAIN_ON_CANCELLATION
);

// Unaligned checkpoints: reduces checkpoint time under backpressure
// (Flink 1.15+ stable; requires exactly-once mode)
cc.enableUnalignedCheckpoints();
```

### Choosing Checkpoint Interval

| Scenario | Suggested Interval |
|----------|-------------------|
| Low-latency, small state | 10–30 seconds |
| Standard stream job | 60 seconds |
| High-throughput, large state | 5–10 minutes |
| Batch-like workloads | Savepoints only |

### Common Checkpoint Failure Causes

1. **Checkpoint timeout** — operators too slow to snapshot. Fix: increase `checkpointTimeout`, enable incremental RocksDB, check for backpressure.
2. **Large state** — checkpoint payload exceeds network or storage bandwidth. Fix: switch to RocksDB + incremental checkpoints, add state TTL.
3. **GC pauses** — JVM stop-the-world GC blocks operator from acknowledging. Fix: tune heap size, use G1GC, reduce in-heap state.
4. **Slow remote storage** — S3/HDFS write latency spikes. Fix: use async checkpointing (default), check S3 endpoint region, increase timeout.
