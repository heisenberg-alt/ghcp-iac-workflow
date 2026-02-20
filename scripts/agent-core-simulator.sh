#!/usr/bin/env bash
set -eu

# Agent Core Simulator
# -----------------------------------------------------------------------------
# Bare-essential behavior distilled from current Go agents:
# 1) policy-checker: parse IaC resources and enforce two default rules.
# 2) cost-estimator: map resource types/SKUs to estimated monthly cost.
# 3) drift-detector: compare desired properties to simulated Azure state.
# 4) security-scanner: detect secret patterns + key security config checks.
# 5) compliance-auditor: run fixed CIS/NIST/SOC2 control checks.
# 6) module-registry: validate module source/catalog/version and recommend.
# 7) impact-analyzer: infer references/dependencies and risk blast radius.
# 8) deploy-promoter: enforce sequential env promotions + approval rules.
# 9) notification-manager: route events to channels using static rules.
# 10) orchestrator: intent-based routing over the above agent actions.
# -----------------------------------------------------------------------------

SCRIPT_NAME="$(basename "$0")"

AGENT=""
FILE_PATH=""
MESSAGE=""
FORMAT_OVERRIDE=""
CODE=""

to_lower() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]'
}

usage() {
  cat <<EOF
Usage:
  $SCRIPT_NAME <agent> [--file PATH] [--message TEXT] [--format terraform|bicep]
  cat main.tf | $SCRIPT_NAME <agent>

Agents:
  policy | cost | drift | security | compliance | modules | impact
  deploy | notify | orchestrator

Examples:
  $SCRIPT_NAME policy --file infra/main.tf
  $SCRIPT_NAME cost < infra/main.tf
  $SCRIPT_NAME deploy --message "promote dev to staging"
  $SCRIPT_NAME orchestrator --message "full analysis" --file infra/security-layer.tf
EOF
}

parse_args() {
  if [[ $# -lt 1 ]]; then
    usage
    exit 1
  fi
  AGENT="$1"
  shift

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --file)
        FILE_PATH="${2:-}"
        shift 2
        ;;
      --message|--text)
        MESSAGE="${2:-}"
        shift 2
        ;;
      --format)
        FORMAT_OVERRIDE="${2:-}"
        shift 2
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        echo "Unknown option: $1" >&2
        usage
        exit 1
        ;;
    esac
  done
}

read_inputs() {
  local stdin_data=""
  if [[ ! -t 0 ]]; then
    stdin_data="$(cat)"
  fi

  if [[ -n "$FILE_PATH" ]]; then
    if [[ ! -f "$FILE_PATH" ]]; then
      echo "File not found: $FILE_PATH" >&2
      exit 1
    fi
    CODE="$(cat "$FILE_PATH")"
  elif [[ -n "$stdin_data" ]]; then
    CODE="$stdin_data"
  fi

  if [[ -z "$CODE" && -n "$MESSAGE" && "$MESSAGE" == *"resource "* ]]; then
    CODE="$MESSAGE"
  fi
}

bicep_to_tf_type() {
  local bicep_type="$1"
  bicep_type="${bicep_type%%@*}"
  case "$bicep_type" in
    Microsoft.Storage/storageAccounts) echo "azurerm_storage_account" ;;
    Microsoft.ContainerService/managedClusters) echo "azurerm_kubernetes_cluster" ;;
    Microsoft.Network/virtualNetworks) echo "azurerm_virtual_network" ;;
    Microsoft.KeyVault/vaults) echo "azurerm_key_vault" ;;
    Microsoft.Compute/virtualMachines) echo "azurerm_virtual_machine" ;;
    Microsoft.Sql/servers/databases) echo "azurerm_mssql_database" ;;
    *) echo "$bicep_type" ;;
  esac
}

detect_iac_type() {
  if [[ -n "$FORMAT_OVERRIDE" ]]; then
    to_lower "$FORMAT_OVERRIDE"
    return
  fi
  if grep -Eq '(^|[[:space:]])resource[[:space:]]+"[^"]+"' <<< "$CODE" || grep -Eq 'terraform[[:space:]]*\{' <<< "$CODE"; then
    echo "terraform"
    return
  fi
  if grep -Eq "^[[:space:]]*resource[[:space:]]+[A-Za-z0-9_]+[[:space:]]+'[^']+'[[:space:]]*=" <<< "$CODE"; then
    echo "bicep"
    return
  fi
  echo "unknown"
}

extract_block_from_line() {
  local start="$1"
  awk -v s="$start" '
    NR < s { next }
    {
      if (NR == s) in_block = 1
      if (!in_block) next
      print
      opens += gsub(/\{/, "{")
      closes += gsub(/\}/, "}")
      if (opens > 0 && opens == closes) exit
    }
  ' <<< "$CODE"
}

RES_TYPES=()
RES_NAMES=()
RES_LINES=()
RES_BLOCKS=()

parse_resources() {
  RES_TYPES=()
  RES_NAMES=()
  RES_LINES=()
  RES_BLOCKS=()

  local line_no=0
  while IFS= read -r line; do
    ((line_no += 1))
    local type=""
    local name=""

    if [[ $line =~ ^[[:space:]]*resource[[:space:]]+\"([^\"]+)\"[[:space:]]+\"([^\"]+)\"[[:space:]]*\{ ]]; then
      type="${BASH_REMATCH[1]}"
      name="${BASH_REMATCH[2]}"
    elif [[ $line =~ ^[[:space:]]*resource[[:space:]]+([A-Za-z0-9_]+)[[:space:]]+\'([^\']+)\'[[:space:]]*=[[:space:]]*\{ ]]; then
      name="${BASH_REMATCH[1]}"
      type="$(bicep_to_tf_type "${BASH_REMATCH[2]}")"
    fi

    if [[ -n "$type" ]]; then
      RES_TYPES+=("$type")
      RES_NAMES+=("$name")
      RES_LINES+=("$line_no")
      RES_BLOCKS+=("$(extract_block_from_line "$line_no")")
    fi
  done <<< "$CODE"
}

block_has() {
  local block="$1"
  local regex="$2"
  grep -Eq "$regex" <<< "$block"
}

resource_count() {
  echo "${#RES_TYPES[@]}"
}

contains_type() {
  local want="$1"
  local i
  for i in "${!RES_TYPES[@]}"; do
    if [[ "${RES_TYPES[$i]}" == "$want" ]]; then
      return 0
    fi
  done
  return 1
}

print_header() {
  local title="$1"
  echo "=== $title ==="
}

# -----------------------------------------------------------------------------
# Policy Checker
# -----------------------------------------------------------------------------
run_policy() {
  print_header "Policy Checker (core)"
  parse_resources
  local total
  total="$(resource_count)"
  echo "Detected resources: $total"
  if [[ "$total" -eq 0 ]]; then
    echo "No resources detected."
    return
  fi

  local violations=()
  local i
  for i in "${!RES_TYPES[@]}"; do
    local t="${RES_TYPES[$i]}"
    local n="${RES_NAMES[$i]}"
    local b="${RES_BLOCKS[$i]}"

    if [[ "$t" == "azurerm_storage_account" ]]; then
      if ! block_has "$b" '(enable_https_traffic_only[[:space:]]*=[[:space:]]*true|supportsHttpsTrafficOnly[[:space:]]*:[[:space:]]*true)'; then
        violations+=("HIGH storage-https-required $t.$n -> set enable_https_traffic_only=true")
      fi
    fi

    if [[ "$t" == "azurerm_kubernetes_cluster" ]]; then
      if ! block_has "$b" 'role_based_access_control_enabled[[:space:]]*=[[:space:]]*true'; then
        violations+=("HIGH aks-rbac-enabled $t.$n -> set role_based_access_control_enabled=true")
      fi
    fi
  done

  if [[ "${#violations[@]}" -eq 0 ]]; then
    echo "PASS: no policy violations."
  else
    echo "FAIL: ${#violations[@]} violation(s)"
    printf '%s\n' "${violations[@]}"
  fi
}

# -----------------------------------------------------------------------------
# Cost Estimator
# -----------------------------------------------------------------------------
vm_hourly_price() {
  case "$1" in
    Standard_B1s) echo "0.0104" ;;
    Standard_B1ms) echo "0.0207" ;;
    Standard_B2s) echo "0.0416" ;;
    Standard_B2ms) echo "0.0832" ;;
    Standard_D2s_v3|Standard_D2s_v4|Standard_D2s_v5) echo "0.096" ;;
    Standard_D4s_v3|Standard_D4s_v4|Standard_D4s_v5) echo "0.192" ;;
    Standard_D8s_v3|Standard_D8s_v4|Standard_D8s_v5) echo "0.384" ;;
    Standard_E2s_v3) echo "0.126" ;;
    Standard_E4s_v3) echo "0.252" ;;
    Standard_E8s_v3) echo "0.504" ;;
    Standard_F2s_v2) echo "0.085" ;;
    Standard_F4s_v2) echo "0.169" ;;
    Standard_F8s_v2) echo "0.338" ;;
    *) echo "0.096" ;;
  esac
}

storage_price_per_gb() {
  case "$1" in
    Standard_LRS|LRS) echo "0.0184" ;;
    Standard_GRS|GRS) echo "0.0368" ;;
    Standard_ZRS|ZRS) echo "0.023" ;;
    Standard_GZRS|GZRS) echo "0.0414" ;;
    RA-GRS) echo "0.046" ;;
    Premium_LRS) echo "0.15" ;;
    *) echo "0.0184" ;;
  esac
}

app_service_monthly() {
  case "$1" in
    F1) echo "0" ;;
    D1) echo "9.49" ;;
    B1) echo "13.14" ;;
    B2) echo "26.28" ;;
    B3) echo "52.56" ;;
    S1) echo "69.35" ;;
    S2) echo "138.70" ;;
    S3) echo "277.40" ;;
    P1v2) echo "73.00" ;;
    P2v2) echo "146.00" ;;
    P3v2) echo "292.00" ;;
    P1v3) echo "95.63" ;;
    P2v3) echo "191.25" ;;
    P3v3) echo "382.50" ;;
    *) echo "13.14" ;;
  esac
}

acr_monthly() {
  case "$1" in
    Basic) echo "5.00" ;;
    Standard) echo "20.00" ;;
    Premium) echo "50.00" ;;
    *) echo "5.00" ;;
  esac
}

run_cost() {
  print_header "Cost Estimator (core)"
  parse_resources
  local total
  total="$(resource_count)"
  echo "Detected resources: $total"
  if [[ "$total" -eq 0 ]]; then
    echo "No resources detected."
    return
  fi

  local monthly_total="0"
  local i
  for i in "${!RES_TYPES[@]}"; do
    local t="${RES_TYPES[$i]}"
    local n="${RES_NAMES[$i]}"
    local b="${RES_BLOCKS[$i]}"
    local cost="0"
    local note=""

    case "$t" in
      azurerm_kubernetes_cluster)
        local vm_size
        vm_size="$(grep -Eo 'vm_size[[:space:]]*=[[:space:]]*"[^"]+"' <<< "$b" | head -n1 | sed -E 's/.*"([^"]+)"/\1/')"
        [[ -z "$vm_size" ]] && vm_size="Standard_D2s_v3"
        local node_count
        node_count="$(grep -Eo 'node_count[[:space:]]*=[[:space:]]*[0-9]+' <<< "$b" | head -n1 | sed -E 's/.*=[[:space:]]*([0-9]+)/\1/')"
        [[ -z "$node_count" ]] && node_count="3"
        local hp
        hp="$(vm_hourly_price "$vm_size")"
        cost="$(awk -v p="$hp" -v c="$node_count" 'BEGIN{printf "%.2f", (p*730*c)+18.25}')"
        note="${node_count}x${vm_size} + LB"
        ;;
      azurerm_virtual_machine|azurerm_linux_virtual_machine|azurerm_windows_virtual_machine)
        local vm_size
        vm_size="$(grep -Eo '(vm_size|size)[[:space:]]*=[[:space:]]*"[^"]+"' <<< "$b" | head -n1 | sed -E 's/.*"([^"]+)"/\1/')"
        [[ -z "$vm_size" ]] && vm_size="Standard_D2s_v3"
        local hp
        hp="$(vm_hourly_price "$vm_size")"
        if [[ "$t" == "azurerm_windows_virtual_machine" ]]; then
          hp="$(awk -v p="$hp" 'BEGIN{printf "%.4f", p*1.5}')"
        fi
        cost="$(awk -v p="$hp" 'BEGIN{printf "%.2f", p*730}')"
        note="$vm_size"
        ;;
      azurerm_storage_account)
        local repl
        repl="$(grep -Eo 'account_replication_type[[:space:]]*=[[:space:]]*"[^"]+"' <<< "$b" | head -n1 | sed -E 's/.*"([^"]+)"/\1/')"
        [[ -z "$repl" ]] && repl="LRS"
        local sku="Standard_${repl}"
        local pgb
        pgb="$(storage_price_per_gb "$sku")"
        cost="$(awk -v p="$pgb" 'BEGIN{printf "%.2f", p*100}')"
        note="$sku, 100GB"
        ;;
      azurerm_app_service_plan)
        local sku
        sku="$(grep -Eo 'sku_name[[:space:]]*=[[:space:]]*"[^"]+"' <<< "$b" | head -n1 | sed -E 's/.*"([^"]+)"/\1/')"
        [[ -z "$sku" ]] && sku="B1"
        cost="$(app_service_monthly "$sku")"
        note="$sku"
        ;;
      azurerm_container_registry)
        local sku
        sku="$(grep -Eo 'sku[[:space:]]*=[[:space:]]*"[^"]+"' <<< "$b" | head -n1 | sed -E 's/.*"([^"]+)"/\1/')"
        [[ -z "$sku" ]] && sku="Basic"
        cost="$(acr_monthly "$sku")"
        note="$sku"
        ;;
      azurerm_key_vault)
        cost="3.00"
        note="Standard estimate"
        ;;
      azurerm_virtual_network|azurerm_subnet|azurerm_network_security_group)
        cost="0.00"
        note="Free resource"
        ;;
      *)
        cost="0.00"
        note="Unknown pricing"
        ;;
    esac

    monthly_total="$(awk -v a="$monthly_total" -v b="$cost" 'BEGIN{printf "%.2f", a+b}')"
    printf "%-45s \$%8s  %s\n" "$t.$n" "$cost" "$note"
  done

  echo "---"
  echo "Estimated monthly total: \$$monthly_total"
}

# -----------------------------------------------------------------------------
# Drift Detector
# -----------------------------------------------------------------------------
drift_expected_bool() {
  local block="$1"
  local tf_key="$2"
  local bicep_key="$3"
  if block_has "$block" "${tf_key}[[:space:]]*=[[:space:]]*true|${bicep_key}[[:space:]]*:[[:space:]]*true"; then
    echo "true"
  elif block_has "$block" "${tf_key}[[:space:]]*=[[:space:]]*false|${bicep_key}[[:space:]]*:[[:space:]]*false"; then
    echo "false"
  else
    echo ""
  fi
}

drift_expected_string() {
  local block="$1"
  local key="$2"
  grep -Eo "${key}[[:space:]]*=[[:space:]]*\"[^\"]+\"" <<< "$block" | head -n1 | sed -E 's/.*"([^"]+)"/\1/'
}

run_drift() {
  print_header "Drift Detector (core)"
  parse_resources
  local total
  total="$(resource_count)"
  echo "Detected resources: $total"
  if [[ "$total" -eq 0 ]]; then
    echo "No resources detected."
    return
  fi

  local drifted=0
  local in_sync=0
  local missing_in_azure=0
  local i
  for i in "${!RES_TYPES[@]}"; do
    local t="${RES_TYPES[$i]}"
    local n="${RES_NAMES[$i]}"
    local b="${RES_BLOCKS[$i]}"

    if [[ "$t" == "azurerm_storage_account" && "$n" == "example" ]]; then
      local expected_https expected_tls expected_public
      expected_https="$(drift_expected_bool "$b" "enable_https_traffic_only" "supportsHttpsTrafficOnly")"
      expected_tls="$(drift_expected_string "$b" "min_tls_version")"
      expected_public="$(drift_expected_bool "$b" "allow_blob_public_access" "allowBlobPublicAccess")"
      local d=0
      [[ -n "$expected_https" && "$expected_https" != "false" ]] && ((d += 1))
      [[ -n "$expected_tls" && "$expected_tls" != "TLS1_0" ]] && ((d += 1))
      [[ -n "$expected_public" && "$expected_public" != "true" ]] && ((d += 1))
      if [[ "$d" -gt 0 ]]; then
        ((drifted += 1))
        echo "DRIFTED $t.$n ($d property difference(s) vs simulated Azure state)"
      else
        ((in_sync += 1))
        echo "IN_SYNC $t.$n"
      fi
    elif [[ "$t" == "azurerm_kubernetes_cluster" && "$n" == "aks" ]]; then
      local expected_rbac expected_k8s
      expected_rbac="$(drift_expected_bool "$b" "role_based_access_control_enabled" "roleBasedAccessControlEnabled")"
      expected_k8s="$(drift_expected_string "$b" "kubernetes_version")"
      local d=0
      [[ -n "$expected_rbac" && "$expected_rbac" != "true" ]] && ((d += 1))
      [[ -n "$expected_k8s" && "$expected_k8s" != "1.27.0" ]] && ((d += 1))
      if [[ "$d" -gt 0 ]]; then
        ((drifted += 1))
        echo "DRIFTED $t.$n ($d property difference(s) vs simulated Azure state)"
      else
        ((in_sync += 1))
        echo "IN_SYNC $t.$n"
      fi
    else
      ((missing_in_azure += 1))
      echo "MISSING_IN_AZURE $t.$n (not present in simulated live state)"
    fi
  done

  echo "---"
  echo "Summary: in_sync=$in_sync drifted=$drifted missing_in_azure=$missing_in_azure"
}

# -----------------------------------------------------------------------------
# Security Scanner
# -----------------------------------------------------------------------------
run_security() {
  print_header "Security Scanner (core)"
  parse_resources
  local total
  total="$(resource_count)"
  echo "Detected resources: $total"

  local findings=()

  # Pattern-based secret scan (SEC001)
  if grep -Eniq '(password|secret|key|token|credential)[[:space:]]*=[[:space:]]*"[^"]{8,}"|api[_-]?key[[:space:]]*=[[:space:]]*"[^"]+|-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----' <<< "$CODE"; then
    findings+=("CRITICAL SEC001 hardcoded-secrets -> use Key Vault/env vars")
  fi

  local i
  for i in "${!RES_TYPES[@]}"; do
    local t="${RES_TYPES[$i]}"
    local n="${RES_NAMES[$i]}"
    local b="${RES_BLOCKS[$i]}"

    if [[ "$t" == "azurerm_storage_account" || "$t" == "azurerm_key_vault" || "$t" == "azurerm_sql_server" ]]; then
      if ! block_has "$b" 'public_network_access_enabled[[:space:]]*=[[:space:]]*false|publicNetworkAccess[[:space:]]*:[[:space:]]*'\''Disabled'\''' ; then
        findings+=("HIGH SEC002 $t.$n public_network_access_enabled should be false")
      fi
    fi

    if [[ "$t" == "azurerm_storage_account" || "$t" == "azurerm_sql_database" ]]; then
      if ! block_has "$b" 'customer_managed_key[[:space:]]*=|keyVaultProperties[[:space:]]*:'; then
        findings+=("HIGH SEC003 $t.$n customer-managed encryption missing")
      fi
    fi

    if [[ "$t" == "azurerm_storage_account" || "$t" == "azurerm_app_service" || "$t" == "azurerm_function_app" ]]; then
      if ! block_has "$b" 'min_tls_version[[:space:]]*=[[:space:]]*"TLS1_2"|minimumTlsVersion[[:space:]]*:[[:space:]]*'\''TLS1_2'\''' ; then
        findings+=("HIGH SEC004 $t.$n minimum TLS should be 1.2")
      fi
      if ! block_has "$b" 'enable_https_traffic_only[[:space:]]*=[[:space:]]*true|supportsHttpsTrafficOnly[[:space:]]*:[[:space:]]*true' ; then
        findings+=("HIGH SEC005 $t.$n HTTPS-only should be enabled")
      fi
    fi
  done

  if [[ "${#findings[@]}" -eq 0 ]]; then
    echo "PASS: no security findings."
  else
    echo "FAIL: ${#findings[@]} finding(s)"
    printf '%s\n' "${findings[@]}"
  fi
}

# -----------------------------------------------------------------------------
# Compliance Auditor
# -----------------------------------------------------------------------------
run_compliance() {
  print_header "Compliance Auditor (core)"
  parse_resources
  local total
  total="$(resource_count)"
  echo "Detected resources: $total"
  if [[ "$total" -eq 0 ]]; then
    echo "No resources detected."
    return
  fi

  local failures=()
  local passes=0
  local i
  for i in "${!RES_TYPES[@]}"; do
    local t="${RES_TYPES[$i]}"
    local n="${RES_NAMES[$i]}"
    local b="${RES_BLOCKS[$i]}"

    # CIS
    if [[ "$t" == "azurerm_storage_account" ]]; then
      if block_has "$b" 'enable_https_traffic_only[[:space:]]*=[[:space:]]*true|supportsHttpsTrafficOnly[[:space:]]*:[[:space:]]*true'; then
        ((passes += 1))
      else
        failures+=("CIS CIS-3.1 $t.$n secure transfer required")
      fi
      if block_has "$b" 'allow_blob_public_access[[:space:]]*=[[:space:]]*false|allowBlobPublicAccess[[:space:]]*:[[:space:]]*false'; then
        ((passes += 1))
      else
        failures+=("CIS CIS-3.7 $t.$n public blob access should be disabled")
      fi
    fi
    if [[ "$t" == "azurerm_kubernetes_cluster" ]]; then
      if block_has "$b" 'role_based_access_control_enabled[[:space:]]*=[[:space:]]*true'; then
        ((passes += 1))
      else
        failures+=("CIS CIS-8.1 $t.$n RBAC should be enabled")
      fi
    fi

    # NIST
    if [[ "$t" == "azurerm_storage_account" || "$t" == "azurerm_app_service" ]]; then
      if block_has "$b" 'enable_https_traffic_only[[:space:]]*=[[:space:]]*true|supportsHttpsTrafficOnly[[:space:]]*:[[:space:]]*true'; then
        ((passes += 1))
      else
        failures+=("NIST NIST-SC-8 $t.$n HTTPS-only traffic required")
      fi
    fi
    if [[ "$t" == "azurerm_storage_account" ]]; then
      if block_has "$b" 'min_tls_version[[:space:]]*=[[:space:]]*"TLS1_2"|minimumTlsVersion[[:space:]]*:[[:space:]]*'\''TLS1_2'\''' ; then
        ((passes += 1))
      else
        failures+=("NIST NIST-SC-28 $t.$n min TLS should be TLS1_2")
      fi
    fi
    if [[ "$t" == "azurerm_kubernetes_cluster" ]]; then
      if block_has "$b" 'role_based_access_control_enabled[[:space:]]*=[[:space:]]*true'; then
        ((passes += 1))
      else
        failures+=("NIST NIST-AC-6 $t.$n least privilege via RBAC required")
      fi
    fi

    # SOC2
    if [[ "$t" == "azurerm_kubernetes_cluster" || "$t" == "azurerm_key_vault" ]]; then
      if block_has "$b" 'role_based_access_control_enabled[[:space:]]*=[[:space:]]*true'; then
        ((passes += 1))
      else
        failures+=("SOC2 SOC2-CC6.1 $t.$n RBAC should be enabled")
      fi
    fi
    if [[ "$t" == "azurerm_storage_account" ]]; then
      if block_has "$b" 'enable_https_traffic_only[[:space:]]*=[[:space:]]*true|supportsHttpsTrafficOnly[[:space:]]*:[[:space:]]*true'; then
        ((passes += 1))
      else
        failures+=("SOC2 SOC2-CC6.7 $t.$n transmission security requires HTTPS")
      fi
    fi
  done

  local failure_count="${#failures[@]}"
  local total_checks=$((passes + failure_count))
  local score="0.0"
  if [[ "$total_checks" -gt 0 ]]; then
    score="$(awk -v p="$passes" -v t="$total_checks" 'BEGIN{printf "%.1f", (p/t)*100}')"
  fi

  if [[ "$failure_count" -eq 0 ]]; then
    echo "PASS: all compliance checks passed."
  else
    echo "FAIL: $failure_count compliance violation(s)"
    printf '%s\n' "${failures[@]}"
  fi
  echo "Compliance score: ${score}% ($passes/$total_checks)"
}

# -----------------------------------------------------------------------------
# Module Registry
# -----------------------------------------------------------------------------
version_lt() {
  local v1="${1:-unspecified}"
  local v2="${2:-0}"
  [[ "$v1" < "$v2" ]]
}

run_modules() {
  print_header "Module Registry (core)"

  if [[ -z "$CODE" ]]; then
    echo "No code supplied. Use --file or stdin."
    return
  fi

  local starts
  starts="$(grep -En '^[[:space:]]*module[[:space:]]+"[^"]+"[[:space:]]*\{' <<< "$CODE" || true)"
  if [[ -z "$starts" ]]; then
    echo "No module blocks found."
    return
  fi

  local allowed_sources=("registry.terraform.io" "github.com/yourorg")
  local issues=0 approved=0

  while IFS= read -r row; do
    [[ -z "$row" ]] && continue
    local line_no="${row%%:*}"
    local line_text="${row#*:}"
    local name
    name="$(sed -E 's/^[[:space:]]*module[[:space:]]+"([^"]+)".*/\1/' <<< "$line_text")"
    local block
    block="$(extract_block_from_line "$line_no")"
    local source version
    source="$(grep -Eo 'source[[:space:]]*=[[:space:]]*"[^"]+"' <<< "$block" | head -n1 | sed -E 's/.*"([^"]+)"/\1/')"
    version="$(grep -Eo 'version[[:space:]]*=[[:space:]]*"[^"]+"' <<< "$block" | head -n1 | sed -E 's/.*"([^"]+)"/\1/')"
    [[ -z "$version" ]] && version="unspecified"

    local source_allowed=0
    local allowed
    for allowed in "${allowed_sources[@]}"; do
      if [[ "$source" == *"$allowed"* ]]; then
        source_allowed=1
        break
      fi
    done

    if [[ "$source_allowed" -eq 0 ]]; then
      ((issues += 1))
      echo "HIGH unknown_source module.$name source='$source' -> use approved source"
      continue
    fi

    # Minimal default catalog logic.
    local min_version=""
    local deprecated=0
    case "$source" in
      *registry.terraform.io/Azure/storage/azurerm*) min_version="2.0.0" ;;
      *registry.terraform.io/Azure/aks/azurerm*) min_version="6.0.0" ;;
      *registry.terraform.io/Azure/keyvault/azurerm*) min_version="2.0.0" ;;
      *registry.terraform.io/Azure/network/azurerm*) min_version="4.0.0" ;;
      *github.com/old-org/terraform-azure-storage*) deprecated=1 ;;
      *) min_version="" ;;
    esac

    if [[ "$deprecated" -eq 1 ]]; then
      ((issues += 1))
      echo "HIGH deprecated module.$name -> migrate to storage-account"
    elif [[ -z "$min_version" ]]; then
      ((issues += 1))
      echo "MEDIUM not_approved module.$name -> not in approved catalog"
    elif [[ "$version" != "unspecified" ]] && version_lt "$version" "$min_version"; then
      ((issues += 1))
      echo "MEDIUM version_mismatch module.$name version=$version < min=$min_version"
    else
      ((approved += 1))
      echo "APPROVED module.$name source='$source' version='$version'"
    fi
  done <<< "$starts"

  echo "---"
  echo "Summary: approved=$approved issues=$issues"
}

# -----------------------------------------------------------------------------
# Impact Analyzer
# -----------------------------------------------------------------------------
run_impact() {
  print_header "Impact Analyzer (core)"
  parse_resources
  local total
  total="$(resource_count)"
  echo "Detected resources: $total"
  if [[ "$total" -eq 0 ]]; then
    echo "No resources detected."
    return
  fi

  local RES_REFS=()
  local i
  for i in "${!RES_TYPES[@]}"; do
    local b="${RES_BLOCKS[$i]}"
    local refs
    refs="$(grep -Eo '[A-Za-z_][A-Za-z0-9_]*\.[A-Za-z_][A-Za-z0-9_]*\.[A-Za-z_][A-Za-z0-9_]*' <<< "$b" | sed -E 's/^([^.]+\.[^.]+)\..*/\1/' | grep -Ev '^(var|local)\.' | sort -u | tr '\n' ' ')"
    RES_REFS+=("$refs")
  done

  local high=0 medium=0 low=0 max_blast=0
  for i in "${!RES_TYPES[@]}"; do
    local t="${RES_TYPES[$i]}"
    local n="${RES_NAMES[$i]}"
    local key="$t.$n"
    local blast=0
    local affected=()
    local j
    for j in "${!RES_TYPES[@]}"; do
      local other="${RES_TYPES[$j]}.${RES_NAMES[$j]}"
      [[ "$other" == "$key" ]] && continue
      if grep -Eq "(^| )${t}\.${n}( |$)" <<< "${RES_REFS[$j]}"; then
        affected+=("$other")
      fi
    done
    blast="${#affected[@]}"
    (( blast > max_blast )) && max_blast="$blast"

    local risk="medium"
    case "$t" in
      azurerm_kubernetes_cluster) risk="critical" ;;
      azurerm_storage_account|azurerm_sql_server|azurerm_sql_database|azurerm_virtual_network|azurerm_subnet|azurerm_key_vault) risk="high" ;;
      *) risk="medium" ;;
    esac

    case "$risk" in
      critical|high) ((high += 1)) ;;
      medium) ((medium += 1)) ;;
      *) ((low += 1)) ;;
    esac

    echo "$risk $key blast_radius=$blast affects='${affected[*]-}'"
  done

  local overall="low"
  if [[ "$high" -gt 0 ]]; then
    overall="high"
  elif [[ "$medium" -gt 0 ]]; then
    overall="medium"
  fi
  echo "---"
  echo "Summary: total=$total high=$high medium=$medium low=$low max_blast_radius=$max_blast overall=$overall"
}

# -----------------------------------------------------------------------------
# Deploy Promoter
# -----------------------------------------------------------------------------
run_deploy() {
  print_header "Deploy Promoter (core)"
  local msg
  msg="$(to_lower "$MESSAGE")"
  local dev_ver="v1.3.0"
  local stg_ver="v1.2.0"
  local prod_ver="v1.1.0"

  if [[ "$msg" == *"status"* ]]; then
    echo "dev=$dev_ver staging=$stg_ver prod=$prod_ver"
    echo "path: dev -> staging -> prod"
    return
  fi

  if [[ "$msg" == *"rollback"* ]]; then
    echo "rollback flow: identify target version -> verify -> plan -> apply with approval"
    return
  fi

  local source="" target=""
  if [[ "$msg" =~ promote[[:space:]]+([a-zA-Z0-9_]+)[[:space:]]+to[[:space:]]+([a-zA-Z0-9_]+) ]]; then
    source="${BASH_REMATCH[1]}"
    target="${BASH_REMATCH[2]}"
  elif [[ "$msg" =~ ([a-zA-Z0-9_]+)[[:space:]]*-[[:space:]]*\>[[:space:]]*([a-zA-Z0-9_]+) ]]; then
    source="${BASH_REMATCH[1]}"
    target="${BASH_REMATCH[2]}"
  elif [[ "$msg" == *"promote"* && "$msg" == *"staging"* ]]; then
    source="dev"; target="staging"
  elif [[ "$msg" == *"promote"* && "$msg" == *"prod"* ]]; then
    source="staging"; target="prod"
  fi

  normalize_env() {
    local env_lower
    env_lower="$(to_lower "$1")"
    case "$env_lower" in
      development|dev) echo "dev" ;;
      stage|staging|stg) echo "staging" ;;
      production|prod|prd) echo "prod" ;;
      *) echo "$env_lower" ;;
    esac
  }

  source="$(normalize_env "$source")"
  target="$(normalize_env "$target")"
  if [[ -z "$source" || -z "$target" ]]; then
    echo "help: use 'promote dev to staging' or 'promote staging to prod'"
    return
  fi

  local level_source=0 level_target=0
  case "$source" in dev) level_source=1 ;; staging) level_source=2 ;; prod) level_source=3 ;; esac
  case "$target" in dev) level_target=1 ;; staging) level_target=2 ;; prod) level_target=3 ;; esac

  local allowed=1
  echo "promotion request: $source -> $target"
  if [[ "$level_source" -eq 0 || "$level_target" -eq 0 ]]; then
    echo "FAIL unknown environment"
    return
  fi
  if (( level_source >= level_target )); then
    allowed=0
    echo "FAIL environment order: must promote upward only"
  else
    echo "PASS environment order"
  fi
  if (( level_target - level_source > 1 )); then
    allowed=0
    echo "FAIL sequential promotion: cannot skip environments"
  else
    echo "PASS sequential promotion"
  fi
  if [[ "$target" == "staging" || "$target" == "prod" ]]; then
    echo "INFO approval required for $target"
  fi

  if [[ "$allowed" -eq 1 ]]; then
    echo "APPROVED: run terraform plan/apply in $target"
  else
    echo "BLOCKED"
  fi
}

# -----------------------------------------------------------------------------
# Notification Manager
# -----------------------------------------------------------------------------
run_notify() {
  print_header "Notification Manager (core)"
  local msg
  msg="$(to_lower "$MESSAGE")"
  local teams_enabled=0 slack_enabled=0 email_enabled=0
  [[ -n "${TEAMS_WEBHOOK_URL:-}" ]] && teams_enabled=1
  [[ -n "${SLACK_WEBHOOK_URL:-}" ]] && slack_enabled=1
  [[ -n "${SMTP_SERVER:-}" ]] && email_enabled=1
  local webhook_enabled=1

  list_channels() {
    echo "teams-alerts type=teams enabled=$teams_enabled"
    echo "slack-devops type=slack enabled=$slack_enabled"
    echo "email-admins type=email enabled=$email_enabled"
    echo "webhook-audit type=webhook enabled=$webhook_enabled"
  }

  show_rules() {
    echo "deployment:info -> slack-devops"
    echo "deployment:error -> teams-alerts,slack-devops,email-admins"
    echo "drift:* -> teams-alerts,slack-devops"
    echo "policy:warning -> slack-devops"
    echo "policy:error -> teams-alerts,email-admins"
    echo "security:* -> teams-alerts,email-admins,webhook-audit"
    echo "cost:warning -> slack-devops"
  }

  if [[ "$msg" == *"channels"* || "$msg" == *"list"* ]]; then
    list_channels
    return
  fi
  if [[ "$msg" == *"rules"* || "$msg" == *"routing"* ]]; then
    show_rules
    return
  fi
  if [[ "$msg" == *"history"* || "$msg" == *"recent"* ]]; then
    echo "history is in-memory per process; none recorded in this run."
    return
  fi
  if [[ "$msg" == *"test"* ]]; then
    local channel="slack-devops"
    [[ "$msg" == *"teams"* ]] && channel="teams-alerts"
    [[ "$msg" == *"email"* ]] && channel="email-admins"
    echo "test notification queued for $channel"
    return
  fi

  if [[ "$msg" == *"send"* || "$msg" == *"notify"* ]]; then
    local event_type="deployment"
    local severity="info"
    [[ "$msg" == *"security"* ]] && event_type="security"
    [[ "$msg" == *"drift"* ]] && event_type="drift"
    [[ "$msg" == *"policy"* ]] && event_type="policy"
    [[ "$msg" == *"cost"* ]] && event_type="cost"
    [[ "$msg" == *"critical"* ]] && severity="critical"
    [[ "$msg" == *"error"* ]] && severity="error"
    [[ "$msg" == *"warning"* ]] && severity="warning"

    echo "event=$event_type severity=$severity"
    local channels=()
    case "$event_type:$severity" in
      deployment:info) channels=("slack-devops") ;;
      deployment:error|deployment:critical) channels=("teams-alerts" "slack-devops" "email-admins") ;;
      drift:*) channels=("teams-alerts" "slack-devops") ;;
      policy:warning) channels=("slack-devops") ;;
      policy:error|policy:critical) channels=("teams-alerts" "email-admins") ;;
      security:*) channels=("teams-alerts" "email-admins" "webhook-audit") ;;
      cost:warning) channels=("slack-devops") ;;
      *) channels=() ;;
    esac

    if [[ "${#channels[@]}" -eq 0 ]]; then
      echo "No routing rule matched."
      return
    fi
    echo "routing to: ${channels[*]-}"
    return
  fi

  echo "help: channels | rules | history | test [teams|email] | send [type] [severity]"
}

# -----------------------------------------------------------------------------
# Orchestrator
# -----------------------------------------------------------------------------
run_orchestrator() {
  print_header "Orchestrator (core)"
  local msg
  msg="$(to_lower "$MESSAGE")"
  if [[ -z "$msg" && -n "$CODE" ]]; then
    msg="full analysis"
  fi

  if [[ "$msg" == *"status"* || "$msg" == *"agents"* ]]; then
    echo "policy cost drift security compliance module impact deploy notification"
    echo "all agents available through this script."
    return
  fi

  if [[ "$msg" == *"review"* && ( "$msg" == *"code"* || "$msg" == *"pr"* ) ]]; then
    echo "workflow=code-review (policy, security, cost, modules)"
    run_policy
    run_security
    run_cost
    run_modules
    return
  fi

  if [[ "$msg" == *"full"* && "$msg" == *"analysis"* ]]; then
    echo "workflow=full-analysis (security, policy, compliance, cost, impact, modules)"
    run_security
    run_policy
    run_compliance
    run_cost
    run_impact
    run_modules
    return
  fi

  if [[ "$msg" == *"deploy"* && ( "$msg" == *"check"* || "$msg" == *"ready"* ) ]]; then
    echo "workflow=deploy-check"
    run_deploy
    run_security
    run_policy
    run_impact
    return
  fi

  if [[ "$msg" == *"policy"* ]]; then run_policy; return; fi
  if [[ "$msg" == *"cost"* || "$msg" == *"pricing"* ]]; then run_cost; return; fi
  if [[ "$msg" == *"drift"* ]]; then run_drift; return; fi
  if [[ "$msg" == *"security"* ]]; then run_security; return; fi
  if [[ "$msg" == *"compliance"* || "$msg" == *"audit"* ]]; then run_compliance; return; fi
  if [[ "$msg" == *"module"* || "$msg" == *"registry"* ]]; then run_modules; return; fi
  if [[ "$msg" == *"impact"* || "$msg" == *"blast"* ]]; then run_impact; return; fi
  if [[ "$msg" == *"notify"* || "$msg" == *"notification"* ]]; then run_notify; return; fi
  if [[ "$msg" == *"promote"* || "$msg" == *"rollback"* ]]; then run_deploy; return; fi

  echo "capabilities: status | code review | full analysis | deploy check | route to single agent"
}

main() {
  parse_args "$@"
  read_inputs

  if [[ "$AGENT" =~ ^(policy|cost|drift|security|compliance|modules|module|impact)$ ]] && [[ -z "$CODE" ]]; then
    echo "This agent requires IaC input via --file or stdin." >&2
    exit 1
  fi

  local iac_type
  iac_type="$(detect_iac_type)"
  if [[ -n "$CODE" ]]; then
    echo "Detected IaC type: $iac_type"
  fi

  case "$(to_lower "$AGENT")" in
    policy) run_policy ;;
    cost) run_cost ;;
    drift) run_drift ;;
    security) run_security ;;
    compliance) run_compliance ;;
    modules|module) run_modules ;;
    impact) run_impact ;;
    deploy) run_deploy ;;
    notify|notification) run_notify ;;
    orchestrator|check) run_orchestrator ;;
    *)
      echo "Unknown agent: $AGENT" >&2
      usage
      exit 1
      ;;
  esac
}

main "$@"
