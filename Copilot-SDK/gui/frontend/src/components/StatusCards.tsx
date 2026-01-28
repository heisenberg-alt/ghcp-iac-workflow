import type { AgentStatus, AnalysisResult } from '../types'

interface StatusCardsProps {
  agentStatus: Record<string, AgentStatus>
  analysisResult: AnalysisResult | null
}

export function StatusCards({ agentStatus, analysisResult }: StatusCardsProps) {
  // Extract metrics from analysis result
  const policyContent = analysisResult?.results?.policy?.content || ''
  const costContent = analysisResult?.results?.cost?.content || ''
  
  // Parse policy violations from content
  const violationMatch = policyContent.match(/(\d+)\s*(?:violation|issue)/i)
  const violations = violationMatch ? parseInt(violationMatch[1]) : null
  
  // Parse cost from content  
  const costMatch = costContent.match(/\$?([\d,]+(?:\.\d{2})?)/i)
  const totalCost = costMatch ? costMatch[1] : null

  // Count online agents
  const onlineAgents = Object.values(agentStatus).filter(a => a.status === 'online').length
  const totalAgents = Object.keys(agentStatus).length

  return (
    <div className="status-cards">
      <div className="status-card agents">
        <div className="card-icon">🤖</div>
        <div className="card-content">
          <div className="card-value">{onlineAgents}/{totalAgents}</div>
          <div className="card-label">Agents Online</div>
        </div>
        <div className="agent-dots">
          {Object.entries(agentStatus).map(([name, status]) => (
            <span 
              key={name}
              className={`agent-dot ${status.status}`}
              title={`${name}: ${status.status} (${status.latency}ms)`}
            />
          ))}
        </div>
      </div>

      <div className={`status-card policy ${violations === 0 ? 'success' : violations ? 'warning' : ''}`}>
        <div className="card-icon">📋</div>
        <div className="card-content">
          <div className="card-value">
            {violations !== null ? (violations === 0 ? '✓ Pass' : `${violations} Issues`) : '—'}
          </div>
          <div className="card-label">Policy Check</div>
        </div>
      </div>

      <div className="status-card cost">
        <div className="card-icon">💰</div>
        <div className="card-content">
          <div className="card-value">
            {totalCost ? `$${totalCost}/mo` : '—'}
          </div>
          <div className="card-label">Estimated Cost</div>
        </div>
      </div>

      <div className="status-card resources">
        <div className="card-icon">📦</div>
        <div className="card-content">
          <div className="card-value">
            {analysisResult?.resources?.length ?? '—'}
          </div>
          <div className="card-label">Resources</div>
        </div>
      </div>
    </div>
  )
}
