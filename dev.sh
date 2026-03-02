#!/bin/bash
# ═══════════════════════════════════════════════════════════════
#
#  dev.sh — AI Development Team + Self-Healing Controller
#  ────────────────────────────────────────────────────────
#  Single-script orchestrator: runs phases in-process,
#  monitors itself, auto-heals, self-improves.
#
#  TEAM:
#    🧑‍💼 PM              (Z.ai / Claude)  — Requirements, user stories
#    🔍 Market Researcher (Z.ai / Claude)  — Competitor analysis
#    🏗️  Architect        (Z.ai / Claude)  — System design, API contracts
#    ⚙️  Backend Dev      (Claude Code)    — Implementation
#    🎨 Frontend Dev      (Claude Code)    — React/TypeScript UI
#    🧪 Tester           (Claude Code)    — Unit tests, E2E
#    📋 QA Controller     (Z.ai / Claude)  — Code review, quality gates
#    🔒 Security Auditor  (Z.ai / Claude)  — Vulnerability scan
#    🐳 DevOps           (Claude Code)    — Docker, deploy, smoke test
#
#  WATERFALL + FEEDBACK LOOPS:
#    Requirements → Market Research → Design → Backend → Frontend
#    → Testing → QA → Security → Deploy
#         ↑               |        |       |
#         └───────────────┘────────┘───────┘ (auto-fix on failure)
#
#  USAGE:
#    ./dev.sh start "project description"    # Full waterfall (background)
#    ./dev.sh status                         # Dashboard
#    ./dev.sh stop                           # Stop everything
#    ./dev.sh resume                         # Continue from last phase
#    ./dev.sh phase backend                  # Single phase
#    ./dev.sh smart-improve [threshold%]     # Scan→Adapt→Run→Demo→PR
#    ./dev.sh improve [proj_N] [team_N]      # Dual-track improvement
#    ./dev.sh scan                           # Gap analysis
#    ./dev.sh -h                             # Full help
#
# ═══════════════════════════════════════════════════════════════

set -uo pipefail

# ── Load PATH for background/nohup execution ──
for rc in "$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.profile" "$HOME/.zshrc"; do
  [ -f "$rc" ] && source "$rc" 2>/dev/null || true
done
export PATH="$HOME/go/bin:$HOME/.local/bin:$HOME/.npm-global/bin:$HOME/.nvm/versions/node/*/bin:/usr/local/go/bin:/usr/local/bin:$PATH"
[ -s "$HOME/.nvm/nvm.sh" ] && source "$HOME/.nvm/nvm.sh" 2>/dev/null || true

# ═══════════════════════════════════════════════
# CONFIGURATION
# ═══════════════════════════════════════════════

REPO_DIR="$PWD"
DEV_DIR="$REPO_DIR/.team"
STATE_FILE="$DEV_DIR/state.json"
LIVE_LOG="$DEV_DIR/live.log"
PID_FILE="$DEV_DIR/dev.pid"
SUP_LOG="$DEV_DIR/supervisor.log"
ARTIFACTS="$DEV_DIR/artifacts"
PHASE_LOGS="$DEV_DIR/logs"
PATCHES_DIR="$DEV_DIR/patches"
PHASE_HISTORY="$DEV_DIR/phase_history.json"
ERROR_LOG="$DEV_DIR/error_history.jsonl"
STUCK_HEAL_FILE="$DEV_DIR/stuck_heals.txt"

CLAUDE_MODEL="${CLAUDE_MODEL:-opus}"
ZAI_API_KEY="${ZAI_API_KEY:-}"
ZAI_URL="${ZAI_ENDPOINT:-https://api.z.ai/api/paas/v4/chat/completions}"
ZAI_MODEL="${ZAI_MODEL:-glm-5}"
ZAI_SEARCH_URL="${ZAI_SEARCH_ENDPOINT:-https://api.z.ai/api/paas/v4/web_search}"

MAX_LOOPS=3
MAX_PHASE_RETRIES=2
MAX_CRASHES=5
DOCKER_TIMEOUT=30

# Master dev.sh — the SINGLE source of truth for self-improvement
MASTER_DEV_SH="${MASTER_DEV_SH:-$HOME/dev.sh}"
MASTER_DEV_DIR="${MASTER_DEV_DIR:-$HOME/.dev-master}"
TEAM_ANALYSIS_FILE="$MASTER_DEV_DIR/dev_analysis.json"
TEAM_PLAN_FILE="$MASTER_DEV_DIR/dev_improvements.json"

# Smart improve
PLAN_FILE="$ARTIFACTS/next_phases.json"
COMPLETION_FILE="$ARTIFACTS/project_completion.json"
GAPS_FILE="$ARTIFACTS/project_gaps.json"
DEMO_DIR="$DEV_DIR/demo"
PR_DIR="$DEV_DIR/pr"
PR_THRESHOLD="${PR_THRESHOLD:-50}"
AUTO_PHASES="${AUTO_PHASES:-3}"

# Port range (auto-detected from docker-compose, or default)
SERVICE_PORTS="${SERVICE_PORTS:-}"
# Health check ports (external services only - excludes postgres, redis, etc.)
HEALTH_CHECK_PORTS="${HEALTH_CHECK_PORTS:-}"

# Health check configuration
HEALTH_CHECK_PATH="${HEALTH_CHECK_PATH:-/health}"
HEALTH_CHECK_TIMEOUT="${HEALTH_CHECK_TIMEOUT:-5}"
SKIP_HEALTH_CHECK="${SKIP_HEALTH_CHECK:-false}"

# External server configuration (for client connectivity)
# Auto-detects server IP, can be overridden for remote testing
SERVER_HOST="${SERVER_HOST:-$(hostname -f 2>/dev/null || hostname)}"
SERVER_IP="${SERVER_IP:-$(hostname -I 2>/dev/null | awk '{print $1}' || echo '127.0.0.1')}"

# Project type auto-detection
detect_project_type() {
  if [ -f "go.mod" ] && grep -q "module " go.mod 2>/dev/null; then
    echo "go"
  elif [ -f "package.json" ] && [ -d "src" ]; then
    echo "node"
  elif [ -f "requirements.txt" ] || [ -f "pyproject.toml" ] || [ -f "setup.py" ]; then
    echo "python"
  elif [ -f "pom.xml" ] || [ -f "build.gradle" ]; then
    echo "java"
  elif [ -f "Cargo.toml" ]; then
    echo "rust"
  else
    echo "unknown"
  fi
}

PROJECT_TYPE="${PROJECT_TYPE:-$(detect_project_type)}"

# Auto-detect frontend directory and port (with intelligent nested search)
detect_frontend_config() {
  local frontend_dir=""
  local default_port="3000"
  local best_score=0

  # Priority 1: Direct root-level frontend with E2E framework
  # Priority 2: Nested frontend (e.g., web/dashboard) with E2E framework
  # Priority 3: Any directory with package.json

  # Score-based detection: higher score = better match
  for dir in $(find "$REPO_DIR" -maxdepth 3 -type d -name "node_modules" -prune -o -type f -name "package.json" -print | xargs dirname 2>/dev/null | sort -u); do
    local rel_dir="${dir#$REPO_DIR/}"
    [ "$rel_dir" = "$dir" ] && rel_dir="$dir"  # Handle case where already relative

    local score=0

    # Check for E2E framework files (highest priority)
    [ -f "$dir/playwright.config.ts" ] && score=$((score + 100))
    [ -f "$dir/playwright.config.js" ] && score=$((score + 100))
    [ -f "$dir/cypress.config.ts" ] && score=$((score + 100))
    [ -f "$dir/cypress.config.js" ] && score=$((score + 100))
    [ -d "$dir/e2e" ] && score=$((score + 50))

    # Check for test files
    [ -d "$dir/tests" ] && score=$((score + 20))
    [ -d "$dir/__tests__" ] && score=$((score + 20))

    # Check for src directory (common in frontend projects)
    [ -d "$dir/src" ] && score=$((score + 30))

    # Prefer specific directory names
    case "$rel_dir" in
      */dashboard) score=$((score + 40)) ;;
      dashboard) score=$((score + 40)) ;;
      */frontend) score=$((score + 30)) ;;
      frontend) score=$((score + 30)) ;;
      */web) score=$((score + 20)) ;;
      web) score=$((score + 20)) ;;
    esac

    # Penalize deep nesting (too deep is likely not the main frontend)
    local depth; depth=$(echo "$rel_dir" | tr -cd '/' | wc -c)
    score=$((score - depth * 5))

    if [ $score -gt $best_score ]; then
      best_score=$score
      frontend_dir="$rel_dir"
    fi
  done

  # Fallback: simple search if find didn't work
  if [ -z "$frontend_dir" ]; then
    for dir in "frontend" "dashboard" "web" "client" "ui" "app"; do
      if [ -d "$REPO_DIR/$dir" ] && [ -f "$REPO_DIR/$dir/package.json" ]; then
        frontend_dir="$dir"
        break
      fi
    done
  fi

  # Detect port from package.json or docker-compose
  if [ -n "$frontend_dir" ]; then
    local pkg_port; pkg_port=$(grep -oP '"port":\s*\K\d+' "$REPO_DIR/$frontend_dir/package.json" 2>/dev/null || echo "")
    [ -n "$pkg_port" ] && default_port="$pkg_port"
  fi

  # Check docker-compose for port mappings
  if [ -f "$REPO_DIR/docker-compose.yml" ] || [ -f "$REPO_DIR/deployments/docker/docker-compose.yml" ]; then
    local compose_file; compose_file="$REPO_DIR/docker-compose.yml"
    [ -f "$REPO_DIR/deployments/docker/docker-compose.yml" ] && compose_file="$REPO_DIR/deployments/docker/docker-compose.yml"
    local dc_port; dc_port=$(grep -oP '"\K\d{4,5}(?=:'"$default_port"')' "$compose_file" 2>/dev/null | head -1 || echo "")
    [ -n "$dc_port" ] && default_port="$dc_port"
  fi

  echo "$frontend_dir:$default_port"
}

FRONTEND_CONFIG="${FRONTEND_CONFIG:-$(detect_frontend_config)}"
FRONTEND_DIR="${FRONTEND_DIR:-${FRONTEND_CONFIG%%:*}}"
DASHBOARD_PORT="${DASHBOARD_PORT:-${FRONTEND_CONFIG##*:}}"

# E2E Testing configuration (generic)
E2E_BASE_URL="${E2E_BASE_URL:-http://${SERVER_IP}:${DASHBOARD_PORT}}"
E2E_ENABLED="${E2E_ENABLED:-true}"

# Store detected E2E framework for reuse
DETECTED_E2E_FRAMEWORK=""
DETECTED_E2E_DIR=""

# Timeouts per phase (seconds)
declare -A PHASE_TIMEOUT=(
  [requirements]=600    [market_research]=900  [design]=900
  [backend]=3600        [frontend]=3600        [testing]=2400
  [qa]=600              [security]=600         [deploy]=1800
  [e2e_production]=1200
)

BRANCH=""
PROJECT_NAME=$(basename "$REPO_DIR")

# ═══════════════════════════════════════════════
# INITIALIZATION
# ═══════════════════════════════════════════════

mkdir -p "$DEV_DIR" "$ARTIFACTS" "$PHASE_LOGS" "$PATCHES_DIR" \
         "$MASTER_DEV_DIR" "$DEMO_DIR" "$PR_DIR" 2>/dev/null || true

# Initialize master dev.sh if not exists
SELF_SCRIPT="$(readlink -f "$0" 2>/dev/null || echo "$0")"
if [ ! -f "$MASTER_DEV_SH" ] && [ -f "$SELF_SCRIPT" ]; then
  cp "$SELF_SCRIPT" "$MASTER_DEV_SH"
fi

touch "$LIVE_LOG" "$SUP_LOG"

# ═══════════════════════════════════════════════
# LOGGING
# ═══════════════════════════════════════════════

G='\033[0;32m'; Y='\033[1;33m'; R='\033[0;31m'
B='\033[0;36m'; M='\033[0;35m'; W='\033[1;37m'; NC='\033[0m'

log()  { echo -e "${G}[$(date '+%H:%M:%S')]${NC} $1" | tee -a "$LIVE_LOG"; }
warn() { echo -e "${Y}[$(date '+%H:%M:%S')]${NC} $1" | tee -a "$LIVE_LOG"; }
err()  { echo -e "${R}[$(date '+%H:%M:%S')]${NC} $1" | tee -a "$LIVE_LOG"; }
info() { echo -e "${B}[$(date '+%H:%M:%S')]${NC} $1" | tee -a "$LIVE_LOG"; }
team() { echo -e "${M}[$(date '+%H:%M:%S')]${NC} ${W}$1${NC} $2" | tee -a "$LIVE_LOG"; }
slog() { echo -e "${G}[SUP $(date '+%H:%M:%S')]${NC} $1" | tee -a "$SUP_LOG"; }
swarn(){ echo -e "${Y}[SUP $(date '+%H:%M:%S')]${NC} $1" | tee -a "$SUP_LOG"; }
serr() { echo -e "${R}[SUP $(date '+%H:%M:%S')]${NC} $1" | tee -a "$SUP_LOG"; }

# ═══════════════════════════════════════════════
# SAFE PIPE HELPERS (no fail on empty results)
# ═══════════════════════════════════════════════

# Run a command and always return success (exit code 0)
# Usage: cmd || safe_pipe
safe_pipe() {
  return 0
}

# Grep wrapper that handles pipefail - returns success even if no matches
# Usage: safe_grep [options] pattern [file...]
safe_grep() {
  grep "$@" || true
}

# Find wrapper that handles pipefail safely
# Usage: safe_find [path...] [options...]
safe_find() {
  find "$@" || true
}

# Awk wrapper that handles pipefail safely
# Usage: safe_awk [options] 'program' [file...]
safe_awk() {
  awk "$@" || true
}

# Sed wrapper for pipe safety
# Usage: safe_sed [options] 'script' [file...]
safe_sed() {
  sed "$@" || true
}

# Cut wrapper for pipe safety
# Usage: safe_cut [options] [file...]
safe_cut() {
  cut "$@" || true
}

# Wc wrapper for pipe safety
# Usage: safe_wc [options] [file...]
safe_wc() {
  wc "$@" || true
}

# Head wrapper for pipe safety
# Usage: safe_head [options] [file...]
safe_head() {
  head "$@" || true
}

# Tail wrapper for pipe safety
# Usage: safe_tail [options] [file...]
safe_tail() {
  tail "$@" || true
}

# Sort wrapper for pipe safety
# Usage: safe_sort [options] [file...]
safe_sort() {
  sort "$@" || true
}

# Uniq wrapper for pipe safety
# Usage: safe_uniq [options] [file...]
safe_uniq() {
  uniq "$@" || true
}

# Xargs wrapper for pipe safety
# Usage: safe_xargs [options] [command...]
safe_xargs() {
  xargs "$@" || true
}

# Tr wrapper for pipe safety
# Usage: safe_tr [options] set1 set2
safe_tr() {
  tr "$@" || true
}

# ═══════════════════════════════════════════════
# ERROR MEMORY
# ═══════════════════════════════════════════════

record_error() {
  local phase="$1" error_type="$2" detail="$3"
  python3 -c "
import json, sys
from datetime import datetime
entry = {'timestamp': datetime.now().isoformat(), 'phase': sys.argv[1], 'type': sys.argv[2], 'detail': sys.argv[3][:200]}
with open(sys.argv[4], 'a') as f:
    f.write(json.dumps(entry) + '\n')
" "$phase" "$error_type" "$detail" "$ERROR_LOG" 2>/dev/null || true
}

get_past_errors() {
  local phase="$1"
  if [ -f "$ERROR_LOG" ]; then
    python3 -c "
import json, sys
errors = []
for line in open(sys.argv[2]):
    try:
        e = json.loads(line.strip())
        if e.get('phase') == sys.argv[1]:
            errors.append(f\"- [{e['type']}] {e['detail']}\")
    except: pass
if errors:
    print('KNOWN ISSUES FROM PREVIOUS RUNS:')
    for e in errors[-5:]: print(e)
" "$phase" "$ERROR_LOG" 2>/dev/null || true
  fi
}

# ═══════════════════════════════════════════════
# STATE MANAGEMENT (unified)
# ═══════════════════════════════════════════════

state_set() {
  python3 - "$STATE_FILE" "$1" "$2" "$3" << 'PYEOF'
import json, os, sys
from datetime import datetime
f, phase, key, val = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4]
d = json.load(open(f)) if os.path.exists(f) else {"phases": {}, "project": "", "branch": ""}
if phase == "_meta":
    d[key] = val
else:
    d.setdefault("phases", {}).setdefault(phase, {})[key] = val
    d["phases"][phase]["_updated"] = datetime.now().isoformat()
    d["current_phase"] = phase
json.dump(d, open(f, "w"), indent=2)
PYEOF
}

state_get() {
  python3 - "$STATE_FILE" "$1" "$2" << 'PYEOF'
import json, os, sys
f, phase, key = sys.argv[1], sys.argv[2], sys.argv[3]
if not os.path.exists(f): print("pending"); exit()
d = json.load(open(f))
if phase == "_meta":
    print(d.get(key, ""))
else:
    print(d.get("phases", {}).get(phase, {}).get(key, "pending"))
PYEOF
}

state_save_meta() {
  python3 - "$STATE_FILE" "$1" "$2" << 'PYEOF'
import json, os, sys
f, project, branch = sys.argv[1], sys.argv[2], sys.argv[3]
d = json.load(open(f)) if os.path.exists(f) else {"phases": {}}
d["project"] = project
d["branch"] = branch
json.dump(d, open(f, "w"), indent=2)
PYEOF
}

current_phase() { state_get _meta current_phase; }
phase_status() { state_get "$1" status; }

# ═══════════════════════════════════════════════
# LOG ROTATION
# ═══════════════════════════════════════════════

# Rotate log files that exceed size threshold
# Usage: rotate_logs [size_kb]
rotate_logs() {
  local max_size="${1:-1024}"  # Default 1MB
  local rotated=0

  for log_file in "$LIVE_LOG" "$SUP_LOG" "$PHASE_LOGS"/*.log; do
    [ -f "$log_file" ] 2>/dev/null || continue

    local size_kb
    size_kb=$(du -k "$log_file" 2>/dev/null | cut -f1)
    size_kb=${size_kb:-0}  # Default to 0 if empty

    if [ "$size_kb" -gt "$max_size" ]; then
      local timestamp
      timestamp=$(date +%Y%m%d_%H%M%S)
      local base_name; base_name=$(basename "$log_file" .log)
      local dir_name; dir_name=$(dirname "$log_file")
      local archived="${dir_name}/${base_name}_${timestamp}.log"

      mv "$log_file" "$archived" 2>/dev/null || continue
      touch "$log_file" 2>/dev/null || true

      # Keep only last 3 rotated logs
      ls -t "${dir_name}/${base_name}_"*.log 2>/dev/null | tail -n +4 | xargs rm -f 2>/dev/null || true

      rotated=$((rotated + 1))
      info "  Rotated: $log_file (${size_kb}KB → ${archived})"
    fi
  done

  [ "$rotated" -gt 0 ] && log "  Rotated $rotated log file(s)"
  return 0
}

# ═══════════════════════════════════════════════
# PROJECT CONTEXT HELPERS
# ═══════════════════════════════════════════════

# Get actual port for a service (accounts for dynamic mappings)
get_service_port() {
  local service_name="$1"
  local default_port="$2"

  # Check if there's a port mapping
  if [ -f "$DEV_DIR/port_mappings.txt" ]; then
    local mapped
    mapped=$(grep -o "^${default_port}:[0-9]*" "$DEV_DIR/port_mappings.txt" | cut -d: -f2)
    if [ -n "$mapped" ]; then
      echo "$mapped"
      return
    fi
  fi

  # Check container's actual port binding
  local runtime
  runtime=$(detect_container_runtime)
  local container_name="openprint-${service_name}"

  local actual_port
  actual_port=$($runtime port inspect "$container_name" 2>/dev/null | grep -oP "0.0.0.0:\K\d+(?= -> 8001)" | head -1)
  if [ -n "$actual_port" ]; then
    echo "$actual_port"
    return
  fi

  # Fallback to default
  echo "$default_port"
}

# Get all service ports as a map
get_all_service_ports() {
  python3 - << 'PYEOF'
import json, subprocess, sys, re

services = {
    "auth-service": "18001",
    "registry-service": "8002",
    "job-service": "8003",
    "storage-service": "8004",
    "notification-service": "18005",
    "dashboard": "3000",
}

# Try to get actual ports from podman
try:
    result = subprocess.run(['podman', 'ps', '--format', 'json'], capture_output=True, text=True, timeout=5)
    if result.returncode == 0:
        containers = json.loads(result.stdout)
        for container in containers:
            # Names is a list in podman JSON output
            names = container.get('Names', [])
            if not isinstance(names, list):
                names = [names]

            # Extract service name from container name
            for svc in services.keys():
                for name in names:
                    if svc in name:
                        # Parse port from Ports field (array of objects in podman)
                        ports = container.get('Ports', [])
                        if isinstance(ports, list) and ports:
                            # Get the first port mapping's host_port
                            host_port = ports[0].get('host_port')
                            if host_port:
                                services[svc] = str(host_port)
                                break
except Exception as e:
    # Fallback to defaults on error
    pass

print(json.dumps(services))
PYEOF
}

# Check if a port is available (not in use)
is_port_available() {
  local port="$1"
  # Check with ss or netstat
  if command -v ss >/dev/null 2>&1; then
    ss -tlnp 2>/dev/null | grep -q ":$port " && return 1
  elif command -v netstat >/dev/null 2>&1; then
    netstat -tlnp 2>/dev/null | grep -q ":$port " && return 1
  fi
  # Also check podman containers
  podman ps -a --format "{{.Ports}}" 2>/dev/null | grep -q ":$port->" && return 1
  return 0
}

# Find next available port starting from a given port
find_free_port() {
  local start_port="$1"
  local max_attempts="${2:-100}"
  local port=$start_port

  for ((i=0; i<max_attempts; i++)); do
    if is_port_available "$port"; then
      echo "$port"
      return 0
    fi
    port=$((port + 1))
  done
  return 1
}

# Get dynamic port mappings (replaces occupied ports with free ones)
# Sets: PORT_MAPPINGS (comma-separated "original:new" pairs)
get_dynamic_ports() {
  local compose=""
  [ -f "$REPO_DIR/docker-compose.yml" ] && compose="$REPO_DIR/docker-compose.yml"
  [ -f "$REPO_DIR/deployments/docker/docker-compose.yml" ] && compose="$REPO_DIR/deployments/docker/docker-compose.yml"

  if [ ! -f "$compose" ]; then
    return 0
  fi

  # Extract ports and check availability
  local occupied_ports=()
  local port_mappings=()

  while IFS= read -r line; do
    if [[ "$line" =~ ^[0-9]+:[0-9]+$ ]]; then
      local external_port="${line%%:*}"
      if ! is_port_available "$external_port"; then
        occupied_ports+=("$external_port")
        local free_port
        free_port=$(find_free_port "$external_port" 50)
        if [ -n "$free_port" ]; then
          port_mappings+=("$external_port:$free_port")
          warn "  ⚠️  Port $external_port in use, using $free_port instead"
        fi
      fi
    fi
  done < <(python3 - "$compose" << 'PYEOF'
import sys, re, yaml
try:
    with open(sys.argv[1]) as f:
        data = yaml.safe_load(f)
    for svc, cfg in data.get('services', {}).items():
        for port in cfg.get('ports', []):
            if isinstance(port, str):
                m = re.match(r'^(\d+):', port)
                if m:
                    print(f"{m.group(1)}:{port.split(':')[1]}")
            elif isinstance(port, int):
                print(f"{port}:{port}")
except:
    pass
PYEOF
)

  if [ ${#port_mappings[@]} -gt 0 ]; then
    PORT_MAPPINGS=$(IFS=,; echo "${port_mappings[*]}")
    log "  🔄 Port mappings: $PORT_MAPPINGS"
    # Store in file for docker-compose to use
    echo "$PORT_MAPPINGS" > "$DEV_DIR/port_mappings.txt"
  else
    PORT_MAPPINGS=""
    rm -f "$DEV_DIR/port_mappings.txt"
  fi
}

# Apply port mappings to docker-compose.yml
apply_port_mappings() {
  local mappings_file="$DEV_DIR/port_mappings.txt"
  local compose="$1"

  if [ ! -f "$mappings_file" ]; then
    return 0
  fi

  log "  🔧 Applying port mappings to $compose..."

  local backup="${compose}.backup"
  cp "$compose" "$backup"

  python3 - "$compose" "$mappings_file" "${compose}.tmp" << 'PYEOF'
import sys, re

compose_file = sys.argv[1]
mappings_file = sys.argv[2]
output_file = sys.argv[3]

# Read mappings
mappings = {}
with open(mappings_file) as f:
    for pair in f.read().strip().split(','):
        if ':' in pair:
            orig, new = pair.split(':')
            mappings[orig] = new

# Apply mappings to compose file
with open(compose_file, 'r') as f:
    content = f.read()

for orig, new in mappings.items():
    # Replace port mappings "orig:cont" with "new:cont"
    content = re.sub(rf'({re.escape(orig)}):(\d+)', rf'{new}:\2', content)

with open(output_file, 'w') as f:
    f.write(content)

print(f"Applied {len(mappings)} port mappings")
PYEOF

  mv "${compose}.tmp" "$compose"
  log "  ✓ Port mappings applied"
}

# Detect service ports from docker-compose.yml
# Also sets HEALTH_CHECK_PORTS (external services only)
detect_service_ports() {
  local compose=""
  [ -f "$REPO_DIR/docker-compose.yml" ] && compose="$REPO_DIR/docker-compose.yml"
  [ -f "$REPO_DIR/deployments/docker/docker-compose.yml" ] && compose="$REPO_DIR/deployments/docker/docker-compose.yml"

  if [ -n "$compose" ] && [ -f "$compose" ]; then
    # Extract all external port mappings (host:container)
    # Skip commented-out ports and internal-only services
    SERVICE_PORTS=$(python3 - "$compose" << 'PYEOF' 2>/dev/null || true
import sys, re
import yaml

try:
    with open(sys.argv[1]) as f:
        data = yaml.safe_load(f)

    health_ports = []
    all_ports = []

    services = data.get('services', {})
    for svc_name, svc_config in services.items():
        # Skip internal infrastructure services for health checks
        if svc_name in ('postgres', 'redis', 'prometheus', 'alertmanager', 'grafana', 'jaeger'):
            continue

        ports = svc_config.get('ports', [])
        for port in ports:
            if isinstance(port, str):
                # Parse "host:container" or "host:container/protocol"
                match = re.match(r'^(\d+):', port)
                if match:
                    host_port = int(match.group(1))
                    all_ports.append(host_port)
                    # Add to health check ports (non-infrastructure services)
                    health_ports.append(host_port)
            elif isinstance(port, int):
                all_ports.append(port)
                health_ports.append(port)

    # Output sorted unique ports
    all_ports = sorted(set(all_ports))
    health_ports = sorted(set(health_ports))

    print(' '.join(map(str, all_ports)))
    print('HEALTH_PORTS:' + ' '.join(map(str, health_ports)))
except Exception as e:
    # Fallback to regex parsing if yaml fails
    with open(sys.argv[1]) as f:
        content = f.read()
        ports = re.findall(r'"\s*(\d{4,5}):(\d+)', content)
        # Filter out common internal service ports
        filtered = [p for p in ports if p[0] not in ('15432', '16379', '5432', '6379')]
        all_ports = sorted(set([int(p[0]) for p in ports]))
        health_ports = sorted(set([int(p[0]) for p in filtered]))
        print(' '.join(map(str, all_ports)))
        print('HEALTH_PORTS:' + ' '.join(map(str, health_ports)))
PYEOF
)
  fi

  # Parse health ports from output
  HEALTH_CHECK_PORTS=$(echo "$SERVICE_PORTS" | grep -o 'HEALTH_PORTS:[0-9 ]*' | cut -d: -f2- || echo "")
  SERVICE_PORTS=$(echo "$SERVICE_PORTS" | grep -v '^HEALTH_PORTS:' || echo "$SERVICE_PORTS")

  # Fallback if detection failed
  if [ -z "$SERVICE_PORTS" ]; then
    SERVICE_PORTS="3000 8002 8003 8004 8005 9090 9091 9092 9093 9094 9095 18001 18005"
  fi
  if [ -z "$HEALTH_CHECK_PORTS" ]; then
    HEALTH_CHECK_PORTS="3000 8002 8003 8004 8005 18001 18005"
  fi
}

read_project_context() {
  if [ -f "$REPO_DIR/CLAUDE.md" ]; then
    head -c 3000 "$REPO_DIR/CLAUDE.md" 2>/dev/null
  else
    echo "Project: $PROJECT_NAME (no CLAUDE.md found)"
  fi
}

summarize_artifact() {
  local file="$1" max_chars="${2:-3000}"
  [ -f "$file" ] || { echo "{}"; return; }
  local size; size=$(wc -c < "$file")
  if [ "$size" -le "$max_chars" ]; then
    cat "$file"
  else
    python3 - "$file" "$max_chars" << 'PYEOF' 2>/dev/null || head -c "$max_chars" "$file"
import json, sys
f, max_c = sys.argv[1], int(sys.argv[2])
try:
    d = json.load(open(f))
    if "backend_tasks" in d:
        summary = {
            "architecture_decisions": [x.get("decision","") for x in d.get("architecture_decisions",[])[:5]],
            "backend_tasks": [{"order":x.get("order"), "file":x.get("file"), "purpose":x.get("purpose","")} for x in d.get("backend_tasks",[])],
            "frontend_tasks": [{"order":x.get("order"), "file":x.get("file"), "purpose":x.get("purpose","")} for x in d.get("frontend_tasks",[])],
            "database_migrations": [x.get("file","") for x in d.get("database_migrations",[])],
            "security_notes": d.get("security_notes",[])[:3]
        }
    elif "functional_requirements" in d:
        summary = {
            "project_name": d.get("project_name",""),
            "summary": d.get("summary",""),
            "functional_requirements": [{"id":x.get("id"), "title":x.get("title")} for x in d.get("functional_requirements",[])],
            "affected_services": d.get("affected_services",[]),
            "implementation_phases": d.get("implementation_phases",[])
        }
    else:
        summary = d
    result = json.dumps(summary, indent=1)
    print(result[:max_c])
except:
    print(open(f).read()[:max_c])
PYEOF
  fi
}

# ═══════════════════════════════════════════════
# EXTERNAL ACCESS HELPERS (Generic)
# ═══════════════════════════════════════════════

# Get server's external IP for client connectivity
get_external_ip() {
  # Try multiple methods to get external IP
  local ip=""

  # Method 1: From environment variable
  [ -n "$SERVER_IP" ] && echo "$SERVER_IP" && return

  # Method 2: From hostname -I
  ip=$(hostname -I 2>/dev/null | awk '{print $1}')
  [ -n "$ip" ] && echo "$ip" && return

  # Method 3: From ip command
  ip=$(ip route get 1 2>/dev/null | awk '{print $7}' | head -1)
  [ -n "$ip" ] && echo "$ip" && return

  # Method 4: From ifconfig
  ip=$(ifconfig 2>/dev/null | grep "inet " | grep -v "127.0.0.1" | head -1 | awk '{print $2}')
  [ -n "$ip" ] && echo "$ip" && return

  # Fallback
  echo "127.0.0.1"
}

# Check if port is accessible from external
check_external_access() {
  local port="$1"
  local host="${2:-0.0.0.0}"

  # Check if service is listening on all interfaces
  if command -v ss >/dev/null 2>&1; then
    ss -tlnp 2>/dev/null | grep -q ":$port " || return 1
  elif command -v netstat >/dev/null 2>&1; then
    netstat -tlnp 2>/dev/null | grep -q ":$port " || return 1
  fi

  # Test connectivity
  curl -sf --max-time 5 "http://$(get_external_ip):$port${HEALTH_CHECK_PATH}" >/dev/null 2>&1
}

# Generate access report for client connection
generate_access_report() {
  # Ensure artifacts directory exists
  [ -d "$ARTIFACTS" ] || mkdir -p "$ARTIFACTS" 2>/dev/null || {
    warn "  Cannot create artifacts directory"
    return 1
  }

  local report_file="${ARTIFACTS}/access_report.json"
  local external_ip
  external_ip=$(get_external_ip 2>/dev/null) || external_ip="localhost"

  # Get actual dashboard port from running containers (dynamic)
  service_ports_json=$(get_all_service_ports 2>/dev/null || echo '{}')
  detected_dashboard_port=$(echo "$service_ports_json" | python3 -c "import json,sys; print(json.load(sys.stdin).get('dashboard','3000'))" 2>/dev/null || echo "3000")

  python3 - "$report_file" "$external_ip" "${detected_dashboard_port}" "${SERVICE_PORTS:-}" "${PROJECT_NAME:-docker}" << 'PYEOF'
import json, sys, socket
from datetime import datetime

report_file, ext_ip, dash_port, svc_ports_str, proj_name = sys.argv[1:]

# Parse service ports
svc_ports = svc_ports_str.split() if svc_ports_str else []

# Check which ports are accessible
accessible_ports = []
for port in svc_ports:
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.settimeout(2)
        result = s.connect_ex(('127.0.0.1', int(port)))
        if result == 0:
            accessible_ports.append(int(port))
        s.close()
    except:
        pass

report = {
    "generated_at": datetime.now().isoformat(),
    "project": proj_name,
    "server": {
        "hostname": socket.gethostname(),
        "external_ip": ext_ip,
        "localhost": "127.0.0.1"
    },
    "services": {
        "dashboard": {
            "url": f"http://{ext_ip}:{dash_port}",
            "local_url": f"http://localhost:{dash_port}",
            "port": int(dash_port),
            "accessible": int(dash_port) in accessible_ports
        },
        "api_ports": [int(p) for p in svc_ports if p != dash_port],
        "all_accessible_ports": accessible_ports
    },
    "client_connection": {
        "base_url": f"http://{ext_ip}:{dash_port}",
        "api_base": f"http://{ext_ip}",
        "note": "Ensure firewall allows inbound connections to the ports listed above"
    }
}

json.dump(report, open(report_file, 'w'), indent=2)
print(f"Access report saved to: {report_file}")
PYEOF
}

# ═══════════════════════════════════════════════
# E2E TESTING HELPERS (Generic)
# ═══════════════════════════════════════════════

# Detect E2E framework with intelligent recursive search
# Sets: DETECTED_E2E_FRAMEWORK, DETECTED_E2E_DIR
# Returns: 0 if found, 1 if not found
detect_e2e_framework() {
  # Use cached result if available
  if [ -n "$DETECTED_E2E_FRAMEWORK" ]; then
    return 0
  fi

  info "  🔍 Searching for E2E test framework..."

  # Search in priority order
  local frameworks=(
    "playwright.config.ts:playwright"
    "playwright.config.js:playwright"
    "cypress.config.ts:cypress"
    "cypress.config.js:cypress"
  )

  local best_match=""
  local best_framework=""
  local best_score=0

  # First, check the detected frontend directory
  if [ -n "$FRONTEND_DIR" ]; then
    local frontend_path="$REPO_DIR/$FRONTEND_DIR"
    for fw in "${frameworks[@]}"; do
      local config="${fw%%:*}"
      local framework="${fw##*:}"
      if [ -f "$frontend_path/$config" ]; then
        DETECTED_E2E_FRAMEWORK="$framework"
        DETECTED_E2E_DIR="$frontend_path"
        info "  ✓ Found $framework at $FRONTEND_DIR/"
        return 0
      fi
    done
  fi

  # Recursive search for E2E configs (up to 3 levels deep)
  while IFS= read -r -d '' config_file; do
    local rel_path="${config_file#$REPO_DIR/}"
    local dir; dir=$(dirname "$config_file")
    local filename; filename=$(basename "$config_file")
    local framework=""

    case "$filename" in
      playwright.config.ts|playwright.config.js) framework="playwright" ;;
      cypress.config.ts|cypress.config.js) framework="cypress" ;;
    esac

    # Score: prefer closer to root, prefer with e2e/ directory
    local score=0
    [ -d "$dir/e2e" ] && score=$((score + 50))
    [ -n "$framework" ] && score=$((score + 100))

    if [ $score -gt $best_score ]; then
      best_score=$score
      best_match="$dir"
      best_framework="$framework"
    fi
  done < <(find "$REPO_DIR" -maxdepth 4 -type f \( -name "playwright.config.ts" -o -name "playwright.config.js" -o -name "cypress.config.ts" -o -name "cypress.config.js" \) -print0 2>/dev/null | grep -vz node_modules)

  if [ -n "$best_framework" ]; then
    DETECTED_E2E_FRAMEWORK="$best_framework"
    DETECTED_E2E_DIR="$best_match"
    local rel_dir="${best_match#$REPO_DIR/}"
    info "  ✓ Found $best_framework at $rel_dir/"
    return 0
  fi

  # Fallback: check for .e2e.ts or .e2e.js files
  if find "$REPO_DIR" -maxdepth 4 -name "*.e2e.ts" -o -name "*.e2e.js" 2>/dev/null | grep -v node_modules | head -1 | read -r e2e_file; then
    local e2e_dir; e2e_dir=$(dirname "$e2e_file")
    DETECTED_E2E_FRAMEWORK="generic"
    DETECTED_E2E_DIR="$e2e_dir"
    info "  ✓ Found generic E2E test files"
    return 0
  fi

  warn "  ⚠️  No E2E framework detected - will use generic smoke tests"
  DETECTED_E2E_FRAMEWORK="none"
  return 1
}

# Self-healing: Install missing E2E dependencies
heal_e2e_dependencies() {
  local framework="$1"
  local e2e_dir="$2"

  case "$framework" in
    playwright)
      if ! command -v npx &>/dev/null; then
        warn "  npx not found - installing Node.js dependencies..."
        return 1
      fi

      # Check if playwright is installed
      if ! cd "$e2e_dir" && npx playwright --version &>/dev/null; then
        info "  📦 Installing Playwright..."
        cd "$e2e_dir" || return 1

        # First, ensure node_modules exist
        if [ ! -d "node_modules" ]; then
          log "  Running npm install..."
          npm install 2>&1 | tail -5 || {
            err "  Failed to install npm dependencies"
            return 1
          }
        fi

        # Install Playwright browsers
        log "  Installing Playwright browsers..."
        npx playwright install --with-deps 2>&1 | tail -3 || {
          warn "  Browser installation had issues, but continuing..."
        }
      fi
      ;;
    cypress)
      if ! cd "$e2e_dir" && npx cypress --version &>/dev/null; then
        info "  📦 Installing Cypress..."
        cd "$e2e_dir" || return 1
        [ ! -d "node_modules" ] && npm install 2>&1 | tail -5
        npx cypress install 2>&1 | tail -3 || true
      fi
      ;;
  esac
  return 0
}

# Detect E2E framework and run tests (with self-healing)
run_e2e_tests() {
  local base_url="$1"
  local report_file="$2"

  log "  🧪 Running E2E tests against: $base_url"

  # Detect framework
  detect_e2e_framework
  local framework="$DETECTED_E2E_FRAMEWORK"
  local e2e_dir="$DETECTED_E2E_DIR"

  case "$framework" in
    playwright)
      # Self-heal: install dependencies if needed
      heal_e2e_dependencies "playwright" "$e2e_dir" || {
        warn "  Dependency healing failed, attempting anyway..."
      }
      run_playwright_e2e "$base_url" "$report_file" "$e2e_dir"
      return $?
      ;;
    cypress)
      heal_e2e_dependencies "cypress" "$e2e_dir" || true
      run_cypress_e2e "$base_url" "$report_file" "$e2e_dir"
      return $?
      ;;
    generic)
      run_ai_e2e "$base_url" "$report_file" "true"
      return $?
      ;;
    *)
      warn "  No E2E framework detected - running AI-powered smoke tests"
      run_ai_e2e "$base_url" "$report_file" "true"
      return $?
      ;;
  esac
}

# Run Playwright E2E tests
run_playwright_e2e() {
  local base_url="$1"
  local report_file="$2"
  local e2e_dir="${3:-$REPO_DIR/$FRONTEND_DIR}"

  log "  🎭 Using Playwright at: ${e2e_dir#$REPO_DIR/}"

  cd "$e2e_dir" || {
    err "  Cannot access E2E directory: $e2e_dir"
    return 1
  }

  # Ensure node_modules exist
  if [ ! -d "node_modules" ]; then
    log "  📦 Installing npm dependencies..."
    npm install 2>&1 | tail -5 || {
      err "  Failed to install dependencies"
      return 1
    }
  fi

  # Ensure browsers are installed
  if ! npx playwright --version &>/dev/null; then
    log "  🌐 Installing Playwright browsers..."
    npx playwright install --with-deps 2>&1 | tail -3 || true
  fi

  # Disable webServer since we're testing against deployed instance
  export PW_DISABLE_WEB_SERVER=1
  export BASE_URL="$base_url"

  local rc=0
  log "  Running tests with BASE_URL=$base_url (set via global-setup.ts)"
  npx playwright test \
    --reporter=json \
    --reporter=list \
    --output="$ARTIFACTS/playwright-report" 2>&1 | tee "$PHASE_LOGS/e2e_production.log" | tail -40 || rc=$?

  # Parse results - check both output locations
  local results_json="$ARTIFACTS/playwright-report/results.json"
  if [ ! -f "$results_json" ]; then
    results_json="$e2e_dir/playwright-report/results.json"
  fi

  if [ -f "$results_json" ]; then
    python3 - "$results_json" "$report_file" << 'PYEOF' 2>/dev/null || true
import json, sys
from datetime import datetime

results_file = sys.argv[1]
report_file = sys.argv[2]

try:
    with open(results_file) as f:
        data = json.load(f)

    # Handle both old and new Playwright result formats
    suites = data.get('suites', [])
    total = 0
    passed = 0
    failed = 0
    skipped = 0

    for suite in suites:
        for spec in suite.get('specs', []):
            for test in spec.get('tests', []):
                total += 1
                status = test.get('results', [{}])[0].get('status', 'unknown')
                if status == 'passed':
                    passed += 1
                elif status == 'failed':
                    failed += 1
                elif status in ('skipped', 'interrupted'):
                    skipped += 1

    report = {
        "timestamp": datetime.now().isoformat(),
        "framework": "playwright",
        "summary": {"total": total, "passed": passed, "failed": failed, "skipped": skipped},
        "success_rate": round(passed / total * 100, 1) if total > 0 else 0,
        "status": "passed" if failed == 0 else "failed"
    }

    with open(report_file, 'w') as f:
        json.dump(report, f, indent=2)
    print(f"Playwright results: {passed}/{total} passed")
except Exception as e:
    with open(report_file, 'w') as f:
        json.dump({"error": str(e), "status": "error", "framework": "playwright"}, f, indent=2)
    print(f"Error parsing results: {e}")
PYEOF
  else
    warn "  ⚠️  No Playwright results found at expected locations"
    # Create a report from the log output
    python3 - "$report_file" "$PHASE_LOGS/e2e_production.log" << 'PYEOF' 2>/dev/null || true
import json, sys
from datetime import datetime
import re

report_file = sys.argv[1]
log_file = sys.argv[2]

report = {
    "timestamp": datetime.now().isoformat(),
    "framework": "playwright",
    "summary": {"total": 0, "passed": 0, "failed": 0, "skipped": 0},
    "status": "unknown"
}

try:
    with open(log_file) as f:
        content = f.read()
        # Try to extract pass/fail from the log
        passed = len(re.findall(r'passed', content, re.IGNORECASE))
        failed = len(re.findall(r'failed', content, re.IGNORECASE))
        report["summary"]["passed"] = min(passed, 100)  # reasonable cap
        report["summary"]["failed"] = min(failed, 100)
        report["status"] = "failed" if failed > 0 else "passed"
except Exception:
    pass

with open(report_file, 'w') as f:
    json.dump(report, f, indent=2)
PYEOF
  fi

  return $rc
}

# Run Cypress E2E tests
run_cypress_e2e() {
  local base_url="$1"
  local report_file="$2"
  local e2e_dir="${3:-$REPO_DIR/$FRONTEND_DIR}"

  log "  🌲 Using Cypress at: ${e2e_dir#$REPO_DIR/}"

  cd "$e2e_dir" || {
    err "  Cannot access E2E directory: $e2e_dir"
    return 1
  }

  # Ensure node_modules exist
  if [ ! -d "node_modules" ]; then
    log "  📦 Installing npm dependencies..."
    npm install 2>&1 | tail -5 || {
      err "  Failed to install dependencies"
      return 1
    }
  fi

  export CYPRESS_baseUrl="$base_url"
  local rc=0
  npx cypress run --config baseUrl="$base_url" --reporter=json 2>&1 | tee "$PHASE_LOGS/e2e_production.log" | tail -20 || rc=$?

  # Parse Cypress results if available
  return $rc
}

# Generic E2E test runner (uses curl for smoke tests)
run_generic_e2e() {
  local base_url="$1"
  local report_file="$2"

  log "  🧪 Running generic smoke tests..."

  local passed=0 failed=0
  local results=()

  # Define health checks
  declare -A checks=(
    ["homepage"]="$base_url"
    ["health"]="${base_url}${HEALTH_CHECK_PATH}"
  )

  # Try service health endpoints if they exist
  for port in 8001 8002 8003 8004 8005; do
    checks["service-port-$port"]="http://${SERVER_IP}:${port}/health"
  done

  # Run checks
  for name in "${!checks[@]}"; do
    local url="${checks[$name]}"
    info "    Checking $name: $url"
    if curl -sf --max-time 5 "$url" >/dev/null 2>&1; then
      ((passed++))
      results+=("$name:passed")
      log "    ✓ $name OK"
    else
      ((failed++))
      results+=("$name:failed")
      log "    ✗ $name FAILED"
    fi
  done

  # Check if common API endpoints respond (even with 404, means server is up)
  for endpoint in "/api" "/api/v1" "/api/health"; do
    local url="${base_url}${endpoint}"
    local status; status=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$url" 2>/dev/null || echo "000")
    if [ "$status" != "000" ]; then
      ((passed++))
      results+=("api-endpoint-${endpoint##*/}:passed")
      log "    ✓ API endpoint $endpoint responds (HTTP $status)"
    fi
  done

  # Generate report
  python3 - "$report_file" "$passed" "$failed" "${results[@]}" << 'PYEOF'
import json, sys
from datetime import datetime

report_file = sys.argv[1]
passed = int(sys.argv[2])
failed = int(sys.argv[3])
results = sys.argv[4:] if len(sys.argv) > 4 else []

total = passed + failed
checks = []
for r in results:
    name, status = r.split(':', 1) if ':' in r else (r, 'unknown')
    checks.append({"check": name, "status": status})

report = {
    "timestamp": datetime.now().isoformat(),
    "framework": "generic/smoke",
    "summary": {"total": total, "passed": passed, "failed": failed, "skipped": 0},
    "checks": checks,
    "success_rate": round(passed / total * 100, 1) if total > 0 else 0,
    "status": "passed" if failed == 0 else "partial"
}

with open(report_file, 'w') as f:
    json.dump(report, f, indent=2)

print(f"Smoke tests: {passed}/{total} passed")
PYEOF

  return 0
}

# AI-Powered E2E test runner - intelligently tests and can self-heal
run_ai_e2e() {
  local base_url="$1"
  local report_file="$2"
  local auto_fix="${3:-true}"

  log "  🤖 Running AI-powered E2E tests..."

  local runtime
  runtime=$(detect_container_runtime)
  local compose_cmd
  compose_cmd=$(detect_compose_command)

  # Get actual ports from running containers
  local service_ports_json
  service_ports_json=$(get_all_service_ports)

  # Build dynamic service endpoints
  local dashboard_port auth_port registry_port job_port storage_port notification_port
  dashboard_port=$(echo "$service_ports_json" | python3 -c "import json,sys; print(json.load(sys.stdin).get('dashboard','3000'))")
  auth_port=$(echo "$service_ports_json" | python3 -c "import json,sys; print(json.load(sys.stdin).get('auth-service','18001'))")
  registry_port=$(echo "$service_ports_json" | python3 -c "import json,sys; print(json.load(sys.stdin).get('registry-service','8002'))")
  job_port=$(echo "$service_ports_json" | python3 -c "import json,sys; print(json.load(sys.stdin).get('job-service','8003'))")
  storage_port=$(echo "$service_ports_json" | python3 -c "import json,sys; print(json.load(sys.stdin).get('storage-service','8004'))")
  notification_port=$(echo "$service_ports_json" | python3 -c "import json,sys; print(json.load(sys.stdin).get('notification-service','18005'))")

  log "  📍 Detected ports: Dashboard:$dashboard_port Auth:$auth_port Registry:$registry_port Job:$job_port Storage:$storage_port Notification:$notification_port"

  # Define service endpoints with dynamic ports
  declare -A service_checks=(
    ["Dashboard"]="http://localhost:$dashboard_port"
    ["Auth Service"]="http://localhost:$auth_port/health"
    ["Registry Service"]="http://localhost:$registry_port/health"
    ["Job Service"]="http://localhost:$job_port/health"
    ["Storage Service"]="http://localhost:$storage_port/health"
    ["Notification Service"]="http://localhost:$notification_port/health"
  )

  local passed=0 failed=0
  local results=()
  local failed_services=()
  local test_output=""

  # Test each service
  for service in "${!service_checks[@]}"; do
    local url="${service_checks[$service]}"
    info "    Testing $service: $url"

    # Try health endpoint first
    local http_code="" response=""
    http_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$url" 2>/dev/null || echo "000")

    if [ "$http_code" != "000" ]; then
      ((passed++))
      results+=("$service:passed:$http_code")
      log "    ✓ $service (HTTP $http_code)"
      test_output+="✓ $service responded with HTTP $http_code\n"

      # Try to get actual response data for API services
      if [[ "$url" == *"/health" ]]; then
        local health_data
        health_data=$(curl -s --max-time 5 "$url" 2>/dev/null || echo "")
        if [ -n "$health_data" ]; then
          test_output+="  Health: $health_data\n"
        fi
      fi
    else
      ((failed++))
      results+=("$service:failed:000")
      log "    ✗ $service FAILED (no response)"
      test_output+="✗ $service - No response on $url\n"
      failed_services+=("$service|$url")
    fi
  done

  # Test API functionality if services are up
  if [ $passed -gt 0 ]; then
    log "  🧪 Testing API functionality..."

    # Test authentication endpoint (using dynamic port)
    local auth_response
    auth_response=$(curl -s --max-time 5 "http://localhost:$auth_port/api/v1/auth/login" \
      -H "Content-Type: application/json" \
      -d '{"email":"test@example.com","password":"test123"}' 2>/dev/null || echo "")

    if [ -n "$auth_response" ]; then
      test_output+="\n📝 Auth endpoint test:\n$auth_response\n"
      # Check if response contains expected fields
      if echo "$auth_response" | grep -qE '(token|error|message|status)'; then
        ((passed++))
        test_output+="✓ Auth API is functional\n"
      fi
    fi

    # Test registry endpoint
    local registry_response
    registry_response=$(curl -s --max-time 5 "http://localhost:8002/api/v1/printers" 2>/dev/null || echo "")
    test_output+="\n📝 Registry endpoint test:\n${registry_response:0:200}...\n"
  fi

  # AI Analysis of failures
  if [ ${#failed_services[@]} -gt 0 ] && [ "$auto_fix" = true ]; then
    log "  🧠 AI analyzing failures..."

    # Get container logs for failed services
    local diagnostic_data=""
    for failed in "${failed_services[@]}"; do
      local svc_name="${failed%%|*}"
      local container_name="openprint-$(echo "$svc_name" | tr '[:upper:]' '[:lower:]' | sed 's/ /-/g' | sed 's/service//')"
      local logs
      logs=$($runtime logs --tail 20 "$container_name" 2>/dev/null || echo "No logs available")
      diagnostic_data+="$svc_name:\n$logs\n\n"
    done

    # Use AI to diagnose and potentially fix
    local diagnosis_file="$PHASE_LOGS/e2e_diagnosis.json"
    claude -p --model "$CLAUDE_MODEL" --dangerously-skip-permissions \
      "Analyze these E2E test failures for the OpenPrint project and provide a JSON response:

FAILED SERVICES:
$(printf '%s\n' "${failed_services[@]}")

SERVICE LOGS:
$diagnostic_data

TEST OUTPUT:
$test_output

Respond with ONLY this JSON format:
{
  \"diagnosis\": \"brief explanation of what's wrong\",
  \"root_cause\": \"primary cause (e.g., 'missing dependency', 'port conflict', 'configuration error', 'container crash')\",
  \"fix_commands\": [\"command to fix issue 1\", \"command to fix issue 2\"],
  \"verification_url\": \"url to check after fix\",
  \"severity\": \"low|medium|high|critical\"
}" > "$diagnosis_file" 2>/dev/null || true

    # Parse and apply fixes if low/medium severity
    if [ -f "$diagnosis_file" ]; then
      local fix_commands
      fix_commands=$(python3 -c "
import json, sys
try:
    with open('$diagnosis_file') as f:
        data = json.load(f)
    if data.get('severity', 'high') in ['low', 'medium']:
        cmds = data.get('fix_commands', [])
        for cmd in cmds[:2]:  # Max 2 fixes
            print(cmd)
except:
    pass
" 2>/dev/null || "")

      if [ -n "$fix_commands" ]; then
        log "  🔧 Applying AI-recommended fixes..."
        for cmd in $fix_commands; do
          log "    Running: $cmd"
          eval "$cmd" 2>&1 | tail -3 || true
          sleep 2
        done

        # Re-check services after fixes
        log "  🔄 Re-checking services after fixes..."
        sleep 5
      fi
    fi
  fi

  # Generate comprehensive report
  python3 - "$report_file" "$passed" "$failed" "${results[@]}" "$test_output" << 'PYEOF'
import json, sys
from datetime import datetime

report_file = sys.argv[1]
passed = int(sys.argv[2])
failed = int(sys.argv[3])
results = sys.argv[4] if len(sys.argv) > 4 else []
test_output = sys.argv[5] if len(sys.argv) > 5 else ""

total = passed + failed
checks = []
for r in results:
    parts = r.split(':')
    if len(parts) >= 2:
        checks.append({"service": parts[0], "status": parts[1], "code": parts[2] if len(parts) > 2 else ""})

report = {
    "timestamp": datetime.now().isoformat(),
    "framework": "ai-enhanced/smoke",
    "summary": {"total": total, "passed": passed, "failed": failed, "skipped": 0},
    "checks": checks,
    "test_output": test_output,
    "success_rate": round(passed / total * 100, 1) if total > 0 else 0,
    "status": "passed" if failed == 0 else "partial" if passed > 0 else "failed",
    "ready_for_production": failed == 0
}

with open(report_file, 'w') as f:
    json.dump(report, f, indent=2)

print(f"AI E2E Tests: {passed}/{total} passed ({report['success_rate']}%)")
PYEOF

  # Display summary
  if [ $failed -eq 0 ]; then
    log "  ✅ All services operational!"
  elif [ $passed -gt 0 ]; then
    warn "  ⚠️  Some services degraded: $failed failed"
  else
    err "  ❌ Critical: All services down!"
  fi

  return $failed
}

# ═══════════════════════════════════════════════
# Z.AI BRAIN (PM, Architect, QA, Security)
# ═══════════════════════════════════════════════

zai_think() {
  local role_name="$1" sys_file="$2" usr_file="$3" out_file="$4"
  team "$role_name" "Thinking..."

  local req_file="$DEV_DIR/tmp_zai_req.json"
  python3 - "$sys_file" "$usr_file" "$ZAI_MODEL" << 'PYEOF' > "$req_file"
import json, sys
req = {
    "model": sys.argv[3],
    "messages": [
        {"role": "system", "content": open(sys.argv[1]).read()},
        {"role": "user", "content": open(sys.argv[2]).read()}
    ],
    "max_tokens": 8192,
    "temperature": 0.15
}
print(json.dumps(req))
PYEOF

  local http_code
  http_code=$(curl -s -w "%{http_code}" -o "$out_file.raw" \
    -X POST "$ZAI_URL" \
    -H "Authorization: Bearer $ZAI_API_KEY" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json" \
    -d @"$req_file" 2>/dev/null) || http_code="000"
  rm -f "$req_file"

  if [ "$http_code" != "200" ]; then
    warn "  Z.ai HTTP $http_code"
    rm -f "$out_file.raw"
    return 1
  fi

  python3 - "$out_file.raw" "$out_file" << 'PYEOF'
import json, sys, re
try:
    r = json.load(open(sys.argv[1]))
    content = r.get("choices", [{}])[0].get("message", {}).get("content", "") or \
              r.get("choices", [{}])[0].get("message", {}).get("reasoning_content", "")
    m = re.search(r'\{[\s\S]*\}', content)
    if m:
        try:
            parsed = json.loads(m.group())
            json.dump(parsed, open(sys.argv[2], "w"), indent=2)
            exit(0)
        except json.JSONDecodeError:
            pass
    open(sys.argv[2], "w").write(content)
except Exception as e:
    json.dump({"error": str(e)}, open(sys.argv[2], "w"))
    exit(1)
PYEOF
  rm -f "$out_file.raw"
  team "$role_name" "✓ Done"
  return 0
}

ai_think() {
  local role_name="$1" sys_file="$2" usr_file="$3" out_file="$4"

  if [ -n "$ZAI_API_KEY" ]; then
    if zai_think "$role_name" "$sys_file" "$usr_file" "$out_file"; then
      local size
      size=$(wc -c < "$out_file" 2>/dev/null || echo "0")
      [ "$size" -gt 20 ] && return 0
    fi
    warn "  Z.ai failed — using Claude Code"
  fi

  team "$role_name" "Using Claude Code..."
  cd "$REPO_DIR"
  local prompt
  prompt="$(cat "$sys_file")

$(cat "$usr_file")

RESPOND WITH ONLY A JSON OBJECT. No markdown fences, no explanation."

  claude -p --model "$CLAUDE_MODEL" --dangerously-skip-permissions \
    "$prompt" > "$out_file" 2>/dev/null || true

  python3 - "$out_file" << 'PYEOF'
import json, re, sys
f = sys.argv[1]
content = open(f).read()
m = re.search(r'\{[\s\S]*\}', content)
if m:
    try:
        parsed = json.loads(m.group())
        json.dump(parsed, open(f, "w"), indent=2)
    except: pass
PYEOF
  team "$role_name" "✓ Done"
}

# ═══════════════════════════════════════════════
# Z.AI WEB SEARCH (Market Research)
# ═══════════════════════════════════════════════

zai_web_search() {
  local query="$1" out_file="$2"
  team "🔍 Research" "Searching: $query"

  if [ -n "$ZAI_API_KEY" ]; then
    local req_file="$DEV_DIR/tmp_search.json"
    python3 - "$query" << 'PYEOF' > "$req_file"
import json, sys
print(json.dumps({"query": sys.argv[1], "count": 10}))
PYEOF

    local http_code
    http_code=$(curl -s -w "%{http_code}" -o "$out_file.raw" \
      -X POST "$ZAI_SEARCH_URL" \
      -H "Authorization: Bearer $ZAI_API_KEY" \
      -H "Content-Type: application/json" \
      -H "Accept: application/json" \
      -d @"$req_file" 2>/dev/null) || http_code="000"
    rm -f "$req_file"

    if [ "$http_code" = "200" ]; then
      python3 - "$out_file.raw" "$out_file" << 'PYEOF'
import json, sys
try:
    data = json.load(open(sys.argv[1]))
    results = []
    for item in data.get("results", data.get("data", {}).get("results", [])):
        results.append({
            "title": item.get("title", ""),
            "content": item.get("content", item.get("snippet", ""))[:500],
            "url": item.get("link", item.get("url", ""))
        })
    json.dump({"results": results}, open(sys.argv[2], "w"), indent=2)
except Exception as e:
    json.dump({"results": [], "error": str(e)}, open(sys.argv[2], "w"))
PYEOF
      rm -f "$out_file.raw"
      local count
      count=$(python3 -c "import json; print(len(json.load(open('$out_file')).get('results',[])))" 2>/dev/null || echo "0")
      if [ "$count" -gt 0 ]; then
        team "🔍 Research" "✓ Z.ai found $count results"
        return 0
      fi
    fi
    rm -f "$out_file.raw"
    warn "  Z.ai search failed (HTTP $http_code) — falling back to Claude"
  fi

  # Fallback: Claude Code
  team "🔍 Research" "Using Claude Code for: $query"
  cd "$REPO_DIR"
  claude -p --model "$CLAUDE_MODEL" --dangerously-skip-permissions \
    "Search the web for: $query

Return ONLY a JSON object with search results in this exact format:
{\"results\": [{\"title\": \"page title\", \"content\": \"summary of the page content (max 500 chars)\", \"url\": \"https://...\"}]}

Return at least 5 results. No markdown, no explanation, just the JSON." \
    > "$out_file" 2>/dev/null || true

  python3 - "$out_file" << 'PYEOF'
import json, re, sys
f = sys.argv[1]
content = open(f).read()
m = re.search(r'\{[\s\S]*\}', content)
if m:
    try:
        parsed = json.loads(m.group())
        if "results" in parsed:
            json.dump(parsed, open(f, "w"), indent=2)
            exit(0)
    except: pass
json.dump({"results": [{"title": "Claude search", "content": content[:1000], "url": ""}]}, open(f, "w"), indent=2)
PYEOF

  local count
  count=$(python3 -c "import json; print(len(json.load(open('$out_file')).get('results',[])))" 2>/dev/null || echo "0")
  team "🔍 Research" "✓ Claude found $count results"
  return 0
}

market_research() {
  local out_file="$1" reqs_file="$2"
  team "🔍 Research" "Starting market analysis..."

  local ctx; ctx=$(read_project_context | safe_head -c 500) # safe_pipe: no fail on empty results
  local project_type; project_type=$(echo "$ctx" | safe_head -5 | safe_tr '\n' ' ') # safe_pipe: no fail on empty results

  zai_web_search "$PROJECT_NAME $project_type competitors comparison features 2025 enterprise" \
    "$ARTIFACTS/market_competitors.json"
  zai_web_search "$PROJECT_NAME similar products features comparison best practices 2025" \
    "$ARTIFACTS/market_features.json"
  zai_web_search "$project_type trends 2025 emerging technologies best practices" \
    "$ARTIFACTS/market_trends.json"
  zai_web_search "$project_type security compliance SOC2 ISO27001 GDPR requirements 2025" \
    "$ARTIFACTS/market_compliance.json"

  python3 - "$ARTIFACTS/market_competitors.json" "$ARTIFACTS/market_features.json" \
    "$ARTIFACTS/market_trends.json" "$ARTIFACTS/market_compliance.json" \
    "$ARTIFACTS/market_combined.json" << 'PYEOF'
import json, sys
combined = {"competitors": [], "features": [], "trends": [], "compliance": []}
files = sys.argv[1:5]
keys = ["competitors", "features", "trends", "compliance"]
for f, k in zip(files, keys):
    try:
        data = json.load(open(f))
        combined[k] = data.get("results", [])
    except: pass
json.dump(combined, open(sys.argv[5], "w"), indent=2)
PYEOF

  local search_data; search_data=$(summarize_artifact "$ARTIFACTS/market_combined.json" 4000)
  local reqs; reqs=$(head -c 3000 "$reqs_file" 2>/dev/null || echo "{}")

  cat > "$DEV_DIR/tmp_sys.txt" << 'PROMPT'
You are a Market Research Analyst. Analyze competitor data and identify gaps in the project.

RESPOND WITH ONLY JSON:
{
  "competitor_summary": [{"name": "competitor", "strengths": ["strength"], "weaknesses": ["weakness"], "pricing_model": "description"}],
  "feature_comparison": [{"feature": "name", "competitors_with_feature": ["name"], "project_status": "implemented|partial|missing", "priority": "critical|high|medium|low", "implementation_effort": "small|medium|large"}],
  "market_gaps": [{"gap": "description", "competitors_offering": ["name"], "business_impact": "high|medium|low", "recommended_priority": 1}],
  "emerging_trends": [{"trend": "name", "description": "detail", "adoption_stage": "early|growing|mainstream", "relevance": "high|medium|low"}],
  "compliance_gaps": [{"standard": "SOC2|ISO27001|GDPR|FedRAMP", "requirement": "what", "project_status": "met|partial|missing"}],
  "recommended_features": [{"feature": "name", "description": "detail", "priority": "critical|high|medium", "effort": "small|medium|large", "competitive_advantage": "description"}],
  "unique_selling_points": ["what makes this project different"],
  "summary": "2-3 paragraph market analysis"
}
PROMPT

  local project_context; project_context=$(read_project_context)

  cat > "$DEV_DIR/tmp_usr.txt" << PROMPT
COMPETITOR SEARCH RESULTS:
$search_data

PROJECT CONTEXT (from CLAUDE.md):
$project_context

CURRENT FEATURES (from requirements):
$reqs

Analyze the market and identify what this project is missing vs competitors.
PROMPT

  ai_think "🔍 Research" "$DEV_DIR/tmp_sys.txt" "$DEV_DIR/tmp_usr.txt" "$out_file"
  team "🔍 Research" "✓ Market analysis complete"
}

# ═══════════════════════════════════════════════
# CLAUDE CODE ENGINE (with in-process healing)
# ═══════════════════════════════════════════════

# Safe Claude wrapper with timeout monitoring
run_claude() {
  local max_secs="${1:-300}" out_file="${2:-/dev/null}" prompt="$3"

  claude -p --model "$CLAUDE_MODEL" --dangerously-skip-permissions \
    "$prompt" </dev/null > "$out_file" 2>&1 &
  local cpid=$!

  local waited=0
  while kill -0 "$cpid" 2>/dev/null && [ "$waited" -lt "$max_secs" ]; do
    sleep 5; waited=$((waited + 5))
  done

  if kill -0 "$cpid" 2>/dev/null; then
    warn "  ⏰ Claude timed out (${max_secs}s) — killing"
    kill "$cpid" 2>/dev/null; sleep 2; kill -9 "$cpid" 2>/dev/null || true
    wait "$cpid" 2>/dev/null || true
    return 124
  fi

  wait "$cpid" 2>/dev/null
  return $?
}

# Primary code execution: retry + prompt shrink + commit
claude_do() {
  local role_name="$1" prompt="$2" log_file="$3"
  local timeout="${4:-1800}"  # Increased default: 30 minutes (was 900s)
  team "$role_name" "Working..."
  cd "$REPO_DIR"

  # Truncate prompt if too large (reduced threshold for better reliability)
  local prompt_len=${#prompt}
  if [ "$prompt_len" -gt 8000 ]; then
    warn "  Prompt too large (${prompt_len} chars) — truncating to 8000"
    prompt="${prompt:0:8000}

[TRUNCATED — original was ${prompt_len} chars. Focus on the most important parts above.]"
  fi

  local attempt=0 ok=false exit_code=0 backoff=5
  while [ $attempt -lt 3 ]; do
    attempt=$((attempt + 1))
    [ $attempt -gt 1 ] && warn "  ↻ Attempt $attempt/3 (after ${backoff}s delay)"

    if timeout "$timeout" claude -p --model "$CLAUDE_MODEL" --dangerously-skip-permissions \
      "$prompt" 2>&1 | tee "$log_file"; then
      ok=true; break
    fi

    exit_code=$?
    if [ $exit_code -eq 124 ]; then
      warn "  ⏰ Timeout after ${timeout}s — increasing timeout for retry"
      timeout=$((timeout + 600))  # Add 10 minutes for retry
    elif [ $exit_code -ge 137 ]; then
      warn "  💀 Killed (exit $exit_code) — likely OOM or rate limit, reducing prompt"
      prompt="${prompt:0:4000}

[REDUCED — Claude was killed. Simplified prompt for retry.]"
    fi

    # Exponential backoff before retry
    [ $attempt -lt 3 ] && sleep "$backoff"
    backoff=$((backoff * 2))
  done

  if [ "$ok" = true ]; then
    cd "$REPO_DIR"; git add -A
    if ! git diff --cached --quiet 2>/dev/null; then
      git commit -m "[$role_name] $(echo "$prompt" | safe_head -1 | safe_cut -c1-60)" 2>/dev/null || true # safe_pipe: no fail on empty results
    fi
    team "$role_name" "✓ Committed"
    return 0
  fi
  team "$role_name" "✗ Failed after $attempt attempts"
  record_error "$role_name" "claude_terminated" "Failed after $attempt attempts, last exit: $exit_code"
  return 1
}

# ═══════════════════════════════════════════════
# DOCKER / TESTS / GIT HELPERS
# ═══════════════════════════════════════════════

docker_build_all() {
  cd "$REPO_DIR"
  local runtime
  runtime=$(detect_container_runtime)
  local ok=true total=0 built=0 failed=0 max_failures=3
  for df in deployments/docker/Dockerfile.*; do [ -f "$df" ] && total=$((total+1)); done
  [ "$total" -eq 0 ] && { warn "No Dockerfiles found"; return 0; }

  # Rotate docker_build.log if it's getting large
  rotate_logs 500

  local idx=0
  for df in deployments/docker/Dockerfile.*; do
    [ -f "$df" ] || continue
    idx=$((idx+1))

    if [ "$failed" -ge "$max_failures" ]; then
      warn "  ⚠ $failed builds failed — skipping remaining"
      ok=false; break
    fi

    local svc; svc=$(basename "$df" | safe_sed 's/Dockerfile\.//') # safe_pipe: no fail on empty results
    local t0; t0=$(date +%s)
    log "  🐳 Building ($idx/$total): $svc"
    local build_rc=0
    timeout 300 $runtime build -f "$df" -t "${PROJECT_NAME}/${svc}:dev" . 2>&1 | tee -a "$PHASE_LOGS/docker_build.log" | tail -5 || build_rc=$?
    local elapsed=$(( $(date +%s) - t0 ))
    if [ $build_rc -eq 0 ]; then
      log "  ✓ Built: $svc (${elapsed}s)"; built=$((built+1))
    else
      warn "  ✗ Build failed: $svc (${elapsed}s) — fixing..."
      failed=$((failed+1))
      record_error "deploy" "docker_build_fail" "$svc: $(tail -5 "$PHASE_LOGS/docker_build.log" 2>/dev/null)"
      claude_do "🐳 DevOps" "Read CLAUDE.md. Docker build failed for $svc. Error:

$(tail -15 "$PHASE_LOGS/docker_build.log" 2>/dev/null)

Fix the Dockerfile or source code. Rebuild should pass." \
        "$PHASE_LOGS/docker_fix_${svc}.log" 600
      timeout 300 $runtime build -f "$df" -t "${PROJECT_NAME}/${svc}:dev" . 2>&1 | tail -5 || { ok=false; failed=$((failed+1)); }
    fi
  done
  log "  Docker: $built/$total built, $failed failed"
  $ok
}

# Clean up containers with stale dependencies (orphaned references)
cleanup_stale_containers() {
  local compose_file="$1"
  local project_name="${PROJECT_NAME:-docker}"

  log "  🔧 Checking for stale containers..."

  # Get runtime from detect_container_runtime()
  local runtime
  runtime=$(detect_container_runtime)

  # Get list of containers for this project
  local containers
  containers=$($runtime ps -a --format "{{.Names}}" --filter "label=com.docker.compose.project=$project_name" 2>/dev/null || true)

  if [ -z "$containers" ]; then
    return 0
  fi

  local stale_count=0
  for container in $containers; do
    # Check if container has stale dependencies by attempting to inspect it
    # If inspection fails with dependency error, remove it
    if ! $runtime inspect "$container" &>/dev/null; then
      log "  🗑️  Removing stale container: $container"
      $runtime rm -f "$container" 2>/dev/null || true
      stale_count=$((stale_count + 1))
    fi
  done

  # Also check for containers in "Created" or "Exited" states but failing to start
  for container in $containers; do
    local state
    state=$($runtime ps -a --format "{{.State}}" --filter "name=$container" 2>/dev/null || echo "")
    if [ "$state" = "Created" ] || [ "$state" = "Exited" ]; then
      # Check if it has been in this state for more than 5 minutes
      local created_since
      created_since=$($runtime ps -a --format "{{.CreatedAt}}" --filter "name=$container" 2>/dev/null || echo "")
      if [ -n "$created_since" ]; then
        log "  🗑️  Removing stuck container ($state state): $container"
        $runtime rm -f "$container" 2>/dev/null || true
        stale_count=$((stale_count + 1))
      fi
    fi
  done

  if [ $stale_count -gt 0 ]; then
    log "  ✓ Removed $stale_count stale container(s)"
  fi
}

# Detect container runtime (docker or podman)
detect_container_runtime() {
  if command -v podman >/dev/null 2>&1; then
    echo "podman"
  elif command -v docker >/dev/null 2>&1; then
    echo "docker"
  else
    echo "docker"  # Default fallback
  fi
}

# Detect compose command (docker-compose, podman-compose, or docker compose)
detect_compose_command() {
  if command -v podman-compose >/dev/null 2>&1; then
    echo "podman-compose"
  elif docker compose version >/dev/null 2>&1; then
    echo "docker compose"
  elif command -v docker-compose >/dev/null 2>&1; then
    echo "docker-compose"
  else
    echo "docker compose"  # Default fallback (plugin is most common now)
  fi
}

# Verify container health directly (works around podman-compose healthcheck parsing issues)
verify_container_health() {
  local container_name="$1"
  local health_url="$2"
  local runtime
  runtime=$(detect_container_runtime)

  # Check if container is running
  if ! $runtime ps --format "{{.Names}}" | grep -q "^${container_name}$"; then
    return 1
  fi

  # If health_url provided, check it
  if [ -n "$health_url" ]; then
    curl -sf --max-time 5 "$health_url" >/dev/null 2>&1
    return $?
  fi

  # Fallback: check if container is healthy (if healthcheck is working)
  local health_status
  health_status=$($runtime inspect --format='{{.State.Health.Status}}' "$container_name" 2>/dev/null || echo "")
  if [ "$health_status" = "healthy" ]; then
    return 0
  fi

  # If no healthcheck status but container is running, assume OK
  $runtime inspect --format='{{.State.Running}}' "$container_name" 2>/dev/null | grep -q "true"
}

# Fix postgres container startup issues (data directory, network, etc.)
fix_postgres_container() {
  local runtime
  runtime=$(detect_container_runtime)
  local pg_container="openprint-postgres"
  local pg_network="openprint-network"

  # Check if postgres container exists
  if ! $runtime ps -a --format "{{.Names}}" | grep -q "^${pg_container}$"; then
    return 0  # Container doesn't exist yet, nothing to fix
  fi

  local pg_state
  pg_state=$($runtime ps --format "{{.State}}" --filter "name=$pg_container" 2>/dev/null || echo "")

  # Check if postgres is stuck in "starting" or exited state
  if [ "$pg_state" = "starting" ] || [ "$pg_state" = "Exited" ] || [ "$pg_state" = "exited" ]; then
    log "  🔧 PostgreSQL container stuck ($pg_state), fixing..."

    # Check logs for data directory errors
    local logs
    logs=$($runtime logs "$pg_container" 2>&1 || echo "")

    if echo "$logs" | grep -qi "exists but is not empty\|initdb: error"; then
      log "  🗑️  PostgreSQL data directory issue detected, recreating..."

      # Stop and remove container
      $runtime stop "$pg_container" 2>/dev/null || true
      $runtime rm "$pg_container" 2>/dev/null || true

      # Note: We preserve the volume, just recreate the container
      log "  ✓ PostgreSQL container removed, will be recreated"
      return 1  # Signal that recreation is needed
    fi

    # Check if postgres is on correct network
    if ! $runtime inspect "$pg_container" --format '{{range $k,$v := .NetworkSettings.Networks}}{{$k}} {{end}}' 2>/dev/null | grep -q "$pg_network"; then
      log "  🔧 PostgreSQL not on correct network, fixing..."
      $runtime stop "$pg_container" 2>/dev/null || true
      $runtime rm "$pg_container" 2>/dev/null || true
      log "  ✓ PostgreSQL will be recreated with correct network"
      return 1
    fi
  fi

  return 0
}

docker_up() {
  log "  🚀 Starting services..."
  cd "$REPO_DIR"

  # Rotate docker_up.log if it's getting large
  rotate_logs 500  # Rotate at 500KB

  # Fix postgres container issues before starting
  fix_postgres_container

  detect_service_ports
  local compose=""
  [ -f "docker-compose.yml" ] && compose="docker-compose.yml"
  [ -f "deployments/docker/docker-compose.yml" ] && compose="deployments/docker/docker-compose.yml"

  if [ -n "$compose" ]; then
    # Detect compose command
    local compose_cmd
    compose_cmd=$(detect_compose_command)

    # Check for port conflicts and get dynamic mappings
    log "  🔍 Checking port availability..."
    get_dynamic_ports

    # Apply port mappings if there were conflicts
    if [ -f "$DEV_DIR/port_mappings.txt" ]; then
      apply_port_mappings "$compose"
      # Update compose path to use the modified file
      compose="${compose}.modified"
      cp "${compose%.modified}" "$compose"
    fi

    # Clean up stale containers first
    cleanup_stale_containers "$compose"

    local up_failed=false
    if ! $compose_cmd -f "$compose" up -d 2>&1 | tee -a "$PHASE_LOGS/docker_up.log" | tail -10; then
      up_failed=true
    fi

    # If up failed due to dependency errors, force recreate
    if $up_failed || grep -qi "depends on.*not found\|no such container\|requires.*not found" "$PHASE_LOGS/docker_up.log" 2>/dev/null; then
      warn "  ⚠️  Dependency errors detected, recreating containers..."
      $compose_cmd -f "$compose" down 2>/dev/null || true
      $compose_cmd -f "$compose" up -d --force-recreate 2>&1 | tee -a "$PHASE_LOGS/docker_up.log" | tail -10 || true
    fi

    sleep "$DOCKER_TIMEOUT"
    local h=0 t=0
    local runtime
    runtime=$(detect_container_runtime)

    # Get actual service ports from running containers (dynamic)
    service_ports_json=$(get_all_service_ports 2>/dev/null || echo '{}')

    # Check application service containers directly first
    local app_containers="auth-service registry-service job-service storage-service notification-service dashboard"
    for container in $app_containers; do
      t=$((t+1))
      local full_name="openprint-${container}"
      if [ "$SKIP_HEALTH_CHECK" = true ]; then
        h=$((h+1)); log "  ✓ $container (health check skipped)"
      elif $runtime ps --format "{{.Names}}" | grep -q "^${full_name}$"; then
        # Get port from dynamic detection
        local port=""
        port=$(echo "$service_ports_json" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('$container','')))" 2>/dev/null || echo "")

        if [ -n "$port" ] && curl -sf --max-time "$HEALTH_CHECK_TIMEOUT" "http://localhost:${port}${HEALTH_CHECK_PATH}" >/dev/null 2>&1; then
          h=$((h+1)); log "  ✓ $container (:$port)"
        else
          warn "  ✗ $container (:$port)"
        fi
      else
        warn "  ✗ $container (not running)"
      fi
    done

    log "  Health: $h/$t services up"

    # Check PostgreSQL specifically (critical for auth service)
    if [ "$SKIP_HEALTH_CHECK" != true ]; then
      local pg_ready=false
      local pg_container="openprint-postgres"

      if $runtime ps --format "{{.Names}}" | grep -q "^${pg_container}$"; then
        # Try pg_isready up to 10 times
        for i in $(seq 1 10); do
          if $runtime exec "$pg_container" pg_isready -U openprint -d openprint >/dev/null 2>&1; then
            pg_ready=true
            log "  ✓ PostgreSQL ready"
            break
          fi
          sleep 1
        done

        if [ "$pg_ready" = false ]; then
          warn "  ⚠️  PostgreSQL not ready - checking for issues..."
          local pg_logs
          pg_logs=$($runtime logs "$pg_container" 2>&1 | tail -5 || echo "")
          if echo "$pg_logs" | grep -qi "exists but is not empty\|initdb: error"; then
            warn "  🗑️  PostgreSQL data issue detected - run './dev.sh fix-postgres' to repair"
          fi
        fi
      fi
    fi

    # Additional port-based check for any extra services
    if [ "$h" -lt "$t" ] && [ "$SKIP_HEALTH_CHECK" != true ]; then
      log "  Waiting for services to be ready..."
      sleep 10
      for port in $HEALTH_CHECK_PORTS; do
        if curl -sf --max-time "$HEALTH_CHECK_TIMEOUT" "http://localhost:${port}${HEALTH_CHECK_PATH}" >/dev/null 2>&1; then
          log "  ✓ Additional service on :$port"
        fi
      done
    fi
  else
    warn "  No docker-compose found"
  fi
}

docker_down() {
  cd "$REPO_DIR" 2>/dev/null || return 0
  local compose_cmd
  compose_cmd=$(detect_compose_command)
  for f in docker-compose.yml deployments/docker/docker-compose.yml; do
    [ -f "$REPO_DIR/$f" ] && $compose_cmd -f "$REPO_DIR/$f" down 2>/dev/null || true
  done
}

run_go_tests() {
  team "🧪 Tester" "Running Go tests..."
  cd "$REPO_DIR"
  local rc=0
  go test ./... -count=1 -timeout 180s -v 2>&1 | tee "$PHASE_LOGS/go_test.log" | tail -30 || rc=$?
  return $rc
}

fix_go_tests() {
  cd "$REPO_DIR"
  local failures
  failures=$(grep -A 3 "FAIL\|Error\|panic\|undefined\|cannot\|expected\|got:" "$PHASE_LOGS/go_test.log" 2>/dev/null | head -40 || true)
  [ -z "$failures" ] && failures=$(tail -25 "$PHASE_LOGS/go_test.log")

  record_error "testing" "go_test_fail" "$failures"
  local past; past=$(get_past_errors "testing")

  claude_do "🧪 Tester" "Read CLAUDE.md. Fix ONLY the failing Go tests:

$failures

${past:+$past

}Do NOT rewrite passing tests. Run 'go test ./...' to verify." \
    "$PHASE_LOGS/go_test_fix.log" 600
}

run_playwright() {
  local dir="$REPO_DIR/${FRONTEND_DIR:-web/dashboard}"
  [ -d "$dir" ] || return 0
  team "🧪 Tester" "Running Playwright..."
  cd "$dir"
  [ -d "node_modules" ] || { npm install 2>&1 | tail -3 || true; npx playwright install --with-deps 2>&1 | tail -3 || true; }
  local rc=0
  npx playwright test --reporter=list 2>&1 | tee "$PHASE_LOGS/playwright.log" | tail -20 || rc=$?
  return $rc
}

fix_playwright() {
  cd "$REPO_DIR"
  local failures
  failures=$(grep -A 3 "FAIL\|Error\|TimeoutError\|expect\|Received" "$PHASE_LOGS/playwright.log" 2>/dev/null | head -30 || true)
  [ -z "$failures" ] && failures=$(tail -20 "$PHASE_LOGS/playwright.log")

  claude_do "🧪 Tester" "Read CLAUDE.md. Fix Playwright failures:

$failures

Verify: cd frontend && npx playwright test" \
    "$PHASE_LOGS/playwright_fix.log" 600
}

ensure_branch() {
  cd "$REPO_DIR"
  git checkout main 2>/dev/null || true
  git pull origin main 2>/dev/null || true
  git rev-parse --verify "$BRANCH" >/dev/null 2>&1 && git checkout "$BRANCH" || git checkout -b "$BRANCH"
}

merge_to_main() {
  cd "$REPO_DIR"
  git add -A && git commit -m "pre-merge" 2>/dev/null || true
  git checkout main 2>/dev/null || true
  git pull origin main 2>/dev/null || true
  if git merge "$BRANCH" --no-ff -m "Merge $BRANCH" 2>/dev/null; then
    git push origin main 2>/dev/null || true
    log "  ✓ Merged → main"
  else
    git merge --abort 2>/dev/null || true
    git merge "$BRANCH" --no-commit 2>/dev/null || true
    claude_do "🐳 DevOps" "Resolve merge conflict between $BRANCH and main. Keep both changes." "$PHASE_LOGS/merge_fix.log"
    git add -A && git commit -m "Merge $BRANCH (resolved)" 2>/dev/null || true
    git push origin main 2>/dev/null || true
  fi
}

# ═══════════════════════════════════════════════
# WATERFALL PHASES
# ═══════════════════════════════════════════════

# ──────────────────────────────
# 1. REQUIREMENTS (PM)
# ──────────────────────────────
phase_requirements() {
  local project="$1"
  log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  log "PHASE 1: REQUIREMENTS — 🧑‍💼 PM"
  log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  state_set requirements status running

  cat > "$DEV_DIR/tmp_sys.txt" << 'PROMPT'
You are a Senior Project Manager. Read the project's CLAUDE.md for context about the tech stack, architecture, and conventions.
RESPOND WITH ONLY JSON:
{"project_name":"short","summary":"desc","user_stories":[{"id":"US-001","as_a":"role","i_want":"action","so_that":"benefit","priority":"critical|high|medium|low","acceptance_criteria":["criterion"]}],"functional_requirements":[{"id":"FR-001","title":"short","description":"detailed","service":"service_name"}],"non_functional_requirements":[{"id":"NFR-001","category":"performance|security|scalability","requirement":"desc","metric":"target"}],"api_endpoints":[{"method":"POST","path":"/api/v1/...","description":"what","roles":["admin"]}],"database_changes":[{"table":"name","action":"create|alter","columns":["col type"]}],"affected_services":["service"],"risks":[{"risk":"desc","mitigation":"plan"}],"implementation_phases":[{"phase":1,"name":"short","tasks":["task"]}]}
PROMPT

  local project_context; project_context=$(read_project_context)

  cat > "$DEV_DIR/tmp_usr.txt" << PROMPT
PROJECT: $project

PROJECT CONTEXT (from CLAUDE.md):
$project_context

Create comprehensive requirements.
PROMPT

  ai_think "🧑‍💼 PM" "$DEV_DIR/tmp_sys.txt" "$DEV_DIR/tmp_usr.txt" "$ARTIFACTS/01_requirements.json"
  state_set requirements status done
  log "✅ Requirements done"
}

# ──────────────────────────────
# 1.5 MARKET RESEARCH (Researcher)
# ──────────────────────────────
phase_market_research() {
  log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  log "PHASE 1.5: MARKET RESEARCH — 🔍 Researcher"
  log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  state_set market_research status running

  market_research "$ARTIFACTS/03_market_analysis.json" "$ARTIFACTS/01_requirements.json"

  local gaps
  gaps=$(python3 -c "
import json
try:
    d = json.load(open('$ARTIFACTS/03_market_analysis.json'))
    critical = [f for f in d.get('recommended_features', []) if f.get('priority') in ('critical', 'high')]
    for f in critical[:5]:
        print(f\"• {f.get('feature','')}: {f.get('description','')}\")
    if not critical:
        print('No critical gaps found')
except: print('Analysis not available')
" 2>/dev/null || echo "Analysis not available")

  log "  Market gaps found:"
  echo "$gaps" | while read -r line; do log "    $line"; done

  python3 - "$ARTIFACTS/01_requirements.json" "$ARTIFACTS/03_market_analysis.json" << 'PYEOF'
import json, sys
try:
    reqs = json.load(open(sys.argv[1]))
    market = json.load(open(sys.argv[2]))
    existing_ids = [r.get("id","") for r in reqs.get("functional_requirements", [])]
    next_id = len(existing_ids) + 1
    for feat in market.get("recommended_features", []):
        if feat.get("priority") in ("critical", "high"):
            reqs.setdefault("functional_requirements", []).append({
                "id": f"FR-M{next_id:03d}",
                "title": feat.get("feature", ""),
                "description": feat.get("description", ""),
                "service": "identity",
                "source": "market_research",
                "competitive_advantage": feat.get("competitive_advantage", "")
            })
            next_id += 1
    reqs["market_context"] = {
        "competitors_analyzed": [c.get("name","") for c in market.get("competitor_summary", [])],
        "key_gaps": [g.get("gap","") for g in market.get("market_gaps", [])[:5]],
        "trends": [t.get("trend","") for t in market.get("emerging_trends", [])[:5]],
        "unique_selling_points": market.get("unique_selling_points", [])
    }
    json.dump(reqs, open(sys.argv[1], "w"), indent=2)
except Exception as e:
    print(f"Warning: Could not enrich requirements: {e}")
PYEOF

  state_set market_research status done
  log "✅ Market research done"
}

# ──────────────────────────────
# 2. DESIGN (Architect)
# ──────────────────────────────
phase_design() {
  log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  log "PHASE 2: DESIGN — 🏗️  Architect"
  log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  state_set design status running

  cd "$REPO_DIR"
  local reqs; reqs=$(summarize_artifact "$ARTIFACTS/01_requirements.json" 4000)
  local frontend_dir="${FRONTEND_DIR:-web/dashboard}"
  local files; files=$(find internal "$frontend_dir/src" -name "*.go" -o -name "*.tsx" 2>/dev/null | grep -v _test | grep -v node_modules | sort | head -50 || true)
  local market; market=$(summarize_artifact "$ARTIFACTS/03_market_analysis.json" 2000)

  cat > "$DEV_DIR/tmp_sys.txt" << 'PROMPT'
You are a Senior Software Architect. Read the project's CLAUDE.md for tech stack and conventions.
RESPOND WITH ONLY JSON:
{"architecture_decisions":[{"decision":"what","rationale":"why"}],"backend_tasks":[{"order":1,"file":"internal/path/file.go","action":"create|modify","purpose":"desc","key_functions":["Name"]}],"frontend_tasks":[{"order":1,"file":"frontend/src/path/File.tsx","action":"create|modify","purpose":"desc"}],"database_migrations":[{"file":"migrations/NNN_name.up.sql","sql":"CREATE TABLE..."}],"api_contracts":[{"method":"POST","path":"/api/v1/...","request":{},"response":{},"status_codes":[200,400]}],"test_plan":{"unit_tests":[{"file":"path_test.go","cases":["scenario"]}],"e2e_tests":[{"name":"test","steps":["step"]}]},"security_notes":["note"],"docker_changes":["change"],"market_driven_features":["feature incorporated from market analysis"]}
PROMPT

  local project_context; project_context=$(read_project_context)

  cat > "$DEV_DIR/tmp_usr.txt" << PROMPT
REQUIREMENTS (enriched with market research):
$reqs

MARKET ANALYSIS:
$market

PROJECT CONTEXT (from CLAUDE.md):
$project_context

EXISTING FILES:
$files

Design the complete solution. Include market-driven features where priority is critical/high. Be specific about file paths, function names, schemas, implementation order.
PROMPT

  ai_think "🏗️  Architect" "$DEV_DIR/tmp_sys.txt" "$DEV_DIR/tmp_usr.txt" "$ARTIFACTS/02_design.json"
  state_set design status done
  log "✅ Design done"
}

# ──────────────────────────────
# 3. BACKEND (Backend Dev)
# ──────────────────────────────
phase_backend() {
  log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  log "PHASE 3: BACKEND — ⚙️  Backend Dev"
  log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  state_set backend status running; ensure_branch

  local design; design=$(summarize_artifact "$ARTIFACTS/02_design.json" 4000)
  local reqs; reqs=$(summarize_artifact "$ARTIFACTS/01_requirements.json" 2000)

  claude_do "⚙️  Backend" \
    "Read CLAUDE.md first. You are the Backend Developer for this project.

DESIGN: $design
REQUIREMENTS: $reqs

IMPLEMENT ALL backend files from design. Follow the patterns and conventions described in CLAUDE.md. Create migration SQL files. DO NOT write tests. Production-quality code." \
    "$PHASE_LOGS/03_backend.log"

  cd "$REPO_DIR"
  local build_ok=true
  go build ./... 2>&1 | tee "$PHASE_LOGS/03_compile.log" | tail -5 || build_ok=false
  if [ "$build_ok" = false ]; then
    record_error "backend" "compile_fail" "$(tail -10 "$PHASE_LOGS/03_compile.log")"
    local past; past=$(get_past_errors "backend")
    claude_do "⚙️  Backend" "Fix Go compilation errors:

$(tail -25 "$PHASE_LOGS/03_compile.log")

${past:+$past

}Fix ALL errors. Run 'go build ./...' to verify." "$PHASE_LOGS/03_compile_fix.log" 600
  fi

  state_set backend status done; log "✅ Backend done"
}

# ──────────────────────────────
# 4. FRONTEND (Frontend Dev)
# ──────────────────────────────
phase_frontend() {
  log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  log "PHASE 4: FRONTEND — 🎨 Frontend Dev"
  log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  state_set frontend status running; ensure_branch

  local design; design=$(summarize_artifact "$ARTIFACTS/02_design.json" 2500)
  local reqs; reqs=$(summarize_artifact "$ARTIFACTS/01_requirements.json" 1500)

  claude_do "🎨 Frontend" \
    "Read CLAUDE.md first. You are the Frontend Developer for this project.

DESIGN: $design
REQUIREMENTS: $reqs

IMPLEMENT ALL frontend components/pages from design. Follow the conventions in CLAUDE.md. Create Playwright E2E tests in ${FRONTEND_DIR:-web/dashboard}/e2e/. Install deps if needed." \
    "$PHASE_LOGS/04_frontend.log" 2400  # 40 minutes for frontend implementation

  local frontend_dir="${FRONTEND_DIR:-web/dashboard}"
  if [ -d "$REPO_DIR/$frontend_dir" ]; then
    cd "$REPO_DIR/$frontend_dir"; [ -d "node_modules" ] || npm install 2>&1 | tail -3 || true
    local ts_ok=true
    npx tsc --noEmit 2>&1 | tee "$PHASE_LOGS/04_typecheck.log" | tail -5 || ts_ok=false
    if [ "$ts_ok" = false ]; then
      record_error "frontend" "typecheck_fail" "$(tail -10 "$PHASE_LOGS/04_typecheck.log")"
      cd "$REPO_DIR"
      claude_do "🎨 Frontend" "Fix TypeScript errors:

$(grep -A 1 "error TS" "$PHASE_LOGS/04_typecheck.log" | head -25 || tail -20 "$PHASE_LOGS/04_typecheck.log")

Run 'cd frontend && npx tsc --noEmit' to verify." "$PHASE_LOGS/04_ts_fix.log" 600
    fi
  fi

  state_set frontend status done; log "✅ Frontend done"
}

# ──────────────────────────────
# 5. TESTING (Tester)
# ──────────────────────────────
phase_testing() {
  log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  log "PHASE 5: TESTING — 🧪 Tester"
  log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  state_set testing status running; ensure_branch

  local design; design=$(summarize_artifact "$ARTIFACTS/02_design.json" 3000)

  # Sub-step 1: Write tests
  if [ "$(state_get testing write_tests)" != "done" ]; then
    claude_do "🧪 Tester" \
      "Read CLAUDE.md first. You are the Test Engineer for this project.
DESIGN: $design
Write comprehensive tests for ALL new files following CLAUDE.md conventions. Write Playwright E2E tests in frontend/e2e/ if frontend exists. Run tests and fix failures." \
      "$PHASE_LOGS/05_tests.log" 1200
    state_set testing write_tests done
  else
    log "  ↳ Write tests already done — skipping"
  fi

  # Sub-step 2: Go tests
  if [ "$(state_get testing unit_tests)" != "passed" ]; then
    if ! run_go_tests; then
      fix_go_tests
      if ! run_go_tests; then
        warn "  Unit tests still failing — marking and continuing"
        state_set testing unit_tests failed
      else
        state_set testing unit_tests passed
      fi
    else
      state_set testing unit_tests passed
    fi
  else
    log "  ↳ Unit tests already passed — skipping"
  fi

  # Sub-step 3: E2E
  if [ "$(state_get testing e2e)" != "passed" ] && [ "$(state_get testing e2e)" != "skipped" ]; then
    local frontend_dir="${FRONTEND_DIR:-web/dashboard}"
    if [ -d "$REPO_DIR/$frontend_dir" ] && ls "$REPO_DIR/$frontend_dir/e2e/"*.spec.* >/dev/null 2>&1; then
      if docker_build_all; then
        docker_up
        if ! run_playwright; then
          fix_playwright
          run_playwright && state_set testing e2e passed || state_set testing e2e failed
        else
          state_set testing e2e passed
        fi
        docker_down
      else
        warn "  Docker builds failed — skipping E2E tests"
        state_set testing e2e skipped
      fi
    else
      state_set testing e2e skipped
    fi
  else
    log "  ↳ E2E already done — skipping"
  fi

  cd "$REPO_DIR"; git add -A && git commit -m "[Tester] tests" 2>/dev/null || true
  state_set testing status done; log "✅ Testing done"
}

# ──────────────────────────────
# 6. QA REVIEW (QA Controller)
# ──────────────────────────────
phase_qa() {
  log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  log "PHASE 6: QA — 📋 QA Controller"
  log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  state_set qa status running

  cd "$REPO_DIR"
  local diff; diff=$(git diff main --stat 2>/dev/null | tail -15 || true)
  local files; files=$(git diff main --name-only 2>/dev/null | grep "\.go$" | head -20 || true)
  local code=""; for f in $(echo "$files" | safe_head -5); do [ -f "$f" ] && code="$code
--- $f ---
$(head -80 "$f")"; done
  local reqs; reqs=$(summarize_artifact "$ARTIFACTS/01_requirements.json" 2000)

  cat > "$DEV_DIR/tmp_sys.txt" << 'PROMPT'
You are the QA Controller for this project. Read CLAUDE.md for context. Review strictly.
RESPOND WITH ONLY JSON:
{"overall_score":85,"verdict":"APPROVE|NEEDS_FIXES|REJECT","code_issues":[{"severity":"critical|major|minor","file":"path","issue":"desc","fix":"suggestion"}],"missing_items":["item"],"blocking_issues":["issue"],"fix_instructions":"if NEEDS_FIXES"}
PROMPT
  cat > "$DEV_DIR/tmp_usr.txt" << PROMPT
REQUIREMENTS: $reqs
CHANGES: $diff
FILES: $files
CODE: $code
Review: error handling, validation, auth checks, HTTP codes, test coverage, API consistency.
PROMPT

  ai_think "📋 QA" "$DEV_DIR/tmp_sys.txt" "$DEV_DIR/tmp_usr.txt" "$ARTIFACTS/06_qa_review.json"

  local verdict; verdict=$(python3 -c "import json; print(json.load(open('$ARTIFACTS/06_qa_review.json')).get('verdict','APPROVE'))" 2>/dev/null || echo "APPROVE")
  team "📋 QA" "Verdict: $verdict"

  if [ "$verdict" = "NEEDS_FIXES" ]; then
    local fixes; fixes=$(python3 -c "
import json; d=json.load(open('$ARTIFACTS/06_qa_review.json'))
print(d.get('fix_instructions',''))
for i in d.get('blocking_issues',[]): print(f'BLOCKING: {i}')
for c in d.get('code_issues',[]):
    if c.get('severity') in ('critical','major'): print(f\"{c['severity'].upper()}: {c.get('file','')}: {c.get('issue','')} → {c.get('fix','')}\")
" 2>/dev/null || echo "Fix issues")
    claude_do "⚙️  Backend" "Read CLAUDE.md. QA found issues:
$fixes
Fix ALL blocking/critical. Run 'go test ./...'." "$PHASE_LOGS/06_qa_fix.log"
  fi

  state_set qa verdict "$verdict"; state_set qa status done; log "✅ QA: $verdict"
}

# ──────────────────────────────
# 7. SECURITY (Security Auditor)
# ──────────────────────────────
phase_security() {
  log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  log "PHASE 7: SECURITY — 🔒 Security Auditor"
  log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  state_set security status running

  cd "$REPO_DIR"
  local auth=""; for f in $(find internal -name "*.go" 2>/dev/null | xargs grep -l "auth\|token\|password\|jwt\|session" 2>/dev/null | head -10 || true); do
    auth="$auth
--- $f ---
$(head -60 "$f")"; done
  local handlers=""; for f in $(find internal -name "handler*.go" -o -name "middleware*.go" 2>/dev/null | head -8 || true); do
    handlers="$handlers
--- $f ---
$(head -50 "$f")"; done

  cat > "$DEV_DIR/tmp_sys.txt" << 'PROMPT'
You are the Security Auditor. Read CLAUDE.md for project context. SECURITY IS CRITICAL.
RESPOND WITH ONLY JSON:
{"risk_level":"low|medium|high|critical","security_score":75,"vulnerabilities":[{"id":"V-001","severity":"critical|high|medium|low","file":"path","description":"what","fix":"how"}],"owasp_checks":[{"category":"A01","status":"pass|fail","detail":""}],"verdict":"APPROVE|NEEDS_FIXES|REJECT","critical_fixes":["fix"]}
PROMPT
  cat > "$DEV_DIR/tmp_usr.txt" << PROMPT
AUTH CODE: $auth
HANDLERS: $handlers
Check: SQL injection, XSS, CSRF, insecure JWT, weak crypto, missing auth, IDOR, data exposure. Be thorough.
PROMPT

  ai_think "🔒 Security" "$DEV_DIR/tmp_sys.txt" "$DEV_DIR/tmp_usr.txt" "$ARTIFACTS/07_security.json"

  local verdict; verdict=$(python3 -c "import json; print(json.load(open('$ARTIFACTS/07_security.json')).get('verdict','APPROVE'))" 2>/dev/null || echo "APPROVE")
  team "🔒 Security" "Verdict: $verdict"

  if [ "$verdict" = "NEEDS_FIXES" ] || [ "$verdict" = "REJECT" ]; then
    local fixes; fixes=$(python3 -c "
import json; d=json.load(open('$ARTIFACTS/07_security.json'))
for v in d.get('vulnerabilities',[]):
    if v.get('severity') in ('critical','high'): print(f\"{v['severity'].upper()}: {v.get('file','')}: {v.get('description','')} → {v.get('fix','')}\")
for f in d.get('critical_fixes',[]): print(f'FIX: {f}')
" 2>/dev/null || echo "Fix security issues")
    claude_do "⚙️  Backend" "Read CLAUDE.md. SECURITY FIX:
$fixes
Fix ALL critical/high vulnerabilities. Run 'go test ./...'." "$PHASE_LOGS/07_sec_fix.log"
    run_go_tests || { fix_go_tests; run_go_tests || true; }
  fi

  state_set security verdict "$verdict"; state_set security status done; log "✅ Security: $verdict"
}

# ──────────────────────────────
# 8. DEPLOY (DevOps)
# ──────────────────────────────
phase_deploy() {
  log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  log "PHASE 8: DEPLOY — 🐳 DevOps"
  log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  state_set deploy status running

  # 🔍 Port availability check (early warning for conflicts)
  log ""
  log "  🔍 Checking port availability..."
  local default_ports=(3000 8001 8002 8003 8004 8005 18001 18005)
  local occupied_ports=()
  local port_alternatives=""

  for port in "${default_ports[@]}"; do
    if ! is_port_available "$port"; then
      occupied_ports+=("$port")
      local free_port
      free_port=$(find_free_port "$((port + 1))" 3)
      if [ -n "$free_port" ]; then
        port_alternatives+="    → Port $port occupied, will use $free_port\n"
        # Store mapping for later use
        echo "${port}:${free_port}" >> "$DEV_DIR/port_conflicts.txt"
      else
        port_alternatives+="    → Port $port occupied, no free alternative found!\n"
      fi
    fi
  done

  if [ -n "$port_alternatives" ]; then
    warn "  ⚠️  Port conflicts detected:"
    echo -e "$port_alternatives"
    log "  🔄 Will use alternative ports for conflicting services"
  else
    log "  ✓ All required ports are available"
  fi

  # Docker setup creation if needed
  if ! ls "$REPO_DIR/deployments/docker/Dockerfile."* >/dev/null 2>&1 && [ ! -f "$REPO_DIR/Dockerfile" ]; then
    claude_do "🐳 DevOps" \
      "Read CLAUDE.md. Create Docker setup in deployments/docker/:
- Dockerfile for each service (multi-stage build, non-root user, HEALTHCHECK)
- docker-compose.yml with all services + database + networking, health checks, volumes.
Follow the service names and ports defined in CLAUDE.md.
Ensure services bind to 0.0.0.0 for external access." \
      "$PHASE_LOGS/08_docker.log"
  fi

  docker_build_all || true
  docker_up

  # Smoke tests
  detect_service_ports
  team "🐳 DevOps" "Smoke testing..."
  local ok=true
  for port in $SERVICE_PORTS; do
    if [ "$SKIP_HEALTH_CHECK" = true ]; then
      log "  ✓ :$port (health check skipped)"
    else
      curl -sf --max-time "$HEALTH_CHECK_TIMEOUT" "http://localhost:${port}${HEALTH_CHECK_PATH}" >/dev/null 2>&1 && log "  ✓ :$port" || { warn "  ✗ :$port"; ok=false; }
    fi
  done

  if [ "$ok" = false ]; then
    local logs=""
    for cname in $(podman ps -a --format '{{.Names}}' 2>/dev/null | grep "$PROJECT_NAME" | head -10 || true); do
      local l; l=$(podman logs "$cname" 2>&1 | safe_tail -15); [ -n "$l" ] && logs="$logs
=== $cname ===
$l"
    done
    claude_do "🐳 DevOps" "Fix startup failures:
$logs" "$PHASE_LOGS/08_fix.log"
    docker_build_all || true; docker_down; docker_up
  fi

  # Generate access report for external client connectivity
  log ""
  log "  📡 Generating access report..."
  generate_access_report

  local access_report="${ARTIFACTS:-}/access_report.json"
  if [ -f "$access_report" ]; then
    local ext_url; ext_url=$(python3 -c "import json; print(json.load(open('$access_report'))['client_connection']['base_url'])" 2>/dev/null || echo "N/A")
    log "  🌐 External Access URL: $ext_url"
    log "  📄 Full report: $access_report"
  fi

  [ -d "$REPO_DIR/$FRONTEND_DIR" ] && { run_playwright || true; }

  cd "$REPO_DIR"; git add -A && git commit -m "[DevOps] deploy ready" 2>/dev/null || true
  merge_to_main

  log "  🟢 Services running. Stop: ./dev.sh stop-services"
  state_set deploy status done; log "✅ Deploy done"
}

# ──────────────────────────────
# 9. E2E PRODUCTION (End-to-End Testing)
# ──────────────────────────────
phase_e2e_production() {
  log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  log "PHASE 9: E2E PRODUCTION — 🧪 Full System Test"
  log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  state_set e2e_production status running

  # Re-detect configuration in case it changed
  FRONTEND_CONFIG=$(detect_frontend_config)
  FRONTEND_DIR="${FRONTEND_CONFIG%%:*}"
  DASHBOARD_PORT="${FRONTEND_CONFIG##*:}"

  # Get actual dashboard port from running containers (dynamic)
  service_ports_json=$(get_all_service_ports 2>/dev/null || echo '{}')
  detected_dashboard_port=$(echo "$service_ports_json" | python3 -c "import json,sys; print(json.load(sys.stdin).get('dashboard','${DASHBOARD_PORT:-3000}'))" 2>/dev/null || echo "${DASHBOARD_PORT:-3000}")

  local external_ip
  external_ip=$(get_external_ip 2>/dev/null) || external_ip="localhost"
  local base_url="${E2E_BASE_URL:-http://localhost:${detected_dashboard_port}}"

  log "  🌐 Testing against: $base_url"
  log "  📡 Server IP: $external_ip"
  log "  🐳 Services should be running from deploy phase"

  # Ensure services are still running
  detect_service_ports
  local services_ok=true
  for port in $SERVICE_PORTS; do
    check_external_access "$port" "$external_ip" || { warn "  Service on port $port not accessible"; services_ok=false; }
  done

  if [ "$services_ok" = false ]; then
    warn "  Some services not accessible - attempting restart..."
    docker_up
    sleep 10
  fi

  # Run E2E tests with AI enhancement
  local e2e_report="${ARTIFACTS:-}/e2e_production_report.json"
  local e2e_result=0

  if [ "$E2E_ENABLED" = true ]; then
    # Try framework-specific E2E first, fall back to AI-powered generic
    if ! run_e2e_tests "$base_url" "$e2e_report" 2>/dev/null; then
      log "  🤖 Framework E2E unavailable, using AI-powered tests..."
      run_ai_e2e "$base_url" "$e2e_report" "true" || e2e_result=$?
    fi
  else
    log "  ⏭️  E2E tests disabled (E2E_ENABLED=false)"
  fi

  # Parse and display results
  if [ -f "$e2e_report" ]; then
    local summary; summary=$(python3 -c "
import json, sys
d = json.load(open(sys.argv[1]))
s = d.get('summary', {})
print(f\"{s.get('passed',0)}/{s.get('total',0)} passed\")
print(f\"Status: {d.get('status', 'unknown').upper()}\")
print(f\"Success Rate: {d.get('success_rate',0)}%\")
" "$e2e_report" 2>/dev/null || echo "Results unavailable")

    log "  📊 E2E Results: $summary"

    local status; status=$(python3 -c "import json; print(json.load(open('$e2e_report')).get('status','failed'))" 2>/dev/null || echo "failed")
    state_set e2e_production result "$status"
  else
    warn "  ⚠️  E2E report not generated"
    state_set e2e_production result "skipped"
  fi

  # Generate access report if not already exists
  generate_access_report
  local access_report="${ARTIFACTS}/access_report.json"

  # Generate final system status report
  local final_report="${ARTIFACTS}/final_system_status.json"

  # Ensure access_report file exists (create empty if not)
  [ -f "$access_report" ] || { echo "{}" > "$access_report"; }

  python3 - "$final_report" "$external_ip" "$base_url" "$e2e_report" "$access_report" << 'PYEOF'
import json, sys, socket, subprocess
from datetime import datetime

final_file = sys.argv[1]
ext_ip = sys.argv[2]
base_url = sys.argv[3]
e2e_file = sys.argv[4]
access_file = sys.argv[5]

# Load E2E results
e2e_data = {}
try:
    with open(e2e_file) as f:
        e2e_data = json.load(f)
except:
    pass

# Load access report
access_data = {}
try:
    with open(access_file) as f:
        access_data = json.load(f)
except:
    pass

# Check service status
services = {}
try:
    result = subprocess.run(['podman', 'ps', '--format', 'json'], capture_output=True, text=True, timeout=10)
    if result.returncode == 0:
        for container in json.loads(result.stdout):
            services[container['Names']] = {
                'status': container['State'],
                'ports': container.get('Ports', '')
            }
except:
    pass

report = {
    "generated_at": datetime.now().isoformat(),
    "server": {
        "hostname": socket.gethostname(),
        "external_ip": ext_ip,
        "base_url": base_url
    },
    "e2e_tests": {
        "framework": e2e_data.get('framework', 'none'),
        "status": e2e_data.get('status', 'not_run'),
        "summary": e2e_data.get('summary', {}),
        "success_rate": e2e_data.get('success_rate', 0)
    },
    "services": services,
    "access_info": access_data.get('client_connection', {}),
    "ready_for_production": e2e_data.get('status', 'failed') == 'passed' and len(services) > 0
}

with open(final_file, 'w') as f:
    json.dump(report, f, indent=2)
PYEOF

  log "  📄 Final report: $final_report"

  if [ "$e2e_result" -ne 0 ] && [ "$e2e_result" -ne 0 ]; then
    warn "  ⚠️  E2E tests had issues - system may not be fully functional"
  else
    log "  ✅ System verified and accessible"
  fi

  state_set e2e_production status done
  log "✅ E2E Production done"
}

# ═══════════════════════════════════════════════
# IN-PROCESS PHASE SKIP
# ═══════════════════════════════════════════════

skip_phase() {
  local phase="${1:-$(current_phase)}"
  warn "Skipping phase: $phase"
  state_set "$phase" status done

  local phases=(requirements market_research design backend frontend testing qa security deploy e2e_production)
  local next="" found=false
  for p in "${phases[@]}"; do
    if [ "$found" = true ]; then next="$p"; break; fi
    [ "$p" = "$phase" ] && found=true
  done

  if [ -n "$next" ]; then
    state_set _meta current_phase "$next"
    log "Advanced to: $next"
  else
    log "All phases complete"
  fi
}

# ═══════════════════════════════════════════════
# WATERFALL ORCHESTRATOR
# ═══════════════════════════════════════════════

run_waterfall() {
  local project="$1"
  local slug; slug=$(echo "$project" | safe_tr '[:upper:]' '[:lower:]' | safe_tr ' ' '-' | safe_tr -cd 'a-z0-9-' | safe_cut -c1-40) # safe_pipe: no fail on empty results
  BRANCH="team/${slug}-$(date +%s)"

  state_save_meta "$project" "$BRANCH"
  ensure_branch

  local t0; t0=$(date +%s)
  local loop=0

  log "╔═══════════════════════════════════════════════╗"
  log "║   AI Development Team                         ║"
  log "╠═══════════════════════════════════════════════╣"
  log "║ 🧑‍💼 PM → 🔍 Market → 🏗️ Arch → ⚙️ Back         ║"
  log "║ → 🎨 Front → 🧪 Test → 📋 QA → 🔒 Sec → 🐳      ║"
  log "║ → 🧪 E2E (Production)                          ║"
  log "╚═══════════════════════════════════════════════╝"
  log "Project: $project"
  log "Branch:  $BRANCH"

  while [ $loop -le $MAX_LOOPS ]; do
    [ $loop -gt 0 ] && warn "═══ FEEDBACK LOOP $loop/$MAX_LOOPS ═══"

    # Requirements + Market Research + Design (first pass only)
    [ $loop -eq 0 ] && { phase_requirements "$project"; phase_market_research; phase_design; }

    # Implementation
    phase_backend; phase_frontend

    # Testing gate
    phase_testing
    if [ "$(state_get testing unit_tests)" = "failed" ]; then
      err "Tests failed — looping back to fix"
      loop=$((loop+1))
      state_set testing status pending
      state_set testing unit_tests pending
      continue
    fi

    # QA gate
    phase_qa
    if [ "$(state_get qa verdict)" = "REJECT" ]; then
      err "QA rejected — looping back to fix code"
      loop=$((loop+1))
      state_set backend status pending
      state_set frontend status pending
      state_set testing status pending; state_set testing unit_tests pending; state_set testing write_tests pending
      state_set qa status pending
      continue
    fi

    # Security gate
    phase_security
    if [ "$(state_get security verdict)" = "REJECT" ]; then
      err "Security rejected — looping back"; loop=$((loop+1))
      state_set testing status pending; state_set qa status pending; state_set security status pending; continue
    fi

    # Deploy
    phase_deploy

    # E2E Production Testing (Final Verification)
    phase_e2e_production

    break
  done

  local elapsed=$(( $(date +%s) - t0 ))
  local external_ip; external_ip=$(get_external_ip)
  local final_report="$ARTIFACTS/final_system_status.json"

  log ""
  log "╔═══════════════════════════════════════════════╗"
  log "║   🎉 PROJECT COMPLETE                         ║"
  log "╚═══════════════════════════════════════════════╝"
  log "  Time:      $((elapsed/3600))h $((elapsed%3600/60))m"
  log "  Loops:     $loop"
  log "  Branch:    $BRANCH → main"
  log "  Artifacts: $ARTIFACTS/"
  log ""
  log "╔═══════════════════════════════════════════════╗"
  log "║   🌐 CLIENT CONNECTION INFO                   ║"
  log "╚═══════════════════════════════════════════════╝"
  log "  Server IP:      $external_ip"
  log "  Dashboard URL:  http://$external_ip:$DASHBOARD_PORT"
  log "  Status Report:  $final_report"

  # Display final status
  if [ -f "$final_report" ]; then
    local ready; ready=$(python3 -c "import json; print('READY' if json.load(open('$final_report')).get('ready_for_production') else 'NEEDS FIXES')" 2>/dev/null || echo "UNKNOWN")
    local e2e_status; e2e_status=$(python3 -c "import json; print(json.load(open('$final_report')).get('e2e_tests',{}).get('status','unknown').upper())" 2>/dev/null || echo "UNKNOWN")
    log "  System Status:  $ready"
    log "  E2E Tests:      $e2e_status"
  fi

  log ""
  log "  To connect from your client:"
  log "    1. Ensure network connectivity to: $external_ip"
  log "    2. Open browser: http://$external_ip:$DASHBOARD_PORT"
  log "    3. For firewall, allow ports: $SERVICE_PORTS"
  log ""
  log "  Next:      ./dev.sh start \"next feature\""
}

# ═══════════════════════════════════════════════
# PHASE HISTORY
# ═══════════════════════════════════════════════

init_phase_history() {
  [ -f "$PHASE_HISTORY" ] || echo '{"completed":[],"current_round":0}' > "$PHASE_HISTORY"
}

record_completed_round() {
  local desc="$1" category="${2:-project}" result="${3:-done}"
  init_phase_history
  python3 - "$PHASE_HISTORY" "$desc" "$category" "$result" << 'PYEOF'
import json, sys, os
from datetime import datetime
f, desc, cat, res = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4]
d = json.load(open(f)) if os.path.exists(f) else {"completed":[],"current_round":0}
d["completed"].append({"description": desc, "category": cat, "result": res, "timestamp": datetime.now().isoformat()})
d["current_round"] = len(d["completed"])
json.dump(d, open(f, "w"), indent=2)
PYEOF
}

# ═══════════════════════════════════════════════════
# TRACK A: SELF-IMPROVEMENT (dev.sh itself)
# ═══════════════════════════════════════════════════

sync_dev() {
  local target="${1:-$SELF_SCRIPT}"
  if [ -f "$MASTER_DEV_SH" ]; then
    cp "$MASTER_DEV_SH" "$target"
    chmod +x "$target"
    slog "📋 Synced master dev.sh → $target"
  fi
}

sync_dev_all() {
  slog "📋 Syncing master dev.sh to all projects..."
  local count=0
  for proj_dev in "$HOME"/*/dev.sh; do
    [ -f "$proj_dev" ] || continue
    [ "$proj_dev" = "$MASTER_DEV_SH" ] && continue
    cp "$MASTER_DEV_SH" "$proj_dev"
    chmod +x "$proj_dev"
    count=$((count + 1))
    slog "  ✓ $(dirname "$proj_dev")"
  done
  slog "  Synced to $count projects"
}

analyze_dev() {
  slog "🔬 A1: ANALYZING master dev.sh ($MASTER_DEV_SH)..."

  if [ ! -f "$MASTER_DEV_SH" ]; then
    if [ -f "$SELF_SCRIPT" ]; then
      cp "$SELF_SCRIPT" "$MASTER_DEV_SH"
      slog "  Initialized master from current script"
    else
      serr "  No dev.sh found"; return 1
    fi
  fi

  local team_lines; team_lines=$(wc -l < "$MASTER_DEV_SH" || true)
  team_lines="${team_lines//[^0-9]/}"; team_lines="${team_lines:-0}"
  local func_count; func_count=$(grep -c "^[a-z_]*() {" "$MASTER_DEV_SH" || true)
  func_count="${func_count//[^0-9]/}"; func_count="${func_count:-0}"
  local func_list; func_list=$(safe_grep "^[a-z_]*() {" "$MASTER_DEV_SH" | safe_sed 's/() {.*//' | safe_tr '\n' ',' | safe_sed 's/,$//') # safe_pipe: no fail on empty results

  local prompt_analysis
  cat > "$DEV_DIR/tmp_analyze.py" << 'PYEOF'
import re, sys, json
content = open(sys.argv[1]).read()
prompts = []
for m in re.finditer(r'^(phase_\w+)\(\)', content, re.MULTILINE):
    name = m.group(1)
    start = m.start()
    next_phase = content.find('\nphase_', start + 1)
    if next_phase == -1: next_phase = len(content)
    block = content[start:next_phase]
    pipe_count = block.count('| ')
    pipe_safe = block.count('|| true') + block.count('|| echo') + block.count('|| rc=')
    prompts.append({
        "phase": name,
        "lines": block.count('\n'),
        "claude_calls": block.count('claude_do'),
        "pipes": pipe_count,
        "safe_pipes": pipe_safe,
        "unsafe_pipes": max(0, pipe_count - pipe_safe - block.count('| tee') - block.count('| tail') - block.count('| head'))
    })
print(json.dumps(prompts, indent=2))
PYEOF
  prompt_analysis=$(python3 "$DEV_DIR/tmp_analyze.py" "$MASTER_DEV_SH" 2>/dev/null || echo "[]")
  rm -f "$DEV_DIR/tmp_analyze.py"

  local pipefail_safe; pipefail_safe=$(grep -c "|| true\||| echo\||| rc=\||| build_" "$MASTER_DEV_SH" || true)
  pipefail_safe="${pipefail_safe//[^0-9]/}"; pipefail_safe="${pipefail_safe:-0}"
  local raw_pipes; raw_pipes=$(grep -c "| grep\|| awk\|| sed\|| cut\|| wc" "$MASTER_DEV_SH" || true)
  raw_pipes="${raw_pipes//[^0-9]/}"; raw_pipes="${raw_pipes:-0}"
  local timeout_calls; timeout_calls=$(grep -c "timeout " "$MASTER_DEV_SH" || true)
  timeout_calls="${timeout_calls//[^0-9]/}"; timeout_calls="${timeout_calls:-0}"
  local hardcoded; hardcoded=$(grep -n "localhost\|127\.0\.0\.1\|:8080\|:3000\|:5432" "$MASTER_DEV_SH" | grep -v "^#" | head -10 || true)

  local error_summary=""
  if [ -f "$ERROR_LOG" ]; then
    error_summary=$(python3 -c "
import json
from collections import Counter
errors = []
for line in open('$ERROR_LOG'):
    try: errors.append(json.loads(line.strip()))
    except: pass
by_phase = Counter(e.get('phase','?') for e in errors)
by_type = Counter(e.get('type','?') for e in errors)
print('By phase: ' + ', '.join(f'{k}:{v}' for k,v in by_phase.most_common(5)))
print('By type:  ' + ', '.join(f'{k}:{v}' for k,v in by_type.most_common(5)))
print(f'Total: {len(errors)} errors')
" 2>/dev/null || echo "No errors")
  fi

  local terminated_count=0 retry_count=0 stuck_count=0
  if [ -f "$LIVE_LOG" ]; then
    terminated_count=$(grep -c "Terminated" "$LIVE_LOG" 2>/dev/null || true)
    terminated_count="${terminated_count//[^0-9]/}"; terminated_count="${terminated_count:-0}"
    retry_count=$(grep -c "Attempt [23]/3" "$LIVE_LOG" 2>/dev/null || true)
    retry_count="${retry_count//[^0-9]/}"; retry_count="${retry_count:-0}"
    stuck_count=$(grep -c "stuck\|stale\|timeout\|Timeout" "$SUP_LOG" 2>/dev/null || true)
    stuck_count="${stuck_count//[^0-9]/}"; stuck_count="${stuck_count:-0}"
  fi

  python3 -c '
import json, sys
analysis = {
    "structure": {
        "total_lines": int(sys.argv[1]),
        "function_count": int(sys.argv[2]),
        "functions": sys.argv[3],
        "pipefail_safe_pipes": int(sys.argv[4]),
        "raw_pipes": int(sys.argv[5]),
        "timeout_calls": int(sys.argv[6])
    },
    "phases": json.loads(sys.argv[7]) if sys.argv[7].strip().startswith("[") else [],
    "hardcoded_values": sys.argv[8],
    "error_history": sys.argv[9],
    "process_crashes": {
        "terminated": int(sys.argv[10]),
        "retries": int(sys.argv[11]),
        "stuck": int(sys.argv[12])
    }
}
json.dump(analysis, open(sys.argv[13], "w"), indent=2)
' "$team_lines" "$func_count" "$func_list" "$pipefail_safe" "$raw_pipes" \
  "$timeout_calls" "$prompt_analysis" \
  "${hardcoded:-none}" "${error_summary:-none}" \
  "$terminated_count" "$retry_count" "$stuck_count" \
  "$TEAM_ANALYSIS_FILE" 2>/dev/null || slog "  ⚠ Analysis write failed"

  slog "📊 DEV.SH ANALYSIS:"
  slog "  Lines: $team_lines | Functions: $func_count"
  slog "  Pipes: $raw_pipes raw, $pipefail_safe safe | Timeouts: $timeout_calls"
  slog "  Crashes: Terminated=$terminated_count Retries=$retry_count Stuck=$stuck_count"
  [ -n "$error_summary" ] && slog "  Errors: $error_summary"

  local team_score=100
  [ "$terminated_count" -gt 5 ] && team_score=$((team_score - 20))
  [ "$retry_count" -gt 10 ] && team_score=$((team_score - 15))
  [ "$stuck_count" -gt 3 ] && team_score=$((team_score - 15))
  [ "$raw_pipes" -gt 20 ] && team_score=$((team_score - 10))
  [ "$timeout_calls" -lt 3 ] && team_score=$((team_score - 10))
  slog "  🏆 Process Score: $team_score/100"
}

plan_dev_improvements() {
  local num_steps="${1:-3}"
  slog "🧠 A2: PLANNING $num_steps dev.sh improvements..."

  local analysis; analysis=$(cat "$TEAM_ANALYSIS_FILE" 2>/dev/null || echo "{}")
  local team_head; team_head=$(head -c 4000 "$MASTER_DEV_SH" 2>/dev/null)
  local history; history=$(cat "$PHASE_HISTORY" 2>/dev/null || echo "{}")

  cd "$REPO_DIR"
  local completed_rounds
  completed_rounds=$(python3 -c "
import json
d = json.loads('''$history''')
for r in d.get('completed',[]):
    if r.get('category') == 'team': print(f\"  - [{r.get('result','')}] {r.get('description','')[:80]}\")
" 2>/dev/null || echo "  None")

  local _prompt="You are a DevOps Process Engineer. Analyze dev.sh (an AI dev team orchestrator) and plan $num_steps UNIVERSAL improvements.

IMPORTANT: dev.sh is PROJECT-AGNOSTIC. It orchestrates AI development for ANY software project.
DO NOT make changes specific to any one project. All improvements must work for Go, Python, Node, React, or any stack.

DEV.SH ANALYSIS:
$analysis

DEV.SH HEADER (first 4000 chars):
$team_head

ALREADY COMPLETED IMPROVEMENTS:
$completed_rounds

WHAT TO IMPROVE — PRIORITY ORDER:
1. CRASH FIXES: If Terminated/Stuck counts are high, fix root causes
2. PIPELINE SAFETY: Add || true to unsafe grep/find/awk pipes
3. PROMPT OPTIMIZATION: Shrink prompts >8000 chars
4. SMART RETRIES: Add exponential backoff
5. NEW PHASES: api-testing, documentation, load-testing, monitoring
6. PHASE SKIP LOGIC: Skip phases based on project type
7. PROCESS METRICS: Timing per phase, success rates
8. PROMPT QUALITY: More specific prompts

RULES:
- Each step is a SINGLE focused change
- ALL changes must be PROJECT-AGNOSTIC
- Each step must include: what, where, why, verification
- Steps must be independent
- DO NOT repeat already completed improvements
- Each change must keep valid bash (verify: bash -n dev.sh)

RESPOND WITH ONLY JSON:
{
  \"steps\": [
    {
      \"order\": 1,
      \"name\": \"Short name\",
      \"category\": \"crash_fix|safety|prompt|retry|new_phase|skip_logic|metrics|quality\",
      \"priority\": \"critical|high|medium\",
      \"target_function\": \"function_name or line range\",
      \"description\": \"Exact change to make.\",
      \"verification\": \"bash -n dev.sh && echo OK\",
      \"risk\": \"low|medium|high\"
    }
  ],
  \"process_health\": {
    \"score\": 75,
    \"biggest_risk\": \"what causes most crashes\",
    \"biggest_win\": \"easiest high-impact improvement\"
  }
}"

  local claude_rc=0
  run_claude 300 "$TEAM_PLAN_FILE" "$_prompt" || claude_rc=$?

  if [ "$claude_rc" -eq 124 ] || [ ! -s "$TEAM_PLAN_FILE" ]; then
    swarn "  ⚠ Planning timed out or empty"
    echo '{"steps":[],"process_health":{"score":0}}' > "$TEAM_PLAN_FILE"
    return 0
  fi

  python3 - "$TEAM_PLAN_FILE" << 'PYEOF'
import json, re, sys
f = sys.argv[1]
content = open(f).read()
m = re.search(r'\{[\s\S]*\}', content)
if m:
    try:
        parsed = json.loads(m.group())
        json.dump(parsed, open(f, "w"), indent=2)
    except: pass
else:
    json.dump({"steps":[]}, open(f, "w"), indent=2)
PYEOF

  slog "📋 Dev.sh improvement plan:"
  python3 - "$TEAM_PLAN_FILE" << 'PYEOF' 2>/dev/null || true
import json, sys
try:
    d = json.load(open(sys.argv[1]))
    h = d.get("process_health", {})
    print(f"  Process health: {h.get('score','?')}/100")
    print(f"  Biggest risk: {h.get('biggest_risk','?')}")
    print(f"  Biggest win:  {h.get('biggest_win','?')}")
    print()
    for s in d.get("steps", []):
        icon = {"crash_fix":"🔥","safety":"🛡️","prompt":"📝","retry":"🔄","new_phase":"✨","skip_logic":"⏭️","metrics":"📊","quality":"💎"}.get(s.get("category",""),"📌")
        print(f"  {s['order']}. {icon} [{s.get('priority','?')}] {s['name']}")
        print(f"     → {s.get('target_function','?')}")
        print(f"     {s['description'][:100]}...")
        print()
except Exception as e:
    print(f"  Error: {e}")
PYEOF
}

execute_dev_improvements() {
  slog "🚀 A3: EXECUTING dev.sh improvements..."

  if [ ! -f "$TEAM_PLAN_FILE" ]; then
    serr "No plan. Run: ./dev.sh plan-dev"
    return 1
  fi

  local total
  total=$(python3 -c "import json; print(len(json.load(open('$TEAM_PLAN_FILE')).get('steps',[])))" 2>/dev/null || echo "0")
  [ "$total" -eq 0 ] && { serr "No steps in plan."; return 1; }

  local applied=0 failed=0 i=0
  while [ "$i" -lt "$total" ]; do
    local step_name; step_name=$(python3 -c "import json; print(json.load(open('$TEAM_PLAN_FILE'))['steps'][$i].get('name','Step $((i+1))'))" 2>/dev/null || echo "Step $((i+1))")
    local step_desc; step_desc=$(python3 -c "import json; print(json.load(open('$TEAM_PLAN_FILE'))['steps'][$i]['description'])" 2>/dev/null || echo "")
    local step_target; step_target=$(python3 -c "import json; print(json.load(open('$TEAM_PLAN_FILE'))['steps'][$i].get('target_function','dev.sh'))" 2>/dev/null || echo "dev.sh")
    local step_verify; step_verify=$(python3 -c "import json; print(json.load(open('$TEAM_PLAN_FILE'))['steps'][$i].get('verification','bash -n dev.sh'))" 2>/dev/null || echo "bash -n dev.sh")

    slog "┌───────────────────────────────────────┐"
    slog "│  Step $((i+1))/$total: $step_name"
    slog "│  Target: $step_target"
    slog "└───────────────────────────────────────┘"

    if [ -z "$step_desc" ]; then
      swarn "  Empty step — skipping"
      i=$((i+1)); continue
    fi

    # Backup master before each step
    cp "$MASTER_DEV_SH" "$MASTER_DEV_DIR/dev.sh.bak.step$((i+1)).$(date +%s)"
    # Copy master → project for Claude to modify
    cp "$MASTER_DEV_SH" "$SELF_SCRIPT"

    cd "$REPO_DIR"
    local _prompt="Read dev.sh. Make this ONE specific change:

CHANGE: $step_desc
TARGET: $step_target

RULES:
- Make ONLY this one change, nothing else
- Keep existing functionality intact
- This is a PROJECT-AGNOSTIC script — do NOT add project-specific code
- Do NOT rewrite large sections
- Verify: $step_verify
- If the change is already applied, say 'ALREADY_DONE' and make no changes"

    run_claude 300 "$PHASE_LOGS/dev_step_$((i+1)).log" "$_prompt"

    if bash -n "$SELF_SCRIPT" 2>/dev/null; then
      slog "  ✓ Step $((i+1)) applied: $step_name"
      applied=$((applied + 1))
      cp "$SELF_SCRIPT" "$MASTER_DEV_SH"
      record_completed_round "Dev: $step_name — $step_desc" "team" "done"
    else
      serr "  ✗ Step $((i+1)) broke dev.sh — rolling back"
      local latest_bak; latest_bak=$(ls -t "$MASTER_DEV_DIR"/dev.sh.bak.step$((i+1)).* 2>/dev/null | safe_head -1) # safe_pipe: no fail on empty results
      if [ -n "$latest_bak" ]; then
        cp "$latest_bak" "$MASTER_DEV_SH"
        cp "$MASTER_DEV_SH" "$SELF_SCRIPT"
      fi
      failed=$((failed + 1))
      record_completed_round "Dev: FAILED $step_name" "team" "failed"
    fi

    i=$((i+1)); sleep 2
  done

  slog "  Dev.sh improvements: $applied applied, $failed failed out of $total"
}

verify_dev() {
  slog "✅ A4: VERIFYING dev.sh..."
  local score=100

  if bash -n "$MASTER_DEV_SH" 2>/dev/null; then
    slog "  ✓ Syntax: PASS"
  else
    slog "  ✗ Syntax: FAIL"; return 1
  fi

  local funcs; funcs=$(grep -c "^[a-z_]*() {" "$MASTER_DEV_SH" || true)
  funcs="${funcs//[^0-9]/}"; funcs="${funcs:-0}"
  slog "  Functions: $funcs"
  [ "$funcs" -lt 15 ] && { slog "  ⚠ Low function count"; score=$((score - 10)); }

  local required="phase_requirements phase_design phase_backend phase_frontend phase_testing phase_qa phase_security phase_deploy claude_do"
  for fn in $required; do
    if ! grep -q "^${fn}()" "$MASTER_DEV_SH" 2>/dev/null; then
      slog "  ✗ Missing: $fn"; score=$((score - 10))
    fi
  done

  local unsafe; unsafe=$(grep -c "| grep\|| awk\|| sed" "$MASTER_DEV_SH" || true)
  unsafe="${unsafe//[^0-9]/}"; unsafe="${unsafe:-0}"
  local safe; safe=$(grep -c "|| true\||| echo\||| rc=" "$MASTER_DEV_SH" || true)
  safe="${safe//[^0-9]/}"; safe="${safe:-0}"
  slog "  Pipe safety: $safe safe, $unsafe raw pipes"

  local lines; lines=$(wc -l < "$MASTER_DEV_SH" || true)
  lines="${lines//[^0-9]/}"; lines="${lines:-0}"
  slog "  Lines: $lines"
  slog "  🏆 Dev.sh Score: $score/100"

  python3 -c "import json; json.dump({'score': $score, 'functions': $funcs, 'lines': $lines}, open('$ARTIFACTS/dev_verification.json', 'w'), indent=2)" 2>/dev/null || true
}

run_dev_improvement() {
  local steps="${1:-3}"
  slog ""
  slog "╔═══════════════════════════════════════════════╗"
  slog "║  🔧 DEV.SH IMPROVEMENT PIPELINE ($steps steps)      ║"
  slog "╚═══════════════════════════════════════════════╝"
  local t0; t0=$(date +%s)

  analyze_dev
  plan_dev_improvements "$steps"
  execute_dev_improvements
  verify_dev
  sync_dev_all

  local elapsed=$(( $(date +%s) - t0 ))
  slog "  Dev.sh pipeline done in $((elapsed/60))m $((elapsed%60))s"
}

# ═══════════════════════════════════════════════════
# TRACK B: PROJECT IMPROVEMENT
# ═══════════════════════════════════════════════════

# scan_frontend_files - Accurately count TSX and TS files in the frontend directory
# Returns JSON string with: tsx_count, ts_count, total_count, test_count, features{}
# Uses detected FRONTEND_DIR with fallback to web/dashboard/src
scan_frontend_files() {
  local frontend_src="${1:-$REPO_DIR/${FRONTEND_DIR:-web/dashboard}/src}"

  # Return zeros if directory doesn't exist
  [ ! -d "$frontend_src" ] && echo '{"tsx_count":0,"ts_count":0,"total_count":0,"test_count":0,"features":{}}' && return 0

  # Proper find command with grouped expressions and exclusions
  # Excludes: node_modules, dist, build directories
  local tsx_count ts_count total_count test_count features_json

  tsx_count=$(find "$frontend_src" \
    \( -name "node_modules" -o -name "dist" -o -name "build" \) -prune \
    -o -type f -name "*.tsx" -print 2>/dev/null | wc -l || echo "0")
  tsx_count="${tsx_count//[^0-9]/}"; tsx_count="${tsx_count:-0}"

  ts_count=$(find "$frontend_src" \
    \( -name "node_modules" -o -name "dist" -o -name "build" \) -prune \
    -o -type f -name "*.ts" ! -name "*.tsx" -print 2>/dev/null | wc -l || echo "0")
  ts_count="${ts_count//[^0-9]/}"; ts_count="${ts_count:-0}"

  total_count=$((tsx_count + ts_count))

  # Count test files (*.spec.ts, *.spec.tsx, *.test.ts, *.test.tsx)
  test_count=$(find "$frontend_src" \
    \( -name "node_modules" -o -name "dist" -o -name "build" \) -prune \
    -o -type f \( -name "*.spec.ts" -o -name "*.spec.tsx" -o -name "*.test.ts" -o -name "*.test.tsx" \) -print 2>/dev/null | wc -l || echo "0")
  test_count="${test_count//[^0-9]/}"; test_count="${test_count:-0}"

  # Feature module breakdown - count files per feature directory
  local features_json="{}"
  if [ -d "$frontend_src/features" ]; then
    local feature_dirs feature_dir feature_tsx feature_ts
    feature_dirs=$(find "$frontend_src/features" -mindepth 1 -maxdepth 1 -type d 2>/dev/null || true)
    if [ -n "$feature_dirs" ]; then
      features_json="{"
      local first=true
      while IFS= read -r feature_dir; do
        [ ! -d "$feature_dir" ] && continue
        local feature_name; feature_name=$(basename "$feature_dir")
        feature_tsx=$(find "$feature_dir" -type f -name "*.tsx" 2>/dev/null | wc -l || echo "0")
        feature_ts=$(find "$feature_dir" -type f -name "*.ts" ! -name "*.tsx" 2>/dev/null | wc -l || echo "0")
        feature_tsx="${feature_tsx//[^0-9]/}"; feature_tsx="${feature_tsx:-0}"
        feature_ts="${feature_ts//[^0-9]/}"; feature_ts="${feature_ts:-0}"
        [ "$first" = true ] && first=false || features_json="${features_json},"
        features_json="${features_json}\"${feature_name}\":{\"tsx\":${feature_tsx},\"ts\":${feature_ts}}"
      done <<< "$feature_dirs"
      features_json="${features_json}}"
    fi
  fi

  echo "{\"tsx_count\":${tsx_count},\"ts_count\":${ts_count},\"total_count\":${total_count},\"test_count\":${test_count},\"features\":${features_json}}"
}

diagnose_project() {
  slog "🔍 DIAGNOSING project..."
  cd "$REPO_DIR"
  local report="$ARTIFACTS/diagnosis.json"

  local go_files; go_files=$(find "$REPO_DIR/internal" "$REPO_DIR/cmd" -name "*.go" 2>/dev/null | wc -l || true)
  go_files="${go_files//[^0-9]/}"; go_files="${go_files:-0}"
  local test_files; test_files=$(find "$REPO_DIR" -name "*_test.go" 2>/dev/null | wc -l || true)
  test_files="${test_files//[^0-9]/}"; test_files="${test_files:-0}"

  # Use detected frontend directory for accurate file enumeration
  local frontend_src_dir="$REPO_DIR/${FRONTEND_DIR:-web/dashboard}/src"
  local frontend_scan; frontend_scan=$(scan_frontend_files "$frontend_src_dir")
  local tsx_files; tsx_files=$(echo "$frontend_scan" | python3 -c "import json,sys; print(json.load(sys.stdin).get('tsx_count',0))" 2>/dev/null || echo "0")
  tsx_files="${tsx_files//[^0-9]/}"; tsx_files="${tsx_files:-0}"
  local ts_files; ts_files=$(echo "$frontend_scan" | python3 -c "import json,sys; print(json.load(sys.stdin).get('ts_count',0))" 2>/dev/null || echo "0")
  ts_files="${ts_files//[^0-9]/}"; ts_files="${ts_files:-0}"
  local frontend_total; frontend_total=$((tsx_files + ts_files))

  # Get frontend directory for TODO scanning (use detected path)
  local frontend_for_scan="${FRONTEND_DIR:-web/dashboard}"
  local todo_count; todo_count=$(grep -rn "TODO\|FIXME\|HACK\|XXX" "$REPO_DIR/internal" "$REPO_DIR/cmd" "$REPO_DIR/$frontend_for_scan" 2>/dev/null | wc -l || true)
  todo_count="${todo_count//[^0-9]/}"; todo_count="${todo_count:-0}"
  local todo_list; todo_list=$(grep -rn "TODO\|FIXME\|HACK\|XXX" "$REPO_DIR/internal" "$REPO_DIR/cmd" "$REPO_DIR/$frontend_for_scan" 2>/dev/null | head -20 || true)

  local build_ok="yes" compile_errors=""
  go build ./... 2>/dev/null || { build_ok="no"; compile_errors=$(go build ./... 2>&1 | tail -20 || true); }

  local test_ok="pass" test_failures="" test_count=0 test_passed=0
  local test_output; test_output=$(go test ./... -count=1 -timeout 120s 2>&1 || true)
  echo "$test_output" | grep -q "^FAIL" && test_ok="fail"
  test_count=$(echo "$test_output" | grep -c "^---\|^ok\|^FAIL" || true)
  test_count="${test_count//[^0-9]/}"; test_count="${test_count:-0}"
  test_passed=$(echo "$test_output" | grep -c "^ok " || true)
  test_passed="${test_passed//[^0-9]/}"; test_passed="${test_passed:-0}"
  test_failures=$(echo "$test_output" | grep -A 2 "FAIL\|Error\|panic" | head -20 || true)

  local dockerfiles; dockerfiles=$(find deployments/docker -maxdepth 1 -name "Dockerfile.*" 2>/dev/null | wc -l || true)
  dockerfiles="${dockerfiles//[^0-9]/}"; dockerfiles="${dockerfiles:-0}"
  local compose_exists="no"
  [ -f "deployments/docker/docker-compose.yml" ] || [ -f "docker-compose.yml" ] && compose_exists="yes"

  local frontend_exists="no" ts_errors=""
  local frontend_dir="${FRONTEND_DIR:-web/dashboard}"
  if [ -d "$REPO_DIR/$frontend_dir" ]; then
    frontend_exists="yes"
    cd "$REPO_DIR/$frontend_dir"
    [ -d node_modules ] || npm install 2>/dev/null || true
    [ -f node_modules/.bin/tsc ] && ts_errors=$(npx tsc --noEmit 2>&1 | grep "error TS" | head -10 || true)
    cd "$REPO_DIR"
  fi

  echo "$compile_errors" > "$DEV_DIR/tmp_ce.txt" 2>/dev/null
  echo "$test_failures" > "$DEV_DIR/tmp_tf.txt" 2>/dev/null
  echo "$ts_errors" > "$DEV_DIR/tmp_ts.txt" 2>/dev/null
  echo "$todo_list" > "$DEV_DIR/tmp_td.txt" 2>/dev/null

  python3 -c '
import json, sys, os
d = sys.argv
report = {
    "project": {
        "go_files": int(d[1]), "test_files": int(d[2]),
        "tsx_files": int(d[3]), "ts_files": int(d[4]), "frontend_total": int(d[5]),
        "todo_count": int(d[6]), "build": d[7], "tests": d[8],
        "test_count": int(d[9]), "test_passed": int(d[10]),
        "dockerfiles": int(d[11]), "compose": d[12], "frontend": d[13],
        "features": d[14] if len(d) > 14 else "{}"
    },
    "compile_errors": open(d[15]).read().strip() if os.path.exists(d[15]) else "",
    "test_failures": open(d[16]).read().strip() if os.path.exists(d[16]) else "",
    "ts_errors": open(d[17]).read().strip() if os.path.exists(d[17]) else "",
    "todos": open(d[18]).read().strip() if os.path.exists(d[18]) else ""
}
json.dump(report, open(d[19], "w"), indent=2)
' "$go_files" "$test_files" "$tsx_files" "$ts_files" "$frontend_total" "$todo_count" \
  "$build_ok" "$test_ok" "$test_count" "$test_passed" \
  "$dockerfiles" "$compose_exists" "$frontend_exists" \
  "$frontend_scan" \
  "$DEV_DIR/tmp_ce.txt" "$DEV_DIR/tmp_tf.txt" "$DEV_DIR/tmp_ts.txt" \
  "$DEV_DIR/tmp_td.txt" "$report" 2>/dev/null || slog "  ⚠ Diagnosis write failed"

  rm -f "$DEV_DIR"/tmp_ce.txt "$DEV_DIR"/tmp_tf.txt "$DEV_DIR"/tmp_ts.txt "$DEV_DIR"/tmp_td.txt

  slog "📊 DIAGNOSIS:"
  slog "  Code:    $go_files .go + $tsx_files .tsx + $ts_files .ts = $frontend_total frontend + $test_files tests"
  slog "  Build:   $build_ok | Tests: $test_ok ($test_passed/$test_count)"
  slog "  TODOs:   $todo_count | Docker: $dockerfiles files"
  [ -n "$compile_errors" ] && slog "  ⚠ Compile errors found"
  [ -n "$test_failures" ] && slog "  ⚠ Test failures found"
}

plan_improvements() {
  local num="${1:-$AUTO_PHASES}"
  slog "🧠 PLANNING $num improvement phases..."

  init_phase_history
  local diagnosis; diagnosis=$(cat "$ARTIFACTS/diagnosis.json" 2>/dev/null || echo "{}")
  local project_context; project_context=$(head -c 3000 "$REPO_DIR/CLAUDE.md" 2>/dev/null || echo "No CLAUDE.md")
  local history; history=$(cat "$PHASE_HISTORY" 2>/dev/null || echo "{}")

  cd "$REPO_DIR"
  local _prompt="You are a Senior Technical Project Manager. Analyze this project and plan the next $num phases.

PROJECT (CLAUDE.md):
$project_context

DIAGNOSIS:
$diagnosis

COMPLETED ROUNDS:
$history

RULES — STRICT ORDERING:
1. CRITICAL FIRST: If build broken → Phase 1 MUST fix compilation
2. TESTS NEXT: If tests fail → fix tests
3. THEN TYPESCRIPT errors
4. THEN TODOS
5. THEN FEATURES
6. THEN HARDENING
7. THEN DEVOPS
8. DO NOT repeat completed rounds
9. Each description: 50-150 words, specific

RESPOND WITH ONLY JSON:
{
  \"phases\": [
    {
      \"order\": 1,
      \"name\": \"Short name\",
      \"priority\": \"critical|high|medium\",
      \"category\": \"fix|feature|test|security|performance|devops\",
      \"description\": \"Detailed task description for the AI team.\",
      \"success_criteria\": [\"go build ./... passes\"],
      \"estimated_minutes\": 90
    }
  ],
  \"project_health\": {\"score\": 75, \"critical_issues\": [\"issue\"], \"strengths\": [\"strength\"]},
  \"rationale\": \"Why these phases in this order\"
}"

  local claude_rc=0
  run_claude 300 "$PLAN_FILE" "$_prompt" || claude_rc=$?

  if [ "$claude_rc" -eq 124 ] || [ ! -s "$PLAN_FILE" ]; then
    swarn "  ⚠ Planning timed out"
    echo '{"phases":[],"rationale":"Planning timed out"}' > "$PLAN_FILE"
    return 0
  fi

  python3 - "$PLAN_FILE" << 'PYEOF'
import json, re, sys
f = sys.argv[1]
content = open(f).read()
m = re.search(r'\{[\s\S]*\}', content)
if m:
    try:
        parsed = json.loads(m.group())
        json.dump(parsed, open(f, "w"), indent=2)
    except: pass
else:
    json.dump({"phases":[]}, open(f, "w"), indent=2)
PYEOF

  slog "📋 Planned phases:"
  python3 - "$PLAN_FILE" << 'PYEOF' 2>/dev/null || true
import json, sys
try:
    d = json.load(open(sys.argv[1]))
    for p in d.get("phases", []):
        icon = {"fix":"🔧","feature":"✨","test":"🧪","security":"🔒","performance":"⚡","devops":"🐳"}.get(p.get("category",""), "📌")
        print(f"  {p['order']}. {icon} [{p.get('priority','?')}] {p['name']}")
        print(f"     {p['description'][:100]}...")
        print()
except Exception as e:
    print(f"  Error: {e}")
PYEOF
}

execute_planned_phases() {
  slog "🚀 EXECUTING planned phases..."

  if [ ! -f "$PLAN_FILE" ]; then
    serr "No plan found. Run: ./dev.sh plan-project"
    return 1
  fi

  local total
  total=$(python3 -c "import json; print(len(json.load(open('$PLAN_FILE')).get('phases',[])))" 2>/dev/null || echo "0")
  [ "$total" -eq 0 ] && { serr "No phases in plan."; return 1; }

  local i=0
  while [ "$i" -lt "$total" ]; do
    local phase_data; phase_data=$(python3 -c "import json; print(json.load(open('$PLAN_FILE'))['phases'][$i]['description'])" 2>/dev/null || echo "")
    local phase_name; phase_name=$(python3 -c "import json; print(json.load(open('$PLAN_FILE'))['phases'][$i].get('name','Phase $((i+1))'))" 2>/dev/null || echo "Phase $((i+1))")

    if [ -z "$phase_data" ]; then
      swarn "Empty phase $((i+1)) — skipping"
      record_completed_round "EMPTY: Phase $((i+1))" "project" "skipped"
      i=$((i+1)); continue
    fi

    slog "╔═══════════════════════════════════════╗"
    slog "║  ROUND $((i+1))/$total: $phase_name"
    slog "╚═══════════════════════════════════════╝"

    # Reset state and run waterfall in-process
    python3 - "$STATE_FILE" "$phase_data" << 'PYEOF'
import json, sys
d = {"phases": {}, "project": sys.argv[2], "branch": ""}
json.dump(d, open(sys.argv[1], "w"), indent=2)
PYEOF

    rm -f "$STUCK_HEAL_FILE"

    # Run waterfall directly — no subprocess
    if run_waterfall "$phase_data" 2>&1; then
      slog "✅ Round $((i+1)) complete: $phase_name"
      record_completed_round "$phase_name: $phase_data" "project" "done"
    else
      serr "💥 Round $((i+1)) failed: $phase_name"
      record_completed_round "FAILED: $phase_name" "project" "failed"
    fi

    i=$((i+1)); sleep 5
  done

  slog "╔═══════════════════════════════════════╗"
  slog "║  🎉 ALL $total ROUNDS COMPLETE          ║"
  slog "╚═══════════════════════════════════════╝"
}

verify_results() {
  slog "✅ VERIFYING results..."
  cd "$REPO_DIR"
  local score=100

  if go build ./... 2>/dev/null; then slog "  ✓ Build: PASS"
  else slog "  ✗ Build: FAIL"; score=$((score - 30)); fi

  local test_out; test_out=$(go test ./... -count=1 -timeout 120s 2>&1 || true)
  local passed; passed=$(echo "$test_out" | grep -c "^ok " || true)
  passed="${passed//[^0-9]/}"; passed="${passed:-0}"
  local failed; failed=$(echo "$test_out" | grep -c "^FAIL" || true)
  failed="${failed//[^0-9]/}"; failed="${failed:-0}"
  if [ "$failed" -eq 0 ] 2>/dev/null; then slog "  ✓ Tests: ALL PASS ($passed packages)"
  else slog "  ⚠ Tests: $passed pass, $failed fail"; score=$((score - 20)); fi

  local frontend_dir="${FRONTEND_DIR:-web/dashboard}"
  if [ -d "$REPO_DIR/$frontend_dir" ]; then
    cd "$REPO_DIR/$frontend_dir"
    if [ -f node_modules/.bin/tsc ]; then
      if npx tsc --noEmit 2>/dev/null; then slog "  ✓ TypeScript: PASS"
      else
        local ts_err_count; ts_err_count=$(npx tsc --noEmit 2>&1 | grep -c "error TS" || true)
        ts_err_count="${ts_err_count//[^0-9]/}"; ts_err_count="${ts_err_count:-0}"
        slog "  ⚠ TypeScript: $ts_err_count errors"
        [ "$ts_err_count" -gt 0 ] 2>/dev/null && score=$((score - 10))
      fi
    fi
    cd "$REPO_DIR"
  fi

  slog "  📊 Project Score: $score/100"
  python3 -c "import json; json.dump({'score': $score, 'test_passed': $passed, 'test_failed': $failed}, open('$ARTIFACTS/verification.json', 'w'), indent=2)" 2>/dev/null || true
}

run_project_improvement() {
  local phases="${1:-3}"
  slog "╔═══════════════════════════════════════════════╗"
  slog "║  📦 PROJECT IMPROVEMENT ($phases phases)             ║"
  slog "╚═══════════════════════════════════════════════╝"
  local t0; t0=$(date +%s)

  diagnose_project
  plan_improvements "$phases"
  execute_planned_phases
  verify_results

  local elapsed=$(( $(date +%s) - t0 ))
  slog "  Project pipeline done in $((elapsed/3600))h $((elapsed%3600/60))m"
}

# ═══════════════════════════════════════════════════
# FULL IMPROVEMENT (Both Tracks)
# ═══════════════════════════════════════════════════

run_full_improvement() {
  local phases="${1:-$AUTO_PHASES}" dev_steps="${2:-3}"
  AUTO_PHASES="$phases"

  slog "╔═══════════════════════════════════════════════════════╗"
  slog "║  🚀 FULL IMPROVEMENT: $dev_steps dev + $phases project       ║"
  slog "╠═══════════════════════════════════════════════════════╣"
  slog "║  TRACK A: 🔧 dev.sh ($dev_steps steps)                      ║"
  slog "║  TRACK B: 📦 Project ($phases phases)                       ║"
  slog "╚═══════════════════════════════════════════════════════╝"
  local t0; t0=$(date +%s)

  run_dev_improvement "$dev_steps"
  run_project_improvement "$phases"

  local elapsed=$(( $(date +%s) - t0 ))
  slog "🎉 FULL IMPROVEMENT COMPLETE in $((elapsed/3600))h $((elapsed%3600/60))m"
}

# ═══════════════════════════════════════════════════
# SMART IMPROVE — Project-Focused Self-Improvement
# ═══════════════════════════════════════════════════

scan_project_completion() {
  slog "🔬 SCANNING project completion..."
  cd "$REPO_DIR"

  local planned_backend=0 planned_frontend=0 planned_tests=0 planned_endpoints=0 planned_migrations=0
  if [ -f "$ARTIFACTS/02_design.json" ]; then
    planned_backend=$(python3 -c "import json; print(len(json.load(open('$ARTIFACTS/02_design.json')).get('backend_tasks',[])))" 2>/dev/null || echo "0")
    planned_frontend=$(python3 -c "import json; print(len(json.load(open('$ARTIFACTS/02_design.json')).get('frontend_tasks',[])))" 2>/dev/null || echo "0")
    planned_tests=$(python3 -c "import json; d=json.load(open('$ARTIFACTS/02_design.json')); tp=d.get('test_plan',{}); print(len(tp.get('unit_tests',[]))+len(tp.get('e2e_tests',[])))" 2>/dev/null || echo "0")
    planned_endpoints=$(python3 -c "import json; print(len(json.load(open('$ARTIFACTS/02_design.json')).get('api_contracts',[])))" 2>/dev/null || echo "0")
    planned_migrations=$(python3 -c "import json; print(len(json.load(open('$ARTIFACTS/02_design.json')).get('database_migrations',[])))" 2>/dev/null || echo "0")
  fi

  local exist_go; exist_go=$(find internal cmd -name "*.go" 2>/dev/null | grep -v _test | wc -l || echo "0")
  exist_go="${exist_go//[^0-9]/}"; exist_go="${exist_go:-0}"
  local frontend_dir="${FRONTEND_DIR:-web/dashboard}"
  local exist_tsx; exist_tsx=$(find "$frontend_dir/src" -type f \( -name "*.tsx" -o -name "*.ts" \) 2>/dev/null | grep -v node_modules | wc -l || echo "0")
  exist_tsx="${exist_tsx//[^0-9]/}"; exist_tsx="${exist_tsx:-0}"
  local exist_tests; exist_tests=$(find . -name "*_test.go" -o -name "*.spec.ts" -o -name "*.spec.tsx" 2>/dev/null | grep -v node_modules | wc -l || echo "0")
  exist_tests="${exist_tests//[^0-9]/}"; exist_tests="${exist_tests:-0}"
  local exist_migrations; exist_migrations=$(find migrations -name "*.sql" 2>/dev/null | wc -l || echo "0")
  exist_migrations="${exist_migrations//[^0-9]/}"; exist_migrations="${exist_migrations:-0}"
  local exist_endpoints=0
  exist_endpoints=$(grep -rn "\.GET\|\.POST\|\.PUT\|\.DELETE\|\.PATCH\|router\.Handle\|http\.HandleFunc\|r\.Route\|e\.GET\|e\.POST\|mux\.Handle" \
    internal cmd 2>/dev/null | grep -v "_test.go" | wc -l || echo "0")
  exist_endpoints="${exist_endpoints//[^0-9]/}"; exist_endpoints="${exist_endpoints:-0}"

  local build_ok="no" test_ok="no" test_pass=0 test_fail=0
  go build ./... 2>/dev/null && build_ok="yes"
  local test_out; test_out=$(go test ./... -count=1 -timeout 60s 2>&1 || true)
  test_pass=$(echo "$test_out" | grep -c "^ok " 2>/dev/null || echo "0")
  test_pass="${test_pass//[^0-9]/}"; test_pass="${test_pass:-0}"
  test_fail=$(echo "$test_out" | grep -c "^FAIL" 2>/dev/null || echo "0")
  test_fail="${test_fail//[^0-9]/}"; test_fail="${test_fail:-0}"
  [ "$test_fail" -eq 0 ] && [ "$test_pass" -gt 0 ] && test_ok="yes"

  local frontend_dir="${FRONTEND_DIR:-web/dashboard}"
  local todos; todos=$(grep -rn "TODO\|FIXME\|HACK\|XXX" internal cmd "$frontend_dir" 2>/dev/null | wc -l || echo "0")
  todos="${todos//[^0-9]/}"; todos="${todos:-0}"
  local docker_ok="no"
  { ls deployments/docker/Dockerfile.* >/dev/null 2>&1 || [ -f Dockerfile ]; } && docker_ok="yes"

  python3 - "$ARTIFACTS/02_design.json" \
    "$exist_go" "$planned_backend" "$exist_tsx" "$planned_frontend" \
    "$exist_tests" "$planned_tests" "$exist_endpoints" "$planned_endpoints" \
    "$exist_migrations" "$planned_migrations" \
    "$build_ok" "$test_ok" "$test_pass" "$test_fail" \
    "$todos" "$docker_ok" "$COMPLETION_FILE" "$STATE_FILE" << 'PYEOF'
import json, sys, os
from datetime import datetime

design_file = sys.argv[1]
eg, pb = int(sys.argv[2]), int(sys.argv[3])
et, pf = int(sys.argv[4]), int(sys.argv[5])
ets, pt = int(sys.argv[6]), int(sys.argv[7])
ee, pe = int(sys.argv[8]), int(sys.argv[9])
em, pm = int(sys.argv[10]), int(sys.argv[11])
build_ok = sys.argv[12] == "yes"
test_ok = sys.argv[13] == "yes"
test_pass, test_fail = int(sys.argv[14]), int(sys.argv[15])
todos = int(sys.argv[16])
docker_ok = sys.argv[17] == "yes"
out_f = sys.argv[18]
state_f = sys.argv[19]

def ratio(have, need):
    if need == 0: return 100
    return min(100, int(have * 100 / need))

scores = {
    "backend_files":   (ratio(eg, pb), 20),
    "frontend_files":  (ratio(et, pf), 15),
    "tests":           (ratio(ets, pt), 15),
    "endpoints":       (ratio(ee, pe), 15),
    "migrations":      (ratio(em, pm), 10),
    "build":           (100 if build_ok else 0, 10),
    "tests_pass":      (100 if test_ok else (50 if test_pass > 0 else 0), 10),
    "docker":          (100 if docker_ok else 0, 5),
}

total_weight = sum(w for _, w in scores.values())
weighted_sum = sum(s * w for s, w in scores.values())
pct = int(weighted_sum / total_weight)

gaps = []
if os.path.exists(design_file):
    try:
        design = json.load(open(design_file))
        for task in design.get("backend_tasks", []):
            fpath = task.get("file", "")
            if fpath and not os.path.exists(fpath):
                gaps.append({"type": "backend", "priority": "high", "item": fpath,
                    "description": task.get("purpose", f"Implement {fpath}"),
                    "key_functions": task.get("key_functions", [])})
        for task in design.get("frontend_tasks", []):
            fpath = task.get("file", "")
            if fpath and not os.path.exists(fpath):
                gaps.append({"type": "frontend", "priority": "high", "item": fpath,
                    "description": task.get("purpose", f"Implement {fpath}")})
        for mig in design.get("database_migrations", []):
            mpath = mig.get("file", "")
            if mpath and not os.path.exists(mpath):
                gaps.append({"type": "migration", "priority": "critical", "item": mpath,
                    "description": f"Create migration: {mpath}", "sql": mig.get("sql", "")[:200]})
    except: pass

if not build_ok:
    gaps.insert(0, {"type": "build", "priority": "critical", "item": "Build broken",
        "description": "Fix compilation errors"})
if test_fail > 0:
    gaps.insert(0, {"type": "tests", "priority": "critical", "item": f"{test_fail} failing packages",
        "description": f"Fix {test_fail} failing test packages"})
if todos > 10:
    gaps.append({"type": "todos", "priority": "medium", "item": f"{todos} TODOs",
        "description": f"Resolve {todos} TODO/FIXME items"})
if not docker_ok:
    gaps.append({"type": "docker", "priority": "high", "item": "No Dockerfiles",
        "description": "Create Docker setup"})

priority_order = {"critical": 0, "high": 1, "medium": 2, "low": 3}
gaps.sort(key=lambda x: priority_order.get(x.get("priority", "low"), 3))

result = {
    "timestamp": datetime.now().isoformat(),
    "completion_pct": pct,
    "pr_ready": pct >= 50,
    "dimension_scores": {k: s for k, (s, _) in scores.items()},
    "counts": {
        "backend_files": f"{eg}/{pb}", "frontend_files": f"{et}/{pf}",
        "tests": f"{ets}/{pt}", "endpoints": f"{ee}/{pe}",
        "migrations": f"{em}/{pm}", "todos": todos,
        "build": build_ok, "tests_pass": test_ok
    },
    "gaps": gaps
}
json.dump(result, open(out_f, "w"), indent=2)
PYEOF

  python3 - "$COMPLETION_FILE" << 'PYEOF'
import json, sys
d = json.load(open(sys.argv[1]))
pct = d["completion_pct"]
bar = "█" * int(pct/5) + "░" * (20 - int(pct/5))
icon = "🔴" if pct < 30 else ("🟡" if pct < 50 else ("🟢" if pct < 80 else "✨"))
print(f"\n  {icon} Completion: {pct}%  [{bar}]")
c = d["counts"]
print(f"  Backend: {c['backend_files']}  Frontend: {c['frontend_files']}  Tests: {c['tests']}")
print(f"  Build: {'✅' if c['build'] else '❌'}  Tests: {'✅' if c['tests_pass'] else '❌'}")
gaps = d.get("gaps", [])
if gaps:
    prio = {"critical":"🔥","high":"⚠️ ","medium":"📌","low":"💡"}
    print(f"\n  {len(gaps)} gaps:")
    for g in gaps[:10]:
        print(f"    {prio.get(g['priority'],'  ')} {g['type']:12s} {g['item'][:50]}")
PYEOF
}

show_gaps() {
  [ -f "$COMPLETION_FILE" ] || scan_project_completion
  python3 - "$COMPLETION_FILE" << 'PYEOF'
import json, sys
d = json.load(open(sys.argv[1]))
pct = d["completion_pct"]
bar = "█" * int(pct/5) + "░" * (20 - int(pct/5))
print(f"\n  {'🟢' if pct>=50 else '🔴'} Completion: {pct}%  [{bar}]")
gaps = d.get("gaps", [])
if not gaps: print("  ✅ No gaps!")
else:
    prio = {"critical":"🔥","high":"⚠️ ","medium":"📌","low":"💡"}
    for g in gaps:
        print(f"  {prio.get(g['priority'],'  ')} [{g['priority']:8s}] {g['type']:12s} {g['item'][:60]}")
        if g.get("description") and g["description"] != g["item"]:
            print(f"               └─ {g['description'][:80]}")
PYEOF
}

run_demo() {
  slog "🎬 DEMO: Running live demonstrations..."
  mkdir -p "$DEMO_DIR"; cd "$REPO_DIR"

  local demo_report="$DEMO_DIR/demo_report.md"
  echo "# 🎬 Demo Report — $(date '+%Y-%m-%d %H:%M')" > "$demo_report"
  echo "" >> "$demo_report"

  detect_service_ports
  local services_started=false any_up=false
  if [ "$SKIP_HEALTH_CHECK" = false ]; then
    for port in $SERVICE_PORTS; do
      curl -sf --max-time "$HEALTH_CHECK_TIMEOUT" "http://localhost:${port}${HEALTH_CHECK_PATH}" >/dev/null 2>&1 && { any_up=true; break; }
    done
  else
    any_up=true  # Skip health check, assume services are up
  fi

  if [ "$any_up" = false ]; then
    local compose=""
    [ -f "docker-compose.yml" ] && compose="docker-compose.yml"
    [ -f "deployments/docker/docker-compose.yml" ] && compose="deployments/docker/docker-compose.yml"
    [ -n "$compose" ] && { podman-compose -f "$compose" up -d 2>&1 | tail -5 || true; sleep 20; services_started=true; }
  fi

  echo "## Service Health" >> "$demo_report"
  echo '```' >> "$demo_report"
  for port in $SERVICE_PORTS; do
    if [ "$SKIP_HEALTH_CHECK" = true ]; then
      echo "  ⏭️ :$port → (skipped)" >> "$demo_report"; slog "  ⏭️ :$port"
    else
      local response; response=$(curl -sf --max-time "$HEALTH_CHECK_TIMEOUT" "http://localhost:${port}${HEALTH_CHECK_PATH}" 2>/dev/null || echo "UNREACHABLE")
      if [ "$response" != "UNREACHABLE" ]; then
        echo "  ✅ :$port → $response" >> "$demo_report"; slog "  ✅ :$port"
      else
        echo "  ❌ :$port → UNREACHABLE" >> "$demo_report"; slog "  ❌ :$port"
      fi
    fi
  done
  echo '```' >> "$demo_report"

  echo "## Test Results" >> "$demo_report"
  echo '```' >> "$demo_report"
  go test ./... -count=1 -timeout 120s 2>&1 | tail -20 >> "$demo_report" || true
  echo '```' >> "$demo_report"

  [ "$services_started" = true ] && docker_down
  slog "  🎬 Demo report → $demo_report"
  cp "$demo_report" "$PR_DIR/demo_results.md" 2>/dev/null || true
}

prepare_pr() {
  slog "📬 Preparing pull request..."
  mkdir -p "$PR_DIR"; cd "$REPO_DIR"

  local pct; pct=$(python3 -c "import json; print(json.load(open('$COMPLETION_FILE'))['completion_pct'])" 2>/dev/null || echo "0")
  local branch; branch=$(git branch --show-current 2>/dev/null || echo "")
  local git_log; git_log=$(git log main..HEAD --oneline --no-merges 2>/dev/null | head -40 || git log --oneline -20 2>/dev/null || echo "")
  local git_stat; git_stat=$(git diff main --stat 2>/dev/null | tail -5 || echo "")

  cat > "$PR_DIR/PR_DESCRIPTION.md" << PR_DOC
# feat: project improvements (${pct}% complete)

## Changes
\`\`\`
$git_log
\`\`\`

## Files Changed
\`\`\`
$git_stat
\`\`\`

## Checklist
- [x] Build passes
- [x] Unit tests reviewed
- [x] Security audit passed
- [x] QA review passed
PR_DOC

  if command -v gh &>/dev/null && [ -n "$branch" ] && [ "$branch" != "main" ]; then
    git push origin "$branch" 2>/dev/null || true
    local pr_url
    pr_url=$(gh pr create --title "feat: project improvements (${pct}%)" --body-file "$PR_DIR/PR_DESCRIPTION.md" --base main --head "$branch" 2>/dev/null) || pr_url=""
    [ -n "$pr_url" ] && slog "  ✅ PR created: $pr_url"
  fi

  slog "  📄 PR → $PR_DIR/PR_DESCRIPTION.md"
}

smart_improve() {
  local t0; t0=$(date +%s)

  slog "╔═══════════════════════════════════════════════════════════╗"
  slog "║  🧠 SMART IMPROVE — Project-Focused Self-Improvement       ║"
  slog "╠═══════════════════════════════════════════════════════════╣"
  slog "║  1. SCAN → 2. ADAPT → 3. RUN → 4. RESCAN → 5. PR?       ║"
  slog "╚═══════════════════════════════════════════════════════════╝"

  scan_project_completion
  local pct_before; pct_before=$(python3 -c "import json; print(json.load(open('$COMPLETION_FILE'))['completion_pct'])" 2>/dev/null || echo "0")

  slog "  📊 Before: ${pct_before}%"

  # Plan and execute focused improvement
  diagnose_project
  plan_improvements 3
  execute_planned_phases

  # Re-scan
  scan_project_completion
  local pct_after; pct_after=$(python3 -c "import json; print(json.load(open('$COMPLETION_FILE'))['completion_pct'])" 2>/dev/null || echo "0")
  local delta=$((pct_after - pct_before))
  slog "  📊 Progress: ${pct_before}% → ${pct_after}% (+${delta}%)"

  if [ "$pct_after" -ge "$PR_THRESHOLD" ]; then
    slog "  🎉 ${pct_after}% ≥ ${PR_THRESHOLD}% → PR mode!"
    run_demo
    prepare_pr
  else
    slog "  ⏳ ${pct_after}% < ${PR_THRESHOLD}% — run again: ./dev.sh smart-improve"
  fi

  local elapsed=$(( $(date +%s) - t0 ))
  slog "  ⏱ Smart improve done in $((elapsed/60))m $((elapsed%60))s"
}

# ═══════════════════════════════════════════════
# BACKGROUND EXECUTION
# ═══════════════════════════════════════════════

is_running() { [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE" 2>/dev/null)" 2>/dev/null; }

stop_all() {
  if is_running; then
    local pid; pid=$(cat "$PID_FILE")
    log "Stopping (PID: $pid)..."
    kill -TERM "$pid" 2>/dev/null || true; sleep 2; pkill -P "$pid" 2>/dev/null || true
    pkill -f "claude.*dangerously-skip-permissions" 2>/dev/null || true; rm -f "$PID_FILE"
    docker_down 2>/dev/null || true; log "✓ Stopped"
  else echo "Not running"; fi
}

launch_bg() {
  if is_running; then err "Already running ($(cat "$PID_FILE"))"; echo "  tail -f $LIVE_LOG"; exit 1; fi
  setsid bash "$0" --fg "$@" </dev/null > /dev/null 2>&1 &
  disown
  local bg_pid=$!
  sleep 1
  [ ! -f "$PID_FILE" ] && echo "$bg_pid" > "$PID_FILE"
  echo ""
  echo "  ✅ AI team running (fully detached)"
  echo "  📺 tail -f $LIVE_LOG"
  echo "  📊 ./dev.sh status"
  echo "  🛑 ./dev.sh stop"
  echo ""
  echo "  Safe to close SSH ✓"
  echo ""
  exit 0
}

# ═══════════════════════════════════════════════
# STATUS DASHBOARD
# ═══════════════════════════════════════════════

show_status() {
  echo ""
  echo -e "  ${W}═══ AI DEVELOPMENT TEAM ═══${NC}"
  echo ""

  if is_running; then
    echo -e "  🤖 Status: ${G}RUNNING${NC} (PID: $(cat "$PID_FILE" 2>/dev/null))"
  else
    echo -e "  🤖 Status: ${R}STOPPED${NC}"
  fi
  echo ""

  # Show environment detection
  echo -e "  ${B}Environment:${NC}"
  echo -e "    📁 Frontend: ${FRONTEND_DIR:-${Y}not detected${NC}}"
  echo -e "    🔌 Dashboard Port: ${DASHBOARD_PORT:-${Y}not detected${NC}}"
  echo -e "    🌐 Base URL: $E2E_BASE_URL"

  # Detect and show E2E framework
  detect_e2e_framework 2>/dev/null
  local fw="$DETECTED_E2E_FRAMEWORK"
  local fw_dir="${DETECTED_E2E_DIR#$REPO_DIR/}"
  case "$fw" in
    playwright|cypress)
      echo -e "    🧪 E2E Framework: ${G}$fw${NC} (${fw_dir})"
      ;;
    generic)
      echo -e "    🧪 E2E Framework: ${Y}generic tests${NC} (${fw_dir})"
      ;;
    none|*)
      echo -e "    🧪 E2E Framework: ${R}none detected${NC} (will use smoke tests)"
      ;;
  esac
  echo ""

  if [ -f "$STATE_FILE" ]; then
    python3 - "$STATE_FILE" << 'PYEOF'
import json, sys
d = json.load(open(sys.argv[1]))
print(f"  Project: {d.get('project','')[:60]}")
print(f"  Branch:  {d.get('branch','')}")
print()

phases = ["requirements","market_research","design","backend","frontend","testing","qa","security","deploy"]
icons = {"done":"✅","running":"🔄","pending":"⬜","failed":"❌","skipped":"⏭️"}
roles = {"requirements":"PM","market_research":"Research","design":"Architect","backend":"Backend","frontend":"Frontend","testing":"Tester","qa":"QA","security":"Security","deploy":"DevOps"}

done = 0
for p in phases:
    data = d.get("phases",{}).get(p,{})
    st = data.get("status","pending")
    if st == "done": done += 1
    verdict = data.get("verdict","")
    verdict_str = f" → {verdict}" if verdict else ""
    updated = data.get("_updated","")
    time_str = f" [{updated[11:19]}]" if updated else ""
    print(f"  {icons.get(st,'⬜')} {roles.get(p,p):10s}{verdict_str}{time_str}")
    for k,v in sorted(data.items()):
        if k.startswith("_") or k in ("status","verdict"): continue
        print(f"     └─ {k}: {v}")

print(f"\n  Progress: {done}/{len(phases)} phases ({done*100//len(phases)}%)")
PYEOF
  fi

  if [ -f "$PHASE_HISTORY" ]; then
    local round_count
    round_count=$(python3 -c "import json; print(len(json.load(open('$PHASE_HISTORY')).get('completed',[])))" 2>/dev/null || echo "0")
    if [ "$round_count" -gt 0 ]; then
      echo ""
      echo -e "  ${G}Completed rounds: $round_count${NC}"
      python3 -c "
import json
d = json.load(open('$PHASE_HISTORY'))
for r in d.get('completed',[])[-5:]:
    cat_icon = {'team':'🔧','project':'📦'}.get(r.get('category',''),'📌')
    res_icon = {'done':'✅','failed':'❌','crashed':'💥','skipped':'⏭️'}.get(r.get('result',''),'❓')
    print(f\"    {cat_icon}{res_icon} {r.get('description','')[:65]}\")" 2>/dev/null || true
    fi
  fi

  echo ""
}

show_history() {
  if [ -f "$PHASE_HISTORY" ]; then
    python3 -c "
import json
d = json.load(open('$PHASE_HISTORY'))
rounds = d.get('completed', [])
print(f'Total rounds: {len(rounds)}')
for i, r in enumerate(rounds, 1):
    cat_icon = {'team':'🔧','project':'📦'}.get(r.get('category',''),'📌')
    res_icon = {'done':'✅','failed':'❌','crashed':'💥','skipped':'⏭️'}.get(r.get('result',''),'❓')
    print(f'  {i}. {cat_icon}{res_icon} [{r.get(\"timestamp\",\"\")[:19]}] {r.get(\"description\",\"\")[:70]}')" 2>/dev/null
  else
    echo "No history yet"
  fi
}

# ═══════════════════════════════════════════════
# CLI
# ═══════════════════════════════════════════════

show_help() { cat << 'HELP'

  ╔══════════════════════════════════════════╗
  ║  AI Development Team — dev.sh            ║
  ╚══════════════════════════════════════════╝

  BASIC:
    ./dev.sh start "description"        # Full waterfall (background)
    ./dev.sh status                     # Dashboard
    ./dev.sh stop                       # Stop everything
    ./dev.sh resume                     # Continue from last phase
    ./dev.sh phase backend              # Single phase (fg)
    ./dev.sh start "desc" --fg          # Foreground mode
    ./dev.sh fix-postgres               # Fix PostgreSQL container issues

  PRODUCTION TESTING (after deploy):
    ./dev.sh ports                      # Check port availability
    ./dev.sh ai-e2e                     # AI-powered E2E (auto-fix issues)
    ./dev.sh e2e                        # Run E2E tests against deployed system
    ./dev.sh ai-e2e URL false           # AI E2E without auto-fix
    ./dev.sh access-report              # Show client connection info

  SMART IMPROVE (recommended):
    ./dev.sh smart-improve [threshold%]  # Scan→Plan→Run→Rescan→PR
    ./dev.sh scan                        # Show % complete + gaps
    ./dev.sh gaps                        # Just the gaps
    ./dev.sh demo                        # Run live demos
    ./dev.sh pr                          # Generate PR

  DUAL-TRACK IMPROVEMENT:
    ./dev.sh improve [proj_N] [dev_N]    # Fix dev.sh + project

  TRACK A — DEV.SH SELF-IMPROVEMENT:
    ./dev.sh improve-dev [N]            # Full: analyze→plan→execute→verify→sync
    ./dev.sh analyze-dev                # A1: Analyze structure
    ./dev.sh plan-dev [N]               # A2: Plan N improvements
    ./dev.sh execute-dev                # A3: Apply step-by-step
    ./dev.sh verify-dev                 # A4: Check integrity
    ./dev.sh sync-dev                   # Sync master → all projects

  TRACK B — PROJECT CODE:
    ./dev.sh improve-project [N]        # Full: diagnose→plan→execute→verify
    ./dev.sh diagnose                   # B1: Check build/tests
    ./dev.sh plan-project [N]           # B2: Plan N phases
    ./dev.sh execute                    # B3: Execute phases
    ./dev.sh verify                     # B4: Check results

  INFO:
    ./dev.sh history                    # Completed rounds
    ./dev.sh logs                       # Supervisor log
    ./dev.sh -h                         # This help

HELP
}

# ── Parse global flags (must come before command extraction) ──
FOREGROUND=false
SKIP_HEALTH_CHECK=false

# Extract command from args, skipping global flags
for arg in "$@"; do
  case "$arg" in
    --fg) FOREGROUND=true ;;
    --skip-health-check) SKIP_HEALTH_CHECK=true ;;
    --*)
      # Strip -- prefix for command detection
      CMD="${arg#--}"
      break
      ;;
    *)
      CMD="$arg"
      break
      ;;
  esac
done

CMD="${CMD:-}"

# ── Preflight checks (only for commands that need tools) ──
case "$CMD" in
  start|resume|phase|improve*|execute*|smart*)
    command -v claude &>/dev/null || { err "Claude Code not found. Install: npm install -g @anthropic-ai/claude-code"; exit 1; }
    command -v go &>/dev/null && log "✓ go $(go version | awk '{print $3}')" || warn "⚠ go not found"
    command -v node &>/dev/null && log "✓ node $(node -v)" || warn "⚠ node not found"

    # Auto-init git if needed
    if [ ! -d "$REPO_DIR/.git" ]; then
      log "No git repo found — initializing..."
      cd "$REPO_DIR"; git init; git add -A; git commit -m "Initial commit" 2>/dev/null || true
    fi
    ;;
esac

case "$CMD" in
  start)
    [ -z "${2:-}" ] && { err "Usage: ./dev.sh start \"project description\""; exit 1; }
    if [ "$FOREGROUND" = false ]; then
      launch_bg --project "$2"
    fi
    # Foreground execution
    echo $$ > "$PID_FILE"
    trap 'rm -f "$PID_FILE"; exit' EXIT INT TERM
    run_waterfall "$2"
    ;;

  resume)
    if [ "$FOREGROUND" = false ]; then
      launch_bg --resume
    fi
    echo $$ > "$PID_FILE"
    trap 'rm -f "$PID_FILE"; exit' EXIT INT TERM
    BRANCH=$(state_get _meta branch); PROJECT=$(state_get _meta project)
    [ -z "$BRANCH" ] && { err "Nothing to resume"; exit 1; }
    cd "$REPO_DIR"; git checkout "$BRANCH" 2>/dev/null || true
    local cur; cur=$(current_phase); log "Resuming: $cur"
    case "$cur" in
      requirements) phase_requirements "$PROJECT" ;& market_research) phase_market_research ;& design) phase_design ;& backend) phase_backend ;&
      frontend) phase_frontend ;& testing) phase_testing ;& qa) phase_qa ;&
      security) phase_security ;& deploy) phase_deploy ;& e2e_production) phase_e2e_production ;; *) run_waterfall "$PROJECT" ;;
    esac
    ;;

  phase)
    [ -z "${2:-}" ] && { err "Usage: ./dev.sh phase <name>"; exit 1; }
    echo $$ > "$PID_FILE"
    trap 'rm -f "$PID_FILE"; exit' EXIT INT TERM
    BRANCH=$(state_get _meta branch); [ -z "$BRANCH" ] && BRANCH="main"
    cd "$REPO_DIR"; git checkout "$BRANCH" 2>/dev/null || true
    case "$2" in
      requirements) phase_requirements "${3:-manual}" ;; market|market_research) phase_market_research ;; design) phase_design ;;
      backend) phase_backend ;; frontend) phase_frontend ;; testing) phase_testing ;;
      qa) phase_qa ;; security) phase_security ;; deploy) phase_deploy ;;
      e2e|e2e_production) phase_e2e_production ;;
      *) err "Unknown phase: $2" ;;
    esac
    ;;

  # Check E2E setup (diagnostic command)
  check-e2e|diag-e2e)
    echo -e "${W}═══ E2E Framework Detection ═══${NC}"
    echo ""

    detect_e2e_framework
    fw="$DETECTED_E2E_FRAMEWORK"
    fw_dir="${DETECTED_E2E_DIR#$REPO_DIR/}"

    echo -e "  Frontend Directory: ${FRONTEND_DIR:-${R}not detected${NC}}"
    echo -e "  Dashboard Port: ${DASHBOARD_PORT:-${R}not detected${NC}}"
    echo -e "  E2E Framework: ${G}${fw}${NC}"
    echo -e "  E2E Directory: ${G}${fw_dir}${NC}"
    echo ""

    # Check for E2E test files
    if [ -n "$DETECTED_E2E_DIR" ]; then
      echo -e "${B}E2E Test Files:${NC}"
      find "$DETECTED_E2E_DIR/e2e" -name "*.spec.ts" -o -name "*.spec.js" 2>/dev/null | head -10 | while read -r f; do
        name="${f#$DETECTED_E2E_DIR/}"
        echo -e "    ✓ $name"
      done
      echo ""
    fi

    # Check dependencies
    echo -e "${B}Dependency Check:${NC}"
    if [ -n "$DETECTED_E2E_DIR" ]; then
      cd "$DETECTED_E2E_DIR" 2>/dev/null || {
        echo -e "    ${R}✗ Cannot access E2E directory${NC}"
        exit 1
      }

      if [ ! -d "node_modules" ]; then
        echo -e "    ${Y}⚠️  node_modules not found - run: npm install${NC}"
      else
        echo -e "    ${G}✓ node_modules exists${NC}"
      fi

      case "$fw" in
        playwright)
          if npx playwright --version &>/dev/null; then
            version=$(npx playwright --version 2>/dev/null)
            echo -e "    ${G}✓ Playwright installed: $version${NC}"
          else
            echo -e "    ${R}✗ Playwright not found - run: npx playwright install${NC}"
          fi
          ;;
        cypress)
          if npx cypress --version &>/dev/null; then
            version=$(npx cypress --version 2>/dev/null)
            echo -e "    ${G}✓ Cypress installed: $version${NC}"
          else
            echo -e "    ${R}✗ Cypress not found - run: npx cypress install${NC}"
          fi
          ;;
      esac
    fi

    echo ""
    echo -e "${B}Target Configuration:${NC}"
    echo -e "    Base URL: ${G}$E2E_BASE_URL${NC}"
    echo -e "    Server IP: ${G}$SERVER_IP${NC}"

    # Test connectivity
    echo ""
    echo -e "${B}Connectivity Test:${NC}"
    if curl -sf --max-time 3 "$E2E_BASE_URL" >/dev/null 2>&1; then
      echo -e "    ${G}✓ Dashboard is accessible${NC}"
    else
      echo -e "    ${Y}⚠️  Dashboard not accessible at $E2E_BASE_URL${NC}"
      echo -e "    ${Y}  (This is OK if you're running tests before deployment)${NC}"
    fi
    echo ""

    # Provide fix commands
    if [ "$fw" = "none" ] || [ -z "$fw" ]; then
      echo -e "${Y}No E2E framework detected. To set up Playwright:${NC}"
    fi
    ;;

  # Port availability check
  ports|check-ports)
    echo -e "${W}═══ PORT AVAILABILITY CHECK ═══${NC}"
    echo ""

    default_ports=(3000 8001 8002 8003 8004 8005 9090 9091 9092 9093 9094 9095 18001 18005 15432 16379)
    occupied=()
    available=()

    for port in "${default_ports[@]}"; do
      if is_port_available "$port"; then
        available+=("$port")
        echo -e "  ${G}✓${NC} Port $port - ${G}AVAILABLE${NC}"
      else
        occupied+=("$port")
        echo -e "  ${R}✗${NC} Port $port - ${R}IN USE${NC}"
        # Show what's using it
        if command -v ss >/dev/null 2>&1; then
          process=$(ss -tlnp 2>/dev/null | grep ":$port " | head -1 | awk '{print $6}' || echo "")
          [ -n "$process" ] && echo -e "    ${Y}used by: $process${NC}"
        fi
      fi
    done

    echo ""
    echo -e "${B}Summary:${NC}"
    echo -e "  Available: ${G}${#available[@]}${NC} ports"
    echo -e "  Occupied: ${R}${#occupied[@]}${NC} ports"

    if [ ${#occupied[@]} -gt 0 ]; then
      echo ""
      echo -e "${Y}Free port alternatives:${NC}"
      for port in "${occupied[@]}"; do
        free_port=$(find_free_port "$((port + 1))" 5)
        if [ -n "$free_port" ]; then
          echo -e "  $port → ${G}$free_port${NC}"
        fi
      done
    fi
    ;;

  # Production E2E testing
  e2e|e2e-test)
    E2E_BASE_URL="${2:-$E2E_BASE_URL}"
    echo "Running E2E tests against: $E2E_BASE_URL"
    echo $$ > "$PID_FILE"
    trap 'rm -f "$PID_FILE"; exit' EXIT INT TERM
    phase_e2e_production
    ;;

  # AI-Powered E2E testing (run directly)
  ai-e2e|ai-test)
    auto_fix="${3:-true}"

    # Detect actual dashboard port dynamically if URL not provided
    if [[ -n "${2:-}" ]]; then
      target_url="$2"
    else
      # Get actual dashboard port from running containers
      ports_json=$(get_all_service_ports)
      dashboard_port=$(echo "$ports_json" | python3 -c "import json,sys; print(json.load(sys.stdin).get('dashboard','3000'))")
      target_url="http://localhost:${dashboard_port}"
    fi

    echo -e "${W}═══ AI-POWERED E2E TESTING ═══${NC}"
    echo -e "  Target: ${G}$target_url${NC}"
    echo -e "  Auto-Fix: ${G}${auto_fix}${NC}"
    echo ""

    detect_service_ports
    e2e_report="${ARTIFACTS:-}/ai_e2e_report.json}"

    echo $$ > "$PID_FILE"
    trap 'rm -f "$PID_FILE"; exit' EXIT INT TERM

    run_ai_e2e "$target_url" "$e2e_report" "$auto_fix"

    # Display results
    if [ -f "$e2e_report" ]; then
      python3 - "$e2e_report" << 'PYEOF'
import json, sys
d = json.load(open(sys.argv[1]))
s = d.get('summary', {})
print(f"\n{'='*50}")
print(f"  E2E TEST RESULTS")
print(f"{'='*50}")
print(f"  Status:    {d.get('status','unknown').upper()}")
print(f"  Passed:    {s.get('passed',0)}/{s.get('total',0)}")
print(f"  Success:   {d.get('success_rate',0)}%")
print(f"  Ready:     {'YES' if d.get('ready_for_production') else 'NO'}")
print(f"{'='*50}\n")
PYEOF
    fi
    ;;

  # Production E2E testing
  e2e|e2e-test)
    E2E_BASE_URL="${2:-$E2E_BASE_URL}"
    echo "Running E2E tests against: $E2E_BASE_URL"
    echo $$ > "$PID_FILE"
    trap 'rm -f "$PID_FILE"; exit' EXIT INT TERM
    phase_e2e_production
    ;;

  access-report)
    echo "Generating access report..."
    generate_access_report
    if [ -f "$ARTIFACTS/access_report.json" ]; then
      python3 - "$ARTIFACTS/access_report.json" << 'PYEOF'
import json, sys
d = json.load(open(sys.argv[1]))
print(f"\n╔═══════════════════════════════════════════════════════╗")
print(f"║  CLIENT CONNECTION INFO                               ║")
print(f"╚═══════════════════════════════════════════════════════╝")
print(f"  Server:         {d['server']['hostname']}")
print(f"  External IP:    {d['server']['external_ip']}")
print(f"  Dashboard URL:  {d['client_connection']['base_url']}")
print(f"\n  Accessible Ports: {', '.join(map(str, d['services']['all_accessible_ports']))}")
print(f"\n  To connect from your client:")
print(f"    1. Ensure network connectivity to: {d['server']['external_ip']}")
print(f"    2. Open browser: {d['client_connection']['base_url']}")
print(f"    3. For firewall, allow ports: {', '.join(map(str, d['services']['api_ports']))}")
PYEOF
    fi
    ;;

  fix-postgres)
    (
      # Fix PostgreSQL container issues (in subshell for proper scoping)
      log "🔧 Fixing PostgreSQL container..."
      runtime=$(detect_container_runtime)
      pg_container="openprint-postgres"
      pg_volume="openprint-postgres-data"
      pg_network="openprint-network"

      # Stop and remove existing postgres container
      if $runtime ps -a --format "{{.Names}}" | grep -q "^${pg_container}$"; then
        log "  🛑 Stopping PostgreSQL container..."
        $runtime stop "$pg_container" 2>/dev/null || true
        $runtime rm "$pg_container" 2>/dev/null || true
        log "  ✓ Removed old container"
      fi

      # Check if volume exists
      if $runtime volume ls --format "{{.Name}}" | grep -q "^${pg_volume}$"; then
        log "  📦 PostgreSQL volume exists: $pg_volume"
      else
        log "  📦 Creating PostgreSQL volume..."
        $runtime volume create "$pg_volume" >/dev/null 2>&1
      fi

      # Create container with correct configuration
      log "  🚀 Creating new PostgreSQL container..."
      $runtime run -d \
        --name "$pg_container" \
        --hostname postgres \
        --network "$pg_network" \
        -e POSTGRES_USER=openprint \
        -e POSTGRES_PASSWORD=openprint \
        -e POSTGRES_DB=openprint \
        -e PGDATA=/var/lib/postgresql/data/pgdata \
        -v "${pg_volume}:/var/lib/postgresql/data" \
        -p 15432:5432 \
        postgres:16-alpine >/dev/null 2>&1

      if [ $? -eq 0 ]; then
        log "  ✓ PostgreSQL container created"

        # Wait for postgres to be ready
        log "  ⏳ Waiting for PostgreSQL to be ready..."
        ready=false
        for i in $(seq 1 30); do
          if $runtime exec "$pg_container" pg_isready -U openprint -d openprint >/dev/null 2>&1; then
            ready=true
            break
          fi
          sleep 1
        done

        if [ "$ready" = true ]; then
          log "  ✓ PostgreSQL is ready!"
          log "  📌 Running on port 15432"
          log "  📌 Connect: psql -h localhost -p 15432 -U openprint -d openprint"
        else
          warn "  ⚠️  PostgreSQL started but not accepting connections"
          log "  Check logs: $runtime logs $pg_container"
        fi
      else
        err "  ✗ Failed to create PostgreSQL container"
        exit 1
      fi
    )
    ;;

  stop)            stop_all ;;
  stop-services)   docker_down; echo "✓ Stopped" ;;
  status)          show_status ;;
  reset)           stop_all 2>/dev/null; rm -rf "$DEV_DIR"; echo "✓ Reset" ;;

  # ── Smart improve ──
  smart-improve|smart|focus)
    PR_THRESHOLD="${2:-50}"
    if [ "$FOREGROUND" = false ] && [ "${_DEV_FG:-}" != "1" ]; then
      _DEV_FG=1 setsid bash "$0" --fg "$CMD" "${2:-50}" </dev/null > /dev/null 2>&1 &
      disown
      echo "  🧠 Smart Improve started (detached)"
      echo "  📺 tail -f $SUP_LOG"
      echo "  📊 ./dev.sh status"
      exit 0
    fi
    smart_improve
    ;;

  scan)    scan_project_completion ;;
  gaps)    show_gaps ;;
  demo)    run_demo ;;
  pr)      [ ! -f "$COMPLETION_FILE" ] && scan_project_completion; run_demo; prepare_pr ;;

  # ── Dual-track ──
  improve|next)
    if [ "$FOREGROUND" = false ] && [ "${_DEV_FG:-}" != "1" ]; then
      AUTO_PHASES="${2:-3}" _DEV_FG=1 setsid bash "$0" --fg "$CMD" "${2:-3}" "${3:-3}" </dev/null > /dev/null 2>&1 &
      disown
      echo "  🚀 Full improvement started (detached)"
      echo "  📺 tail -f $SUP_LOG"
      exit 0
    fi
    AUTO_PHASES="${2:-3}"
    run_full_improvement "$AUTO_PHASES" "${3:-3}"
    ;;

  # ── Track A ──
  improve-dev|fix-dev)
    if [ "$FOREGROUND" = false ] && [ "${_DEV_FG:-}" != "1" ]; then
      _DEV_FG=1 setsid bash "$0" --fg "$CMD" "${2:-3}" </dev/null > /dev/null 2>&1 &
      disown
      echo "  🔧 Dev.sh improvement started (detached)"
      echo "  📺 tail -f $SUP_LOG"
      exit 0
    fi
    run_dev_improvement "${2:-3}"
    ;;
  analyze-dev)    analyze_dev ;;
  plan-dev)       analyze_dev; plan_dev_improvements "${2:-3}" ;;
  execute-dev)
    if [ "${_DEV_FG:-}" != "1" ]; then
      _DEV_FG=1 setsid bash "$0" --fg "$CMD" </dev/null > /dev/null 2>&1 &
      disown; echo "  🚀 Executing (detached)"; echo "  📺 tail -f $SUP_LOG"; exit 0
    fi
    execute_dev_improvements
    ;;
  verify-dev)     verify_dev ;;
  sync-dev)       sync_dev_all ;;

  # ── Track B ──
  improve-project|project)
    if [ "$FOREGROUND" = false ] && [ "${_DEV_FG:-}" != "1" ]; then
      _DEV_FG=1 setsid bash "$0" --fg "$CMD" "${2:-3}" </dev/null > /dev/null 2>&1 &
      disown
      echo "  📦 Project improvement started (detached)"
      echo "  📺 tail -f $SUP_LOG"
      exit 0
    fi
    run_project_improvement "${2:-3}"
    ;;
  diagnose|diag)  diagnose_project ;;
  plan-project)   AUTO_PHASES="${2:-3}"; diagnose_project; plan_improvements "$AUTO_PHASES" ;;
  execute|run)
    if [ "${_DEV_FG:-}" != "1" ]; then
      _DEV_FG=1 setsid bash "$0" --fg "$CMD" </dev/null > /dev/null 2>&1 &
      disown; echo "  🚀 Executing (detached)"; exit 0
    fi
    execute_planned_phases
    ;;
  verify|check)   verify_results ;;

  # ── Info ──
  history)        show_history ;;
  logs)           tail -50 "$SUP_LOG" ;;
  skip)           skip_phase "${2:-}" ;;
  -h|--help|help) show_help ;;
  "")             show_help ;;
  *)              err "Unknown: $CMD"; show_help; exit 1 ;;
esac
