import React, { useState } from 'react';
import { Copy, ChevronRight, Code, AlertCircle, CheckCircle } from 'lucide-react';

const methodBadgeClasses = {
  GET: 'wax-seal-normal',
  POST: 'wax-seal-critical',
  PUT: 'wax-seal-warning',
  DELETE: 'wax-seal-offline'
};

const methodColors = {
  GET: 'text-forest-moss',
  POST: 'text-cardinal',
  PUT: 'text-gold-dim',
  DELETE: 'text-ink-muted'
};

function CodeBlock({ code, language = 'json', label, onCopy, copyLabel }) {
  const [copied, setCopied] = useState(false);
  
  const handleCopy = () => {
    onCopy(code, copyLabel || label);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="relative group">
      <div className="flex items-center justify-between mb-1.5">
        <span className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider">{label}</span>
        <button
          onClick={handleCopy}
          className="opacity-0 group-hover:opacity-100 transition-opacity p-1.5 bg-parchment-dark border border-ink-muted/40 hover:bg-ink/5 rounded-sm focus-ring"
          aria-label={`Copy ${label}`}
        >
          {copied ? (
            <CheckCircle className="w-4 h-4 text-forest-moss" strokeWidth={2.5} />
          ) : (
            <Copy className="w-4 h-4 text-ink-muted hover:text-ink" strokeWidth={2} />
          )}
        </button>
      </div>
      <pre className="bg-ink/95 border border-ink-muted/30 rounded-sm p-4 overflow-x-auto max-h-64">
        <code className={`language-${language} font-mono-ui text-[11px] text-parchment`}>
          {code}
        </code>
      </pre>
    </div>
  );
}

function ParamTable({ params, title }) {
  if (!params || params.length === 0) return null;
  
  return (
    <div className="mb-4">
      <h5 className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider mb-2">{title}</h5>
      <div className="overflow-x-auto">
        <table className="w-full text-[11px] font-mono-ui border-collapse">
          <thead>
            <tr className="border-b border-ink-muted/20 text-ink-muted uppercase tracking-wider">
              <th className="pb-2 pl-4 text-left font-medium">Name</th>
              <th className="pb-2 pl-4 text-left font-medium">In</th>
              <th className="pb-2 pl-4 text-left font-medium">Required</th>
              <th className="pb-2 pl-4 text-left font-medium">Type</th>
              <th className="pb-2 pl-4 text-left font-medium">Description</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-ink-muted/15 text-ink">
            {params.map((p, i) => (
              <tr key={`${p.name}-${i}`} className="hover:bg-ink/3">
                <td className="py-2 pl-4 font-medium text-ink">{p.name}</td>
                <td className="py-2 pl-4 text-ink-muted">
                  <span className={`inline-block px-1.5 py-0.5 text-[9px] rounded-sm ${p.in === 'path' ? 'bg-gold-leaf/20 text-gold-dim' : p.in === 'query' ? 'bg-moss/20 text-moss' : 'bg-ink-muted/20 text-ink-muted'}`}>
                    {p.in}
                  </span>
                </td>
                <td className="py-2 pl-4">
                  {p.required ? (
                    <span className="inline-flex items-center gap-1 text-cardinal">
                      <span className="w-1.5 h-1.5 rounded-full bg-cardinal" />
                      Required
                    </span>
                  ) : (
                    <span className="text-ink-muted">Optional</span>
                  )}
                </td>
                <td className="py-2 pl-4 font-mono-ui text-[10px] text-ink-muted">{p.type + (p.enum ? ` (${p.enum.join(', ')})` : '') + (p.pattern ? ` / pattern: ${p.pattern}` : '') + (p.minimum !== undefined ? ` / min: ${p.minimum}` : '') + (p.maximum !== undefined ? ` / max: ${p.maximum}` : '') + (p.maxLength ? ` / maxLen: ${p.maxLength}` : '')}</td>
                <td className="py-2 pl-4 pr-4 text-ink-muted">{p.description}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function JsonSchema({ schema }) {
  if (!schema) return null;
  
  const renderSchema = (s, level = 0) => {
    if (!s) return null;
    const pad = '  '.repeat(level);
    
    if (s.type === 'object' && s.properties) {
      const entries = Object.entries(s.properties);
      if (entries.length === 0) return '{ }';
      return `{\n${entries.map(([key, val]) => 
        `${'  '.repeat(level + 1)}${key}: ${renderSchema(val, level + 1)}${s.required?.includes(key) ? '' : '?'},`
      ).join('\n')}\n${pad}}`;
    }
    if (s.type === 'array' && s.items) {
      return `[\n${'  '.repeat(level + 1)}${renderSchema(s.items, level + 1)}\n${pad}]`;
    }
    if (s.type === 'string' && s.enum) {
      return `string { ${s.enum.map(e => `"${e}"`).join(', ')} }`;
    }
    if (s.type === 'string' && s.format) {
      return `string (${s.format})`;
    }
    if (s.type === 'number') {
      let str = s.format === 'float' ? 'float' : 'integer';
      if (s.minimum !== undefined || s.maximum !== undefined) {
        str += ` (${s.minimum !== undefined ? `≥${s.minimum}` : ''}${s.minimum !== undefined && s.maximum !== undefined ? ', ' : ''}${s.maximum !== undefined ? `≤${s.maximum}` : ''})`;
      }
      return str;
    }
    return s.type || 'any';
  };
  
  return (
    <CodeBlock 
      code={renderSchema(schema)} 
      language="json" 
      label="Request Schema" 
      onCopy={() => {}}
    />
  );
}

export default function EndpointCard({ endpoint, environment, onTryIt, onCopy }) {
  const [showRequestSchema, setShowRequestSchema] = useState(false);

  const methodColorClass = methodColors[endpoint.method] || 'text-ink';
  const endpointId = endpoint.id;
  
  const requestExample = endpoint.requestBody?.example ? JSON.stringify(endpoint.requestBody.example, null, 2) : '';
  const curlExample = endpoint.curl || '';
  
  // Generate cURL with environment base URL
  const fullCurl = curlExample.replace('http://localhost:8081', environment.baseUrl);
  
  // Build path params array for table
  const pathParams = (endpoint.parameters || []).filter(p => p.in === 'path');
  const queryParams = (endpoint.parameters || []).filter(p => p.in === 'query');
  const hasRequestBody = endpoint.requestBody && endpoint.requestBody.required;

  return (
    <article 
      id={endpointId}
      data-endpoint-id={endpointId}
      className="ledger-card-elevated p-6 space-y-6"
      tabIndex={0}
      role="listitem"
      aria-labelledby={`${endpointId}-title`}
    >
      {/* Header */}
      <header className="flex flex-col md:flex-row md:items-start justify-between gap-4 pb-4 border-b border-ink-muted/20">
        <div className="flex flex-col md:flex-row md:items-center gap-3">
          <span 
            className={`${methodBadgeClasses[endpoint.method] || 'wax-seal-normal'} stamp text-[10px] font-mono-ui uppercase tracking-wider whitespace-nowrap`}
          >
            {endpoint.method}
          </span>
          <div>
            <h3 
              id={`${endpointId}-title`}
              className="font-display text-xl font-semibold text-ink m-0 font-mono-ui"
            >
              {endpoint.path}
            </h3>
            <p className="font-mono-ui text-[11px] text-ink-muted mt-1 m-0">{endpoint.summary}</p>
          </div>
        </div>
        
        <div className="flex flex-wrap items-center gap-2">
          <span className={`font-mono-ui text-[10px] px-2 py-0.5 rounded-sm ${methodColorClass}`}>
            {endpoint.tag}
          </span>
          <button
            onClick={() => onTryIt?.(endpoint)}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-parchment-dark border border-ink-muted/40 text-ink text-[10px] font-mono-ui uppercase tracking-wider hover:bg-ink/5 transition focus-ring rounded-sm"
          >
            <Code className="w-3.5 h-3.5" strokeWidth={2} />
            Try It
          </button>
        </div>
      </header>

      {/* Description */}
      <div className="text-[11px] text-ink-muted leading-relaxed">
        {endpoint.description}
      </div>

      {/* Auth Info */}
      {endpoint.auth && (
        <div className="bg-parchment-dark border border-ink-muted/30 p-3 rounded-sm">
          <div className="flex items-center gap-2 mb-2">
            <AlertCircle className="w-4 h-4 text-gold-dim" strokeWidth={2} />
            <span className="font-mono-ui text-[10px] uppercase tracking-wider text-gold-dim">Authentication</span>
          </div>
          <p className="font-mono-ui text-[11px] text-ink-muted mb-2">{endpoint.auth.description}</p>
          <div className="flex flex-wrap gap-1.5">
            {endpoint.auth.headers?.map((h, i) => (
              <span key={i} className="font-mono-ui text-[10px] bg-ink/50 px-2 py-0.5 rounded-sm text-ink-muted">
                {h}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Path Parameters */}
      {pathParams.length > 0 && (
        <ParamTable params={pathParams} title="Path Parameters" />
      )}

      {/* Query Parameters */}
      {queryParams.length > 0 && (
        <ParamTable params={queryParams} title="Query Parameters" />
      )}

      {/* Request Body */}
      {hasRequestBody && (
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <h5 className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider">Request Body</h5>
            <button
              onClick={() => setShowRequestSchema(!showRequestSchema)}
              className="flex items-center gap-1 px-2 py-1 text-[10px] font-mono-ui text-ink-muted hover:text-ink transition"
            >
              <ChevronRight className={`w-3 h-3 transition-transform ${showRequestSchema ? 'rotate-90' : ''}`} strokeWidth={2} />
              {showRequestSchema ? 'Hide Schema' : 'Show Schema'}
            </button>
          </div>
          
          {showRequestSchema && endpoint.requestBody.schema && (
            <JsonSchema schema={endpoint.requestBody.schema} />
          )}
          
          {requestExample && (
            <CodeBlock 
              code={requestExample} 
              language="json" 
              label="Request Example" 
              onCopy={onCopy}
              copyLabel="Request Example"
            />
          )}
        </div>
      )}

      {/* Responses */}
      <details className="group" defaultOpen>
        <summary className="flex items-center justify-between cursor-pointer py-2">
          <h5 className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider">Responses</h5>
          <ChevronRight className="w-4 h-4 transition-transform group-open:rotate-90" strokeWidth={2} />
        </summary>
        
        <div className="space-y-4 mt-2">
          {Object.entries(endpoint.responses).map(([code, resp]) => (
            <div key={code} className="space-y-2">
              <div className="flex items-center gap-2">
                <span className={`font-mono-ui text-[11px] font-bold px-2 py-0.5 rounded-sm ${code.startsWith('2') ? 'bg-forest-moss/20 text-forest-moss' : code.startsWith('4') ? 'bg-cardinal/20 text-cardinal' : code.startsWith('5') ? 'bg-cardinal/20 text-cardinal' : 'bg-ink-muted/20 text-ink-muted'}`}>
                  {code}
                </span>
                <span className="font-mono-ui text-[11px] text-ink-muted">{resp.description}</span>
              </div>
              {resp.example && (
                <CodeBlock 
                  code={JSON.stringify(resp.example, null, 2)} 
                  language="json" 
                  label={`Response ${code}`} 
                  onCopy={onCopy}
                  copyLabel={`Response ${code}`}
                />
              )}
            </div>
          ))}
        </div>
      </details>

      {/* cURL Example */}
      {curlExample && (
        <details className="group">
          <summary className="flex items-center justify-between cursor-pointer py-2">
            <h5 className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider">cURL Example</h5>
            <ChevronRight className="w-4 h-4 transition-transform group-open:rotate-90" strokeWidth={2} />
          </summary>
          <div className="mt-2">
            <CodeBlock 
              code={fullCurl} 
              language="bash" 
              label="cURL" 
              onCopy={onCopy}
              copyLabel="cURL Command"
            />
          </div>
        </details>
      )}
    </article>
  );
}