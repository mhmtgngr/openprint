# 🎬 Demo Report — 2026-02-28 11:50

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
--- FAIL: TestJobAssignmentRepository_UpdateHeartbeat (0.00s)
    job_assignment_test.go:359: failed to create test print job: failed to connect to `user=openprint database=openprint`: 127.0.0.1:5432 (localhost): dial error: dial tcp 127.0.0.1:5432: connect: connection refused
--- FAIL: TestJobAssignmentRepository_IncrementRetry (0.00s)
    job_assignment_test.go:359: failed to create test print job: failed to connect to `user=openprint database=openprint`: 127.0.0.1:5432 (localhost): dial error: dial tcp 127.0.0.1:5432: connect: connection refused
--- FAIL: TestJobAssignmentRepository_SetError (0.00s)
    job_assignment_test.go:359: failed to create test print job: failed to connect to `user=openprint database=openprint`: 127.0.0.1:5432 (localhost): dial error: dial tcp 127.0.0.1:5432: connect: connection refused
--- FAIL: TestJobAssignmentRepository_GetStaleAssignments (0.00s)
    job_assignment_test.go:359: failed to create test print job: failed to connect to `user=openprint database=openprint`: 127.0.0.1:5432 (localhost): dial error: dial tcp 127.0.0.1:5432: connect: connection refused
FAIL
FAIL	github.com/openprint/openprint/services/job-service/repository	0.016s
?   	github.com/openprint/openprint/services/notification-service	[no test files]
ok  	github.com/openprint/openprint/services/notification-service/websocket	0.023s
?   	github.com/openprint/openprint/services/registry-service	[no test files]
?   	github.com/openprint/openprint/services/registry-service/handler	[no test files]
ok  	github.com/openprint/openprint/services/registry-service/repository	0.012s
?   	github.com/openprint/openprint/services/storage-service	[no test files]
ok  	github.com/openprint/openprint/services/storage-service/handler	0.017s
ok  	github.com/openprint/openprint/services/storage-service/storage	0.233s
ok  	github.com/openprint/openprint/tests/integration	2.844s
FAIL
```
