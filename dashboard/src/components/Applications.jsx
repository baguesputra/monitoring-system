import React, { useState, useEffect } from 'react';
import { Package, Search, Server } from 'lucide-react';

export default function Applications() {
  const [apps, setApps] = useState([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');

  useEffect(() => {
    fetchApps();
  }, []);

  const fetchApps = async () => {
    try {
      const res = await fetch('/api/servers');
      const servers = await res.json();
      
      const allApps = [];
      for (const s of (servers || [])) {
        try {
          const appRes = await fetch(`/api/servers/${s.server_id}/applications`);
          if (appRes.ok) {
            const appData = await appRes.json();
            for (const app of (appData || [])) {
              allApps.push({
                ...app,
                server_id: s.server_id
              });
            }
          }
        } catch {
          // ignore individual fetch errors
        }
      }
      setApps(allApps);
    } catch (err) {
      console.error('Failed to fetch applications:', err);
    } finally {
      setLoading(false);
    }
  };

  const filteredApps = apps.filter(app => 
    (app.app_name && app.app_name.toLowerCase().includes(search.toLowerCase())) ||
    (app.publisher && app.publisher.toLowerCase().includes(search.toLowerCase())) ||
    (app.server_id && app.server_id.toLowerCase().includes(search.toLowerCase()))
  );

  return (
    <div className="space-y-8">
      {/* Ledger Header */}
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-4 pb-4 border-b border-ink-muted/30">
        <div>
          <h2 className="font-display text-2xl font-semibold text-ink m-0">Provisions Register</h2>
          <p className="font-mono-ui text-[11px] text-ink-muted uppercase tracking-wider mt-1 m-0">
            {apps.length} provisions catalogued across {new Set(apps.map(a => a.server_id)).size} holdings
          </p>
        </div>
      </div>

      {/* Search Bar */}
      <div className="bg-parchment-dark border border-ink-muted/30 p-4">
        <div className="relative w-full md:w-96">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-ink-muted" strokeWidth={2} />
          <input 
            type="text"
            placeholder="Search provisions by name, publisher, or holding..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full bg-parchment border border-ink-muted/40 pl-10 pr-4 py-2.5 text-sm text-ink focus:outline-none focus:border-gold transition focus-ring rounded-sm"
          />
        </div>
      </div>

      {/* Provisions Table */}
      {loading ? (
        <div className="text-center py-16 text-ink-muted font-mono-ui">Consulting the provision ledgers...</div>
      ) : filteredApps.length === 0 ? (
        <div className="text-center py-16 text-ink-muted bg-parchment-dark border border-ink-muted/30">
          <Package className="w-14 h-14 mx-auto mb-4 opacity-30 text-ink-muted" strokeWidth={1.5} />
          <p className="font-display text-lg text-ink m-0">No provisions recorded</p>
          <p className="font-mono-ui text-[11px] mt-1 m-0">Applications are reported from Windows Registry by agents.</p>
        </div>
      ) : (
        <div className="ledger-card-elevated overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse font-mono-ui text-[11px]">
              <thead>
                <tr className="bg-ink/5 border-b border-ink-muted/30 text-ink-muted uppercase tracking-wider">
                  <th className="p-4 text-left font-medium">Holding</th>
                  <th className="p-4 text-left font-medium">Provision</th>
                  <th className="p-4 text-left font-medium">Version</th>
                  <th className="p-4 text-left font-medium">Purveyor</th>
                  <th className="p-4 text-left font-medium">Acquired</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-ink-muted/20 text-ink">
                {filteredApps.map((app, idx) => (
                  <tr key={idx} className="hover:bg-ink/3 transition">
                    <td className="p-4">
                      <div className="flex items-center gap-2">
                        <Server className="w-3.5 h-3.5 text-gold-dim" strokeWidth={1.5} />
                        <span className="font-mono-ui text-sm text-ink font-medium">{app.server_id}</span>
                      </div>
                    </td>
                    <td className="p-4 font-medium text-ink">{app.app_name}</td>
                    <td className="p-4 text-[11px] text-ink-muted">{app.app_version || '—'}</td>
                    <td className="p-4 text-[11px] text-ink-muted max-w-xs truncate">{app.publisher || 'Unknown'}</td>
                    <td className="p-4 text-[11px] text-ink-muted">{app.install_date || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}