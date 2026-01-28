# ============================================================
# IaC Governance Demo Script
# ============================================================
# Run this script while recording with ScreenToGif or OBS
# Recommended: Font size 16pt, terminal width 100 chars
# ============================================================

# Helper function for pauses (gives viewer time to read)
function Pause-Demo {
    param([int]$Seconds = 3)
    Start-Sleep -Seconds $Seconds
}

# Helper to wait for keypress (manual advance)
function Wait-ForKey {
    param([string]$Message = "Press any key to continue...")
    Write-Host ""
    Write-Host $Message -ForegroundColor DarkGray
    $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
    Write-Host ""
}

Clear-Host

# ============================================================
# SCENE 1: Introduction
# ============================================================
Write-Host ""
Write-Host "============================================" -ForegroundColor DarkCyan
Write-Host "  IaC Governance Demo - gh iac extension" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor DarkCyan
Write-Host ""
Pause-Demo -Seconds 3

# Show version
Write-Host "$ gh iac version" -ForegroundColor Green
gh iac version
Pause-Demo -Seconds 3

# Show status
Write-Host ""
Write-Host "$ gh iac status" -ForegroundColor Green
gh iac status
Wait-ForKey

# ============================================================
# SCENE 2: Policy Check (Bad Code)
# ============================================================
Clear-Host
Write-Host ""
Write-Host "============================================" -ForegroundColor DarkCyan
Write-Host "  SCENE 2: Policy Check" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor DarkCyan
Write-Host ""
Pause-Demo -Seconds 2

Write-Host "# Let's look at some non-compliant Terraform code:" -ForegroundColor Yellow
Write-Host ""
Write-Host "$ cat bad-storage.tf" -ForegroundColor Green
Write-Host ""
Get-Content .\bad-storage.tf
Pause-Demo -Seconds 4

Write-Host ""
Write-Host "# Check against organization policies:" -ForegroundColor Yellow
Write-Host ""
Write-Host '$ gh iac policy $(cat bad-storage.tf)' -ForegroundColor Green
Write-Host ""
$code = Get-Content .\bad-storage.tf -Raw
gh iac policy $code
Wait-ForKey

# ============================================================
# SCENE 3: Security Scan
# ============================================================
Clear-Host
Write-Host ""
Write-Host "============================================" -ForegroundColor DarkCyan
Write-Host "  SCENE 3: Security Scan" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor DarkCyan
Write-Host ""
Pause-Demo -Seconds 2

Write-Host "# Scan the same code for security vulnerabilities:" -ForegroundColor Yellow
Write-Host ""
Write-Host '$ gh iac security $(cat bad-storage.tf)' -ForegroundColor Green
Write-Host ""
$code = Get-Content .\bad-storage.tf -Raw
gh iac security $code
Wait-ForKey

# ============================================================
# SCENE 4: Cost Estimation
# ============================================================
Clear-Host
Write-Host ""
Write-Host "============================================" -ForegroundColor DarkCyan
Write-Host "  SCENE 4: Cost Estimation" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor DarkCyan
Write-Host ""
Pause-Demo -Seconds 2

Write-Host "# Now let's check compliant code and estimate costs:" -ForegroundColor Yellow
Write-Host ""
Write-Host "$ cat good-storage.tf" -ForegroundColor Green
Write-Host ""
Get-Content .\good-storage.tf
Pause-Demo -Seconds 4

Write-Host ""
Write-Host "# Estimate monthly Azure costs:" -ForegroundColor Yellow
Write-Host ""
Write-Host '$ gh iac cost $(cat good-storage.tf)' -ForegroundColor Green
Write-Host ""
$code = Get-Content .\good-storage.tf -Raw
gh iac cost $code
Wait-ForKey

# ============================================================
# SCENE 5: Compliance Audit (Bicep)
# ============================================================
Clear-Host
Write-Host ""
Write-Host "============================================" -ForegroundColor DarkCyan
Write-Host "  SCENE 5: Compliance Audit (Bicep)" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor DarkCyan
Write-Host ""
Pause-Demo -Seconds 2

Write-Host "# Check Bicep code against CIS/NIST/SOC2:" -ForegroundColor Yellow
Write-Host ""
Write-Host "$ cat bad-keyvault.bicep" -ForegroundColor Green
Write-Host ""
Get-Content .\bad-keyvault.bicep
Pause-Demo -Seconds 4

Write-Host ""
Write-Host '$ gh iac compliance $(cat bad-keyvault.bicep)' -ForegroundColor Green
Write-Host ""
$code = Get-Content .\bad-keyvault.bicep -Raw
gh iac compliance $code
Wait-ForKey

# ============================================================
# SCENE 6: Full Governance Check
# ============================================================
Clear-Host
Write-Host ""
Write-Host "============================================" -ForegroundColor DarkCyan
Write-Host "  SCENE 6: Full Governance Check" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor DarkCyan
Write-Host ""
Pause-Demo -Seconds 2

Write-Host "# Run comprehensive check via orchestrator:" -ForegroundColor Yellow
Write-Host ""
Write-Host '$ gh iac check $(cat good-storage.tf)' -ForegroundColor Green
Write-Host ""
$code = Get-Content .\good-storage.tf -Raw
gh iac check $code
Wait-ForKey

# ============================================================
# END
# ============================================================
Clear-Host
Write-Host ""
Write-Host ""
Write-Host "  ============================================" -ForegroundColor Green
Write-Host "           Demo Complete!" -ForegroundColor Green
Write-Host "  ============================================" -ForegroundColor Green
Write-Host ""
Write-Host "  gh iac - IaC Governance CLI Extension" -ForegroundColor Cyan
Write-Host ""
Write-Host "  Powered by GitHub Copilot SDK" -ForegroundColor White
Write-Host ""
Write-Host "  10 Agents | Ports 8081-8090" -ForegroundColor DarkGray
Write-Host ""
Write-Host "  Commands: policy, cost, security, compliance," -ForegroundColor DarkGray
Write-Host "            drift, impact, deploy, modules," -ForegroundColor DarkGray
Write-Host "            notify, check" -ForegroundColor DarkGray
Write-Host ""
Pause-Demo -Seconds 5
