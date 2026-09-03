# Decisions

## Deterministic policy

Currency is EUR. The latest daily-sale price is current price. A promotion returns `promotion_active`; fewer than fourteen observations returns `low_confidence`; a less-than-1% gross-margin improvement returns `hold`.

## Scope boundaries

No authentication, real retailer integration, ORM, queues, cache, ML training, Kubernetes, or cloud deployment is included. This keeps the repository focused on explainable pricing logic and directly inspectable SQL.

## Trade-offs

The importer uses a transactional upsert boundary and reports invalid rows without discarding parsed valid rows. The sample system processes small data synchronously; production workloads would use bounded batch workers, durable job execution, richer audit trails, and a reviewed commercial pricing policy.
