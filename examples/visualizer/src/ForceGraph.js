import React, { useEffect, useRef, useState } from 'react';
import * as d3 from 'd3';

const ForceGraph = ({ data, width = 800, height = 600 }) => {
  const svgRef = useRef();
  const [selectedNode, setSelectedNode] = useState(null);

  useEffect(() => {
    if (!data || !data.nodes || !data.links) return;

    const svg = d3.select(svgRef.current);
    svg.selectAll("*").remove(); // Clear previous render

    // Create zoom behavior
    const zoom = d3.zoom()
      .scaleExtent([0.1, 10])
      .on("zoom", (event) => {
        container.attr("transform", event.transform);
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
      .attr("stroke", "#999")
      .attr("stroke-opacity", d => d.similarity * 0.8 + 0.2)
      .attr("stroke-width", d => d.similarity * 3 + 1);

    // Create nodes
    const node = container.append("g")
      .attr("class", "nodes")
      .selectAll("circle")
      .data(data.nodes)
      .enter().append("circle")
      .attr("r", 8)
      .attr("fill", d => d3.schemeCategory10[d.index % 10])
      .attr("stroke", "#fff")
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
        d3.select(this).attr("r", 12);
        
        // Show tooltip
        const tooltip = container.append("g")
          .attr("id", "tooltip")
          .attr("transform", `translate(${d.x + 15}, ${d.y - 15})`);
        
        const text = tooltip.append("text")
          .attr("class", "tooltip-text")
          .style("font-size", "12px")
          .style("fill", "black")
          .style("background", "white")
          .text(d.text.substring(0, 50) + (d.text.length > 50 ? "..." : ""));
        
        const bbox = text.node().getBBox();
        tooltip.insert("rect", "text")
          .attr("x", bbox.x - 5)
          .attr("y", bbox.y - 2)
          .attr("width", bbox.width + 10)
          .attr("height", bbox.height + 4)
          .style("fill", "white")
          .style("stroke", "black")
          .style("stroke-width", 1);
      })
      .on("mouseout", function(event, d) {
        d3.select(this).attr("r", 8);
        container.select("#tooltip").remove();
      });

    // Add labels
    const label = container.append("g")
      .attr("class", "labels")
      .selectAll("text")
      .data(data.nodes)
      .enter().append("text")
      .attr("text-anchor", "middle")
      .attr("dy", 25)
      .style("font-size", "10px")
      .style("fill", "black")
      .text(d => `Chunk ${d.index}`);

    // Update positions on simulation tick
    simulation.on("tick", () => {
      link
        .attr("x1", d => d.source.x)
        .attr("y1", d => d.source.y)
        .attr("x2", d => d.target.x)
        .attr("y2", d => d.target.y);

      node
        .attr("cx", d => d.x)
        .attr("cy", d => d.y);

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
    <div style={{ display: 'flex', gap: '20px' }}>
      <div style={{ position: 'relative' }}>
        <svg 
          ref={svgRef} 
          width={width} 
          height={height}
          style={{ border: '1px solid #ccc', background: '#fafafa', cursor: 'grab' }}
        />
        <div style={{ 
          position: 'absolute', 
          top: '10px', 
          right: '10px',
          display: 'flex',
          flexDirection: 'column',
          gap: '5px'
        }}>
          <button 
            onClick={handleResetZoom}
            style={{ 
              padding: '5px 10px',
              border: '1px solid #ccc',
              borderRadius: '4px',
              backgroundColor: 'white',
              cursor: 'pointer',
              fontSize: '12px'
            }}
          >
            Reset Zoom
          </button>
          <div style={{
            padding: '5px',
            backgroundColor: 'rgba(255,255,255,0.9)',
            borderRadius: '4px',
            fontSize: '10px',
            border: '1px solid #ccc'
          }}>
            <div>🖱️ Drag to pan</div>
            <div>🔍 Scroll to zoom</div>
            <div>🎯 Drag nodes</div>
          </div>
        </div>
      </div>
      
      {selectedNode && (
        <div style={{ 
          width: '300px', 
          padding: '20px', 
          border: '1px solid #ccc',
          borderRadius: '8px',
          backgroundColor: '#f9f9f9'
        }}>
          <h3>Chunk {selectedNode.index}</h3>
          <p style={{ fontSize: '14px', lineHeight: '1.4' }}>
            {selectedNode.text}
          </p>
          <button 
            onClick={() => setSelectedNode(null)}
            style={{ 
              marginTop: '10px',
              padding: '5px 10px',
              border: '1px solid #ccc',
              borderRadius: '4px',
              cursor: 'pointer'
            }}
          >
            Close
          </button>
        </div>
      )}
    </div>
  );
};

export default ForceGraph;