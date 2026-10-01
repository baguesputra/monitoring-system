import React, { useState, useEffect } from 'react';
import { Server, Laptop, HardDrive, Cpu, MemoryStick, Search, RefreshCw, ArrowRight } from 'lucide-react';

export default function Overview({ onSelectDevice }) {
  const [servers, setServers] = useState([]);
  const [statuses, setStatuses] = useState({});
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [typeFilter, setTypeFilter] = useState('all');

  const fetchData = async () => {
    try {
      const res = await fetch('/api/servers');
      const data = await res.json();
      setServers(data || []);

      const statusMap = {};
      for (const s of (data || [])) {
        try {
          const sRes = await fetch(`/api/servers/${s.server_id}/status`);
          if (sRes.ok) {
            const sData = await sRes.json();
            statusMap[s.server_id] = sData;
          }
        } catch (e) {
          // ignore individual fetch errors
        }
      }
      setStatuses(statusMap);
} catch (_err) {
        console.error('Failed to fetch servers:', _err);
      } finally {
        setLoading(false);
      }
    };

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 10000);
    return () => clearInterval(interval);
  }, []);

  const filteredServers = servers.filter(s => {
    const q = search.toLowerCase();
    const matchSearch = s.server_id.toLowerCase().includes(q) ||
      (s.hostname && s.hostname.toLowerCase().includes(q)) ||
      (s.ip_address && s.ip_address.includes(q)) ||
      (s.owner_name && s.owner_name.toLowerCase().includes(q));
    const matchType = typeFilter === 'all' || (s.device_type && s.device_type.toLowerCase() === typeFilter);
    return matchSearch && matchType;
  });

  const getSealClass = (st) => {
    if (!st || !st.last_seen_at) return 'wax-seal-offline';
    const cpu = st.cpu_percent || 0;
    const ram = st.ram_percent || 0;
    const disk = st.disk_percent || 0;
    if (cpu >= 90 || ram >= 95 || disk >= 95) return 'wax-seal-critical';
    if (cpu >= 75 || ram >= 85 || disk >= 85) return 'wax-seal-warning';
    return 'wax-seal-normal';
  };

  const getSealLabel = (st) => {
    if (!st) return 'OFFLINE';
    const cpu = st.cpu_percent || 0;
    const ram = st.ram_percent || 0;
    const disk = st.disk_percent || 0;
    if (cpu >= 90 || ram >= 95 || disk >= 95) return 'CRITICAL';
    if (cpu >= 75 || ram >= 85 || disk >= 85) return 'ATTENTION';
    return 'NOMINAL';
  };

  return (
    <div className="space-y-8">
      {/* Ledger Header */}
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-4 pb-4 border-b border-ink-muted/30">
        <div>
          <h2 className="font-display text-2xl font-semibold text-ink m-0">Daily Ledger</h2>
          <p className="font-mono-ui text-[11px] text-ink-muted uppercase tracking-wider mt-1 m-0">
            Estate Infrastructure Register
          </p>
        </div>
        <button onClick={fetchData} className="flex items-center gap-2 px-4 py-2 bg-parchment-dark border border-ink-muted/40 text-ink text-[11px] font-mono-ui uppercase tracking-wider hover:bg-ink/5 transition focus-ring rounded-sm">
          <RefreshCw className="w-4 h-4" strokeWidth={2} />
          Refresh Ledger
        </button>
      </div>

      {/* Summary Registers */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="ledger-card p-5">
          <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Total Holdings</p>
          <p className="font-display text-3xl font-semibold text-ink mt-1 m-0">{servers.length}</p>
          <p className="font-mono-ui text-[11px] text-ink-muted mt-2 m-0">Managed Estates</p>
        </div>
        <div className="ledger-card p-5">
          <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Nominal</p>
          <p className="font-display text-3xl font-semibold text-moss mt-1 m-0">
            {Object.values(statuses).filter(st => getSealClass(st) === 'wax-seal-normal').length}
          </p>
          <p className="font-mono-ui text-[11px] text-ink-muted mt-2 m-0">Operating within bounds</p>
        </div>
        <div className="ledger-card p-5">
          <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Attention Required</p>
          <p className="font-display text-3xl font-semibold text-gold-dim mt-1 m-0">
            {Object.values(statuses).filter(st => getSealClass(st) === 'wax-seal-warning').length}
          </p>
          <p className="font-mono-ui text-[11px] text-ink-muted mt-2 m-0">Elevated metrics detected</p>
        </div>
        <div className="ledger-card p-5">
          <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Critical</p>
          <p className="font-display text-3xl font-semibold text-cardinal mt-1 m-0">
            {Object.values(statuses).filter(st => getSealClass(st) === 'wax-seal-critical').length}
          </p>
          <p className="font-mono-ui text-[11px] text-ink-muted mt-2 m-0">Immediate steward action</p>
        </div>
      </div>

      {/* Search & Filter Bar */}
      <div className="flex flex-col md:flex-row gap-4 items-start justify-between bg-parchment-dark border border-ink-muted/30 p-4">
        <div className="relative w-full md:w-96">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-ink-muted" strokeWidth={2} />
          <input 
            type="text"
            placeholder="Search holdings by name, hostname, or IP..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full bg-parchment border border-ink-muted/40 pl-10 pr-4 py-2.5 text-sm text-ink focus:outline-none focus:border-gold transition focus-ring rounded-sm"
          />
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {['all', 'server', 'pc', 'laptop'].map((type) => (
            <button
              key={type}
              onClick={() => setTypeFilter(type)}
              className={`px-3 py-1.5 text-[10px] font-mono-ui uppercase tracking-wider transition focus-ring rounded-sm ${
                typeFilter === type 
                  ? 'bg-ink text-parchment' 
                  : 'text-ink-muted bg-parchment border border-ink-muted/30 hover:text-ink hover:border-ink/50'
              }`}
            >
              {type === 'all' ? 'All Holdings' : type.charAt(0).toUpperCase() + type.slice(1)}
            </button>
          ))}
        </div>
      </div>

      {/* Holdings Grid */}
      {loading ? (
        <div className="text-center py-16 text-ink-muted font-mono-ui">Recording ledger entries...</div>
      ) : filteredServers.length === 0 ? (
        <div className="text-center py-16 text-ink-muted bg-parchment-dark border border-ink-muted/30">
          <Server className="w-14 h-14 mx-auto mb-4 opacity-30 text-ink-muted" strokeWidth={1.5} />
          <p className="font-display text-lg text-ink m-0">No holdings recorded</p>
          <p className="font-mono-ui text-[11px] mt-1 m-0">Ensure agents are reporting to the steward.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
          {filteredServers.map((s) => {
            const st = statuses[s.server_id] || {};
            const cpu = st.cpu_percent || 0;
            const ram = st.ram_percent || 0;
            const disk = st.disk_percent || 0;
            const sealClass = getSealClass(st);
            const sealLabel = getSealLabel(st);

            return (
              <article 
                key={s.server_id}
                onClick={() => onSelectDevice(s.server_id)}
                className="ledger-card-elevated p-5 cursor-pointer transition hover:border-gold/50 group"
                tabIndex={0}
                onKeyDown={(e) => e.key === 'Enter' && onSelectDevice(s.server_id)}
                role="button"
                aria-label={`View details for ${s.server_id}`}
              >
                <div className="flex items-start justify-between mb-4">
                  <div className="flex items-center gap-3">
                    <div className={`p-2 rounded-sm ${s.device_type === 'laptop' ? 'bg-moss/10 text-moss' : 'bg-gold-leaf/10 text-gold-dim'}`}>
                      {s.device_type === 'laptop' ? <Laptop className="w-5 h-5" strokeWidth={1.5} /> : <Server className="w-5 h-5" strokeWidth={1.5} />}
                    </div>
                    <h3 className="font-display text-lg font-semibold text-ink group-hover:text-gold-dim transition">{s.server_id}</h3>
                  </div>
                  <span className={`wax-seal ${sealClass} stamp`}>{sealLabel}</span>
                </div>

                <p className="font-mono-ui text-[11px] text-ink-muted mb-4">{s.hostname || 'Unknown host'}</p>

                <div className="space-y-3 mb-4">
                  <div>
                    <div className="flex justify-between text-[11px] mb-1 font-mono-ui">
                      <span className="text-ink-muted flex items-center gap-1.5"><Cpu className="w-3 h-3" strokeWidth={2} /> CPU</span>
                      <span className={cpu > 85 ? 'text-cardinal font-bold' : cpu > 70 ? 'text-gold-dim font-bold' : 'text-ink'}>
                        {cpu.toFixed(1)}%
                      </span>
                    </div>
                    <div className="w-full bg-parchment-dark h-1.5 rounded-sm overflow-hidden border border-ink-muted/20">
                      <div 
                        className={`h-full rounded-sm transition-all duration-500 ${cpu > 85 ? 'bg-cardinal' : cpu > 70 ? 'bg-gold-dim' : 'bg-moss'}`}
                        style={{ width: `${Math.min(cpu, 100)}%` }}
                      ></div>
                    </div>
                  </div>

                  <div>
                    <div className="flex justify-between text-[11px] mb-1 font-mono-ui">
                      <span className="text-ink-muted flex items-center gap-1.5"><MemoryStick className="w-3 h-3" strokeWidth={2} /> RAM</span>
                      <span className={ram > 85 ? 'text-cardinal font-bold' : ram > 70 ? 'text-gold-dim font-bold' : 'text-ink'}>
                        {ram.toFixed(1)}%
                      </span>
                    </div>
                    <div className="w-full bg-parchment-dark h-1.5 rounded-sm overflow-hidden border border-ink-muted/20">
                      <div 
                        className={`h-full rounded-sm transition-all duration-500 ${ram > 85 ? 'bg-cardinal' : ram > 70 ? 'bg-gold-dim' : 'bg-blue-600'}`}
                        style={{ width: `${Math.min(ram, 100)}%` }}
                      ></div>
                    </div>
                  </div>

                  <div>
                    <div className="flex justify-between text-[11px] mb-1 font-mono-ui">
                      <span className="text-ink-muted flex items-center gap-1.5"><HardDrive className="w-3 h-3" strokeWidth={2} /> Disk</span>
                      <span className={disk > 85 ? 'text-cardinal font-bold' : disk > 70 ? 'text-gold-dim font-bold' : 'text-ink'}>
                        {disk.toFixed(1)}%
                      </span>
                    </div>
                    <div className="w-full bg-parchment-dark h-1.5 rounded-sm overflow-hidden border border-ink-muted/20">
                      <div 
                        className={`h-full rounded-sm transition-all duration-500 ${disk > 85 ? 'bg-cardinal' : disk > 70 ? 'bg-gold-dim' : 'bg-emerald-600'}`}
                        style={{ width: `${Math.min(disk, 100)}%` }}
                      ></div>
                    </div>
                  </div>
                </div>

                <div className="pt-3 border-t border-ink-muted/20 flex items-center justify-between text-[11px] font-mono-ui text-ink-muted">
                  <span>IP: {s.ip_address || st.ip_address || '—'}</span>
                  <span className="text-gold-dim group-hover:translate-x-0.5 transition flex items-center gap-1">
                    Inspect <ArrowRight className="w-3 h-3" strokeWidth={2.5} />
                  </span>
                </div>
              </article>
            );
          })}
        </div>
      )}
    </div>
  );
}