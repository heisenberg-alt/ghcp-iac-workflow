import { useEffect, useRef } from 'react'
import * as d3 from 'd3'
import type { CostResult } from '../types'

interface CostChartProps {
  result?: CostResult
}

export function CostChart({ result }: CostChartProps) {
  const svgRef = useRef<SVGSVGElement>(null)

  useEffect(() => {
    if (!result?.content || !svgRef.current) return

    // Parse cost data from content
    const costData = parseCostContent(result.content)
    if (costData.length === 0) return

    const svg = d3.select(svgRef.current)
    svg.selectAll('*').remove()

    const width = 500
    const height = 300
    const margin = { top: 20, right: 30, bottom: 60, left: 60 }
    const innerWidth = width - margin.left - margin.right
    const innerHeight = height - margin.top - margin.bottom

    const container = svg
      .attr('viewBox', `0 0 ${width} ${height}`)
      .append('g')
      .attr('transform', `translate(${margin.left},${margin.top})`)

    // Color scale
    const color = d3.scaleOrdinal<string>()
      .domain(costData.map(d => d.name))
      .range(d3.schemeTableau10)

    // X scale
    const x = d3.scaleBand()
      .domain(costData.map(d => d.name))
      .range([0, innerWidth])
      .padding(0.3)

    // Y scale
    const maxCost = d3.max(costData, d => d.cost) || 100
    const y = d3.scaleLinear()
      .domain([0, maxCost * 1.1])
      .range([innerHeight, 0])

    // Draw bars
    container.selectAll('rect')
      .data(costData)
      .enter()
      .append('rect')
      .attr('x', d => x(d.name)!)
      .attr('y', d => y(d.cost))
      .attr('width', x.bandwidth())
      .attr('height', d => innerHeight - y(d.cost))
      .attr('fill', d => color(d.name))
      .attr('rx', 4)
      .style('filter', 'drop-shadow(0 2px 4px rgba(0,0,0,0.1))')

    // Draw cost labels on bars
    container.selectAll('.cost-label')
      .data(costData)
      .enter()
      .append('text')
      .attr('class', 'cost-label')
      .attr('x', d => x(d.name)! + x.bandwidth() / 2)
      .attr('y', d => y(d.cost) - 5)
      .attr('text-anchor', 'middle')
      .attr('font-size', '12px')
      .attr('font-weight', 'bold')
      .attr('fill', '#333')
      .text(d => `$${d.cost.toFixed(2)}`)

    // X axis
    container.append('g')
      .attr('transform', `translate(0,${innerHeight})`)
      .call(d3.axisBottom(x))
      .selectAll('text')
      .attr('transform', 'rotate(-45)')
      .attr('text-anchor', 'end')
      .attr('font-size', '10px')

    // Y axis
    container.append('g')
      .call(d3.axisLeft(y).tickFormat(d => `$${d}`))
      .selectAll('text')
      .attr('font-size', '10px')

    // Y axis label
    container.append('text')
      .attr('transform', 'rotate(-90)')
      .attr('y', -45)
      .attr('x', -innerHeight / 2)
      .attr('text-anchor', 'middle')
      .attr('font-size', '12px')
      .attr('fill', '#666')
      .text('Monthly Cost ($)')

    // Total cost annotation
    const totalCost = d3.sum(costData, d => d.cost)
    container.append('text')
      .attr('x', innerWidth)
      .attr('y', -5)
      .attr('text-anchor', 'end')
      .attr('font-size', '14px')
      .attr('font-weight', 'bold')
      .attr('fill', '#333')
      .text(`Total: $${totalCost.toFixed(2)}/mo`)

  }, [result])

  if (!result?.content) {
    return (
      <div className="graph-placeholder">
        <div className="placeholder-icon">💰</div>
        <p>Click <strong>Analyze</strong> to see cost breakdown</p>
      </div>
    )
  }

  return (
    <div className="cost-chart">
      <svg ref={svgRef} width="100%" height="100%" />
      <div className="cost-summary">
        <div className="summary-item">
          <span className="summary-label">Estimate Period:</span>
          <span className="summary-value">Monthly</span>
        </div>
        <div className="summary-item">
          <span className="summary-label">Currency:</span>
          <span className="summary-value">USD</span>
        </div>
        <div className="summary-item">
          <span className="summary-label">Region:</span>
          <span className="summary-value">East US</span>
        </div>
      </div>
    </div>
  )
}

interface CostItem {
  name: string
  cost: number
  type: string
}

function parseCostContent(content: string): CostItem[] {
  const items: CostItem[] = []
  
  // Try to parse cost items from the content
  // Format: "Resource: $XX.XX/month"
  const lines = content.split('\n')
  
  for (const line of lines) {
    // Match patterns like "Storage: $12.50" or "azurerm_storage_account: $12.50/month"
    const match = line.match(/([a-zA-Z_\s]+):\s*\$?([\d,]+\.?\d*)/i)
    if (match) {
      const name = match[1].trim()
        .replace('azurerm_', '')
        .replace(/_/g, ' ')
        .split(' ')
        .map(w => w.charAt(0).toUpperCase() + w.slice(1))
        .join(' ')
      const cost = parseFloat(match[2].replace(',', ''))
      
      if (!isNaN(cost) && cost > 0 && !name.toLowerCase().includes('total')) {
        items.push({ name, cost, type: match[1].trim() })
      }
    }
  }

  // If no items found, try a different pattern
  if (items.length === 0) {
    // Look for table-like format
    const tableMatch = content.match(/\|\s*([^|]+)\s*\|\s*\$?([\d,]+\.?\d*)\s*\|/g)
    if (tableMatch) {
      for (const row of tableMatch) {
        const parts = row.match(/\|\s*([^|]+)\s*\|\s*\$?([\d,]+\.?\d*)\s*\|/)
        if (parts && parts[1] && parts[2]) {
          const name = parts[1].trim()
          const cost = parseFloat(parts[2].replace(',', ''))
          if (!isNaN(cost) && cost > 0) {
            items.push({ name, cost, type: name })
          }
        }
      }
    }
  }

  return items
}
