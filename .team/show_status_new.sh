# Enhanced status dashboard with three-layer orchestrator view

show_status() {
  echo ""
  echo -e "  ${W}╔════════════════════════════════════════╗${NC}"
  echo -e "  ${W}║  🤖 AI DEVELOPMENT TEAM - ORCHESTRATOR      ║${NC}"
  echo -e "  ${W}╚════════════════════════════════════════╝${NC}"
  echo ""

  # System status
  if is_running; then
    local pid; pid=$(cat "$PID_FILE" 2>/dev/null)
    local uptime
    uptime=$(ps -o etime= -p "$pid" 2>/dev/null | awk '{print $1}' || echo "?")
    echo -e "  🤖 Status: ${G}RUNNING${NC} (PID: $pid | Uptime: ${uptime}s)"
  else
    echo -e "  🤖 Status: ${R}STOPPED${NC}"
  fi
  echo ""

  # Project info
  if [ -f "$STATE_FILE" ]; then
    python3 - "$STATE_FILE" << 'PYEOF'
import json, sys
from datetime import datetime, timedelta
d = json.load(open(sys.argv[1]))

project = d.get('project','')[:60]
branch = d.get('branch','')
current = d.get('current_phase','')

# Calculate project age
started = d.get('_meta',{}).get('_updated','')
if started:
    try:
        start = datetime.fromisoformat(started)
        age = datetime.now() - start
        if age < timedelta(minutes=1):
            age_str = f"{age.seconds}s"
        elif age < timedelta(hours=1):
            age_str = f"{age.seconds//60}m"
        else:
            age_str = f"{age.seconds//3600}h"
    except:
        age_str = "?"
else:
    age_str = "?"

print(f"  📋 Project: {project}")
print(f"  🌿 Branch:  {branch}")
print(f"  ⏱️  Age:     {age_str}")
print(f"  🔄 Phase:   {current}")
PYEOF
  fi

  echo ""

  # Three-layer status (Orchestrator → PMs → Engineers)
  echo -e "  ${B}┌─ THREE-LAYER ORCHESTRATOR ─────────────────┐${NC}"

  # Layer 1: Orchestrator
  local orchestrator_state="idle"
  if [ -f "$ORCHESTRATOR_STATE" ]; then
    orchestrator_state=$(python3 -c "import json; print(json.load(open('$ORCHESTRATOR_STATE')).get('global_state','idle')" 2>/dev/null || echo "idle")
  fi

  # Count active Claude processes
  local claude_count
  claude_count=$(pgrep -f "claude.*dangerously-skip-permissions" 2>/dev/null | wc -l || echo "0")

  local layer1_icon="💤"
  [ "$orchestrator_state" = "active" ] && layer1_icon="🟢"

  echo -e "  ${layer1_icon}  Layer 1: ORCHESTRATOR ($orchestrator_state)"
  echo -e "       └─ Monitoring $claude_count Claude agents"
  echo ""

  # Layer 2: Project Managers
  echo -e "  📋  Layer 2: PROJECT MANAGERS"

  # Show phases with PMs
  if [ -f "$STATE_FILE" ]; then
    python3 - "$STATE_FILE" << 'PYEOF'
import json, sys
from datetime import datetime

d = json.load(open(sys.argv[1]))
phases = ["requirements","market_research","design","backend","frontend","testing","qa","security","deploy"]
icons = {"done":"✅","running":"🔄","pending":"⬜","failed":"❌","skipped":"⏭️","blocked":"🚫"}
roles = {"requirements":"🧑‍💼 PM","market_research":"🔍 Research","design":"🏗️  Architect","backend":"⚙️","frontend":"🎨","testing":"🧪","qa":"📋","security":"🔒","deploy":"🐳"}

for p in phases:
    data = d.get("phases",{}).get(p,{})
    st = data.get("status","pending")
    icon = icons.get(st,'⬜')
    role = roles.get(p,'')

    # Show timestamp if running/recently done
    updated = data.get("_updated","")
    time_str = ""
    if updated and st in ("running","done"):
        try:
            ts = datetime.fromisoformat(updated)
            ago = (datetime.now() - ts).total_seconds()
            if ago < 60: time_str = f"{int(ago)}s ago"
            elif ago < 3600: time_str = f"{int(ago/60)}m ago"
            else: time_str = f"{int(ago/3600)}h ago"
        except: pass

    # Calculate progress percentage
    total = len(phases)
    done = len([p for p in phases if d.get("phases",{}).get(p,{}).get("status") == "done"])
    pct = (done * 100 // total) if total > 0 else 0

    bar_len=20
    filled = int(bar_len * pct / 100)
    bar="█" * filled + "░" * (bar_len - filled)

    print(f"    {icon} {role:13s} {st:8s} {time_str:15s} [{bar}] {pct}%")
PYEOF
  fi
  echo ""

  # Layer 3: Engineers (Claude processes)
  echo -e "  🔧  Layer 3: ENGINEERS (Claude Agents)"
  echo -e "       ├─ Active: $claude_count processes"

  # Show recent checkpoints
  local recent_checkpoints
  recent_checkpoints=$(ls -t "$CHECKPOINT_DIR"/*.json 2>/dev/null | head -3 || true)
  if [ -n "$recent_checkpoints" ]; then
    echo "       └─ Recent checkpoints:"
    for cp in $recent_checkpoints; do
      local cp_name=$(basename "$cp")
      echo "          • $cp_name"
    done
  fi
  echo ""

  # SBFL rankings if available
  if [ -f "$ARTIFACTS/sbfl_rankings.json" ]; then
    echo -e "  ${Y}┌─ FAULT LOCALIZATION (SBFL) ───────────────────┐${NC}"
    python3 - "$ARTIFACTS/sbfl_rankings.json" << 'PYEOF'
import json
r = json.load(open(sys.argv[1]))
rankings = r.get('rankings', [])[:3]
if rankings:
    print("  Top suspicious files:")
    for item in rankings:
        print(f"    • {item['file']}:{item['line']} (score:{item['score']})")
PYEOF
    echo ""
  fi

  # Error memory
  if [ -f "$ERROR_LOG" ] && [ -s "$ERROR_LOG" ]; then
    local recent_errors
    recent_errors=$(tail -3 "$ERROR_LOG" 2>/dev/null | python3 -c "
import json, sys
lines = []
for line in sys.stdin:
    try:
        e = json.loads(line.strip())
        lines.append(f\"  - {e.get('phase','?')}: {e.get('type','?')}\")
    except: pass
print('\n'.join(lines[:3])" 2>/dev/null || true)
    if [ -n "$recent_errors" ]; then
      echo -e "  ${R}Recent errors:${NC}"
      echo "$recent_errors"
      echo ""
    fi
  fi

  # Quick actions
  echo -e "  ${W}╰──────────────────────────────────────────────╯${NC}"
  echo -e "     ${W}./dev.sh logs${NC}        - Supervisor log"
  echo -e "     ${W}./dev.sh gaps${NC}        - Show gaps"
  echo -e "     ${W}./dev.sh stop${NC}        - Stop all"
  echo ""
}
