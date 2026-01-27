# =============================================================================
# IaC Lab Interactive Learning Experience - PowerShell Version
# =============================================================================
# Native Windows PowerShell implementation
# =============================================================================

$ErrorActionPreference = "Stop"

# Script paths
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$LabRoot = Split-Path -Parent $ScriptDir
$ProgressFile = Join-Path $env:USERPROFILE ".iac-lab-progress.json"

# =============================================================================
# Colors and Formatting
# =============================================================================

function Write-ColorText {
    param(
        [string]$Text,
        [ConsoleColor]$Color = "White"
    )
    Write-Host $Text -ForegroundColor $Color -NoNewline
}

function Write-ColorLine {
    param(
        [string]$Text,
        [ConsoleColor]$Color = "White"
    )
    Write-Host $Text -ForegroundColor $Color
}

# =============================================================================
# ASCII Art
# =============================================================================

function Show-Logo {
    Clear-Host
    Write-ColorLine @"

    ██╗ █████╗  ██████╗    ██╗      █████╗ ██████╗ 
    ██║██╔══██╗██╔════╝    ██║     ██╔══██╗██╔══██╗
    ██║███████║██║         ██║     ███████║██████╔╝
    ██║██╔══██║██║         ██║     ██╔══██║██╔══██╗
    ██║██║  ██║╚██████╗    ███████╗██║  ██║██████╔╝
    ╚═╝╚═╝  ╚═╝ ╚═════╝    ╚══════╝╚═╝  ╚═╝╚═════╝ 
                                                    
"@ -Color Cyan

    Write-ColorLine @"
     ╔══════════════════════════════════════════════════════╗
     ║   🚀 Infrastructure as Code Learning Experience 🚀   ║
     ║        Powered by GitHub Copilot & Azure            ║
     ╚══════════════════════════════════════════════════════╝
"@ -Color Magenta
    Write-Host ""
}

function Show-TerraformLogo {
    Write-ColorLine @"
    ████████╗███████╗██████╗ ██████╗  █████╗ ███████╗ ██████╗ ██████╗ ███╗   ███╗
    ╚══██╔══╝██╔════╝██╔══██╗██╔══██╗██╔══██╗██╔════╝██╔═══██╗██╔══██╗████╗ ████║
       ██║   █████╗  ██████╔╝██████╔╝███████║█████╗  ██║   ██║██████╔╝██╔████╔██║
       ██║   ██╔══╝  ██╔══██╗██╔══██╗██╔══██║██╔══╝  ██║   ██║██╔══██╗██║╚██╔╝██║
       ██║   ███████╗██║  ██║██║  ██║██║  ██║██║     ╚██████╔╝██║  ██║██║ ╚═╝ ██║
       ╚═╝   ╚══════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝      ╚═════╝ ╚═╝  ╚═╝╚═╝     ╚═╝
"@ -Color Magenta
}

function Show-BicepLogo {
    Write-ColorLine @"
    ██████╗ ██╗ ██████╗███████╗██████╗ 
    ██╔══██╗██║██╔════╝██╔════╝██╔══██╗
    ██████╔╝██║██║     █████╗  ██████╔╝
    ██╔══██╗██║██║     ██╔══╝  ██╔═══╝ 
    ██████╔╝██║╚██████╗███████╗██║     
    ╚═════╝ ╚═╝ ╚═════╝╚══════╝╚═╝     
                                        
       💪 Azure's Muscle for IaC 💪
"@ -Color Blue
}

# =============================================================================
# Progress Management
# =============================================================================

function Initialize-Progress {
    if (-not (Test-Path $ProgressFile)) {
        $progress = @{
            LEVEL1_TF = @(0, 0, 0)
            LEVEL1_BICEP = @(0, 0, 0)
            LEVEL2_TF = @(0, 0, 0)
            LEVEL2_BICEP = @(0, 0, 0)
            LEVEL3_TF = @(0, 0, 0)
            LEVEL3_BICEP = @(0, 0, 0)
            LEVEL4_TF = @(0, 0, 0)
            LEVEL4_BICEP = @(0, 0, 0)
            XP_POINTS = 0
        }
        $progress | ConvertTo-Json | Set-Content $ProgressFile
    }
    return Get-Content $ProgressFile | ConvertFrom-Json
}

function Save-Progress {
    param($Progress)
    $Progress | ConvertTo-Json | Set-Content $ProgressFile
}

function Get-TotalCompleted {
    param($Progress)
    $count = 0
    foreach ($prop in $Progress.PSObject.Properties) {
        if ($prop.Name -like "LEVEL*" -and $prop.Value -is [array]) {
            $count += ($prop.Value | Where-Object { $_ -eq 1 }).Count
        }
    }
    return $count
}

# =============================================================================
# Celebration
# =============================================================================

function Show-Celebration {
    param([string]$Message)
    
    Write-Host ""
    Write-ColorLine @"
    
    ╔═══════════════════════════════════════════════════════════╗
    ║                                                           ║
    ║   🎉🎊  CHALLENGE COMPLETED!  🎊🎉                        ║
    ║                                                           ║
    ╚═══════════════════════════════════════════════════════════╝
    
           ⭐    ⭐    ⭐    ⭐    ⭐
              ★       ★       ★
                  ✨     ✨
                     🏆
    
"@ -Color Green

    Write-ColorLine "    $Message" -Color Yellow
    Write-Host ""
    
    # Play a sound
    [Console]::Beep(800, 200)
    [Console]::Beep(1000, 200)
    [Console]::Beep(1200, 400)
}

# =============================================================================
# Progress Bar
# =============================================================================

function Show-ProgressBar {
    param(
        [int]$Current,
        [int]$Total
    )
    
    $width = 40
    $percentage = [math]::Round(($Current / $Total) * 100)
    $filled = [math]::Round(($Current / $Total) * $width)
    $empty = $width - $filled
    
    Write-ColorText "[" -Color Cyan
    Write-ColorText ("█" * $filled) -Color Green
    Write-ColorText ("░" * $empty) -Color DarkGray
    Write-ColorText "] $percentage% " -Color Cyan
    Write-Host "($Current/$Total challenges)"
}

# =============================================================================
# Prerequisites Check
# =============================================================================

function Test-Prerequisites {
    Clear-Host
    Write-ColorLine @"
  ╔═══════════════════════════════════════════════════════════════╗
  ║              ⚙️  PREREQUISITES CHECK ⚙️                        ║
  ╚═══════════════════════════════════════════════════════════════╝
"@ -Color Cyan
    Write-Host ""
    
    $allGood = $true
    
    # Terraform
    Write-ColorText "  Terraform CLI:     " -Color White
    if (Get-Command terraform -ErrorAction SilentlyContinue) {
        $ver = terraform version -json 2>$null | ConvertFrom-Json
        Write-ColorLine "✅ Installed ($($ver.terraform_version))" -Color Green
    } else {
        Write-ColorLine "❌ Not installed" -Color Red
        $allGood = $false
    }
    
    # Azure CLI
    Write-ColorText "  Azure CLI:         " -Color White
    if (Get-Command az -ErrorAction SilentlyContinue) {
        $ver = (az version 2>$null | ConvertFrom-Json).'azure-cli'
        Write-ColorLine "✅ Installed ($ver)" -Color Green
    } else {
        Write-ColorLine "❌ Not installed" -Color Red
        $allGood = $false
    }
    
    # Bicep
    Write-ColorText "  Bicep CLI:         " -Color White
    try {
        $null = az bicep version 2>$null
        Write-ColorLine "✅ Installed" -Color Green
    } catch {
        Write-ColorLine "⚠️  Run: az bicep install" -Color Yellow
    }
    
    # VS Code
    Write-ColorText "  VS Code:           " -Color White
    if (Get-Command code -ErrorAction SilentlyContinue) {
        Write-ColorLine "✅ Installed" -Color Green
    } else {
        Write-ColorLine "⚠️  Not in PATH" -Color Yellow
    }
    
    # Git
    Write-ColorText "  Git:               " -Color White
    if (Get-Command git -ErrorAction SilentlyContinue) {
        $ver = git --version
        Write-ColorLine "✅ $ver" -Color Green
    } else {
        Write-ColorLine "❌ Not installed" -Color Red
        $allGood = $false
    }
    
    # GitHub CLI
    Write-ColorText "  GitHub CLI:        " -Color White
    if (Get-Command gh -ErrorAction SilentlyContinue) {
        Write-ColorLine "✅ Installed" -Color Green
    } else {
        Write-ColorLine "⚠️  Optional - for CLI demos" -Color Yellow
    }
    
    Write-Host ""
    if ($allGood) {
        Write-ColorLine "  🎉 All prerequisites are met! You're ready to learn!" -Color Green
    } else {
        Write-ColorLine "  ⚠️  Some prerequisites are missing." -Color Yellow
    }
    
    Write-Host ""
    Read-Host "  Press Enter to continue"
}

# =============================================================================
# Main Menu
# =============================================================================

function Show-MainMenu {
    $progress = Initialize-Progress
    
    Show-Logo
    
    $completed = Get-TotalCompleted -Progress $progress
    
    Write-ColorLine "  Your Progress:" -Color White
    Write-Host -NoNewline "  "
    Show-ProgressBar -Current $completed -Total 24
    Write-Host ""
    Write-ColorLine "  XP Points: $($progress.XP_POINTS)" -Color Yellow
    Write-Host ""
    
    Write-ColorLine "  ═══════════════════════════════════════════════════" -Color White
    Write-Host ""
    Write-ColorLine "  [1] 📚 Start Learning Journey" -Color Cyan
    Write-ColorLine "  [2] 🎯 Select Specific Challenge" -Color Cyan
    Write-ColorLine "  [3] 🤖 Copilot Demo Scenarios" -Color Cyan
    Write-ColorLine "  [4] 📊 View Progress & Achievements" -Color Cyan
    Write-ColorLine "  [5] ⚙️  Prerequisites Check" -Color Cyan
    Write-ColorLine "  [6] 📖 Quick Reference Guides" -Color Cyan
    Write-ColorLine "  [q] 🚪 Exit" -Color Cyan
    Write-Host ""
    Write-ColorLine "  ═══════════════════════════════════════════════════" -Color White
    Write-Host ""
    
    return Read-Host "  Select an option"
}

function Show-LevelMenu {
    Clear-Host
    Show-Logo
    
    Write-ColorLine "  📚 SELECT YOUR LEVEL" -Color White
    Write-Host ""
    Write-ColorLine "  [1] 🌱 Level 1 - Fundamentals (Beginner)" -Color Green
    Write-ColorLine "      Resource groups, storage accounts, basic syntax" -Color DarkGray
    Write-Host ""
    Write-ColorLine "  [2] 🌿 Level 2 - Intermediate (Developing)" -Color Yellow
    Write-ColorLine "      Networking, compute, App Services" -Color DarkGray
    Write-Host ""
    Write-ColorLine "  [3] 🌳 Level 3 - Advanced (Experienced)" -Color Blue
    Write-ColorLine "      Modules, state management, AKS" -Color DarkGray
    Write-Host ""
    Write-ColorLine "  [4] 🏔️  Level 4 - Enterprise (Expert)" -Color Magenta
    Write-ColorLine "      Multi-region, policy-as-code, CI/CD" -Color DarkGray
    Write-Host ""
    Write-ColorLine "  [b] ⬅️  Back to Main Menu" -Color Cyan
    Write-Host ""
    
    return Read-Host "  Select a level"
}

function Show-TrackMenu {
    param([int]$Level)
    
    Clear-Host
    Show-Logo
    
    Write-ColorLine "  🛤️  SELECT YOUR TRACK - Level $Level" -Color White
    Write-Host ""
    Write-ColorLine "  [1] Terraform - HashiCorp's IaC tool" -Color Magenta
    Write-ColorLine "  [2] Bicep - Azure-native DSL" -Color Blue
    Write-Host ""
    Write-ColorLine "  [b] ⬅️  Back" -Color Cyan
    Write-Host ""
    
    return Read-Host "  Select a track"
}

function Open-Challenge {
    param(
        [int]$Level,
        [string]$Track,
        [int]$ChallengeNum
    )
    
    $levelNames = @{
        1 = "Level-1-Fundamentals"
        2 = "Level-2-Intermediate"
        3 = "Level-3-Advanced"
        4 = "Level-4-Enterprise"
    }
    
    $exerciseDirs = @{
        "1-terraform" = @("01-hello-azure", "02-storage-account", "03-outputs-locals")
        "1-bicep" = @("01-hello-azure", "02-storage-account", "03-outputs-variables")
        "2-terraform" = @("01-networking", "02-compute", "03-app-service")
        "2-bicep" = @("01-networking", "02-compute", "03-app-service")
        "3-terraform" = @("01-modules", "02-state-management", "03-aks-cluster")
        "3-bicep" = @("01-modules", "02-deployment-stacks", "03-aks-cluster")
        "4-terraform" = @("01-multi-region", "02-policy-as-code", "03-cicd-integration")
        "4-bicep" = @("01-multi-region", "02-policy-as-code", "03-cicd-integration")
    }
    
    $key = "$Level-$Track"
    $exercise = $exerciseDirs[$key][$ChallengeNum - 1]
    $challengePath = Join-Path $LabRoot "$($levelNames[$Level])\$Track\$exercise\challenge"
    
    if (Test-Path $challengePath) {
        code $challengePath
    } else {
        $fallback = Join-Path $LabRoot "$($levelNames[$Level])\$Track\$exercise"
        code $fallback
    }
}

# =============================================================================
# Main Loop
# =============================================================================

function Start-IaCLab {
    while ($true) {
        $choice = Show-MainMenu
        
        switch ($choice) {
            "1" {
                $level = Show-LevelMenu
                if ($level -match "^[1-4]$") {
                    $track = Show-TrackMenu -Level ([int]$level)
                    switch ($track) {
                        "1" { Open-Challenge -Level ([int]$level) -Track "terraform" -ChallengeNum 1 }
                        "2" { Open-Challenge -Level ([int]$level) -Track "bicep" -ChallengeNum 1 }
                    }
                }
            }
            "2" {
                $level = Show-LevelMenu
                if ($level -match "^[1-4]$") {
                    $track = Show-TrackMenu -Level ([int]$level)
                    $trackName = if ($track -eq "1") { "terraform" } else { "bicep" }
                    if ($track -match "^[1-2]$") {
                        Write-Host ""
                        Write-ColorLine "  Select challenge (1-3): " -Color Yellow
                        $challenge = Read-Host
                        if ($challenge -match "^[1-3]$") {
                            Open-Challenge -Level ([int]$level) -Track $trackName -ChallengeNum ([int]$challenge)
                        }
                    }
                }
            }
            "3" {
                Clear-Host
                Write-ColorLine @"
     ██████╗ ██████╗ ██████╗ ██╗██╗      ██████╗ ████████╗
    ██╔════╝██╔═══██╗██╔══██╗██║██║     ██╔═══██╗╚══██╔══╝
    ██║     ██║   ██║██████╔╝██║██║     ██║   ██║   ██║   
    ██║     ██║   ██║██╔═══╝ ██║██║     ██║   ██║   ██║   
    ╚██████╗╚██████╔╝██║     ██║███████╗╚██████╔╝   ██║   
     ╚═════╝ ╚═════╝ ╚═╝     ╚═╝╚══════╝ ╚═════╝    ╚═╝   
"@ -Color Green
                Write-Host ""
                Write-ColorLine "  Opening Copilot Demos folder..." -Color Cyan
                code (Join-Path $LabRoot "Copilot-Demos")
                Read-Host "  Press Enter to continue"
            }
            "4" {
                $progress = Initialize-Progress
                Clear-Host
                Write-ColorLine @"
  ╔═══════════════════════════════════════════════════════════════╗
  ║                 📊 YOUR PROGRESS 📊                           ║
  ╚═══════════════════════════════════════════════════════════════╝
"@ -Color Cyan
                Write-Host ""
                $completed = Get-TotalCompleted -Progress $progress
                Write-Host -NoNewline "  "
                Show-ProgressBar -Current $completed -Total 24
                Write-Host ""
                Write-ColorLine "  ⭐ XP Points: $($progress.XP_POINTS)" -Color Yellow
                Write-Host ""
                Read-Host "  Press Enter to continue"
            }
            "5" {
                Test-Prerequisites
            }
            "6" {
                code (Join-Path $LabRoot "Solutions")
                Read-Host "  Press Enter to continue"
            }
            "q" {
                Clear-Host
                Write-ColorLine @"
    
     ████████╗██╗  ██╗ █████╗ ███╗   ██╗██╗  ██╗███████╗██╗
     ╚══██╔══╝██║  ██║██╔══██╗████╗  ██║██║ ██╔╝██╔════╝██║
        ██║   ███████║███████║██╔██╗ ██║█████╔╝ ███████╗██║
        ██║   ██╔══██║██╔══██║██║╚██╗██║██╔═██╗ ╚════██║╚═╝
        ██║   ██║  ██║██║  ██║██║ ╚████║██║  ██╗███████║██╗
        ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝  ╚═╝╚══════╝╚═╝
    
         Keep learning, keep building! 🚀
    
"@ -Color Cyan
                return
            }
        }
    }
}

# Run
Start-IaCLab
