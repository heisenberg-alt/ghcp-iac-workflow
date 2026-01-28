import type { StatusResponse, AnalysisResult } from './types'

const API_BASE = ''

export async function getStatus(): Promise<StatusResponse> {
  const response = await fetch(`${API_BASE}/api/status`)
  if (!response.ok) throw new Error('Failed to fetch status')
  return response.json()
}

export async function analyzeCode(
  code: string, 
  checks: string[] = ['policy', 'cost']
): Promise<AnalysisResult> {
  const response = await fetch(`${API_BASE}/api/analyze`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ code, checks })
  })
  if (!response.ok) throw new Error('Analysis failed')
  return response.json()
}

export async function* streamCopilot(
  message: string, 
  context: string
): AsyncGenerator<string> {
  const response = await fetch(`${API_BASE}/api/copilot`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ message, context })
  })

  if (!response.ok) throw new Error('Copilot request failed')
  if (!response.body) throw new Error('No response body')

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    buffer = lines.pop() || ''

    for (const line of lines) {
      if (line.startsWith('data: ')) {
        try {
          const data = JSON.parse(line.slice(6))
          if (data.content) {
            yield data.content
          }
        } catch {
          // Skip non-JSON lines
        }
      }
    }
  }
}
