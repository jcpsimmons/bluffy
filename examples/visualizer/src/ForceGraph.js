import * as d3 from 'd3';
import React, { useEffect, useRef, useState } from 'react';

const ForceGraph = ({ data, width = 800, height = 600 }) => {
  const svgRef = useRef();
  const [selectedNode, setSelectedNode] = useState(null);
  const [currentZoom, setCurrentZoom] = useState(1);

  useEffect(() => {
    if (!data || !data.nodes || !data.links) return;

    const svg = d3.select(svgRef.current);
    svg.selectAll("*").remove(); // Clear previous render

    // Create zoom behavior
    const zoom = d3.zoom()
      .scaleExtent([0.05, 8])
      .on("zoom", (event) => {
        container.attr("transform", event.transform);
        setCurrentZoom(event.transform.k);
        // Update node sizes immediately on zoom
        node.attr("r", d => Math.max(12, 15 / Math.sqrt(event.transform.k)));
      });

    // Apply zoom to SVG
    svg.call(zoom);

    // Create container group for zoom/pan
    const container = svg.append("g");

    // Create force simulation
    const simulation = d3.forceSimulation(data.nodes)
      .force("link", d3.forceLink(data.links).id(d => d.id).distance(d => (1 - d.similarity) * 200 + 50))
      .force("charge", d3.forceManyBody().strength(-300))
      .force("center", d3.forceCenter(width / 2, height / 2))
      .force("collision", d3.forceCollide().radius(20));

    // Create links
    const link = container.append("g")
      .attr("class", "links")
      .selectAll("line")
      .data(data.links)
      .enter().append("line")
      .attr("stroke", "#475569")
      .attr("stroke-opacity", d => d.similarity * 0.6 + 0.3)
      .attr("stroke-width", d => d.similarity * 2 + 0.5);

    // Create nodes
    const node = container.append("g")
      .attr("class", "nodes")
      .selectAll("circle")
      .data(data.nodes)
      .enter().append("circle")
      .attr("r", 10)
      .attr("fill", d => {
        const colors = ['#3b82f6', '#8b5cf6', '#06b6d4', '#10b981', '#f59e0b', '#ef4444', '#ec4899', '#84cc16'];
        return colors[d.index % colors.length];
      })
      .attr("stroke", "#1a1a2e")
      .attr("stroke-width", 2)
      .style("cursor", "pointer")
      .call(d3.drag()
        .on("start", dragstarted)
        .on("drag", dragged)
        .on("end", dragended))
      .on("click", (event, d) => {
        event.stopPropagation();
        setSelectedNode(d);
      })
      .on("mouseover", function(event, d) {
        d3.select(this).attr("r", Math.max(16, 18 / Math.sqrt(currentZoom)));
        console.log(d)
        
        // Show tooltip
        const tooltip = container.append("g")
          .attr("id", "tooltip")
          .attr("transform", `translate(${d.x + 15}, ${d.y - 15})`);
        
        const text = tooltip.append("text")
          .attr("class", "tooltip-text")
          .style("font-size", "12px")
          .style("fill", "#e2e8f0")
          .style("font-weight", "500")
          .text(d.summary || d.text.substring(0, 40) + (d.text.length > 40 ? "..." : ""));
        
        const bbox = text.node().getBBox();
        tooltip.insert("rect", "text")
          .attr("x", bbox.x - 8)
          .attr("y", bbox.y - 4)
          .attr("width", bbox.width + 16)
          .attr("height", bbox.height + 8)
          .attr("rx", 6)
          .style("fill", "#1a1a2e")
          .style("stroke", "#2a2a54")
          .style("stroke-width", 1)
          .style("filter", "drop-shadow(0 4px 6px rgba(0, 0, 0, 0.3))");
      })
      .on("mouseout", function(event, d) {
        d3.select(this).attr("r", Math.max(12, 15 / Math.sqrt(currentZoom)));
        container.select("#tooltip").remove();
      });

    // Add labels
    const label = container.append("g")
      .attr("class", "labels")
      .selectAll("text")
      .data(data.nodes)
      .enter().append("text")
      .attr("text-anchor", "middle")
      .attr("dy", 28)
      .style("font-size", "11px")
      .style("font-weight", "500")
      .style("fill", "#94a3b8")
      .style("pointer-events", "none")
      .text(d => d.summary || `C${d.index}`);

    // Update positions on simulation tick
    simulation.on("tick", () => {
      link
        .attr("x1", d => d.source.x)
        .attr("y1", d => d.source.y)
        .attr("x2", d => d.target.x)
        .attr("y2", d => d.target.y);

      node
        .attr("cx", d => d.x)
        .attr("cy", d => d.y)
        .attr("r", d => Math.max(12, 15 / Math.sqrt(currentZoom)));

      label
        .attr("x", d => d.x)
        .attr("y", d => d.y);
    });

    function dragstarted(event, d) {
      if (!event.active) simulation.alphaTarget(0.3).restart();
      d.fx = d.x;
      d.fy = d.y;
    }

    function dragged(event, d) {
      d.fx = event.x;
      d.fy = event.y;
    }

    function dragended(event, d) {
      if (!event.active) simulation.alphaTarget(0);
      d.fx = null;
      d.fy = null;
    }

    // Add reset zoom button functionality
    const resetZoom = () => {
      svg.transition()
        .duration(750)
        .call(zoom.transform, d3.zoomIdentity);
    };

    // Store reset function for external access
    svg.node().resetZoom = resetZoom;

    return () => {
      simulation.stop();
    };

  }, [data, width, height]);

  const handleResetZoom = () => {
    if (svgRef.current && svgRef.current.resetZoom) {
      svgRef.current.resetZoom();
    }
  };

  return (
    <div className="flex gap-6 h-full flex-col">
      <div className="relative flex-1">
        <svg 
          ref={svgRef} 
          width={width} 
          height={height}
          className="bg-dark-surface border border-dark-border rounded-lg cursor-grab"
        />
        
        {/* Controls Overlay */}
        <div className="absolute top-4 right-4 flex flex-col gap-3">
          <button 
            onClick={handleResetZoom}
            className="btn-secondary text-xs shadow-lg"
          >
            <svg className="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1z" clipRule="evenodd" />
            </svg>
            Reset Zoom
          </button>
          
          <div className="glass p-3 rounded-lg text-xs text-dark-muted space-y-1">
            <div className="flex items-center space-x-2">
              <span>🖱️</span>
              <span>Drag to pan</span>
            </div>
            <div className="flex items-center space-x-2">
              <span>🔍</span>
              <span>Scroll to zoom</span>
            </div>
            <div className="flex items-center space-x-2">
              <span>🎯</span>
              <span>Drag nodes</span>
            </div>
            <div className="flex items-center space-x-2">
              <span>👆</span>
              <span>Click for details</span>
            </div>
          </div>
        </div>
      </div>
      
      {selectedNode && (
        <div className="card w-full">
          <div className="p-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center space-x-3">
                <div className="w-8 h-8 bg-gradient-to-br from-dark-primary to-dark-secondary rounded-lg flex items-center justify-center text-white text-sm font-bold">
                  {selectedNode.index}
                </div>
                <h3 className="text-lg font-semibold text-dark-text">
                {selectedNode.summary}
                </h3>
              </div>
              <button 
                onClick={() => setSelectedNode(null)}
                className="p-1 hover:bg-dark-hover rounded-md transition-colors"
              >
                <svg className="w-5 h-5 text-dark-muted" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
                </svg>
              </button>
            </div>
            
            
            <div className="space-y-3">
              <div>
                <label className="text-xs font-medium text-dark-muted uppercase tracking-wide">
                  Content
                </label>
                <div className="mt-1 p-3 bg-dark-surface rounded-lg border border-dark-border">
                  <p className="text-sm text-dark-text leading-relaxed">
                    {selectedNode.text}
                  </p>
                </div>
              </div>
              
              <div className="flex items-center justify-between text-xs text-dark-muted">
                <span>Characters: {selectedNode.text.length}</span>
                <span>Words: {selectedNode.text.split(' ').length}</span>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default ForceGraph;
