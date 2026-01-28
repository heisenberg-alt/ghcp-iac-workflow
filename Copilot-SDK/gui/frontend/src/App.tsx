import { useState, useEffect, useCallback } from 'react'
import { Header } from './components/Header'
import { StatusCards } from './components/StatusCards'
import { ResourceGraph } from './components/ResourceGraph'
import { CostChart } from './components/CostChart'
import { CopilotChat } from './components/CopilotChat'
import { CodeEditor } from './components/CodeEditor'
import { useWebSocket } from './hooks/useWebSocket'
import { analyzeCode, getStatus } from './api'
import type { AnalysisResult, AgentStatus, GraphData } from './types'

function App() {
  const [code, setCode] = useState(SAMPLE_TERRAFORM)
  const [analysisResult, setAnalysisResult] = useState<AnalysisResult | null>(null)
  const [agentStatus, setAgentStatus] = useState<Record<string, AgentStatus>>({})
  const [isAnalyzing, setIsAnalyzing] = useState(false)
  const [activeTab, setActiveTab] = useState<'graph' | 'cost'>('graph')

  // WebSocket for real-time updates
  const { lastMessage, isConnected } = useWebSocket()

  // Fetch agent status on mount
  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const status = await getStatus()
        setAgentStatus(status.agents)
      } catch (err) {
        console.error('Failed to fetch status:', err)
      }
    }
    fetchStatus()
    const interval = setInterval(fetchStatus, 30000)
    return () => clearInterval(interval)
  }, [])

  // Handle WebSocket messages
  useEffect(() => {
    if (lastMessage?.type === 'analysis_completed') {
      setAnalysisResult(lastMessage.payload as AnalysisResult)
      setIsAnalyzing(false)
    }
  }, [lastMessage])

  const handleAnalyze = useCallback(async () => {
    setIsAnalyzing(true)
    try {
      const result = await analyzeCode(code, ['policy', 'cost'])
      setAnalysisResult(result)
    } catch (err) {
      console.error('Analysis failed:', err)
    } finally {
      setIsAnalyzing(false)
    }
  }, [code])

  return (
    <div className="app">
      <Header isConnected={isConnected} />
      
      <main className="main-content">
        <div className="left-panel">
          <StatusCards 
            agentStatus={agentStatus} 
            analysisResult={analysisResult}
          />
          
          <div className="editor-section">
            <div className="section-header">
              <h2>📝 Infrastructure Code</h2>
              <button 
                className="analyze-btn"
                onClick={handleAnalyze}
                disabled={isAnalyzing}
              >
                {isAnalyzing ? '⏳ Analyzing...' : '🔍 Analyze'}
              </button>
            </div>
            <CodeEditor value={code} onChange={setCode} />
          </div>
        </div>
        
        <div className="right-panel">
          <div className="visualization-section">
            <div className="tab-header">
              <button 
                className={`tab ${activeTab === 'graph' ? 'active' : ''}`}
                onClick={() => setActiveTab('graph')}
              >
                🔗 Resource Graph
              </button>
              <button 
                className={`tab ${activeTab === 'cost' ? 'active' : ''}`}
                onClick={() => setActiveTab('cost')}
              >
                💰 Cost Breakdown
              </button>
            </div>
            
            <div className="visualization-content">
              {activeTab === 'graph' ? (
                <ResourceGraph data={analysisResult?.graph} />
              ) : (
                <CostChart result={analysisResult?.results?.cost} />
              )}
            </div>
          </div>
          
          <div className="chat-section">
            <h2>💬 Copilot Chat</h2>
            <CopilotChat context={code} />
          </div>
        </div>
      </main>
    </div>
  )
}

const SAMPLE_TERRAFORM = `# Azure Infrastructure - Storage and Networking
resource "azurerm_resource_group" "main" {
  name     = "rg-iac-demo"
  location = "eastus"
}

resource "azurerm_virtual_network" "main" {
  name                = "vnet-main"
  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
}

resource "azurerm_subnet" "app" {
  name                 = "subnet-app"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.1.0/24"]
}

resource "azurerm_storage_account" "main" {
  name                     = "stiacdemostorage"
  resource_group_name      = azurerm_resource_group.main.name
  location                 = azurerm_resource_group.main.location
  account_tier             = "Standard"
  account_replication_type = "GRS"
  
  enable_https_traffic_only = true
  min_tls_version           = "TLS1_2"
  allow_blob_public_access  = false
}

resource "azurerm_key_vault" "main" {
  name                = "kv-iac-demo"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  tenant_id           = data.azurerm_client_config.current.tenant_id
  sku_name            = "standard"
  
  soft_delete_retention_days = 90
  purge_protection_enabled   = true
}`

export default App
