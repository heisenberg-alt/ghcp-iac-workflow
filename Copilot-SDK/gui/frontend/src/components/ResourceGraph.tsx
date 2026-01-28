import { useEffect, useRef, useState } from 'react'
import * as d3 from 'd3'
import type { GraphData, GraphNode, GraphLink } from '../types'

interface ResourceGraphProps {
  data?: GraphData
}

export function ResourceGraph({ data }: ResourceGraphProps) {
  const svgRef = useRef<SVGSVGElement>(null)
  const [dimensions, setDimensions] = useState({ width: 600, height: 400 })

  useEffect(() => {
    const handleResize = () => {
      if (svgRef.current?.parentElement) {
        const { width, height } = svgRef.current.parentElement.getBoundingClientRect()
        setDimensions({ width: width || 600, height: height || 400 })
      }
    }
    handleResize()
    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [])

  useEffect(() => {
    if (!data || !svgRef.current) return

    const svg = d3.select(svgRef.current)
    svg.selectAll('*').remove()

    const { width, height } = dimensions

    // Color scale for categories
    const colorScale: Record<string, string> = {
      network: '#4CAF50',
      storage: '#2196F3',
      compute: '#FF9800',
      database: '#9C27B0',
      security: '#F44336',
      application: '#00BCD4',
      other: '#607D8B'
    }

    // Icon mappings for resource types
    const iconMap: Record<string, string> = {
      network: '🌐',
      storage: '💾',
      compute: '🖥️',
      database: '🗄️',
      security: '🔐',
      application: '📱',
      other: '📦'
    }

    // Create container with zoom
    const container = svg.append('g')

    const zoom = d3.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.3, 3])
      .on('zoom', (event) => {
        container.attr('transform', event.transform)
      })

    svg.call(zoom)

    // Arrow marker for links
    svg.append('defs').append('marker')
      .attr('id', 'arrowhead')
      .attr('viewBox', '-0 -5 10 10')
      .attr('refX', 25)
      .attr('refY', 0)
      .attr('orient', 'auto')
      .attr('markerWidth', 6)
      .attr('markerHeight', 6)
      .append('path')
      .attr('d', 'M 0,-5 L 10,0 L 0,5')
      .attr('fill', '#999')

    // Create force simulation
    const simulation = d3.forceSimulation<GraphNode>(data.nodes)
      .force('link', d3.forceLink<GraphNode, GraphLink>(data.links)
        .id(d => d.id)
        .distance(120))
      .force('charge', d3.forceManyBody().strength(-400))
      .force('center', d3.forceCenter(width / 2, height / 2))
      .force('collision', d3.forceCollide().radius(50))

    // Draw links
    const link = container.append('g')
      .attr('class', 'links')
      .selectAll('line')
      .data(data.links)
      .enter()
      .append('line')
      .attr('stroke', '#999')
      .attr('stroke-opacity', 0.6)
      .attr('stroke-width', 2)
      .attr('marker-end', 'url(#arrowhead)')

    // Draw nodes
    const node = container.append('g')
      .attr('class', 'nodes')
      .selectAll('g')
      .data(data.nodes)
      .enter()
      .append('g')
      .attr('class', 'node')
      .call(d3.drag<SVGGElement, GraphNode>()
        .on('start', (event, d) => {
          if (!event.active) simulation.alphaTarget(0.3).restart()
          d.fx = d.x
          d.fy = d.y
        })
        .on('drag', (event, d) => {
          d.fx = event.x
          d.fy = event.y
        })
        .on('end', (event, d) => {
          if (!event.active) simulation.alphaTarget(0)
          d.fx = null
          d.fy = null
        }))

    // Node circles
    node.append('circle')
      .attr('r', 30)
      .attr('fill', d => colorScale[d.category] || colorScale.other)
      .attr('stroke', d => d.status === 'error' ? '#F44336' : d.status === 'warning' ? '#FF9800' : '#fff')
      .attr('stroke-width', 3)
      .style('filter', 'drop-shadow(0 2px 4px rgba(0,0,0,0.2))')

    // Node icons
    node.append('text')
      .attr('text-anchor', 'middle')
      .attr('dominant-baseline', 'central')
      .attr('font-size', '20px')
      .text(d => iconMap[d.category] || iconMap.other)

    // Node labels
    node.append('text')
      .attr('dy', 45)
      .attr('text-anchor', 'middle')
      .attr('font-size', '11px')
      .attr('font-weight', 'bold')
      .attr('fill', '#333')
      .text(d => d.name)

    // Node type labels
    node.append('text')
      .attr('dy', 58)
      .attr('text-anchor', 'middle')
      .attr('font-size', '9px')
      .attr('fill', '#666')
      .text(d => {
        const parts = d.type.split('_')
        return parts.slice(-1)[0]
      })

    // Tooltip
    const tooltip = d3.select('body').append('div')
      .attr('class', 'graph-tooltip')
      .style('position', 'absolute')
      .style('visibility', 'hidden')
      .style('background', 'white')
      .style('border', '1px solid #ddd')
      .style('border-radius', '8px')
      .style('padding', '12px')
      .style('box-shadow', '0 4px 12px rgba(0,0,0,0.15)')
      .style('font-size', '12px')
      .style('z-index', '1000')

    node.on('mouseover', (event, d) => {
      tooltip
        .style('visibility', 'visible')
        .html(`
          <strong>${d.name}</strong><br/>
          <span style="color: #666">Type:</span> ${d.type}<br/>
          <span style="color: #666">Category:</span> ${d.category}<br/>
          <span style="color: ${d.status === 'ok' ? 'green' : d.status === 'warning' ? 'orange' : 'red'}">
            Status: ${d.status}
          </span>
        `)
    })
    .on('mousemove', (event) => {
      tooltip
        .style('top', (event.pageY - 10) + 'px')
        .style('left', (event.pageX + 10) + 'px')
    })
    .on('mouseout', () => {
      tooltip.style('visibility', 'hidden')
    })

    // Update positions on tick
    simulation.on('tick', () => {
      link
        .attr('x1', d => (d.source as GraphNode).x!)
        .attr('y1', d => (d.source as GraphNode).y!)
        .attr('x2', d => (d.target as GraphNode).x!)
        .attr('y2', d => (d.target as GraphNode).y!)

      node.attr('transform', d => `translate(${d.x},${d.y})`)
    })

    // Cleanup
    return () => {
      simulation.stop()
      tooltip.remove()
    }
  }, [data, dimensions])

  if (!data || data.nodes.length === 0) {
    return (
      <div className="graph-placeholder">
        <div className="placeholder-icon">🔗</div>
        <p>Click <strong>Analyze</strong> to visualize resource dependencies</p>
      </div>
    )
  }

  return (
    <div className="resource-graph">
      <svg ref={svgRef} width="100%" height="100%" />
      <div className="graph-legend">
        <div className="legend-item">
          <span className="legend-color" style={{ background: '#4CAF50' }} />
          Network
        </div>
        <div className="legend-item">
          <span className="legend-color" style={{ background: '#2196F3' }} />
          Storage
        </div>
        <div className="legend-item">
          <span className="legend-color" style={{ background: '#F44336' }} />
          Security
        </div>
        <div className="legend-item">
          <span className="legend-color" style={{ background: '#00BCD4' }} />
          Application
        </div>
      </div>
    </div>
  )
}
