# Demo Recording Instructions

## Prerequisites

1. All 10 agents running (`gh iac status` shows all online)
2. `gh iac` extension installed
3. Screen recording software

## Recommended Recording Software

### Option 1: ScreenToGif (Simplest)
```powershell
# Install via winget
winget install ScreenToGif

# Or download from: https://www.screentogif.com/
```

### Option 2: OBS Studio (More Control)
```powershell
winget install OBSStudio
```

## Recording Steps

### 1. Prepare Terminal
```powershell
# Set font size to 16pt in Windows Terminal settings
# Set window width to ~100 characters
# Use dark theme
```

### 2. Navigate to Demo Folder
```powershell
cd C:\GHCP-IaC\copilot-iac\Copilot-SDK\enterprise\demo
```

### 3. Start Recording
- **ScreenToGif**: Click record, select terminal window
- **OBS**: Add Window Capture, select terminal

### 4. Run Demo Script
```powershell
.\demo-script.ps1
```

### 5. Stop Recording & Export
- Export as MP4
- Recommended: 1920x1080, 30fps
- Save to: `enterprise/dashboard/static/demo.mp4`

## Manual Demo (if you prefer)

Run these commands with pauses between each:

```powershell
# Scene 1: Status
gh iac version
gh iac status

# Scene 2: Policy (bad code)
cat .\bad-storage.tf
cat .\bad-storage.tf | gh iac policy

# Scene 3: Security
gh iac security < .\bad-storage.tf

# Scene 4: Cost
cat .\good-storage.tf | gh iac cost

# Scene 5: Compliance (Bicep)
cat .\bad-keyvault.bicep
gh iac compliance < .\bad-keyvault.bicep

# Scene 6: Full Check
gh iac check < .\good-storage.tf
```

## After Recording

1. Place `demo.mp4` in `enterprise/dashboard/static/`
2. The dashboard will automatically display it

## Tips

- Keep terminal font large (14-16pt)
- Pause 2-3 seconds after each output
- Total duration: ~2 minutes ideal
- No need for mouse movements
