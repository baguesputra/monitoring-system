import React, { useState, useEffect, useRef } from 'react';
import { X, Send, Copy, Loader2, ChevronRight } from 'lucide-react';

const methodBadgeClasses = {
  GET: 'wax-seal-normal',
  POST: 'wax-seal-critical',
  PUT: 'wax-seal-warning',
  DELETE: 'wax-seal-offline'
};

export default function TryItPanel({ endpoint, environment, onClose }) {
  const [pathParams, setPathParams] = useState({});
  const [queryParams, setQueryParams] = useState({});
  const [requestBody, setRequestBody] = useState('');
  const [headers, setHeaders] = useState({ 'Content-Type': 'application/json' });
  const [response, setResponse] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [responseTab, setResponseTab] = useState('body');
  const bodyEditorRef = useRef(null);

  // Initialize form values from endpoint
  useEffect(() => {
    if (endpoint) {
      // Initialize path params
      const pathParamsInit = {};
      (endpoint.parameters || []).filter(p => p.in === 'path').forEach(p => {
        pathParamsInit[p.name] = p.example || '';
      });
      setPathParams(pathParamsInit);

      // Initialize query params
      const queryParamsInit = {};
      (endpoint.parameters || []).filter(p => p.in === 'query').forEach(p => {
        queryParamsInit[p.name] = p.default || '';
      });
      setQueryParams(queryParamsInit);

      // Initialize request body
      if (endpoint.requestBody?.example) {
        setRequestBody(JSON.stringify(endpoint.requestBody.example, null, 2));
      } else if (endpoint.requestBody?.schema) {
        // Generate minimal example from schema
        const generateExample = (schema) => {
          if (schema.type === 'object' && schema.properties) {
            const obj = {};
            Object.entries(schema.properties).forEach(([key, val]) => {
              if (schema.required?.includes(key) || Math.random() > 0.5) {
                obj[key] = generateExampleValue(val);
              }
            });
            return obj;
          }
          if (schema.type === 'array' && schema.items) {
            return [generateExampleValue(schema.items)];
          }
          return generateExampleValue(schema);
        };
        
        const generateExampleValue = (s) => {
          if (s.type === 'string') {
            if (s.enum) return s.enum[0];
            if (s.format === 'date-time') return new Date().toISOString();
            if (s.format === 'ipv4') return '192.168.1.100';
            return s.example || 'string';
          }
          if (s.type === 'number') {
            if (s.format === 'float') return s.example || 45.2;
            return s.example || 42;
          }
          if (s.type === 'integer') return s.example || 8;
          if (s.type === 'boolean') return true;
          if (s.type === 'array') return s.items ? [generateExampleValue(s.items)] : [];
          if (s.type === 'object') return {};
          return null;
        };
        
        const example = generateExample(endpoint.requestBody.schema);
        setRequestBody(JSON.stringify(example, null, 2));
      } else {
        setRequestBody('');
      }
    }
  }, [endpoint]);

  const buildUrl = () => {
    let url = `${environment.baseUrl}${endpoint.path}`;
    
    // Replace path parameters
    Object.entries(pathParams).forEach(([key, value]) => {
      url = url.replace(`{${key}}`, encodeURIComponent(value));
    });
    
    // Add query parameters
    const queryEntries = Object.entries(queryParams).filter(([, v]) => v !== '');
    if (queryEntries.length > 0) {
      const queryString = queryEntries.map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v)}`).join('&');
      url += `?${queryString}`;
    }
    
    return url;
  };

  const generateCurl = () => {
    const url = buildUrl();
    const headerStr = Object.entries(headers)
      .filter(([, v]) => v)
      .map(([k, v]) => `-H "${k}: ${v}"`)
      .join(' \\\n  ');
    
    let curl = `curl -X ${endpoint.method} "${url}"`;
    
    if (headerStr) {
      curl += ` \\\n  ${headerStr}`;
    }
    
    if (endpoint.method !== 'GET' && requestBody.trim()) {
      curl += ` \\\n  -d '${requestBody.replace(/'/g, "'\\''")}'`;
    }
    
    return curl;
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setResponse(null);

    try {
      const url = buildUrl();
      const options = {
        method: endpoint.method,
        headers: headers,
      };

      if (endpoint.method !== 'GET' && requestBody.trim()) {
        options.body = requestBody;
      }

      // Add credentials for session cookie
      options.credentials = 'include';

      const res = await fetch(url, options);
      const responseText = await res.text();
      let responseJson;
      
      try {
        responseJson = JSON.parse(responseText);
      } catch {
        responseJson = responseText;
      }

      setResponse({
        status: res.status,
        statusText: res.statusText,
        headers: Object.fromEntries(res.headers.entries()),
        body: responseJson,
        raw: responseText
      });
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const copyToClipboard = async (text) => {
    try {
      await navigator.clipboard.writeText(text);
      // Could show toast
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  const formatJson = (obj) => {
    try {
      return JSON.stringify(obj, null, 2);
    } catch {
      return String(obj);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-ink/60 backdrop-blur-sm"
        onClick={onClose}
        aria-hidden="true"
      />
      
      {/* Modal */}
      <div className="relative w-full max-w-5xl max-h-[90vh] bg-parchment border border-ink-muted/30 rounded-sm shadow-seal overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-ink-muted/30">
          <div className="flex items-center gap-3">
            <span className={`${methodBadgeClasses[endpoint.method] || 'wax-seal-normal'} text-[10px] font-mono-ui uppercase tracking-wider`}>
              {endpoint.method}
            </span>
            <div>
              <h3 className="font-display text-lg font-semibold text-ink m-0 font-mono-ui">{endpoint.path}</h3>
              <p className="font-mono-ui text-[10px] text-ink-muted m-0">{endpoint.summary}</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-2 bg-parchment-dark border border-ink-muted/40 text-ink hover:bg-ink/5 transition focus-ring rounded-sm"
            aria-label="Close Try It panel"
          >
            <X className="w-5 h-5" strokeWidth={2} />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-4 space-y-6">
          {/* Request Configuration */}
          <div className="space-y-4">
            {/* URL Preview */}
            <div className="bg-parchment-dark border border-ink-muted/30 p-3 rounded-sm">
              <div className="flex items-center justify-between mb-2">
                <span className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider">Request URL</span>
                <button
                  onClick={() => copyToClipboard(buildUrl(), 'Request URL')}
                  className="text-[10px] text-gold-dim hover:text-gold-leaf font-mono-ui"
                >
                  Copy URL
                </button>
              </div>
              <code className="font-mono-ui text-[11px] text-ink break-all block bg-ink/30 px-3 py-2 rounded-sm">
                {buildUrl()}
              </code>
            </div>

            {/* Path Parameters */}
            {Object.keys(pathParams).length > 0 && (
              <div className="space-y-2">
                <h5 className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider">Path Parameters</h5>
                <div className="grid gap-2 sm:grid-cols-2">
                  {Object.entries(pathParams).map(([key, value]) => (
                    <div key={key} className="bg-parchment-dark border border-ink-muted/30 p-3 rounded-sm">
                      <label className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider block mb-1">{key} *</label>
                      <input
                        type="text"
                        value={value}
                        onChange={(e) => setPathParams({ ...pathParams, [key]: e.target.value })}
                        className="w-full bg-parchment border border-ink-muted/40 px-3 py-2 text-sm text-ink focus:outline-none focus:border-gold transition focus-ring rounded-sm font-mono-ui"
                        placeholder={`Enter ${key}`}
                      />
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Query Parameters */}
            {Object.keys(queryParams).length > 0 && (
              <div className="space-y-2">
                <h5 className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider">Query Parameters</h5>
                <div className="grid gap-2 sm:grid-cols-2">
                  {Object.entries(queryParams).map(([key, value]) => (
                    <div key={key} className="bg-parchment-dark border border-ink-muted/30 p-3 rounded-sm">
                      <label className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider block mb-1">{key}</label>
                      <input
                        type="text"
                        value={value}
                        onChange={(e) => setQueryParams({ ...queryParams, [key]: e.target.value })}
                        className="w-full bg-parchment border border-ink-muted/40 px-3 py-2 text-sm text-ink focus:outline-none focus:border-gold transition focus-ring rounded-sm font-mono-ui"
                        placeholder={endpoint.parameters?.find(p => p.name === key)?.default || ''}
                      />
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Headers */}
            <div className="space-y-2">
              <h5 className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider">Headers</h5>
              <div className="bg-parchment-dark border border-ink-muted/30 p-3 rounded-sm">
                <div className="space-y-2">
                  {Object.entries(headers).map(([key, value]) => (
                    <div key={key} className="flex gap-2">
                      <input
                        type="text"
                        value={key}
                        onChange={(e) => setHeaders({ ...headers, [e.target.value]: headers[key], ...(e.target.value !== key ? { [key]: undefined } : {}) })}
                        className="flex-1 bg-parchment border border-ink-muted/40 px-3 py-2 text-sm text-ink focus:outline-none focus:border-gold transition focus-ring rounded-sm font-mono-ui text-[11px]"
                      />
                      <input
                        type="text"
                        value={value}
                        onChange={(e) => setHeaders({ ...headers, [key]: e.target.value })}
                        className="flex-1 bg-parchment border border-ink-muted/40 px-3 py-2 text-sm text-ink focus:outline-none focus:border-gold transition focus-ring rounded-sm font-mono-ui text-[11px]"
                      />
                      <button
                        onClick={() => {
                          const newHeaders = { ...headers };
                          delete newHeaders[key];
                          setHeaders(newHeaders);
                        }}
                        className="p-1.5 bg-ink/10 hover:bg-ink/20 text-ink-muted hover:text-cardinal rounded-sm transition"
                        aria-label="Remove header"
                      >
                        <X className="w-4 h-4" strokeWidth={2} />
                      </button>
                    </div>
                  ))}
                  <button
                    onClick={() => setHeaders({ ...headers, 'X-Custom-Header': '' })}
                    className="text-[10px] text-gold-dim hover:text-gold-leaf font-mono-ui flex items-center gap-1"
                  >
                    <ChevronRight className="w-3 h-3" strokeWidth={2} />
                    Add Header
                  </button>
                </div>
              </div>
            </div>

            {/* Request Body */}
            {['POST', 'PUT', 'PATCH'].includes(endpoint.method) && (
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <h5 className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider">Request Body</h5>
                  <button
                    onClick={() => copyToClipboard(requestBody, 'Request Body')}
                    className="text-[10px] text-gold-dim hover:text-gold-leaf font-mono-ui"
                  >
                    Copy
                  </button>
                </div>

                <div className="relative">
                  <textarea
                    ref={bodyEditorRef}
                    value={requestBody}
                    onChange={(e) => setRequestBody(e.target.value)}
                    className="w-full bg-ink/20 border border-ink-muted/30 px-3 py-3 text-sm text-parchment focus:outline-none focus:border-gold transition focus-ring rounded-sm font-mono-ui text-[11px] resize-none"
                    rows={10}
                    spellCheck={false}
                    placeholder='{"key": "value"}'
                  />
                </div>
              </div>
            )}

            {/* Send Button */}
            <div className="flex items-center gap-3 pt-2 border-t border-ink-muted/20">
              <button
                onClick={handleSubmit}
                disabled={loading}
                className="flex items-center gap-2 px-6 py-2.5 bg-ink text-parchment text-[11px] font-mono-ui uppercase tracking-wider hover:bg-ink/90 transition focus-ring rounded-sm disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loading ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin" strokeWidth={2} />
                    Sending...
                  </>
                ) : (
                  <>
                    <Send className="w-4 h-4" strokeWidth={2} />
                    Send Request
                  </>
                )}
              </button>
              
              <button
                onClick={() => copyToClipboard(generateCurl(), 'cURL Command')}
                className="flex items-center gap-1.5 px-4 py-2 bg-parchment-dark border border-ink-muted/40 text-ink text-[10px] font-mono-ui uppercase tracking-wider hover:bg-ink/5 transition focus-ring rounded-sm"
              >
                <Copy className="w-4 h-4" strokeWidth={2} />
                Copy cURL
              </button>
            </div>
          </div>

          {/* Response */}
          {response && (
            <div className="space-y-4 pt-4 border-t border-ink-muted/20">
              <div className="flex items-center justify-between">
                <h4 className="font-display text-lg font-semibold text-ink m-0">Response</h4>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => copyToClipboard(formatJson(response.body), 'Response Body')}
                    className="text-[10px] text-gold-dim hover:text-gold-leaf font-mono-ui"
                  >
                    Copy Response
                  </button>
                  <button
                    onClick={() => copyToClipboard(generateCurl(), 'cURL Command')}
                    className="text-[10px] text-gold-dim hover:text-gold-leaf font-mono-ui"
                  >
                    Copy cURL
                  </button>
                </div>
              </div>
              
              {/* Status Badge */}
              <div className="flex items-center gap-3 mb-4">
                <span className={`font-mono-ui text-[11px] font-bold px-3 py-1 rounded-sm ${response.status >= 200 && response.status < 300 ? 'bg-forest-moss/20 text-forest-moss' : response.status >= 400 && response.status < 500 ? 'bg-cardinal/20 text-cardinal' : 'bg-gold-dim/20 text-gold-dim'}`}>
                  {response.status} {response.statusText}
                </span>
                <span className="font-mono-ui text-[10px] text-ink-muted">Completed</span>
              </div>

              {/* Response Tabs */}
              <div className="flex gap-1 mb-2" role="tablist">
                {['body', 'headers', 'raw'].map(tab => (
                  <button
                    key={tab}
                    role="tab"
                    aria-selected={responseTab === tab}
                    onClick={() => setResponseTab(tab)}
                    className={`px-3 py-1.5 text-[10px] font-mono-ui uppercase tracking-wider rounded-sm transition ${responseTab === tab ? 'bg-ink text-parchment' : 'text-ink-muted hover:text-ink hover:bg-ink/5'}`}
                  >
                    {tab.charAt(0).toUpperCase() + tab.slice(1)}
                  </button>
                ))}
              </div>

              {/* Response Content */}
              <div className="bg-ink/20 border border-ink-muted/30 rounded-sm overflow-hidden">
                <pre className="p-4 max-h-96 overflow-auto">
                  <code className={`font-mono-ui text-[11px] ${responseTab === 'raw' ? 'text-slate-300' : 'text-parchment'}`}>
                    {responseTab === 'body' ? formatJson(response.body) : responseTab === 'headers' ? formatJson(response.headers) : response.raw}
                  </code>
                </pre>
              </div>
            </div>
          )}

          {error && (
            <div className="bg-cardinal/10 border border-cardinal/30 p-4 rounded-sm text-cardinal text-[11px] font-mono-ui">
              Error: {error}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}