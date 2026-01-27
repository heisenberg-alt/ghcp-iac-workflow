#!/bin/bash

# =============================================================================
# IaC Lab Interactive Learning Experience
# =============================================================================
# An engaging, gamified learning experience for Terraform and Bicep
# =============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
GRAY='\033[0;90m'
NC='\033[0m' # No Color
BOLD='\033[1m'
DIM='\033[2m'

# Progress file
PROGRESS_FILE="$HOME/.iac-lab-progress"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LAB_ROOT="$(dirname "$SCRIPT_DIR")"

# =============================================================================
# ASCII Art & Visual Elements
# =============================================================================

show_logo() {
    clear
    echo -e "${CYAN}"
    cat << 'EOF'
    
    ██╗ █████╗  ██████╗    ██╗      █████╗ ██████╗ 
    ██║██╔══██╗██╔════╝    ██║     ██╔══██╗██╔══██╗
    ██║███████║██║         ██║     ███████║██████╔╝
    ██║██╔══██║██║         ██║     ██╔══██║██╔══██╗
    ██║██║  ██║╚██████╗    ███████╗██║  ██║██████╔╝
    ╚═╝╚═╝  ╚═╝ ╚═════╝    ╚══════╝╚═╝  ╚═╝╚═════╝ 
                                                    
EOF
    echo -e "${PURPLE}"
    cat << 'EOF'
     ╔══════════════════════════════════════════════════════╗
     ║   🚀 Infrastructure as Code Learning Experience 🚀   ║
     ║        Powered by GitHub Copilot & Azure            ║
     ╚══════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"
}

show_terraform_logo() {
    echo -e "${PURPLE}"
    cat << 'EOF'
    ████████╗███████╗██████╗ ██████╗  █████╗ ███████╗ ██████╗ ██████╗ ███╗   ███╗
    ╚══██╔══╝██╔════╝██╔══██╗██╔══██╗██╔══██╗██╔════╝██╔═══██╗██╔══██╗████╗ ████║
       ██║   █████╗  ██████╔╝██████╔╝███████║█████╗  ██║   ██║██████╔╝██╔████╔██║
       ██║   ██╔══╝  ██╔══██╗██╔══██╗██╔══██║██╔══╝  ██║   ██║██╔══██╗██║╚██╔╝██║
       ██║   ███████╗██║  ██║██║  ██║██║  ██║██║     ╚██████╔╝██║  ██║██║ ╚═╝ ██║
       ╚═╝   ╚══════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝      ╚═════╝ ╚═╝  ╚═╝╚═╝     ╚═╝
EOF
    echo -e "${NC}"
}

show_bicep_logo() {
    echo -e "${BLUE}"
    cat << 'EOF'
    ██████╗ ██╗ ██████╗███████╗██████╗ 
    ██╔══██╗██║██╔════╝██╔════╝██╔══██╗
    ██████╔╝██║██║     █████╗  ██████╔╝
    ██╔══██╗██║██║     ██╔══╝  ██╔═══╝ 
    ██████╔╝██║╚██████╗███████╗██║     
    ╚═════╝ ╚═╝ ╚═════╝╚══════╝╚═╝     
                                        
       💪 Azure's Muscle for IaC 💪
EOF
    echo -e "${NC}"
}

show_copilot_logo() {
    echo -e "${GREEN}"
    cat << 'EOF'
     ██████╗ ██████╗ ██████╗ ██╗██╗      ██████╗ ████████╗
    ██╔════╝██╔═══██╗██╔══██╗██║██║     ██╔═══██╗╚══██╔══╝
    ██║     ██║   ██║██████╔╝██║██║     ██║   ██║   ██║   
    ██║     ██║   ██║██╔═══╝ ██║██║     ██║   ██║   ██║   
    ╚██████╗╚██████╔╝██║     ██║███████╗╚██████╔╝   ██║   
     ╚═════╝ ╚═════╝ ╚═╝     ╚═╝╚══════╝ ╚═════╝    ╚═╝   
                                                           
        🤖 Your AI Pair Programmer 🤖
EOF
    echo -e "${NC}"
}

# =============================================================================
# Celebration Animations
# =============================================================================

celebrate_success() {
    local message="$1"
    echo ""
    echo -e "${GREEN}${BOLD}"
    cat << 'EOF'
    
    ╔═══════════════════════════════════════════════════════════╗
    ║                                                           ║
    ║   🎉🎊  CHALLENGE COMPLETED!  🎊🎉                        ║
    ║                                                           ║
    ╚═══════════════════════════════════════════════════════════╝
    
           ⭐    ⭐    ⭐    ⭐    ⭐
              ★       ★       ★
                  ✨     ✨
                     🏆
    
EOF
    echo -e "${NC}"
    echo -e "${YELLOW}${BOLD}    $message${NC}"
    echo ""
    
    # Play a sound if available (macOS)
    if command -v afplay &> /dev/null; then
        afplay /System/Library/Sounds/Glass.aiff 2>/dev/null &
    fi
}

celebrate_level_complete() {
    local level="$1"
    echo ""
    echo -e "${CYAN}${BOLD}"
    cat << 'EOF'
    
    ███████╗██████╗ ██╗ ██████╗    ██╗    ██╗██╗███╗   ██╗██╗
    ██╔════╝██╔══██╗██║██╔════╝    ██║    ██║██║████╗  ██║██║
    █████╗  ██████╔╝██║██║         ██║ █╗ ██║██║██╔██╗ ██║██║
    ██╔══╝  ██╔═══╝ ██║██║         ██║███╗██║██║██║╚██╗██║╚═╝
    ███████╗██║     ██║╚██████╗    ╚███╔███╔╝██║██║ ╚████║██╗
    ╚══════╝╚═╝     ╚═╝ ╚═════╝     ╚══╝╚══╝ ╚═╝╚═╝  ╚═══╝╚═╝
    
EOF
    echo -e "${NC}"
    echo -e "${YELLOW}${BOLD}          🏅 You've mastered Level $level! 🏅${NC}"
    echo ""
    
    # Fireworks animation
    for i in {1..3}; do
        echo -e "${YELLOW}    💥 ${RED}💥 ${GREEN}💥 ${BLUE}💥 ${PURPLE}💥 ${CYAN}💥${NC}"
        sleep 0.3
        echo -e "${CYAN}    ✨ ${PURPLE}✨ ${BLUE}✨ ${GREEN}✨ ${RED}✨ ${YELLOW}✨${NC}"
        sleep 0.3
    done
}

show_progress_bar() {
    local current=$1
    local total=$2
    local width=40
    local percentage=$((current * 100 / total))
    local filled=$((current * width / total))
    local empty=$((width - filled))
    
    printf "${CYAN}["
    printf "%${filled}s" | tr ' ' '█'
    printf "%${empty}s" | tr ' ' '░'
    printf "] ${percentage}%% ${NC}"
    echo " ($current/$total challenges)"
}

# =============================================================================
# Progress Management
# =============================================================================

init_progress() {
    if [[ ! -f "$PROGRESS_FILE" ]]; then
        cat > "$PROGRESS_FILE" << 'EOF'
# IaC Lab Progress
LEVEL1_TF_01=0
LEVEL1_TF_02=0
LEVEL1_TF_03=0
LEVEL1_BICEP_01=0
LEVEL1_BICEP_02=0
LEVEL1_BICEP_03=0
LEVEL2_TF_01=0
LEVEL2_TF_02=0
LEVEL2_TF_03=0
LEVEL2_BICEP_01=0
LEVEL2_BICEP_02=0
LEVEL2_BICEP_03=0
LEVEL3_TF_01=0
LEVEL3_TF_02=0
LEVEL3_TF_03=0
LEVEL3_BICEP_01=0
LEVEL3_BICEP_02=0
LEVEL3_BICEP_03=0
LEVEL4_TF_01=0
LEVEL4_TF_02=0
LEVEL4_TF_03=0
LEVEL4_BICEP_01=0
LEVEL4_BICEP_02=0
LEVEL4_BICEP_03=0
XP_POINTS=0
EOF
    fi
    source "$PROGRESS_FILE"
}

save_progress() {
    local key=$1
    local value=$2
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/^$key=.*/$key=$value/" "$PROGRESS_FILE"
    else
        sed -i "s/^$key=.*/$key=$value/" "$PROGRESS_FILE"
    fi
    source "$PROGRESS_FILE"
}

get_total_completed() {
    source "$PROGRESS_FILE"
    local count=0
    for var in LEVEL1_TF_01 LEVEL1_TF_02 LEVEL1_TF_03 LEVEL1_BICEP_01 LEVEL1_BICEP_02 LEVEL1_BICEP_03 \
               LEVEL2_TF_01 LEVEL2_TF_02 LEVEL2_TF_03 LEVEL2_BICEP_01 LEVEL2_BICEP_02 LEVEL2_BICEP_03 \
               LEVEL3_TF_01 LEVEL3_TF_02 LEVEL3_TF_03 LEVEL3_BICEP_01 LEVEL3_BICEP_02 LEVEL3_BICEP_03 \
               LEVEL4_TF_01 LEVEL4_TF_02 LEVEL4_TF_03 LEVEL4_BICEP_01 LEVEL4_BICEP_02 LEVEL4_BICEP_03; do
        if [[ "${!var}" == "1" ]]; then
            ((count++))
        fi
    done
    echo $count
}

add_xp() {
    local points=$1
    source "$PROGRESS_FILE"
    XP_POINTS=$((XP_POINTS + points))
    save_progress "XP_POINTS" "$XP_POINTS"
    echo -e "${YELLOW}  ✨ +${points} XP earned! (Total: ${XP_POINTS} XP)${NC}"
}

# =============================================================================
# Challenge Verification
# =============================================================================

verify_terraform_challenge() {
    local challenge_dir="$1"
    local challenge_file="$challenge_dir/main.tf"
    
    if [[ ! -f "$challenge_file" ]]; then
        echo -e "${RED}❌ Challenge file not found: $challenge_file${NC}"
        return 1
    fi
    
    if grep -q "# TODO:" "$challenge_file"; then
        echo -e "${YELLOW}⚠️  Some TODOs are still incomplete${NC}"
        return 1
    fi
    
    echo -e "${CYAN}🔍 Validating Terraform configuration...${NC}"
    cd "$challenge_dir"
    
    if terraform init -backend=false > /dev/null 2>&1; then
        if terraform validate > /dev/null 2>&1; then
            echo -e "${GREEN}✅ Terraform validation passed!${NC}"
            return 0
        else
            echo -e "${RED}❌ Terraform validation failed${NC}"
            terraform validate
            return 1
        fi
    else
        echo -e "${RED}❌ Terraform init failed${NC}"
        return 1
    fi
}

verify_bicep_challenge() {
    local challenge_dir="$1"
    local challenge_file="$challenge_dir/main.bicep"
    
    if [[ ! -f "$challenge_file" ]]; then
        echo -e "${RED}❌ Challenge file not found: $challenge_file${NC}"
        return 1
    fi
    
    if grep -q "// TODO:" "$challenge_file"; then
        echo -e "${YELLOW}⚠️  Some TODOs are still incomplete${NC}"
        return 1
    fi
    
    echo -e "${CYAN}🔍 Validating Bicep configuration...${NC}"
    
    if az bicep build --file "$challenge_file" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Bicep validation passed!${NC}"
        return 0
    else
        echo -e "${RED}❌ Bicep validation failed${NC}"
        az bicep build --file "$challenge_file"
        return 1
    fi
}

# =============================================================================
# Menu System
# =============================================================================

show_main_menu() {
    show_logo
    
    local completed=$(get_total_completed)
    local total=24
    
    echo -e "${WHITE}${BOLD}  Your Progress:${NC}"
    echo -n "  "
    show_progress_bar $completed $total
    echo ""
    echo -e "${GRAY}  XP Points: ${YELLOW}${BOLD}${XP_POINTS:-0}${NC}"
    echo ""
    
    echo -e "${WHITE}${BOLD}  ═══════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "  ${CYAN}[1]${NC} 📚 ${WHITE}Start Learning Journey${NC}"
    echo -e "  ${CYAN}[2]${NC} 🎯 ${WHITE}Select Specific Challenge${NC}"
    echo -e "  ${CYAN}[3]${NC} 🤖 ${WHITE}Copilot Demo Scenarios${NC}"
    echo -e "  ${CYAN}[4]${NC} 📊 ${WHITE}View Progress & Achievements${NC}"
    echo -e "  ${CYAN}[5]${NC} ⚙️  ${WHITE}Prerequisites Check${NC}"
    echo -e "  ${CYAN}[6]${NC} 📖 ${WHITE}Quick Reference Guides${NC}"
    echo -e "  ${CYAN}[q]${NC} 🚪 ${WHITE}Exit${NC}"
    echo ""
    echo -e "${WHITE}${BOLD}  ═══════════════════════════════════════════════════${NC}"
    echo ""
    echo -ne "  ${YELLOW}Select an option: ${NC}"
}

show_level_menu() {
    clear
    show_logo
    
    echo -e "${WHITE}${BOLD}  📚 SELECT YOUR LEVEL${NC}"
    echo ""
    echo -e "  ${GREEN}[1]${NC} 🌱 ${WHITE}Level 1 - Fundamentals${NC} ${DIM}(Beginner)${NC}"
    echo -e "      ${GRAY}Resource groups, storage accounts, basic syntax${NC}"
    echo ""
    echo -e "  ${YELLOW}[2]${NC} 🌿 ${WHITE}Level 2 - Intermediate${NC} ${DIM}(Developing)${NC}"
    echo -e "      ${GRAY}Networking, compute, App Services${NC}"
    echo ""
    echo -e "  ${BLUE}[3]${NC} 🌳 ${WHITE}Level 3 - Advanced${NC} ${DIM}(Experienced)${NC}"
    echo -e "      ${GRAY}Modules, state management, AKS${NC}"
    echo ""
    echo -e "  ${PURPLE}[4]${NC} 🏔️  ${WHITE}Level 4 - Enterprise${NC} ${DIM}(Expert)${NC}"
    echo -e "      ${GRAY}Multi-region, policy-as-code, CI/CD${NC}"
    echo ""
    echo -e "  ${CYAN}[b]${NC} ⬅️  ${WHITE}Back to Main Menu${NC}"
    echo ""
    echo -ne "  ${YELLOW}Select a level: ${NC}"
}

show_track_menu() {
    local level=$1
    clear
    show_logo
    
    echo -e "${WHITE}${BOLD}  🛤️  SELECT YOUR TRACK - Level $level${NC}"
    echo ""
    echo -e "  ${PURPLE}[1]${NC} ${PURPLE}Terraform${NC} - HashiCorp's IaC tool"
    echo -e "  ${BLUE}[2]${NC} ${BLUE}Bicep${NC} - Azure-native DSL"
    echo ""
    echo -e "  ${CYAN}[b]${NC} ⬅️  ${WHITE}Back${NC}"
    echo ""
    echo -ne "  ${YELLOW}Select a track: ${NC}"
}

show_challenges_menu() {
    local level=$1
    local track=$2
    clear
    
    if [[ "$track" == "terraform" ]]; then
        show_terraform_logo
    else
        show_bicep_logo
    fi
    
    echo -e "${WHITE}${BOLD}  📝 Level $level - ${track^} Challenges${NC}"
    echo ""
    
    local challenges=()
    local progress_vars=()
    
    case "$level-$track" in
        "1-terraform")
            challenges=("01 - Hello Azure (Resource Group)" "02 - Storage Account" "03 - Outputs & Locals")
            progress_vars=("LEVEL1_TF_01" "LEVEL1_TF_02" "LEVEL1_TF_03")
            ;;
        "1-bicep")
            challenges=("01 - Hello Azure (Resource Group)" "02 - Storage Account" "03 - Outputs & Variables")
            progress_vars=("LEVEL1_BICEP_01" "LEVEL1_BICEP_02" "LEVEL1_BICEP_03")
            ;;
        "2-terraform")
            challenges=("01 - Networking (VNet & NSG)" "02 - Compute (VMs)" "03 - App Service")
            progress_vars=("LEVEL2_TF_01" "LEVEL2_TF_02" "LEVEL2_TF_03")
            ;;
        "2-bicep")
            challenges=("01 - Networking (VNet & NSG)" "02 - Compute (VMs)" "03 - App Service")
            progress_vars=("LEVEL2_BICEP_01" "LEVEL2_BICEP_02" "LEVEL2_BICEP_03")
            ;;
        "3-terraform")
            challenges=("01 - Modules" "02 - State Management" "03 - AKS Cluster")
            progress_vars=("LEVEL3_TF_01" "LEVEL3_TF_02" "LEVEL3_TF_03")
            ;;
        "3-bicep")
            challenges=("01 - Modules" "02 - Deployment Stacks" "03 - AKS Cluster")
            progress_vars=("LEVEL3_BICEP_01" "LEVEL3_BICEP_02" "LEVEL3_BICEP_03")
            ;;
        "4-terraform")
            challenges=("01 - Multi-Region" "02 - Policy as Code" "03 - CI/CD Integration")
            progress_vars=("LEVEL4_TF_01" "LEVEL4_TF_02" "LEVEL4_TF_03")
            ;;
        "4-bicep")
            challenges=("01 - Multi-Region" "02 - Policy as Code" "03 - CI/CD Integration")
            progress_vars=("LEVEL4_BICEP_01" "LEVEL4_BICEP_02" "LEVEL4_BICEP_03")
            ;;
    esac
    
    source "$PROGRESS_FILE"
    
    for i in "${!challenges[@]}"; do
        local status_icon="⬜"
        local status_color="${GRAY}"
        local var_name="${progress_vars[$i]}"
        
        if [[ "${!var_name}" == "1" ]]; then
            status_icon="✅"
            status_color="${GREEN}"
        fi
        
        echo -e "  ${CYAN}[$((i+1))]${NC} ${status_icon} ${status_color}${challenges[$i]}${NC}"
    done
    
    echo ""
    echo -e "  ${CYAN}[b]${NC} ⬅️  ${WHITE}Back${NC}"
    echo ""
    echo -ne "  ${YELLOW}Select a challenge: ${NC}"
}

# =============================================================================
# Challenge Runner
# =============================================================================

run_challenge() {
    local level=$1
    local track=$2
    local challenge_num=$3
    
    clear
    
    local level_name=""
    case $level in
        1) level_name="Level-1-Fundamentals" ;;
        2) level_name="Level-2-Intermediate" ;;
        3) level_name="Level-3-Advanced" ;;
        4) level_name="Level-4-Enterprise" ;;
    esac
    
    local exercise_dirs=()
    local progress_var=""
    
    case "$level-$track" in
        "1-terraform")
            exercise_dirs=("01-hello-azure" "02-storage-account" "03-outputs-locals")
            progress_var="LEVEL1_TF_0$challenge_num"
            ;;
        "1-bicep")
            exercise_dirs=("01-hello-azure" "02-storage-account" "03-outputs-variables")
            progress_var="LEVEL1_BICEP_0$challenge_num"
            ;;
        "2-terraform"|"2-bicep")
            exercise_dirs=("01-networking" "02-compute" "03-app-service")
            [[ "$track" == "terraform" ]] && progress_var="LEVEL2_TF_0$challenge_num" || progress_var="LEVEL2_BICEP_0$challenge_num"
            ;;
        "3-terraform")
            exercise_dirs=("01-modules" "02-state-management" "03-aks-cluster")
            progress_var="LEVEL3_TF_0$challenge_num"
            ;;
        "3-bicep")
            exercise_dirs=("01-modules" "02-deployment-stacks" "03-aks-cluster")
            progress_var="LEVEL3_BICEP_0$challenge_num"
            ;;
        "4-terraform"|"4-bicep")
            exercise_dirs=("01-multi-region" "02-policy-as-code" "03-cicd-integration")
            [[ "$track" == "terraform" ]] && progress_var="LEVEL4_TF_0$challenge_num" || progress_var="LEVEL4_BICEP_0$challenge_num"
            ;;
    esac
    
    local exercise="${exercise_dirs[$((challenge_num-1))]}"
    local challenge_dir="$LAB_ROOT/$level_name/$track/$exercise"
    
    echo -e "${CYAN}${BOLD}"
    echo "  ╔═══════════════════════════════════════════════════════════════╗"
    echo "  ║                     🎯 CHALLENGE MODE 🎯                      ║"
    echo "  ╚═══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    
    echo -e "${WHITE}${BOLD}  Challenge: ${YELLOW}$exercise${NC}"
    echo -e "${WHITE}  Track: ${track^} | Level: $level${NC}"
    echo ""
    
    if [[ -f "$challenge_dir/README.md" ]]; then
        echo -e "${GRAY}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        head -30 "$challenge_dir/README.md" | while IFS= read -r line; do
            echo -e "  ${WHITE}$line${NC}"
        done
        echo -e "${GRAY}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    fi
    
    echo ""
    echo -e "${CYAN}${BOLD}  Actions:${NC}"
    echo -e "  ${GREEN}[o]${NC} 📂 Open challenge in VS Code"
    echo -e "  ${GREEN}[v]${NC} ✅ Verify my solution"
    echo -e "  ${GREEN}[h]${NC} 💡 Get a hint"
    echo -e "  ${GREEN}[s]${NC} 👀 View solution"
    echo -e "  ${GREEN}[b]${NC} ⬅️  Back"
    echo ""
    echo -ne "  ${YELLOW}Select action: ${NC}"
    
    read -r action
    
    case $action in
        o|O)
            echo -e "${CYAN}  Opening VS Code...${NC}"
            code "$challenge_dir/challenge" 2>/dev/null || code "$challenge_dir" 2>/dev/null
            run_challenge $level $track $challenge_num
            ;;
        v|V)
            echo ""
            if [[ "$track" == "terraform" ]]; then
                if verify_terraform_challenge "$challenge_dir/challenge"; then
                    save_progress "$progress_var" "1"
                    add_xp $((level * 100))
                    celebrate_success "You've completed the $exercise challenge!"
                    check_level_complete $level $track
                fi
            else
                if verify_bicep_challenge "$challenge_dir/challenge"; then
                    save_progress "$progress_var" "1"
                    add_xp $((level * 100))
                    celebrate_success "You've completed the $exercise challenge!"
                    check_level_complete $level $track
                fi
            fi
            echo ""
            echo -ne "  ${YELLOW}Press Enter to continue...${NC}"
            read -r
            run_challenge $level $track $challenge_num
            ;;
        h|H)
            show_hint $level $track $challenge_num
            run_challenge $level $track $challenge_num
            ;;
        s|S)
            echo -e "${CYAN}  Opening solution...${NC}"
            code "$challenge_dir/solution" 2>/dev/null
            run_challenge $level $track $challenge_num
            ;;
        b|B)
            return
            ;;
        *)
            run_challenge $level $track $challenge_num
            ;;
    esac
}

show_hint() {
    local level=$1
    local track=$2
    local challenge=$3
    
    echo ""
    echo -e "${YELLOW}${BOLD}  💡 HINT - Use GitHub Copilot!${NC}"
    echo ""
    echo -e "${WHITE}  Try these approaches:${NC}"
    echo ""
    echo -e "${CYAN}  1. Open Copilot Chat (Ctrl+Shift+I) and ask:${NC}"
    echo -e "${WHITE}     \"Complete the TODO items in this file\"${NC}"
    echo ""
    echo -e "${CYAN}  2. Position cursor after a TODO and wait for suggestions${NC}"
    echo ""
    echo -e "${CYAN}  3. Use inline chat (Ctrl+I) for specific help${NC}"
    echo ""
    echo -ne "  ${YELLOW}Press Enter to continue...${NC}"
    read -r
}

check_level_complete() {
    local level=$1
    local track=$2
    source "$PROGRESS_FILE"
    
    local all_complete=true
    local prefix=""
    
    case "$level-$track" in
        "1-terraform") prefix="LEVEL1_TF" ;;
        "1-bicep") prefix="LEVEL1_BICEP" ;;
        "2-terraform") prefix="LEVEL2_TF" ;;
        "2-bicep") prefix="LEVEL2_BICEP" ;;
        "3-terraform") prefix="LEVEL3_TF" ;;
        "3-bicep") prefix="LEVEL3_BICEP" ;;
        "4-terraform") prefix="LEVEL4_TF" ;;
        "4-bicep") prefix="LEVEL4_BICEP" ;;
    esac
    
    for i in 01 02 03; do
        local var="${prefix}_$i"
        if [[ "${!var}" != "1" ]]; then
            all_complete=false
            break
        fi
    done
    
    if $all_complete; then
        celebrate_level_complete "$level ($track)"
        add_xp $((level * 500))
    fi
}

# =============================================================================
# Prerequisites Check
# =============================================================================

check_prerequisites() {
    clear
    echo -e "${CYAN}${BOLD}"
    echo "  ╔═══════════════════════════════════════════════════════════════╗"
    echo "  ║              ⚙️  PREREQUISITES CHECK ⚙️                        ║"
    echo "  ╚═══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    echo ""
    
    local all_good=true
    
    echo -ne "  Terraform CLI:     "
    if command -v terraform &> /dev/null; then
        local tf_ver=$(terraform version -json 2>/dev/null | grep -o '"terraform_version":"[^"]*"' | cut -d'"' -f4)
        echo -e "${GREEN}✅ Installed ($tf_ver)${NC}"
    else
        echo -e "${RED}❌ Not installed${NC}"
        all_good=false
    fi
    
    echo -ne "  Azure CLI:         "
    if command -v az &> /dev/null; then
        local az_ver=$(az version --query '"azure-cli"' -o tsv 2>/dev/null)
        echo -e "${GREEN}✅ Installed ($az_ver)${NC}"
    else
        echo -e "${RED}❌ Not installed${NC}"
        all_good=false
    fi
    
    echo -ne "  Bicep CLI:         "
    if az bicep version &> /dev/null 2>&1; then
        local bicep_ver=$(az bicep version 2>/dev/null | grep -oP 'v[\d.]+')
        echo -e "${GREEN}✅ Installed ($bicep_ver)${NC}"
    else
        echo -e "${YELLOW}⚠️  Run: az bicep install${NC}"
    fi
    
    echo -ne "  VS Code:           "
    if command -v code &> /dev/null; then
        echo -e "${GREEN}✅ Installed${NC}"
    else
        echo -e "${YELLOW}⚠️  Not in PATH${NC}"
    fi
    
    echo -ne "  Git:               "
    if command -v git &> /dev/null; then
        local git_ver=$(git --version | cut -d' ' -f3)
        echo -e "${GREEN}✅ Installed ($git_ver)${NC}"
    else
        echo -e "${RED}❌ Not installed${NC}"
        all_good=false
    fi
    
    echo -ne "  GitHub CLI:        "
    if command -v gh &> /dev/null; then
        local gh_ver=$(gh --version | head -1 | cut -d' ' -f3)
        echo -e "${GREEN}✅ Installed ($gh_ver)${NC}"
    else
        echo -e "${YELLOW}⚠️  Optional - for CLI demos${NC}"
    fi
    
    echo -ne "  Azure Login:       "
    if az account show &> /dev/null 2>&1; then
        local sub=$(az account show --query name -o tsv 2>/dev/null)
        echo -e "${GREEN}✅ Logged in ($sub)${NC}"
    else
        echo -e "${YELLOW}⚠️  Run: az login${NC}"
    fi
    
    echo ""
    
    if $all_good; then
        echo -e "${GREEN}${BOLD}  🎉 All prerequisites are met! You're ready to learn!${NC}"
    else
        echo -e "${YELLOW}${BOLD}  ⚠️  Some prerequisites are missing. Install them for the full experience.${NC}"
    fi
    
    echo ""
    echo -ne "  ${YELLOW}Press Enter to continue...${NC}"
    read -r
}

# =============================================================================
# Copilot Demos
# =============================================================================

show_copilot_demos() {
    clear
    show_copilot_logo
    
    echo -e "${WHITE}${BOLD}  🤖 COPILOT DEMO SCENARIOS${NC}"
    echo ""
    echo -e "  ${GREEN}[1]${NC} ✨ ${WHITE}Code Generation${NC} - Generate IaC from comments"
    echo -e "  ${GREEN}[2]${NC} 📖 ${WHITE}Code Explanation${NC} - Understand complex configs"
    echo -e "  ${GREEN}[3]${NC} 🔧 ${WHITE}Error Fixing${NC} - Debug and fix issues"
    echo -e "  ${GREEN}[4]${NC} 🔄 ${WHITE}Refactoring${NC} - Improve code quality"
    echo -e "  ${GREEN}[5]${NC} 💻 ${WHITE}CLI Integration${NC} - GitHub Copilot CLI"
    echo ""
    echo -e "  ${CYAN}[b]${NC} ⬅️  ${WHITE}Back${NC}"
    echo ""
    echo -ne "  ${YELLOW}Select a demo: ${NC}"
    
    read -r choice
    
    case $choice in
        1) 
            echo -e "${CYAN}  Opening Code Generation demos...${NC}"
            code "$LAB_ROOT/Copilot-Demos/01-code-generation" 2>/dev/null
            ;;
        2) 
            echo -e "${CYAN}  Opening Code Explanation demos...${NC}"
            code "$LAB_ROOT/Copilot-Demos/02-code-explanation" 2>/dev/null
            ;;
        3) 
            echo -e "${CYAN}  Opening Error Fixing demos...${NC}"
            code "$LAB_ROOT/Copilot-Demos/03-error-fixing" 2>/dev/null
            ;;
        4) 
            echo -e "${CYAN}  Opening Refactoring demos...${NC}"
            code "$LAB_ROOT/Copilot-Demos/04-refactoring" 2>/dev/null
            ;;
        5) run_cli_demo ;;
        b|B) return ;;
    esac
    
    show_copilot_demos
}

run_cli_demo() {
    clear
    echo -e "${GREEN}${BOLD}"
    echo "  ╔═══════════════════════════════════════════════════════════════╗"
    echo "  ║           💻 GitHub Copilot CLI Demo 💻                       ║"
    echo "  ╚═══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    echo ""
    echo -e "${WHITE}  GitHub Copilot CLI brings AI assistance to your terminal!${NC}"
    echo ""
    echo -e "${CYAN}${BOLD}  Installation:${NC}"
    echo -e "${WHITE}  gh extension install github/gh-copilot${NC}"
    echo ""
    echo -e "${CYAN}${BOLD}  Try these commands:${NC}"
    echo ""
    echo -e "${YELLOW}  # Explain a command${NC}"
    echo -e "${WHITE}  gh copilot explain \"terraform state mv\"${NC}"
    echo ""
    echo -e "${YELLOW}  # Get suggestions${NC}"
    echo -e "${WHITE}  gh copilot suggest \"import existing Azure resource to Terraform\"${NC}"
    echo ""
    echo -e "${YELLOW}  # Explain errors${NC}"
    echo -e "${WHITE}  gh copilot explain \"Error: A resource with the ID already exists\"${NC}"
    echo ""
    echo -ne "  ${YELLOW}Press Enter to continue...${NC}"
    read -r
}

# =============================================================================
# Progress View
# =============================================================================

show_progress() {
    clear
    source "$PROGRESS_FILE"
    
    echo -e "${CYAN}${BOLD}"
    echo "  ╔═══════════════════════════════════════════════════════════════╗"
    echo "  ║                 📊 YOUR PROGRESS 📊                           ║"
    echo "  ╚═══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    echo ""
    
    local completed=$(get_total_completed)
    echo -e "${WHITE}${BOLD}  Overall Progress:${NC}"
    echo -n "  "
    show_progress_bar $completed 24
    echo ""
    echo -e "  ${YELLOW}${BOLD}  ⭐ XP Points: $XP_POINTS${NC}"
    echo ""
    
    # Rank calculation
    local rank="🌱 Seedling"
    if [[ $XP_POINTS -ge 5000 ]]; then
        rank="👑 IaC Master"
    elif [[ $XP_POINTS -ge 3000 ]]; then
        rank="🏆 Expert"
    elif [[ $XP_POINTS -ge 1500 ]]; then
        rank="🌳 Advanced"
    elif [[ $XP_POINTS -ge 500 ]]; then
        rank="🌿 Intermediate"
    fi
    echo -e "  ${PURPLE}${BOLD}  Rank: $rank${NC}"
    echo ""
    
    echo -e "${WHITE}${BOLD}  Progress by Level:${NC}"
    echo ""
    
    for level in 1 2 3 4; do
        local tf_count=0
        local bicep_count=0
        
        for i in 01 02 03; do
            local tf_var="LEVEL${level}_TF_$i"
            local bicep_var="LEVEL${level}_BICEP_$i"
            [[ "${!tf_var}" == "1" ]] && ((tf_count++))
            [[ "${!bicep_var}" == "1" ]] && ((bicep_count++))
        done
        
        local level_icon=""
        case $level in
            1) level_icon="🌱" ;;
            2) level_icon="🌿" ;;
            3) level_icon="🌳" ;;
            4) level_icon="🏔️" ;;
        esac
        
        echo -e "  ${level_icon} ${WHITE}Level $level:${NC}"
        echo -e "     ${PURPLE}Terraform:${NC} $(get_stars $tf_count) ($tf_count/3)"
        echo -e "     ${BLUE}Bicep:${NC}     $(get_stars $bicep_count) ($bicep_count/3)"
        echo ""
    done
    
    echo -e "${WHITE}${BOLD}  Achievements:${NC}"
    echo ""
    
    # Check achievements
    if [[ $completed -ge 1 ]]; then
        echo -e "  🏅 ${GREEN}First Steps${NC} - Complete your first challenge"
    fi
    if [[ $completed -ge 6 ]]; then
        echo -e "  🎖️  ${GREEN}Level 1 Graduate${NC} - Complete all Level 1 challenges"
    fi
    if [[ $completed -ge 12 ]]; then
        echo -e "  🥉 ${GREEN}Rising Star${NC} - Complete all Level 1 & 2 challenges"
    fi
    if [[ $completed -ge 18 ]]; then
        echo -e "  🥈 ${GREEN}IaC Professional${NC} - Complete all Level 1-3 challenges"
    fi
    if [[ $completed -ge 24 ]]; then
        echo -e "  🥇 ${GREEN}IaC Master${NC} - Complete ALL challenges!"
    fi
    
    echo ""
    echo -ne "  ${YELLOW}Press Enter to continue...${NC}"
    read -r
}

get_stars() {
    local count=$1
    case $count in
        3) echo "⭐⭐⭐" ;;
        2) echo "⭐⭐☆" ;;
        1) echo "⭐☆☆" ;;
        0) echo "☆☆☆" ;;
    esac
}

# =============================================================================
# Reference Guides
# =============================================================================

show_reference_guides() {
    clear
    echo -e "${CYAN}${BOLD}"
    echo "  ╔═══════════════════════════════════════════════════════════════╗"
    echo "  ║                📖 QUICK REFERENCE GUIDES 📖                   ║"
    echo "  ╚═══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    echo ""
    echo -e "  ${GREEN}[1]${NC} ${PURPLE}Terraform Patterns${NC} - Common patterns & best practices"
    echo -e "  ${GREEN}[2]${NC} ${BLUE}Bicep Patterns${NC} - Azure-native IaC patterns"
    echo -e "  ${GREEN}[3]${NC} ${GREEN}Copilot Prompts${NC} - Effective prompts for IaC"
    echo -e "  ${GREEN}[4]${NC} ${WHITE}Main README${NC} - Lab overview and instructions"
    echo ""
    echo -e "  ${CYAN}[b]${NC} ⬅️  ${WHITE}Back${NC}"
    echo ""
    echo -ne "  ${YELLOW}Select a guide: ${NC}"
    
    read -r choice
    
    case $choice in
        1) code "$LAB_ROOT/Solutions/terraform-patterns.md" 2>/dev/null ;;
        2) code "$LAB_ROOT/Solutions/bicep-patterns.md" 2>/dev/null ;;
        3) code "$LAB_ROOT/Solutions/copilot-prompts.md" 2>/dev/null ;;
        4) code "$LAB_ROOT/README.md" 2>/dev/null ;;
        b|B) return ;;
    esac
    
    show_reference_guides
}

# =============================================================================
# Main Loop
# =============================================================================

main() {
    init_progress
    
    while true; do
        show_main_menu
        read -r choice
        
        case $choice in
            1|2)
                show_level_menu
                read -r level
                case $level in
                    1|2|3|4)
                        show_track_menu $level
                        read -r track
                        case $track in
                            1) track="terraform" ;;
                            2) track="bicep" ;;
                            b|B) continue ;;
                            *) continue ;;
                        esac
                        
                        while true; do
                            show_challenges_menu $level $track
                            read -r challenge
                            case $challenge in
                                1|2|3) run_challenge $level $track $challenge ;;
                                b|B) break ;;
                            esac
                        done
                        ;;
                    b|B) continue ;;
                esac
                ;;
            3) show_copilot_demos ;;
            4) show_progress ;;
            5) check_prerequisites ;;
            6) show_reference_guides ;;
            q|Q)
                clear
                echo -e "${CYAN}"
                cat << 'EOF'
    
     ████████╗██╗  ██╗ █████╗ ███╗   ██╗██╗  ██╗███████╗██╗
     ╚══██╔══╝██║  ██║██╔══██╗████╗  ██║██║ ██╔╝██╔════╝██║
        ██║   ███████║███████║██╔██╗ ██║█████╔╝ ███████╗██║
        ██║   ██╔══██║██╔══██║██║╚██╗██║██╔═██╗ ╚════██║╚═╝
        ██║   ██║  ██║██║  ██║██║ ╚████║██║  ██╗███████║██╗
        ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝  ╚═╝╚══════╝╚═╝
    
         Keep learning, keep building! 🚀
    
EOF
                echo -e "${NC}"
                exit 0
                ;;
        esac
    done
}

# Run main
main "$@"
