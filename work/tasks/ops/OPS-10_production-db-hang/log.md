
## 2026-11-12 — Coordinator acceptance

Accepted H.1–H.4 (commits 3dfab7b, d7c8526, 2ed1e31, d07924c, 5c2cba2): sqlite 1.62.0 + libc 1.81.0 pinned; status from cache (0.156 s → 0.0006 s); one writer + read-only pool with per-connection pragmas and IMMEDIATE write transactions; /api/health 503 within 2 s. H.5 awaits the owner. The full jobs list refetched every 2.5 s (H.4) goes to INBOX-2 as L.4d, which owns the poll loop.
