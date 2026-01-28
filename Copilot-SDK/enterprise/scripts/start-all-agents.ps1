# =============================================================================
# Enterprise IaC Governance Platform - Agent Startup Script
# =============================================================================
# This script builds, starts, and verifies all agents in the platform.
#
# Usage:
#   .\start-all-agents.ps1              # Start all agents
#   .\start-all-agents.ps1 -BuildOnly   # Only build, don't start
#   .\start-all-agents.ps1 -StatusOnly  # Only check status
#   .\start-all-agents.ps1 -StopAll     # Stop all running agents
# =============================================================================

param(
    [switch]$BuildOnly,
    [switch]$StatusOnly,
    [switch]$StopAll,
    [switch]$Verbose
)

$ErrorActionPreference = "Continue"

# =============================================================================
# Configuration
# =============================================================================

$BaseDir = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$EnterpriseAgentsDir = Join-Path $BaseDir "enterprise\agents"
$LegacyAgentsDir = $BaseDir

# Agent definitions: Name, Port, Directory, Executable
$Agents = @(
    @{ Name = "Policy Checker";       Port = 8081; Dir = "03-policy-agent";                    Exe = "policy-agent.exe";        Legacy = $true },
    @{ Name = "Cost Estimator";       Port = 8082; Dir = "04-cost-estimator";                  Exe = "cost-estimator.exe";      Legacy = $true },
    @{ Name = "Drift Detector";       Port = 8083; Dir = "enterprise\agents\drift-detector";   Exe = "drift-detector.exe";      Legacy = $false },
    @{ Name = "Security Scanner";     Port = 8084; Dir = "enterprise\agents\security-scanner"; Exe = "security-scanner.exe";    Legacy = $false },
    @{ Name = "Compliance Auditor";   Port = 8085; Dir = "enterprise\agents\compliance-auditor"; Exe = "compliance-auditor.exe"; Legacy = $false },
    @{ Name = "Module Registry";      Port = 8086; Dir = "enterprise\agents\module-registry";  Exe = "module-registry.exe";     Legacy = $false },
    @{ Name = "Impact Analyzer";      Port = 8087; Dir = "enterprise\agents\impact-analyzer";  Exe = "impact-analyzer.exe";     Legacy = $false },
    @{ Name = "Deploy Promoter";      Port = 8088; Dir = "enterprise\agents\deploy-promoter";  Exe = "deploy-promoter.exe";     Legacy = $false },
    @{ Name = "Notification Manager"; Port = 8089; Dir = "enterprise\agents\notification-manager"; Exe = "notification-manager.exe"; Legacy = $false },
    @{ Name = "Orchestrator";         Port = 8090; Dir = "enterprise\agents\orchestrator";     Exe = "orchestrator.exe";        Legacy = $false }
)

# =============================================================================
# Helper Functions
# =============================================================================

function Write-Header {
    param([string]$Message)
    Write-Host ""
    Write-Host ("=" * 70) -ForegroundColor Cyan
    Write-Host " $Message" -ForegroundColor Cyan
    Write-Host ("=" * 70) -ForegroundColor Cyan
    Write-Host ""
}

function Write-AgentStatus {
    param(
        [string]$Name,
        [int]$Port,
        [string]$Status,
        [string]$Color = "White"
    )
    $portStr = $Port.ToString().PadRight(6)
    $nameStr = $Name.PadRight(22)
    Write-Host "  [$portStr] $nameStr : " -NoNewline
    Write-Host $Status -ForegroundColor $Color
}

function Get-AgentPath {
    param($Agent)
    return Join-Path $BaseDir $Agent.Dir
}

function Test-AgentHealth {
    param([int]$Port)
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:$Port/health" -TimeoutSec 2 -UseBasicParsing -ErrorAction SilentlyContinue
        return $response.StatusCode -eq 200
    } catch {
        return $false
    }
}

function Stop-AgentOnPort {
    param([int]$Port)
    $process = Get-NetTCPConnection -LocalPort $Port -ErrorAction SilentlyContinue | 
               Select-Object -ExpandProperty OwningProcess -First 1
    if ($process) {
        Stop-Process -Id $process -Force -ErrorAction SilentlyContinue
        return $true
    }
    return $false
}

function Wait-ForAgent {
    param(
        [int]$Port,
        [int]$TimeoutSeconds = 10
    )
    $elapsed = 0
    while ($elapsed -lt $TimeoutSeconds) {
        if (Test-AgentHealth -Port $Port) {
            return $true
        }
        Start-Sleep -Milliseconds 500
        $elapsed += 0.5
    }
    return $false
}

# =============================================================================
# Main Functions
# =============================================================================

function Show-Status {
    Write-Header "Agent Status Check"
    
    $online = 0
    $offline = 0
    
    Write-Host "  Port    Agent                  Status" -ForegroundColor Gray
    Write-Host "  ------  ---------------------- --------" -ForegroundColor Gray
    
    foreach ($agent in $Agents) {
        if (Test-AgentHealth -Port $agent.Port) {
            Write-AgentStatus -Name $agent.Name -Port $agent.Port -Status "ONLINE" -Color "Green"
            $online++
        } else {
            Write-AgentStatus -Name $agent.Name -Port $agent.Port -Status "OFFLINE" -Color "Red"
            $offline++
        }
    }
    
    Write-Host ""
    Write-Host "  Summary: " -NoNewline
    Write-Host "$online online" -ForegroundColor Green -NoNewline
    Write-Host ", " -NoNewline
    Write-Host "$offline offline" -ForegroundColor $(if ($offline -gt 0) { "Red" } else { "Gray" })
    Write-Host ""
    
    return $online
}

function Build-AllAgents {
    Write-Header "Building All Agents"
    
    $success = 0
    $failed = 0
    
    foreach ($agent in $Agents) {
        $agentPath = Get-AgentPath -Agent $agent
        $exePath = Join-Path $agentPath $agent.Exe
        
        Write-Host "  Building $($agent.Name)..." -NoNewline
        
        if (-not (Test-Path $agentPath)) {
            Write-Host " SKIP (directory not found)" -ForegroundColor Yellow
            continue
        }
        
        Push-Location $agentPath
        try {
            $output = go build -o $agent.Exe . 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-Host " OK" -ForegroundColor Green
                $success++
            } else {
                Write-Host " FAILED" -ForegroundColor Red
                if ($Verbose) {
                    Write-Host "    $output" -ForegroundColor Red
                }
                $failed++
            }
        } catch {
            Write-Host " ERROR: $_" -ForegroundColor Red
            $failed++
        } finally {
            Pop-Location
        }
    }
    
    Write-Host ""
    Write-Host "  Build complete: " -NoNewline
    Write-Host "$success succeeded" -ForegroundColor Green -NoNewline
    Write-Host ", " -NoNewline
    Write-Host "$failed failed" -ForegroundColor $(if ($failed -gt 0) { "Red" } else { "Gray" })
    Write-Host ""
    
    return $failed -eq 0
}

function Start-AllAgents {
    Write-Header "Starting All Agents"
    
    $started = 0
    $failed = 0
    $skipped = 0
    
    foreach ($agent in $Agents) {
        $agentPath = Get-AgentPath -Agent $agent
        $exePath = Join-Path $agentPath $agent.Exe
        
        Write-Host "  Starting $($agent.Name) on port $($agent.Port)..." -NoNewline
        
        # Check if already running
        if (Test-AgentHealth -Port $agent.Port) {
            Write-Host " ALREADY RUNNING" -ForegroundColor Yellow
            $skipped++
            continue
        }
        
        # Check if executable exists
        if (-not (Test-Path $exePath)) {
            Write-Host " SKIP (not built)" -ForegroundColor Yellow
            $skipped++
            continue
        }
        
        # Start agent in background
        try {
            $env:PORT = $agent.Port
            Start-Process -FilePath $exePath -WorkingDirectory $agentPath -WindowStyle Hidden
            
            # Wait for agent to be healthy
            if (Wait-ForAgent -Port $agent.Port -TimeoutSeconds 5) {
                Write-Host " OK" -ForegroundColor Green
                $started++
            } else {
                Write-Host " TIMEOUT" -ForegroundColor Red
                $failed++
            }
        } catch {
            Write-Host " ERROR: $_" -ForegroundColor Red
            $failed++
        }
    }
    
    Write-Host ""
    Write-Host "  Startup complete: " -NoNewline
    Write-Host "$started started" -ForegroundColor Green -NoNewline
    Write-Host ", $skipped skipped, " -NoNewline
    Write-Host "$failed failed" -ForegroundColor $(if ($failed -gt 0) { "Red" } else { "Gray" })
    Write-Host ""
    
    return $failed -eq 0
}

function Stop-AllAgents {
    Write-Header "Stopping All Agents"
    
    $stopped = 0
    $notRunning = 0
    
    # Stop in reverse order (orchestrator first)
    $reversedAgents = $Agents | Sort-Object { $_.Port } -Descending
    
    foreach ($agent in $reversedAgents) {
        Write-Host "  Stopping $($agent.Name) on port $($agent.Port)..." -NoNewline
        
        if (Stop-AgentOnPort -Port $agent.Port) {
            Write-Host " STOPPED" -ForegroundColor Green
            $stopped++
        } else {
            Write-Host " NOT RUNNING" -ForegroundColor Gray
            $notRunning++
        }
    }
    
    Write-Host ""
    Write-Host "  Shutdown complete: $stopped stopped, $notRunning were not running"
    Write-Host ""
}

function Test-FullPlatform {
    Write-Header "Platform Integration Test"
    
    Write-Host "  Testing Orchestrator connectivity to all agents..."
    Write-Host ""
    
    try {
        $body = @{
            messages = @(
                @{
                    role = "user"
                    content = "status"
                }
            )
        } | ConvertTo-Json -Depth 3
        
        $response = Invoke-WebRequest -Uri "http://localhost:8090/agents" -Method GET -TimeoutSec 10 -UseBasicParsing
        $agents = $response.Content | ConvertFrom-Json
        
        Write-Host "  Orchestrator reports:" -ForegroundColor Cyan
        foreach ($agent in $agents) {
            $status = if ($agent.status -eq "online") { "ONLINE" } else { "OFFLINE" }
            $color = if ($agent.status -eq "online") { "Green" } else { "Red" }
            Write-Host "    - $($agent.name): " -NoNewline
            Write-Host $status -ForegroundColor $color
        }
        Write-Host ""
        Write-Host "  Platform integration: " -NoNewline -ForegroundColor Green
        Write-Host "OK" -ForegroundColor Green
    } catch {
        Write-Host "  Platform integration: " -NoNewline
        Write-Host "FAILED - Orchestrator not responding" -ForegroundColor Red
    }
    
    Write-Host ""
}

# =============================================================================
# Main Entry Point
# =============================================================================

Write-Host ""
Write-Host "+======================================================================+" -ForegroundColor Magenta
Write-Host "|       Enterprise IaC Governance Platform - Agent Manager             |" -ForegroundColor Magenta
Write-Host "+======================================================================+" -ForegroundColor Magenta

if ($StopAll) {
    Stop-AllAgents
    exit 0
}

if ($StatusOnly) {
    $online = Show-Status
    exit $(if ($online -eq $Agents.Count) { 0 } else { 1 })
}

if ($BuildOnly) {
    $buildSuccess = Build-AllAgents
    exit $(if ($buildSuccess) { 0 } else { 1 })
}

# Full startup sequence
$buildSuccess = Build-AllAgents
if (-not $buildSuccess) {
    Write-Host "  Build failed. Continuing with available agents..." -ForegroundColor Yellow
}

$startSuccess = Start-AllAgents
Start-Sleep -Seconds 2

Show-Status

if ($startSuccess) {
    Test-FullPlatform
}

Write-Host ("=" * 70) -ForegroundColor Cyan
Write-Host " Startup sequence complete" -ForegroundColor Cyan
Write-Host ("=" * 70) -ForegroundColor Cyan
Write-Host ""
