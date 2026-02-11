# Capacity and Performance

## Rule count (规则数量)

- **No hard limit in code.** The system loads all enabled rules for:
  - **List API** (`GET /api/v1/rules`): returns all rules (no pagination).
  - **Scheduler**: loads all rules with `enabled = true` and non-empty `query_expression`; each such rule gets **one long-lived goroutine** that runs on its configured `check_interval`.
  - **Engine** (inbound alert processing): loads all enabled rules when matching an incoming alert.

- **Practical limits:**
  - **Memory**: Full rule list is held in memory when loading; rule struct is small, so hundreds to a few thousand rules are fine on typical deployments.
  - **Goroutines**: One goroutine per enabled rule with a query. ~100–1000 rules ⇒ 100–1000 goroutines; Go handles this well, but very large numbers (e.g. 10k+) may need tuning (e.g. worker pool in a future version).
  - **Database**: Rule table size and query latency for `Find(&rules)`; indexing on `enabled` is recommended for large rule sets.

- **Recommendation:** Use in the **hundreds to low thousands** of rules per instance without changes. For larger scale, consider splitting by datasource or introducing pagination/limits in the List API and a bounded worker pool in the scheduler.

---

## Concurrency (并发性能)

- **Scheduler**
  - Each rule runs in its **own goroutine** with a fixed-interval timer (no global worker pool).
  - **Concurrent execution**: Many rules can evaluate at the same time (e.g. when intervals align).
  - **Per-rule timeout**: Each evaluation uses a **30s** context timeout (`evaluateRule`).
  - **Single evaluation flow**: For one rule, execution is sequential: query datasource (Prometheus/VM) → update alert state → call engine to send notifications.

- **Bottlenecks**
  - **Datasource**: Concurrent requests = number of rules that fire in the same time window. A single Prometheus/VictoriaMetrics may need rate limiting or scaling if many rules hit it at once.
  - **Notification channels**:
    - **Lark (飞书)**: Token-bucket rate limiter **5 req/s**, burst 3 (see `sender/larkRateLimiter`). High alert volume to Lark will queue/wait.
    - **Telegram / others**: No in-app limiter; external API limits apply.

- **Inbound (webhook) path**: Alertmanager/VM/ES/Doris push alerts to the server; each request is handled in a new goroutine. Matching is over all enabled rules in memory; no per-rule goroutine here.

---

## Summary

| Item              | Behavior / limit |
|-------------------|------------------|
| Max rules (code)  | No hard limit    |
| Suggested scale   | Hundreds – low thousands of rules |
| Concurrency model | One goroutine per rule (scheduler); inbound = one goroutine per request |
| Per-rule timeout  | 30s (scheduler evaluation) |
| Lark send rate    | 5 req/s (burst 3) |
| Report export     | Alerts list capped at 10,000 rows |
