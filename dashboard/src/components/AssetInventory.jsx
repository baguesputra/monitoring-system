import React, { useState, useEffect } from 'react';
import { HardDrive, BatteryCharging, Download, Search, Server, Laptop, AlertCircle, CheckCircle } from 'lucide-react';
import { formatDeviceModel } from '../utils/formatters';

export default function AssetInventory() {
  const [assets, setAssets] = useState([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');

  useEffect(() => {
    fetchAssets();
  }, []);

  const fetchAssets = async () => {
    try {
      const res = await fetch('/api/servers');
      const servers = await res.json();
      
      const assetList = [];
      for (const s of (servers || [])) {
        try {
          const aRes = await fetch(`/api/servers/${s.server_id}/asset-info`);
          if (aRes.ok) {
            const aData = await aRes.json();
            assetList.push({
              ...aData,
              device_type: s.device_type,
              owner_name: s.owner_name,
              department: s.department,
            });
          } else {
            assetList.push({ server_id: s.server_id, ip_address: s.ip_address, device_type: s.device_type, owner_name: s.owner_name, department: s.department });
          }
        } catch {
          assetList.push({ server_id: s.server_id, ip_address: s.ip_address, device_type: s.device_type });
        }
      }
      setAssets(assetList);
    } catch (err) {
      console.error('Failed to fetch asset inventory:', err);
    } finally {
      setLoading(false);
    }
  };

  const exportCSV = () => {
    const headers = ['Server ID', 'Device Type', 'Model', 'BIOS Date', 'Owner', 'Department', 'OS Version', 'IP Address', 'Processor', 'GPU', 'RAM (GB)', 'RAM Spec', 'Disk Type', 'Disk Size (GB)', 'Disk Health %', 'Disk Health Status', 'Battery %', 'Serial Number'];
    const rows = assets.map(a => [
      a.server_id,
      a.device_type || '',
      `"${formatDeviceModel(a.device_manufacturer, a.device_model)}"`,
      a.bios_release_date || '',
      `"${a.owner_name || ''}"`,
      `"${a.department || ''}"`,
      `"${a.os_version || ''}"`,
      a.ip_address || '',
      `"${a.processor_model || ''} (${a.processor_cores || 0} Cores)"`,
      `"${a.gpu_model || ''}"`,
      a.ram_total_gb || 0,
      `"${a.ram_ddr_type || ''} ${a.ram_speed_mhz ? a.ram_speed_mhz + 'MHz' : ''} ${a.ram_form_factor || ''} (${a.ram_slots_used || 0}/${a.ram_slots_total || 0})"`,
      a.disk_type || '',
      a.disk_size_gb || 0,
      a.disk_health_percent !== null && a.disk_health_percent !== undefined ? a.disk_health_percent : 'N/A',
      a.disk_health_status || 'UNKNOWN',
      a.battery_health_percent !== null && a.battery_health_percent !== undefined ? Math.round(a.battery_health_percent) : 'N/A',
      a.serial_number || ''
    ]);

    const csvContent = "data:text/csv;charset=utf-8," + [headers.join(','), ...rows.map(e => e.join(','))].join('\n');
    const encodedUri = encodeURI(csvContent);
    const link = document.createElement('a');
    link.setAttribute('href', encodedUri);
    link.setAttribute('download', `estate-inventory_${new Date().toISOString().slice(0,10)}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

const filteredAssets = assets.filter(a => 
    (a.server_id && a.server_id.toLowerCase().includes(search.toLowerCase())) ||
    (a.ip_address && a.ip_address.includes(search)) ||
    (a.serial_number && a.serial_number.toLowerCase().includes(search.toLowerCase())) ||
    (a.device_model && a.device_model.toLowerCase().includes(search.toLowerCase())) ||
    (a.device_manufacturer && a.device_manufacturer.toLowerCase().includes(search.toLowerCase())) ||
    (a.owner_name && a.owner_name.toLowerCase().includes(search.toLowerCase())) ||
    (a.processor_model && a.processor_model.toLowerCase().includes(search.toLowerCase())) ||
    (a.os_version && a.os_version.toLowerCase().includes(search.toLowerCase()))
);

  const getDiskHealthClass = (percent) => {
    if (percent === null || percent === undefined) return 'bg-ink-muted/10 text-ink-muted border-ink-muted/30';
    if (percent < 50) return 'bg-cardinal/10 text-cardinal border-cardinal/30';
    if (percent < 80) return 'bg-gold-dim/10 text-gold-dim border-gold-dim/30';
    return 'bg-moss/10 text-moss border-moss/30';
  };

  const getDiskHealthLabel = (percent) => {
    if (percent === null || percent === undefined) return 'N/A';
    if (percent < 50) return `${percent}% — DEGRADED`;
    if (percent < 80) return `${percent}% — WEARING`;
    return `${percent}% — PRISTINE`;
  };

  const getSmartBadge = (status) => {
    if (status === 'WARNING' || status === 'FAIL') {
      return (
        <span className="flex items-center gap-1 px-2 py-1 bg-cardinal/10 text-cardinal border border-cardinal/30 text-[10px] font-mono-ui uppercase tracking-wider rounded-sm">
          <AlertCircle className="w-3 h-3" strokeWidth={2.5} />
          SMART: {status}
        </span>
      );
    }
    return (
      <span className="flex items-center gap-1 px-2 py-1 bg-moss/10 text-moss border border-moss/30 text-[10px] font-mono-ui uppercase tracking-wider rounded-sm">
        <CheckCircle className="w-3 h-3" strokeWidth={2.5} />
        SMART: {status || 'OK'}
      </span>
    );
  };

  return (
    <div className="space-y-8">
      {/* Ledger Header */}
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-4 pb-4 border-b border-ink-muted/30">
        <div>
          <h2 className="font-display text-2xl font-semibold text-ink m-0">Estate Inventory</h2>
          <p className="font-mono-ui text-[11px] text-ink-muted uppercase tracking-wider mt-1 m-0">
            Hardware Register — {assets.length} holdings recorded
          </p>
        </div>
        <button 
          onClick={exportCSV}
          className="flex items-center gap-2 px-4 py-2 bg-ink text-parchment text-[11px] font-mono-ui uppercase tracking-wider hover:bg-ink/90 transition focus-ring rounded-sm"
        >
          <Download className="w-4 h-4" strokeWidth={2} />
          Export Register
        </button>
      </div>

      {/* Search Bar */}
      <div className="flex flex-col md:flex-row gap-4 items-start justify-between bg-parchment-dark border border-ink-muted/30 p-4">
        <div className="relative w-full md:w-96">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-ink-muted" strokeWidth={2} />
          <input 
            type="text"
            placeholder="Search holdings by name, IP, processor, or serial..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full bg-parchment border border-ink-muted/40 pl-10 pr-4 py-2.5 text-sm text-ink focus:outline-none focus:border-gold transition focus-ring rounded-sm"
          />
        </div>
      </div>

      {/* Inventory Register */}
      {loading ? (
        <div className="text-center py-16 text-ink-muted font-mono-ui">Consulting the estate registers...</div>
      ) : filteredAssets.length === 0 ? (
        <div className="text-center py-16 text-ink-muted bg-parchment-dark border border-ink-muted/30">
          <HardDrive className="w-14 h-14 mx-auto mb-4 opacity-30 text-ink-muted" strokeWidth={1.5} />
          <p className="font-display text-lg text-ink m-0">No inventory recorded</p>
          <p className="font-mono-ui text-[11px] mt-1 m-0">Assets are reported by agents every 24 hours.</p>
        </div>
      ) : (
        <div className="ledger-card-elevated overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse font-mono-ui text-[11px]">
              <thead>
                <tr className="bg-ink/5 border-b border-ink-muted/30 text-ink-muted uppercase tracking-wider">
                  <th className="p-4 text-left font-medium">Holding</th>
                  <th className="p-4 text-left font-medium">Processor & Memory</th>
                  <th className="p-4 text-left font-medium">Storage Health</th>
                  <th className="p-4 text-left font-medium">Power Reserve</th>
                  <th className="p-4 text-left font-medium">Identity</th>
                  <th className="p-4 text-left font-medium">Network</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-ink-muted/20 text-ink">
                {filteredAssets.map((a) => (
                  <tr key={a.server_id} className="hover:bg-ink/3 transition">
                    <td className="p-4">
                      <div className="flex items-center gap-3">
                        <div className={`p-2 rounded-sm ${a.device_type === 'laptop' ? 'bg-moss/10 text-moss' : 'bg-gold-leaf/10 text-gold-dim'}`}>
                          {a.device_type === 'laptop' ? <Laptop className="w-5 h-5" strokeWidth={1.5} /> : <Server className="w-5 h-5" strokeWidth={1.5} />}
                        </div>
                        <div>
                          <div className="font-display text-base font-semibold">{a.server_id}</div>
                          <div className="text-[10px] text-ink-muted uppercase tracking-wider">
                            {a.device_type?.charAt(0).toUpperCase() + a.device_type?.slice(1) || 'Server'}
                            {a.owner_name ? ` · ${a.owner_name}` : ''}
                          </div>
                          {formatDeviceModel(a.device_manufacturer, a.device_model) && (
                            <div className="text-[10px] text-ink mt-0.5">{formatDeviceModel(a.device_manufacturer, a.device_model)}</div>
                          )}
                        </div>
                      </div>
                    </td>
                    <td className="p-4">
                      <div className="font-medium text-ink">{a.processor_model || '—'}</div>
                      <div className="text-[10px] text-ink-muted mt-0.5">
                        {a.processor_cores || 0} Cores @ {a.processor_clock_ghz ? a.processor_clock_ghz.toFixed(1) : '?'} GHz
                      </div>
                      <div className="text-[10px] text-ink-muted mt-0.5">
                        {a.ram_total_gb || 0} GB ({a.ram_ddr_type || 'DDR'}{a.ram_speed_mhz > 0 ? ` ${a.ram_speed_mhz}MHz` : ''}{a.ram_form_factor ? ` ${a.ram_form_factor}` : ''}) &middot; {a.ram_slots_used || 0}/{a.ram_slots_total || 0} slots
                        {a.ram_max_gb > 0 ? ` · max ${a.ram_max_gb.toFixed(0)}GB` : ''}
                      </div>
                      {a.gpu_model && (
                        <div className="text-[10px] text-ink-muted mt-0.5 truncate max-w-xs" title={a.gpu_model}>GPU: {a.gpu_model}</div>
                      )}
                    </td>
                    <td className="p-4">
                      <div className="text-[11px] text-ink-muted mb-1">
                        {a.disk_type || 'Disk'} ({a.disk_size_gb ? a.disk_size_gb.toFixed(0) : 0} GB)
                      </div>
                      <div className="flex flex-wrap items-center gap-2">
                        <span className={`inline-flex items-center px-2 py-1 rounded-sm font-mono-ui text-[10px] uppercase tracking-wider ${getDiskHealthClass(a.disk_health_percent)}`}>
                          {getDiskHealthLabel(a.disk_health_percent)}
                        </span>
                        {getSmartBadge(a.disk_health_status)}
                      </div>
                      {a.disk_volumes?.length > 0 && (
                        <div className="text-[10px] text-ink-muted mt-1">
                          {a.disk_volumes.map(v => `${v.drive} ${v.used_percent.toFixed(0)}%/${v.total_gb.toFixed(0)}G`).join(' · ')}
                        </div>
                      )}
                    </td>
                    <td className="p-4">
                      {a.battery_health_percent !== null && a.battery_health_percent !== undefined ? (
                        <div className="flex items-center gap-2">
                          <BatteryCharging className="w-4 h-4 text-gold-dim" strokeWidth={1.5} />
                          <span className={`font-mono-ui ${a.battery_health_percent < 50 ? 'text-cardinal' : a.battery_health_percent < 80 ? 'text-gold-dim' : 'text-moss'} font-bold`}>
                            {a.battery_health_percent.toFixed(0)}%
                          </span>
                        </div>
                      ) : (
                        <span className="text-ink-muted text-[11px]">— (Mains Power)</span>
                      )}
                    </td>
                    <td className="p-4">
                      <div className="font-mono-ui text-sm text-ink font-medium">{a.serial_number || '—'}</div>
                      <div className="text-[10px] text-ink-muted mt-0.5 truncate max-w-xs">{a.os_version || 'Windows'}</div>
                      {a.bios_release_date && (
                        <div className="text-[10px] text-ink-muted mt-0.5">BIOS: {a.bios_release_date}</div>
                      )}
                    </td>
                    <td className="p-4">
                      <div className="font-mono-ui text-[11px] text-gold-dim">{a.ip_address || '—'}</div>
                      <div className="text-[10px] text-ink-muted mt-0.5">{a.mac_address || '—'}</div>
                    </td>
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