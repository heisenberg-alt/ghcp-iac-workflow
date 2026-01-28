import { useState, useRef, useEffect } from 'react'
import { streamCopilot } from '../api'
import type { ChatMessage } from '../types'

interface CopilotChatProps {
  context: string
}

export function CopilotChat({ context }: CopilotChatProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([
    {
      id: 'welcome',
      role: 'assistant',
      content: '👋 Hi! I\'m your IaC Governance assistant. I can help you:\n\n• **Check policies** - "Are there any policy violations?"\n• **Estimate costs** - "How much will this cost per month?"\n• **Security review** - "Check for security issues"\n• **Explain resources** - "What does this storage account do?"',
      timestamp: new Date()
    }
  ])
  const [input, setInput] = useState('')
  const [isStreaming, setIsStreaming] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    scrollToBottom()
  }, [messages])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!input.trim() || isStreaming) return

    const userMessage: ChatMessage = {
      id: `user-${Date.now()}`,
      role: 'user',
      content: input.trim(),
      timestamp: new Date()
    }

    setMessages(prev => [...prev, userMessage])
    setInput('')
    setIsStreaming(true)

    // Create placeholder for assistant message
    const assistantId = `assistant-${Date.now()}`
    setMessages(prev => [...prev, {
      id: assistantId,
      role: 'assistant',
      content: '',
      timestamp: new Date()
    }])

    try {
      for await (const chunk of streamCopilot(userMessage.content, context)) {
        setMessages(prev => prev.map(msg => 
          msg.id === assistantId 
            ? { ...msg, content: msg.content + chunk }
            : msg
        ))
      }
    } catch (err) {
      setMessages(prev => prev.map(msg => 
        msg.id === assistantId 
          ? { ...msg, content: '❌ Sorry, I encountered an error. Please try again.' }
          : msg
      ))
    } finally {
      setIsStreaming(false)
    }
  }

  const quickActions = [
    { label: '📋 Check Policies', message: 'Check this code for policy violations' },
    { label: '💰 Estimate Cost', message: 'Estimate the monthly cost for this infrastructure' },
    { label: '🔒 Security Review', message: 'Review this code for security issues' },
  ]

  return (
    <div className="copilot-chat">
      <div className="chat-messages">
        {messages.map(msg => (
          <div key={msg.id} className={`chat-message ${msg.role}`}>
            <div className="message-avatar">
              {msg.role === 'user' ? '👤' : '🤖'}
            </div>
            <div className="message-content">
              <div className="message-text">
                {msg.content || (
                  <span className="typing-indicator">
                    <span>●</span><span>●</span><span>●</span>
                  </span>
                )}
              </div>
              <div className="message-time">
                {msg.timestamp.toLocaleTimeString()}
              </div>
            </div>
          </div>
        ))}
        <div ref={messagesEndRef} />
      </div>

      <div className="quick-actions">
        {quickActions.map(action => (
          <button
            key={action.label}
            className="quick-action-btn"
            onClick={() => setInput(action.message)}
            disabled={isStreaming}
          >
            {action.label}
          </button>
        ))}
      </div>

      <form className="chat-input-form" onSubmit={handleSubmit}>
        <input
          type="text"
          className="chat-input"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Ask about your infrastructure..."
          disabled={isStreaming}
        />
        <button 
          type="submit" 
          className="send-btn"
          disabled={!input.trim() || isStreaming}
        >
          {isStreaming ? '⏳' : '➤'}
        </button>
      </form>
    </div>
  )
}
