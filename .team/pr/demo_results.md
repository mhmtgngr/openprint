# 🎬 Demo Report — 2026-02-28 05:04

## Service Health
```
  ❌ :15432 → UNREACHABLE
  ❌ :16379 → UNREACHABLE
  ✅ :18001 → {"status":"healthy","service":"auth-service"}
  ✅ :18005 → {"status":"healthy","service":"notification-service"}
  ✅ :8002 → {"status":"healthy","service":"registry-service"}
  ✅ :8003 → {"status":"healthy","service":"job-service"}
  ✅ :8004 → {"status":"healthy","service":"storage-service"}
```
## Test Results
```
?   	github.com/openprint/openprint/internal/middleware	[no test files]
ok  	github.com/openprint/openprint/internal/shared/errors	0.010s
?   	github.com/openprint/openprint/internal/shared/middleware	[no test files]
ok  	github.com/openprint/openprint/internal/shared/telemetry	0.032s
?   	github.com/openprint/openprint/services/auth-service	[no test files]
?   	github.com/openprint/openprint/services/auth-service/handler	[no test files]
ok  	github.com/openprint/openprint/services/auth-service/repository	0.008s
?   	github.com/openprint/openprint/services/job-service	[no test files]
ok  	github.com/openprint/openprint/services/job-service/handler	0.015s
ok  	github.com/openprint/openprint/services/job-service/processor	0.009s
ok  	github.com/openprint/openprint/services/job-service/repository	0.012s
?   	github.com/openprint/openprint/services/notification-service	[no test files]
ok  	github.com/openprint/openprint/services/notification-service/websocket	0.025s
?   	github.com/openprint/openprint/services/registry-service	[no test files]
?   	github.com/openprint/openprint/services/registry-service/handler	[no test files]
ok  	github.com/openprint/openprint/services/registry-service/repository	0.012s
?   	github.com/openprint/openprint/services/storage-service	[no test files]
ok  	github.com/openprint/openprint/services/storage-service/handler	0.010s
ok  	github.com/openprint/openprint/services/storage-service/storage	0.240s
ok  	github.com/openprint/openprint/tests/integration	2.905s
```
