# 🚀 IaC Lab Interactive CLI

An engaging, gamified command-line interface for learning Infrastructure as Code with Terraform and Bicep, powered by GitHub Copilot.

![IaC Lab](https://img.shields.io/badge/IaC-Lab-blue)
![Terraform](https://img.shields.io/badge/Terraform-4.0+-purple)
![Bicep](https://img.shields.io/badge/Bicep-Latest-blue)
![Copilot](https://img.shields.io/badge/GitHub-Copilot-green)

## ✨ Features

- 🎮 **Gamified Learning** - XP points, achievements, and progress tracking
- 🎨 **Beautiful ASCII Art** - Eye-catching logos and animations
- 🎉 **Celebration Animations** - Fireworks when you complete challenges!
- 📊 **Progress Dashboard** - Track your journey across all levels
- ✅ **Auto-Validation** - Instant feedback on your solutions
- 🔧 **Prerequisites Check** - Verify your environment is ready
- 💻 **Cross-Platform** - Works on macOS, Linux, and Windows

## 🖥️ Screenshots

```
    ██╗ █████╗  ██████╗    ██╗      █████╗ ██████╗ 
    ██║██╔══██╗██╔════╝    ██║     ██╔══██╗██╔══██╗
    ██║███████║██║         ██║     ███████║██████╔╝
    ██║██╔══██║██║         ██║     ██╔══██║██╔══██╗
    ██║██║  ██║╚██████╗    ███████╗██║  ██║██████╔╝
    ╚═╝╚═╝  ╚═╝ ╚═════╝    ╚══════╝╚═╝  ╚═╝╚═════╝ 
```

## 📦 Installation

### Quick Start (All Platforms)

```bash
# Clone the repository
git clone https://github.com/heisenberg-alt/copilot-IaC-lab.git
cd copilot-IaC-lab

# Run the installer
./cli/install.sh
```

### macOS / Linux

```bash
# Option 1: Run directly
./cli/iac-lab.sh

# Option 2: Install to PATH
./cli/install.sh
# Then restart terminal and run:
iac-lab
```

### Windows

#### Using Git Bash or WSL (Recommended)
```bash
# Navigate to the cli folder
cd cli

# Run the bash script
bash iac-lab.sh
```

#### Using PowerShell (Native)
```powershell
# Navigate to the cli folder
cd cli

# Run the PowerShell script
.\iac-lab.ps1
```

#### Add to PATH (Windows)
```powershell
# Add the cli folder to your PATH environment variable
# Or create a batch file in a PATH location:
# iac-lab.cmd with content: @powershell -ExecutionPolicy Bypass -File "C:\path\to\cli\iac-lab.ps1"
```

## 🎯 Usage

### Main Menu Options

| Option | Description |
|--------|-------------|
| `1` | Start Learning Journey - Begin from Level 1 |
| `2` | Select Specific Challenge - Jump to any challenge |
| `3` | Copilot Demo Scenarios - See Copilot in action |
| `4` | View Progress & Achievements - Check your stats |
| `5` | Prerequisites Check - Verify your setup |
| `6` | Quick Reference Guides - Access cheat sheets |
| `q` | Exit |

### Challenge Actions

When viewing a challenge:

| Key | Action |
|-----|--------|
| `o` | Open challenge in VS Code |
| `v` | Verify your solution |
| `h` | Get a hint (use Copilot!) |
| `s` | View the solution |
| `b` | Go back |

## 🏆 Gamification

### XP System

| Level | XP per Challenge | XP for Level Completion |
|-------|------------------|-------------------------|
| Level 1 | 100 XP | 500 XP |
| Level 2 | 200 XP | 1000 XP |
| Level 3 | 300 XP | 1500 XP |
| Level 4 | 400 XP | 2000 XP |

### Ranks

| XP Required | Rank |
|-------------|------|
| 0 | 🌱 Seedling |
| 500 | 🌿 Intermediate |
| 1500 | 🌳 Advanced |
| 3000 | 🏆 Expert |
| 5000 | 👑 IaC Master |

### Achievements

- 🏅 **First Steps** - Complete your first challenge
- 🎖️ **Level 1 Graduate** - Complete all Level 1 challenges
- 🥉 **Rising Star** - Complete all Level 1 & 2 challenges
- 🥈 **IaC Professional** - Complete all Level 1-3 challenges
- 🥇 **IaC Master** - Complete ALL challenges!

## 📁 Progress Storage

Your progress is saved locally:

- **Bash version**: `~/.iac-lab-progress`
- **PowerShell version**: `~/.iac-lab-progress.json`

To reset progress, simply delete this file.

## 🔧 Prerequisites

The CLI will check for these tools:

| Tool | Required | Purpose |
|------|----------|---------|
| Terraform | ✅ | Terraform challenges |
| Azure CLI | ✅ | Authentication & Bicep |
| Bicep CLI | ⚠️ | Bicep challenges |
| VS Code | ⚠️ | Opening challenges |
| Git | ✅ | Version control |
| GitHub CLI | ⚡ | Copilot CLI demos |

## 🐛 Troubleshooting

### Script won't run on macOS/Linux
```bash
chmod +x cli/iac-lab.sh
chmod +x cli/install.sh
```

### PowerShell execution policy
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### Unicode/Emoji not displaying
- Ensure your terminal supports UTF-8
- On Windows, use Windows Terminal for best results
- Try: `chcp 65001` in Command Prompt

### Colors not showing
- Use a terminal that supports ANSI colors
- Windows Terminal, iTerm2, or modern Linux terminals work best

## 🤝 Contributing

Found a bug or want to add a feature? PRs welcome!

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## 📄 License

MIT License - feel free to use this for your own learning labs!

---

**Happy Learning! 🚀**

*Built with ❤️ and GitHub Copilot*
