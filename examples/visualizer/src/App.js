import React, { useEffect, useState } from 'react';
import ForceGraph from './ForceGraph';

function App() {
  const [graphData, setGraphData] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [minSimilarity, setMinSimilarity] = useState(0.8);
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
  }, [minSimilarity, apiUrl]); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="min-h-screen bg-dark-bg">
      {/* Header */}
      <header className="glass border-b border-dark-border/30 sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-6 py-4">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center space-x-3">
              <div className="w-8 h-8 bg-gradient-to-br from-dark-primary to-dark-secondary rounded-lg flex items-center justify-center">
                <svg className="w-5 h-5 text-white" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M3 4a1 1 0 011-1h12a1 1 0 011 1v2a1 1 0 01-1 1H4a1 1 0 01-1-1V4zM3 10a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H4a1 1 0 01-1-1v-6zM14 9a1 1 0 00-1 1v6a1 1 0 001 1h2a1 1 0 001-1v-6a1 1 0 00-1-1h-2z" />
                </svg>
              </div>
              <h1 className="text-2xl font-bold bg-gradient-to-r from-dark-text to-dark-muted bg-clip-text text-transparent">
                Text Embeddings Visualizer
              </h1>
            </div>
            
            {graphData && (
              <div className="flex items-center space-x-4">
                <div className="badge-primary">
                  {graphData.nodes?.length || 0} chunks
                </div>
                <div className="badge-secondary">
                  {graphData.links?.length || 0} connections
                </div>
              </div>
            )}
          </div>
          
          {/* Controls */}
          <div className="flex flex-wrap items-center gap-6">
            <div className="flex items-center space-x-3">
              <label className="text-sm font-medium text-dark-muted">API URL:</label>
              <input 
                type="text" 
                value={apiUrl}
                onChange={(e) => setApiUrl(e.target.value)}
                className="input w-64"
                placeholder="http://localhost:8080"
              />
            </div>
            
            <div className="flex items-center space-x-3">
              <label className="text-sm font-medium text-dark-muted">Min Similarity:</label>
              <input 
                type="range"
                min="0"
                max="1"
                step="0.01"
                value={minSimilarity}
                onChange={(e) => setMinSimilarity(parseFloat(e.target.value))}
                className="w-32 h-2 bg-dark-surface rounded-lg appearance-none cursor-pointer slider"
              />
              <span className="text-sm font-mono text-dark-primary min-w-[3rem]">
                {minSimilarity.toFixed(2)}
              </span>
            </div>
            
            <button 
              onClick={fetchGraphData}
              disabled={loading}
              className={`btn-primary ${loading ? 'opacity-50 cursor-not-allowed' : ''}`}
            >
              {loading ? (
                <div className="flex items-center space-x-2">
                  <div className="animate-spin rounded-full h-4 w-4 border-2 border-white border-t-transparent"></div>
                  <span>Loading...</span>
                </div>
              ) : (
                <div className="flex items-center space-x-2">
                  <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clipRule="evenodd" />
                  </svg>
                  <span>Refresh</span>
                </div>
              )}
            </button>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-6 py-6">
        {error && (
          <div className="card border-red-500/30 bg-red-500/10 mb-6">
            <div className="p-4 flex items-center space-x-3">
              <svg className="w-5 h-5 text-red-400 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
              </svg>
              <div>
                <h3 className="text-red-400 font-medium">Connection Error</h3>
                <p className="text-red-300/80 text-sm mt-1">{error}</p>
              </div>
            </div>
          </div>
        )}

        {loading && !graphData && (
          <div className="card">
            <div className="p-8 text-center">
              <div className="animate-spin rounded-full h-12 w-12 border-2 border-dark-primary border-t-transparent mx-auto mb-4"></div>
              <h3 className="text-lg font-medium text-dark-text mb-2">Loading graph data...</h3>
              <p className="text-dark-muted">Fetching embeddings and similarities</p>
            </div>
          </div>
        )}
        
        {graphData && !loading && (
          <div className="card overflow-hidden">
            <ForceGraph 
              data={graphData} 
              width={1000} 
              height={700} 
            />
          </div>
        )}

        {!graphData && !loading && !error && (
          <div className="card">
            <div className="p-8 text-center">
              <div className="w-16 h-16 bg-dark-surface rounded-full flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-dark-muted" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm3.293-7.707a1 1 0 011.414 0L9 10.586V3a1 1 0 112 0v7.586l1.293-1.293a1 1 0 111.414 1.414l-3 3a1 1 0 01-1.414 0l-3-3a1 1 0 010-1.414z" clipRule="evenodd" />
                </svg>
              </div>
              <h3 className="text-xl font-medium text-dark-text mb-2">No Data Available</h3>
              <p className="text-dark-muted mb-6">To get started, make sure:</p>
              <div className="inline-flex flex-col items-start space-y-2 text-sm text-dark-muted">
                <div className="flex items-center space-x-2">
                  <div className="w-1.5 h-1.5 bg-dark-primary rounded-full"></div>
                  <span>The API server is running</span>
                </div>
                <div className="flex items-center space-x-2">
                  <div className="w-1.5 h-1.5 bg-dark-primary rounded-full"></div>
                  <span>You have generated embeddings with the CLI tool</span>
                </div>
                <div className="flex items-center space-x-2">
                  <div className="w-1.5 h-1.5 bg-dark-primary rounded-full"></div>
                  <span>The database path is correct</span>
                </div>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}

export default App;
