import { useState, useEffect } from 'react'
import type { WSMessage } from '../types'

export function useWebSocket() {
  const [lastMessage, setLastMessage] = useState<WSMessage | null>(null)
  const [isConnected, setIsConnected] = useState(false)

  useEffect(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/ws`
    
    let ws: WebSocket | null = null
    let reconnectTimeout: number | null = null

    const connect = () => {
      ws = new WebSocket(wsUrl)

      ws.onopen = () => {
        console.log('WebSocket connected')
        setIsConnected(true)
      }

      ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data) as WSMessage
          setLastMessage(message)
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err)
        }
      }

      ws.onclose = () => {
        console.log('WebSocket disconnected')
        setIsConnected(false)
        // Reconnect after 3 seconds
        reconnectTimeout = window.setTimeout(connect, 3000)
      }

      ws.onerror = (error) => {
        console.error('WebSocket error:', error)
      }
    }

    connect()

    // Ping to keep connection alive
    const pingInterval = setInterval(() => {
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'ping' }))
      }
    }, 30000)

    return () => {
      if (reconnectTimeout) clearTimeout(reconnectTimeout)
      clearInterval(pingInterval)
      ws?.close()
    }
  }, [])

  return { lastMessage, isConnected }
}
