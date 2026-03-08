import { useEffect, useState } from 'react';
import { Rocket, Shield, Globe, ExternalLink, RefreshCw } from 'lucide-react';

function App() {
  const [apps, setApps] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [currentTime, setCurrentTime] = useState(new Date().toLocaleTimeString());

  const fetchApps = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/apps');
      if (!response.ok) throw new Error('Failed to fetch apps');
      const data = await response.json();
      setApps(data);
      setError(null);
    } catch (err) {
      console.error('Error fetching apps:', err);
      setError('Could not load applications. Please try again later.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchApps();
    const timer = setInterval(() => {
      setCurrentTime(new Date().toLocaleTimeString());
    }, 1000);
    return () => clearInterval(timer);
  }, []);

  return (
    <div className="container animate-fadeIn">
      <header style={{ textAlign: 'center' }}>
        <h1 className="title">Welcome to MyOS</h1>
        <p className="subtitle">
          Your centralized gateway for enterprise-grade microservices and applications.
          Managed with precision, delivered with speed.
        </p>
      </header>

      <main className="glass-panel">
        <div className="section-header">
          <Rocket className="text-primary" size={24} />
          <h2 className="section-title">Deployed Applications</h2>
          <div style={{ marginLeft: 'auto' }}>
            <button
              onClick={fetchApps}
              className="refresh-button"
              style={{
                background: 'transparent',
                border: 'none',
                color: 'var(--text-muted)',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: '0.5rem'
              }}
            >
              <RefreshCw size={16} className={loading ? 'animate-spin' : ''} />
              Refresh
            </button>
          </div>
        </div>

        {loading ? (
          <div className="app-grid">
            {[1, 2, 3].map((i) => (
              <div key={i} className="app-card loading-shimmer" style={{ height: '140px' }} />
            ))}
          </div>
        ) : error ? (
          <div className="empty-state">
            <p style={{ color: '#f87171' }}>{error}</p>
          </div>
        ) : apps.length > 0 ? (
          <div className="app-grid">
            {apps.map((app) => (
              <a
                key={app}
                href={`http://${app}`}
                className="app-card"
                target="_blank"
                rel="noopener noreferrer"
              >
                <div className="app-icon-wrapper">
                  {app.charAt(0).toUpperCase()}
                </div>
                <div className="app-name">{app}</div>
                <div className="app-url" style={{ display: 'flex', alignItems: 'center', gap: '0.25rem' }}>
                  <Globe size={12} />
                  <span>{app}</span>
                  <ExternalLink size={12} style={{ marginLeft: 'auto' }} />
                </div>
              </a>
            ))}
          </div>
        ) : (
          <div className="empty-state">
            <p>No applications currently deployed on this instance.</p>
          </div>
        )}
      </main>

      <footer className="footer">
        <div className="status-indicator">
          <span className="status-dot"></span>
          <span>System Online</span>
        </div>
        <span style={{ color: 'var(--glass-border)' }}>|</span>
        <div className="status-indicator">
          <Shield size={14} />
          <span>Secure Gateway Connected</span>
        </div>
        <span style={{ color: 'var(--glass-border)' }}>|</span>
        <span>{currentTime}</span>
      </footer>
    </div>
  );
}

export default App;
