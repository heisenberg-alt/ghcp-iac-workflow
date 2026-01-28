#!/bin/bash
# =============================================================================
# Enterprise IaC Governance Platform - Agent Startup Script (Bash)
# =============================================================================
# This script builds, starts, and verifies all agents in the platform.
#
# Usage:
#   ./start-all-agents.sh              # Start all agents
#   ./start-all-agents.sh --build      # Only build, don't start
#   ./start-all-agents.sh --status     # Only check status
#   ./start-all-agents.sh --stop       # Stop all running agents
# =============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
GRAY='\033[0;37m'
NC='\033[0m' # No Color

# =============================================================================
# Configuration
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Agent definitions: Name|Port|Directory|Executable
declare -a AGENTS=(
    "Policy Checker|8081|03-policy-agent|policy-agent"
    "Cost Estimator|8082|04-cost-estimator|cost-estimator"
    "Drift Detector|8083|enterprise/agents/drift-detector|drift-detector"
    "Security Scanner|8084|enterprise/agents/security-scanner|security-scanner"
    "Compliance Auditor|8085|enterprise/agents/compliance-auditor|compliance-auditor"
    "Module Registry|8086|enterprise/agents/module-registry|module-registry"
    "Impact Analyzer|8087|enterprise/agents/impact-analyzer|impact-analyzer"
    "Deploy Promoter|8088|enterprise/agents/deploy-promoter|deploy-promoter"
    "Notification Manager|8089|enterprise/agents/notification-manager|notification-manager"
    "Orchestrator|8090|enterprise/agents/orchestrator|orchestrator"
)

# =============================================================================
# Helper Functions
# =============================================================================

print_header() {
    echo ""
    echo -e "${CYAN}======================================================================${NC}"
    echo -e "${CYAN} $1${NC}"
    echo -e "${CYAN}======================================================================${NC}"
    echo ""
}

print_banner() {
    echo ""
    echo -e "${MAGENTA}╔══════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${MAGENTA}║       Enterprise IaC Governance Platform - Agent Manager             ║${NC}"
    echo -e "${MAGENTA}╚══════════════════════════════════════════════════════════════════════╝${NC}"
}

check_health() {
    local port=$1
    local response=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 2 "http://localhost:$port/health" 2>/dev/null || echo "000")
    [[ "$response" == "200" ]]
}

wait_for_agent() {
    local port=$1
    local timeout=${2:-10}
    local elapsed=0
    
    while [[ $elapsed -lt $timeout ]]; do
        if check_health $port; then
            return 0
        fi
        sleep 0.5
        elapsed=$((elapsed + 1))
    done
    return 1
}

stop_agent_on_port() {
    local port=$1
    local pid=$(lsof -ti :$port 2>/dev/null || echo "")
    if [[ -n "$pid" ]]; then
        kill -9 $pid 2>/dev/null
        return 0
    fi
    return 1
}

get_agent_field() {
    local agent_str=$1
    local field=$2
    echo "$agent_str" | cut -d'|' -f$field
}

# =============================================================================
# Main Functions
# =============================================================================

show_status() {
    print_header "Agent Status Check"
    
    local online=0
    local offline=0
    
    printf "  ${GRAY}%-8s %-24s %s${NC}\n" "Port" "Agent" "Status"
    printf "  ${GRAY}%-8s %-24s %s${NC}\n" "--------" "------------------------" "--------"
    
    for agent in "${AGENTS[@]}"; do
        local name=$(get_agent_field "$agent" 1)
        local port=$(get_agent_field "$agent" 2)
        
        if check_health $port; then
            printf "  %-8s %-24s ${GREEN}%s${NC}\n" "[$port]" "$name" "ONLINE"
            ((online++))
        else
            printf "  %-8s %-24s ${RED}%s${NC}\n" "[$port]" "$name" "OFFLINE"
            ((offline++))
        fi
    done
    
    echo ""
    echo -e "  Summary: ${GREEN}$online online${NC}, ${RED}$offline offline${NC}"
    echo ""
    
    return $online
}

build_all_agents() {
    print_header "Building All Agents"
    
    local success=0
    local failed=0
    
    for agent in "${AGENTS[@]}"; do
        local name=$(get_agent_field "$agent" 1)
        local dir=$(get_agent_field "$agent" 3)
        local exe=$(get_agent_field "$agent" 4)
        local agent_path="$BASE_DIR/$dir"
        
        printf "  Building %-24s" "$name..."
        
        if [[ ! -d "$agent_path" ]]; then
            echo -e " ${YELLOW}SKIP (directory not found)${NC}"
            continue
        fi
        
        cd "$agent_path"
        if go build -o "$exe" . 2>/dev/null; then
            echo -e " ${GREEN}OK${NC}"
            ((success++))
        else
            echo -e " ${RED}FAILED${NC}"
            ((failed++))
        fi
    done
    
    cd "$SCRIPT_DIR"
    
    echo ""
    echo -e "  Build complete: ${GREEN}$success succeeded${NC}, ${RED}$failed failed${NC}"
    echo ""
    
    [[ $failed -eq 0 ]]
}

start_all_agents() {
    print_header "Starting All Agents"
    
    local started=0
    local failed=0
    local skipped=0
    
    for agent in "${AGENTS[@]}"; do
        local name=$(get_agent_field "$agent" 1)
        local port=$(get_agent_field "$agent" 2)
        local dir=$(get_agent_field "$agent" 3)
        local exe=$(get_agent_field "$agent" 4)
        local agent_path="$BASE_DIR/$dir"
        local exe_path="$agent_path/$exe"
        
        printf "  Starting %-24s on port %s..." "$name" "$port"
        
        # Check if already running
        if check_health $port; then
            echo -e " ${YELLOW}ALREADY RUNNING${NC}"
            ((skipped++))
            continue
        fi
        
        # Check if executable exists
        if [[ ! -f "$exe_path" ]]; then
            echo -e " ${YELLOW}SKIP (not built)${NC}"
            ((skipped++))
            continue
        fi
        
        # Start agent in background
        cd "$agent_path"
        PORT=$port nohup "./$exe" > /dev/null 2>&1 &
        
        # Wait for agent to be healthy
        if wait_for_agent $port 5; then
            echo -e " ${GREEN}OK${NC}"
            ((started++))
        else
            echo -e " ${RED}TIMEOUT${NC}"
            ((failed++))
        fi
    done
    
    cd "$SCRIPT_DIR"
    
    echo ""
    echo -e "  Startup complete: ${GREEN}$started started${NC}, $skipped skipped, ${RED}$failed failed${NC}"
    echo ""
    
    [[ $failed -eq 0 ]]
}

stop_all_agents() {
    print_header "Stopping All Agents"
    
    local stopped=0
    local not_running=0
    
    # Stop in reverse order (orchestrator first)
    for ((i=${#AGENTS[@]}-1; i>=0; i--)); do
        local agent="${AGENTS[$i]}"
        local name=$(get_agent_field "$agent" 1)
        local port=$(get_agent_field "$agent" 2)
        
        printf "  Stopping %-24s on port %s..." "$name" "$port"
        
        if stop_agent_on_port $port; then
            echo -e " ${GREEN}STOPPED${NC}"
            ((stopped++))
        else
            echo -e " ${GRAY}NOT RUNNING${NC}"
            ((not_running++))
        fi
    done
    
    echo ""
    echo "  Shutdown complete: $stopped stopped, $not_running were not running"
    echo ""
}

test_full_platform() {
    print_header "Platform Integration Test"
    
    echo "  Testing Orchestrator connectivity to all agents..."
    echo ""
    
    if check_health 8090; then
        local response=$(curl -s "http://localhost:8090/agents" 2>/dev/null)
        
        if [[ -n "$response" ]]; then
            echo -e "  ${CYAN}Orchestrator reports:${NC}"
            echo "$response" | jq -r '.[] | "    - \(.name): \(.status)"' 2>/dev/null || echo "    (raw response: $response)"
            echo ""
            echo -e "  Platform integration: ${GREEN}OK${NC}"
        else
            echo -e "  Platform integration: ${RED}FAILED - No response${NC}"
        fi
    else
        echo -e "  Platform integration: ${RED}FAILED - Orchestrator not responding${NC}"
    fi
    
    echo ""
}

show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --build, -b     Only build agents (don't start)"
    echo "  --status, -s    Only show agent status"
    echo "  --stop          Stop all running agents"
    echo "  --help, -h      Show this help message"
    echo ""
    echo "Without options, the script will build, start, and verify all agents."
}

# =============================================================================
# Main Entry Point
# =============================================================================

main() {
    case "${1:-}" in
        --help|-h)
            show_help
            exit 0
            ;;
        --stop)
            print_banner
            stop_all_agents
            exit 0
            ;;
        --status|-s)
            print_banner
            show_status
            exit $?
            ;;
        --build|-b)
            print_banner
            build_all_agents
            exit $?
            ;;
        "")
            print_banner
            
            # Full startup sequence
            build_all_agents || echo -e "  ${YELLOW}Build had failures. Continuing with available agents...${NC}"
            
            start_all_agents
            sleep 2
            
            show_status
            test_full_platform
            
            echo -e "${CYAN}======================================================================${NC}"
            echo -e "${CYAN} Startup sequence complete${NC}"
            echo -e "${CYAN}======================================================================${NC}"
            echo ""
            ;;
        *)
            echo "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
