import React, { useState, useEffect } from 'react';
import ForceGraph from './ForceGraph';
import './App.css';

function App() {
  const [graphData, setGraphData] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [minSimilarity, setMinSimilarity] = useState(0.5);
  const [apiUrl, setApiUrl] = useState('http://localhost:8080');

  const fetchGraphData = async () => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch(`${apiUrl}/api/graph?min_similarity=${minSimilarity}`);
      const result = await response.json();
      
      if (result.success) {
        setGraphData(result.data);
      } else {
        setError(result.error || 'Failed to fetch data');
      }
    } catch (err) {
      setError(`Connection error: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchGraphData();
  }, [minSimilarity, apiUrl]);

  return (
    <div className="App" style={{ padding: '20px' }}>
      <header style={{ marginBottom: '20px' }}>
        <h1>Text Embeddings Visualizer</h1>
        
        <div style={{ display: 'flex', gap: '20px', alignItems: 'center', marginBottom: '20px' }}>
          <div>
            <label>
              API URL:
              <input 
                type="text" 
                value={apiUrl}
                onChange={(e) => setApiUrl(e.target.value)}
                style={{ marginLeft: '10px', padding: '5px', width: '200px' }}
              />
            </label>
          </div>
          
          <div>
            <label>
              Min Similarity:
              <input 
                type="range"
                min="0"
                max="1"
                step="0.1"
                value={minSimilarity}
                onChange={(e) => setMinSimilarity(parseFloat(e.target.value))}
                style={{ marginLeft: '10px' }}
              />
              <span style={{ marginLeft: '10px' }}>{minSimilarity}</span>
            </label>
          </div>
          
          <button 
            onClick={fetchGraphData}
            disabled={loading}
            style={{
              padding: '8px 16px',
              border: '1px solid #ccc',
              borderRadius: '4px',
              cursor: loading ? 'not-allowed' : 'pointer',
              opacity: loading ? 0.6 : 1
            }}
          >
            {loading ? 'Loading...' : 'Refresh'}
          </button>
        </div>

        {error && (
          <div style={{ 
            color: 'red', 
            padding: '10px', 
            border: '1px solid red', 
            borderRadius: '4px',
            marginBottom: '20px'
          }}>
            Error: {error}
          </div>
        )}

        {graphData && (
          <div style={{ marginBottom: '10px', fontSize: '14px', color: '#666' }}>
            Showing {graphData.nodes?.length || 0} chunks and {graphData.links?.length || 0} connections
          </div>
        )}
      </header>

      {loading && <div>Loading graph data...</div>}
      
      {graphData && !loading && (
        <ForceGraph 
          data={graphData} 
          width={1000} 
          height={700} 
        />
      )}

      {!graphData && !loading && !error && (
        <div style={{ textAlign: 'center', marginTop: '50px' }}>
          <p>No data available. Make sure:</p>
          <ul style={{ textAlign: 'left', display: 'inline-block' }}>
            <li>The API server is running</li>
            <li>You have generated embeddings with the CLI tool</li>
            <li>The database path is correct</li>
          </ul>
        </div>
      )}
    </div>
  );
}

export default App;
