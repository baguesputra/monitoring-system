import React, { useState, useEffect, useRef } from 'react';
import { 
  BookOpen, 
  Copy, 
  ChevronDown, 
  Search, 
  Globe, 
  Download, 
  Menu 
} from 'lucide-react';
import { apiEndpoints, apiTags, environments, getEndpointsByTag, buildOpenApiSpec } from '../data/apiEndpoints';
import EndpointCard from './EndpointCard';
import TryItPanel from './TryItPanel';

export default function ApiReference() {
  const [activeTag, setActiveTag] = useState('Agent');
  const [searchQuery, setSearchQuery] = useState('');
  const [environment, setEnvironment] = useState('development');
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);
  const [tryItOpen, setTryItOpen] = useState(false);
  const [tryItEndpoint, setTryItEndpoint] = useState(null);
  const mainRef = useRef(null);
  const sidebarRef = useRef(null);

  const filteredEndpoints = (tagId) => {
    const endpoints = getEndpointsByTag(tagId);
    if (!searchQuery.trim()) return endpoints;
    const q = searchQuery.toLowerCase();
    return endpoints.filter(ep => 
      ep.id.toLowerCase().includes(q) ||
      ep.path.toLowerCase().includes(q) ||
      ep.summary.toLowerCase().includes(q) ||
      ep.method.toLowerCase().includes(q)
    );
  };

  useEffect(() => {
    const hash = window.location.hash.slice(1);
    if (hash) {
      const el = document.getElementById(hash);
      if (el) {
        el.scrollIntoView({ behavior: 'smooth', block: 'start' });
        el.focus();
      }
    }
  }, []);

  useEffect(() => {
    const handleKeyDown = (e) => {
      if ((e.key === 'ArrowDown' || e.key === 'ArrowUp') && (e.ctrlKey || e.metaKey)) {
        e.preventDefault();
        const allCards = Array.from(mainRef.current?.querySelectorAll('[data-endpoint-id]') || []);
        const currentIndex = allCards.findIndex(el => el.id === window.location.hash.slice(1));
        const nextIndex = e.key === 'ArrowDown' ? currentIndex + 1 : currentIndex - 1;
        if (allCards[nextIndex]) {
          allCards[nextIndex].focus();
          allCards[nextIndex].scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  const openTryIt = (endpoint) => {
    setTryItEndpoint(endpoint);
    setTryItOpen(true);
  };

  const renderSidebarTags = () => {
    const authErrorsSection = (searchQuery && activeTag === 'AuthErrors') ? (
      <details className="group" defaultOpen>
        <summary className="flex items-center justify-between px-3 py-2.5 rounded-sm cursor-pointer text-ink-muted hover:text-ink hover:bg-ink/5 transition">
          <span className="font-mono-ui text-[11px] uppercase tracking-wider flex items-center gap-2">
            🔐 Authentication & Errors
          </span>
          <ChevronDown className="w-4 h-4" strokeWidth={2} />
        </summary>
        <ul className="mt-1 ml-2 space-y-1 border-l border-ink-muted/20 pl-3 text-[11px] text-ink-muted">
          <li className="py-1">Current: No authentication required (development)</li>
          <li className="py-1">Planned: Per-agent token auth (Phase 6)</li>
          <li className="py-1">Error codes: 400, 401, 404, 409, 500</li>
        </ul>
      </details>
    ) : null;

    const tagElements = apiTags.map(tag => {
      const isActive = activeTag === tag.id;
      return (
        <details key={tag.id} className="group" defaultOpen={activeTag === tag.id || activeTag === ''}>
          <summary
            onClick={() => setActiveTag(tag.id)}
            className={`flex items-center justify-between px-3 py-2.5 rounded-sm cursor-pointer transition ${isActive ? 'bg-ink text-parchment' : 'text-ink-muted hover:text-ink hover:bg-ink/5'}`}
            aria-expanded={activeTag === tag.id}
          >
            <span className="font-mono-ui text-[11px] uppercase tracking-wider flex items-center gap-2">
              {tag.label}
            </span>
            <ChevronDown className={`w-4 h-4 transition-transform ${activeTag === tag.id ? 'rotate-180' : ''}`} strokeWidth={2} />
          </summary>
          <ul className="mt-1 ml-2 space-y-1 border-l border-ink-muted/20 pl-3">
            {filteredEndpoints(tag.id).map(ep => (
              <li key={ep.id}>
                <button
                  onClick={() => {
                    window.location.hash = ep.id;
                    if (window.innerWidth < 768) setMobileSidebarOpen(false);
                  }}
                  className={`w-full text-left px-2 py-1.5 text-[11px] font-mono-ui rounded-sm transition ${window.location.hash === '#' + ep.id ? 'bg-ink text-parchment' : 'text-ink-muted hover:text-ink hover:bg-ink/5'}`}
                >
                  <span className={`inline-block w-16 ${getMethodBadgeClass(ep.method)} text-[9px] py-0.5 px-1.5 rounded`}>{getMethodLabel(ep.method)}</span>
                  <span className="ml-2 truncate block">{ep.path}</span>
                </button>
              </li>
            ))}
            {filteredEndpoints(tag.id).length === 0 && searchQuery && (
              <li className="text-ink-muted text-[11px] py-2">No matching endpoints</li>
            )}
          </ul>
        </details>
      );
    });

    return (
      <>
        {tagElements}
        {authErrorsSection}
      </>
    );
  };

  const copyToClipboard = async (text, label) => {
    try {
      await navigator.clipboard.writeText(text);
      console.log(`Copied ${label} to clipboard`);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  const exportOpenApi = () => {
    const blob = new Blob([JSON.stringify(buildOpenApiSpec(), null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'seneschal-openapi.json';
    a.click();
    URL.revokeObjectURL(url);
  };

  const getMethodBadgeClass = (method) => {
    switch (method) {
      case 'GET': return 'wax-seal-normal';
      case 'POST': return 'wax-seal-critical';
      case 'PUT': return 'wax-seal-warning';
      case 'DELETE': return 'wax-seal-offline';
      default: return 'wax-seal-normal';
    }
  };

  const getMethodLabel = (method) => method;

  const currentEnv = environments.find(e => e.id === environment) || environments[0];

  return (
    <div className="min-h-screen bg-parchment text-ink flex flex-col font-body antialiased">
      <header className="sticky top-0 z-50 bg-parchment border-b border-ink-muted/30 px-6 py-4">
        <div className="max-w-7xl mx-auto">
          <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
            <div className="flex items-center gap-4">
              <div className="p-3 bg-ink rounded-sm border border-ink shadow-wax flex-shrink-0">
                <BookOpen className="w-7 h-7 text-gold-leaf" strokeWidth={1.5} />
              </div>
              <div>
                <h1 className="font-display text-2xl md:text-3xl font-semibold text-ink tracking-tight leading-tight m-0">
                  SENESCHAL
                </h1>
                <p className="font-mono-ui text-[11px] text-ink-muted uppercase tracking-widest mt-0.5 m-0">
                  API Reference · Collector REST API
                </p>
              </div>
            </div>

            <div className="flex flex-wrap items-center gap-3">
              <div className="relative">
                <button
                  onClick={() => setMobileSidebarOpen(!mobileSidebarOpen)}
                  className="md:hidden p-2 bg-parchment-dark border border-ink-muted/30 text-ink hover:bg-ink/5 transition focus-ring rounded-sm"
                  aria-label="Toggle sidebar"
                >
                  <Menu className="w-5 h-5" strokeWidth={2} />
                </button>
                
                <select
                  value={environment}
                  onChange={(e) => setEnvironment(e.target.value)}
                  className={`appearance-none bg-parchment-dark border border-ink-muted/40 pl-10 pr-32 py-2 text-sm text-ink focus:outline-none focus:border-gold transition focus-ring rounded-sm cursor-pointer`}
                >
                  {environments.map(env => (
                    <option key={env.id} value={env.id}>{env.name}</option>
                  ))}
                </select>
                <Globe className="absolute right-8 top-1/2 -translate-y-1/2 w-4 h-4 text-ink-muted pointer-events-none" strokeWidth={2} />
              </div>

              <button
                onClick={() => navigator.clipboard.writeText(currentEnv.baseUrl)}
                className="flex items-center gap-2 px-4 py-2 bg-parchment-dark border border-ink-muted/40 text-ink text-[11px] font-mono-ui uppercase tracking-wider hover:bg-ink/5 transition focus-ring rounded-sm"
              >
                <Copy className="w-4 h-4" strokeWidth={2} />
                <span>Copy Base URL</span>
              </button>

              <button
                onClick={exportOpenApi}
                className="flex items-center gap-2 px-4 py-2 bg-ink text-parchment text-[11px] font-mono-ui uppercase tracking-wider hover:bg-ink/90 transition focus-ring rounded-sm"
              >
                <Download className="w-4 h-4" strokeWidth={2} />
                <span>Export OpenAPI</span>
              </button>
            </div>

            <div className="md:hidden mt-4">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-ink-muted" strokeWidth={2} />
                <input
                  type="text"
                  placeholder="Search endpoints..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-full bg-parchment border border-ink-muted/40 pl-10 pr-4 py-2.5 text-sm text-ink focus:outline-none focus:border-gold transition focus-ring rounded-sm"
                />
              </div>
            </div>
          </div>
        </div>
      </header>

      <div className="flex flex-1">
        <aside
          ref={sidebarRef}
          className={`${mobileSidebarOpen ? 'translate-x-0' : '-translate-x-full'} md:translate-x-0 fixed md:static inset-y-0 left-0 z-40 w-80 md:w-80 bg-parchment-dark border-r border-ink-muted/30 transform transition-transform duration-300 ease-in-out flex flex-col overflow-y-auto`}
          aria-label="API Navigation"
        >
          <div className="hidden md:block p-4 border-b border-ink-muted/30">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-ink-muted" strokeWidth={2} />
              <input
                type="text"
                placeholder="Search endpoints..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full bg-parchment border border-ink-muted/40 pl-10 pr-4 py-2.5 text-sm text-ink focus:outline-none focus:border-gold transition focus-ring rounded-sm"
              />
            </div>
          </div>

          <nav className="p-4 space-y-1" role="navigation" aria-label="API Endpoint Groups">
            {renderSidebarTags()}
          </nav>

          <div className="absolute bottom-0 left-0 right-0 p-4 border-t border-ink-muted/30 bg-parchment/50">
            <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0 text-center">
              Collector v2.0 · {apiEndpoints.length} endpoints
            </p>
          </div>
        </aside>

        {mobileSidebarOpen && (
          <div 
            className="fixed inset-0 z-30 bg-ink/50 md:hidden"
            onClick={() => setMobileSidebarOpen(false)}
            aria-hidden="true"
          />
        )}

        <main 
          ref={mainRef}
          className="flex-1 max-w-7xl w-full mx-auto px-6 py-8 overflow-auto"
          role="main"
          aria-label="API Documentation"
        >
          <div className="mb-8 pb-6 border-b border-ink-muted/30">
            <div className="flex flex-col md:flex-row md:items-end justify-between gap-4">
              <div>
                <h2 className="font-display text-2xl font-semibold text-ink m-0">API Reference</h2>
                <p className="font-mono-ui text-[11px] text-ink-muted uppercase tracking-wider mt-1 m-0">
                  Collector REST API — Base URL: <code className="text-gold-leaf font-mono-ui">{currentEnv.baseUrl}</code>
                </p>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => navigator.clipboard.writeText(currentEnv.baseUrl)}
                  className="flex items-center gap-2 px-3 py-1.5 bg-parchment-dark border border-ink-muted/40 text-ink text-[10px] font-mono-ui uppercase tracking-wider hover:bg-ink/5 transition focus-ring rounded-sm"
                >
                  <Copy className="w-3.5 h-3.5" strokeWidth={2} />
                  Copy Base URL
                </button>
              </div>
            </div>
          </div>

          <div className="space-y-8" role="list" aria-label="API Endpoints">
            {apiTags.filter(t => t.id !== 'AuthErrors').map(tag => {
              const filtered = filteredEndpoints(tag.id);
              if (filtered.length === 0) return null;
              return (
                <section key={tag.id} className="space-y-6" aria-labelledby={`tag-${tag.id}`}>
                  <div className="flex items-center justify-between pb-3 border-b border-ink-muted/20" id={`tag-${tag.id}`}>
                    <h3 className="font-display text-lg font-semibold text-ink flex items-center gap-2 m-0">
                      {tag.label}
                    </h3>
                    <span className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider">
                      {filtered.length} endpoint{filtered.length !== 1 ? 's' : ''}
                    </span>
                  </div>
                  <div className="space-y-6" role="list">
                    {filtered.map(ep => (
                      <EndpointCard
                        key={ep.id}
                        endpoint={ep}
                        environment={currentEnv}
                        onTryIt={openTryIt}
                        onCopy={copyToClipboard}
                      />
                    ))}
                  </div>
                </section>
              )}
            )}

            <section className="space-y-6" aria-labelledby="tag-AuthErrors">
              <div className="flex items-center justify-between pb-3 border-b border-ink-muted/20" id="tag-AuthErrors">
                <h3 className="font-display text-lg font-semibold text-ink flex items-center gap-2 m-0">
                  🔐 Authentication & Errors
                </h3>
              </div>
              <div className="ledger-card-elevated p-6 space-y-4">
                <div>
                  <h4 className="font-display text-sm font-semibold text-ink m-0 mb-2">Current Authentication</h4>
                  <p className="font-mono-ui text-[11px] text-ink-muted m-0">
                    Development mode: No authentication required. All endpoints accessible without tokens.
                  </p>
                </div>
                <div className="pt-4 border-t border-ink-muted/20">
                  <h4 className="font-display text-sm font-semibold text-ink m-0 mb-2">Planned: Per-Agent Token Auth (Phase 6)</h4>
                  <ul className="font-mono-ui text-[11px] text-ink-muted space-y-1 list-disc list-inside">
                    <li>Each agent receives unique token via secure provisioning</li>
                    <li>Tokens validated on every POST /api/metrics and POST /api/asset-info</li>
                    <li>Dashboard endpoints require session cookie authentication</li>
                    <li>Rate limiting per token (planned)</li>
                  </ul>
                </div>
                <div className="pt-4 border-t border-ink-muted/20">
                  <h4 className="font-display text-sm font-semibold text-ink m-0 mb-2">Error Code Reference</h4>
                  <table className="w-full text-[11px] font-mono-ui border-collapse">
                    <thead>
                      <tr className="border-b border-ink-muted/20 text-ink-muted uppercase tracking-wider">
                        <th className="pb-2 pl-4 text-left font-medium">Code</th>
                        <th className="pb-2 pl-4 text-left font-medium">Meaning</th>
                        <th className="pb-2 pl-4 text-left font-medium">When It Occurs</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-ink-muted/15 text-ink">
                      <tr className="hover:bg-ink/3"><td className="py-2 pl-4 font-medium text-cardinal">400</td><td className="py-2 pl-4">Bad Request</td><td className="py-2 pl-4">Invalid JSON, missing fields, validation failed</td></tr>
                      <tr className="hover:bg-ink/3"><td className="py-2 pl-4 font-medium text-cardinal">401</td><td className="py-2 pl-4">Unauthorized</td><td className="py-2 pl-4">Missing/invalid token (when auth enabled)</td></tr>
                      <tr className="hover:bg-ink/3"><td className="py-2 pl-4 font-medium text-cardinal">404</td><td className="py-2 pl-4">Not Found</td><td className="py-2 pl-4">Server ID not found, no data for device</td></tr>
                      <tr className="hover:bg-ink/3"><td className="py-2 pl-4 font-medium text-cardinal">409</td><td className="py-2 pl-4">Conflict</td><td className="py-2 pl-4">Duplicate server_id on rename</td></tr>
                      <tr className="hover:bg-ink/3"><td className="py-2 pl-4 font-medium text-cardinal">500</td><td className="py-2 pl-4">Internal Server Error</td><td className="py-2 pl-4">Database connection, unexpected failure</td></tr>
                    </tbody>
                  </table>
                </div>
              </div>
            </section>
          </div>
        </main>
      </div>

      {tryItOpen && tryItEndpoint && (
        <TryItPanel
          endpoint={tryItEndpoint}
          environment={currentEnv}
          onClose={() => { setTryItOpen(false); setTryItEndpoint(null); }}
        />
      )}

      <footer className="border-t border-ink-muted/30 bg-parchment-dark py-6 px-6">
        <div className="max-w-7xl mx-auto flex flex-col md:flex-row items-center justify-between gap-4 text-center md:text-left">
          <p className="font-mono-ui text-[11px] text-ink-muted uppercase tracking-wider m-0">
            SENESCHAL Estate Steward &copy; 2026 PT Tata Optima Property
          </p>
          <p className="font-mono-ui text-[11px] text-ink-muted m-0">
            API Reference · Collector REST API v2.0
          </p>
        </div>
      </footer>
    </div>
  );
}