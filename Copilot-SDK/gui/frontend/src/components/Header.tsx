interface HeaderProps {
  isConnected: boolean
}

export function Header({ isConnected }: HeaderProps) {
  return (
    <header className="header">
      <div className="header-left">
        <h1>🎨 IaC Governance Dashboard</h1>
        <span className="subtitle">Powered by GitHub Copilot Agents</span>
      </div>
      <div className="header-right">
        <div className={`connection-status ${isConnected ? 'connected' : 'disconnected'}`}>
          <span className="status-dot" />
          {isConnected ? 'Connected' : 'Reconnecting...'}
        </div>
      </div>
    </header>
  )
}
