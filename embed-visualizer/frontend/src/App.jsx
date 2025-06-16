import React, { useState, useEffect } from 'react';
import ForceGraph from './ForceGraph';
import { ProcessFile, OpenDatabase, GetGraphData, SelectFile, SelectDirectory, SelectDatabase } from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";

function App() {
  const [currentView, setCurrentView] = useState('process'); // 'process' or 'visualize'
  const [graphData, setGraphData] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [minSimilarity, setMinSimilarity] = useState(0.8);
  
  // Process form state
  const [filePath, setFilePath] = useState('');
  const [outputDir, setOutputDir] = useState('');
  const [ollamaHost, setOllamaHost] = useState('http://localhost:11434');
  const [maxWorkers, setMaxWorkers] = useState(4);
  const [progress, setProgress] = useState('');

  // Database state
  const [dbPath, setDbPath] = useState('');

  useEffect(() => {
    // Set up event listeners immediately when component mounts
    const embeddingProgressListener = (data) => {
      setProgress(`Generating embeddings: ${data.completed}/${data.total}`);
    };

    const summaryProgressListener = (data) => {
      setProgress(`Generating summaries: ${data.completed}/${data.total}`);
    };

    const similarityProgressListener = (data) => {
      setProgress(data.message);
    };

    const processingCompleteListener = (data) => {
      setProgress('');
      setLoading(false);
      setDbPath(data.dbPath);
      setCurrentView('visualize');
      fetchGraphData(data.dbPath);
    };

    // Register event listeners
    EventsOn('embedding-progress', embeddingProgressListener);
    EventsOn('summary-progress', summaryProgressListener);
    EventsOn('similarity-progress', similarityProgressListener);
    EventsOn('processing-complete', processingCompleteListener);

    // Cleanup event listeners on unmount
    return () => {
      // Note: EventsOff is not available in Wails, but we can clean up by not referencing old functions
    };
  }, []);

  const handleSelectFile = async () => {
    try {
      const path = await SelectFile();
      setFilePath(path);
    } catch (err) {
      console.error('Error selecting file:', err);
    }
  };

  const handleSelectDirectory = async () => {
    try {
      const path = await SelectDirectory();
      setOutputDir(path);
    } catch (err) {
      console.error('Error selecting directory:', err);
    }
  };

  const handleSelectDatabase = async () => {
    try {
      const path = await SelectDatabase();
      setDbPath(path);
      await OpenDatabase(path);
      fetchGraphData(path);
    } catch (err) {
      setError(`Failed to open database: ${err}`);
    }
  };

  const handleProcessFile = async () => {
    if (!filePath || !outputDir) {
      setError('Please select both a file and output directory');
      return;
    }

    setLoading(true);
    setError(null);
    setProgress('Starting processing...');

    try {
      await ProcessFile(filePath, outputDir, ollamaHost, maxWorkers);
    } catch (err) {
      setError(`Processing failed: ${err}`);
      setLoading(false);
      setProgress('');
    }
  };

  const fetchGraphData = async (path = dbPath) => {
    if (!path) return;
    
    setLoading(true);
    setError(null);
    
    try {
      const data = await GetGraphData(minSimilarity);
      setGraphData(data);
    } catch (err) {
      setError(`Failed to fetch graph data: ${err}`);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (dbPath && currentView === 'visualize') {
      fetchGraphData();
    }
  }, [minSimilarity, dbPath, currentView]);

  return (
    <div className="min-h-screen bg-dark-bg">
      {/* Header */}
      <header className="glass border-b border-dark-border/30 sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
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
            
            <div className="flex items-center space-x-4">
              <button 
                onClick={() => setCurrentView('process')}
                className={`btn ${currentView === 'process' ? 'btn-primary' : 'btn-secondary'}`}
              >
                Process
              </button>
              <button 
                onClick={() => setCurrentView('visualize')}
                className={`btn ${currentView === 'visualize' ? 'btn-primary' : 'btn-secondary'}`}
              >
                Visualize
              </button>
            </div>
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
                <h3 className="text-red-400 font-medium">Error</h3>
                <p className="text-red-300/80 text-sm mt-1">{error}</p>
              </div>
            </div>
          </div>
        )}

        {currentView === 'process' && (
          <div className="card">
            <div className="p-6">
              <h2 className="text-xl font-semibold text-dark-text mb-6">Process Text File</h2>
              
              <div className="space-y-6">
                <div>
                  <label className="block text-sm font-medium text-dark-muted mb-2">
                    Text File
                  </label>
                  <div className="flex items-center space-x-3">
                    <input 
                      type="text" 
                      value={filePath} 
                      readOnly
                      className="input flex-1" 
                      placeholder="Select a .txt or .md file"
                    />
                    <button onClick={handleSelectFile} className="btn-secondary">
                      Browse
                    </button>
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-dark-muted mb-2">
                    Output Directory
                  </label>
                  <div className="flex items-center space-x-3">
                    <input 
                      type="text" 
                      value={outputDir} 
                      readOnly
                      className="input flex-1" 
                      placeholder="Select output directory"
                    />
                    <button onClick={handleSelectDirectory} className="btn-secondary">
                      Browse
                    </button>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-6">
                  <div>
                    <label className="block text-sm font-medium text-dark-muted mb-2">
                      Ollama Host
                    </label>
                    <input 
                      type="text" 
                      value={ollamaHost}
                      onChange={(e) => setOllamaHost(e.target.value)}
                      className="input w-full"
                      placeholder="http://localhost:11434"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-dark-muted mb-2">
                      Max Workers
                    </label>
                    <input 
                      type="number" 
                      value={maxWorkers}
                      onChange={(e) => setMaxWorkers(parseInt(e.target.value))}
                      className="input w-full"
                      min="1"
                      max="16"
                    />
                  </div>
                </div>

                {progress && (
                  <div className="p-4 bg-dark-surface rounded-lg border border-dark-border">
                    <div className="flex items-center space-x-3">
                      <div className="animate-spin rounded-full h-4 w-4 border-2 border-dark-primary border-t-transparent"></div>
                      <span className="text-dark-text">{progress}</span>
                    </div>
                  </div>
                )}

                <button 
                  onClick={handleProcessFile}
                  disabled={loading || !filePath || !outputDir}
                  className={`btn-primary w-full ${loading ? 'opacity-50 cursor-not-allowed' : ''}`}
                >
                  {loading ? 'Processing...' : 'Process File'}
                </button>
              </div>
            </div>
          </div>
        )}

        {currentView === 'visualize' && (
          <div className="space-y-6">
            <div className="card">
              <div className="p-6">
                <div className="flex items-center justify-between mb-4">
                  <h2 className="text-xl font-semibold text-dark-text">Visualization</h2>
                  
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
                
                <div className="flex items-center space-x-6 mb-6">
                  <div className="flex items-center space-x-3">
                    <label className="text-sm font-medium text-dark-muted">Database:</label>
                    <div className="flex items-center space-x-2">
                      <input 
                        type="text" 
                        value={dbPath} 
                        readOnly
                        className="input w-64" 
                        placeholder="Select database file"
                      />
                      <button onClick={handleSelectDatabase} className="btn-secondary">
                        Browse
                      </button>
                    </div>
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
                </div>
              </div>
            </div>

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

            {!graphData && !loading && !dbPath && (
              <div className="card">
                <div className="p-8 text-center">
                  <div className="w-16 h-16 bg-dark-surface rounded-full flex items-center justify-center mx-auto mb-4">
                    <svg className="w-8 h-8 text-dark-muted" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M3 17a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm3.293-7.707a1 1 0 011.414 0L9 10.586V3a1 1 0 112 0v7.586l1.293-1.293a1 1 0 111.414 1.414l-3 3a1 1 0 01-1.414 0l-3-3a1 1 0 010-1.414z" clipRule="evenodd" />
                    </svg>
                  </div>
                  <h3 className="text-xl font-medium text-dark-text mb-2">No Database Selected</h3>
                  <p className="text-dark-muted mb-6">Select a database file to visualize embeddings</p>
                  <button onClick={handleSelectDatabase} className="btn-primary">
                    Select Database
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </main>
    </div>
  );
}

export default App;
