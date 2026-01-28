# IaC Governance GUI

A web-based dashboard for visualizing Infrastructure-as-Code governance with embedded GitHub Copilot chat integration.

![Dashboard Screenshot](docs/dashboard.png)

## Features

- 🔗 **Resource Dependency Graph** - D3.js force-directed visualization of IaC resources
- 💰 **Cost Breakdown Charts** - Interactive cost estimation charts
- 📋 **Policy Status Cards** - Real-time policy compliance status
- 💬 **Embedded Copilot Chat** - Conversational interface to governance agents
- ⚡ **Real-time Updates** - WebSocket-powered live status updates
- 🎨 **Modern React UI** - Clean, responsive dashboard design

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│              IaC Governance Dashboard                    │
│  ┌─────────────────────────────────────────────────────┐│
│  │ React Frontend (Vite + TypeScript)                  ││
│  │  • D3.js Resource Graph                             ││
│  │  • Cost Charts                                      ││
│  │  • Copilot Chat Panel                               ││
│  └─────────────────────────────────────────────────────┘│
│                          │                              │
│  ┌─────────────────────────────────────────────────────┐│
│  │ Go Backend (HTTP + WebSocket)                       ││
│  │  • REST API (/api/*)                                ││
│  │  • SSE Proxy (/api/copilot)                         ││
│  │  • WebSocket (/ws)                                  ││
│  │  • Static file serving                              ││
│  └─────────────────────────────────────────────────────┘│
└────────────────────│────────────────────────────────────┘
                     │
     ┌───────────────┼───────────────┐
     │               │               │
     ▼               ▼               ▼
┌─────────┐   ┌───────────┐   ┌──────────────┐
│ Policy  │   │   Cost    │   │ Orchestrator │
│ Agent   │   │ Estimator │   │    Agent     │
│ :8081   │   │   :8082   │   │    :8090     │
└─────────┘   └───────────┘   └──────────────┘
```

## Quick Start

### Prerequisites

- Go 1.21+
- Node.js 18+
- Running IaC agents (Policy Agent on 8081, Cost Estimator on 8082)

### Build & Run

```bash
# Build frontend
cd frontend
npm install
npm run build
cd ..

# Build Go backend with embedded frontend
go build -o gui.exe .

# Start the server
$env:PORT="3000"
.\gui.exe
```

Open http://localhost:3000 in your browser.

### Development Mode

For frontend hot-reload during development:

```bash
# Terminal 1: Start Go backend
go run .

# Terminal 2: Start Vite dev server
cd frontend
npm run dev
```

Open http://localhost:5173 (Vite proxies API calls to :3000)

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `PORT` | `3000` | GUI server port |
| `POLICY_AGENT_URL` | `http://localhost:8081` | Policy Agent URL |
| `COST_AGENT_URL` | `http://localhost:8082` | Cost Estimator URL |
| `ORCHESTRATOR_URL` | `http://localhost:8090` | Orchestrator URL |
| `ENABLE_CORS` | `true` | Enable CORS headers |

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Serve dashboard UI |
| `/api/health` | GET | Health check |
| `/api/status` | GET | Agent status |
| `/api/analyze` | POST | Analyze IaC code |
| `/api/copilot` | POST | Chat with agents (SSE) |
| `/ws` | WS | Real-time updates |

## Usage

### 1. Paste Infrastructure Code

Paste your Terraform or Bicep code in the editor panel.

### 2. Click Analyze

The system will:
- Parse resources and build dependency graph
- Run policy checks
- Estimate costs
- Display results in real-time

### 3. Explore Visualizations

- **Resource Graph**: Drag nodes, zoom, and hover for details
- **Cost Chart**: View cost breakdown by resource

### 4. Chat with Copilot

Ask questions like:
- "Are there any policy violations?"
- "How much will this cost per month?"
- "Check for security issues"

## Tech Stack

### Frontend
- React 18 + TypeScript
- D3.js 7 for visualizations
- Vite for build tooling
- CSS with custom properties

### Backend
- Go 1.21
- Gorilla WebSocket
- Embedded filesystem (go:embed)

## Project Structure

```
gui/
├── main.go                 # Go backend server
├── go.mod                  # Go dependencies
├── gui.exe                 # Built executable
└── frontend/
    ├── package.json        # npm dependencies
    ├── vite.config.ts      # Vite configuration
    ├── tsconfig.json       # TypeScript config
    ├── index.html          # Entry HTML
    └── src/
        ├── main.tsx        # React entry
        ├── App.tsx         # Main app component
        ├── api.ts          # API client
        ├── types.ts        # TypeScript types
        ├── hooks/
        │   └── useWebSocket.ts
        ├── components/
        │   ├── Header.tsx
        │   ├── StatusCards.tsx
        │   ├── CodeEditor.tsx
        │   ├── ResourceGraph.tsx
        │   ├── CostChart.tsx
        │   └── CopilotChat.tsx
        └── styles/
            └── global.css
```

## Next Version: Desktop App

The current web version can be packaged as a desktop app using [Wails](https://wails.io/):

```bash
# Install Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Create Wails project
wails init -n iac-governance-desktop -t react-ts

# Copy components and build
wails build
```

## License

MIT
