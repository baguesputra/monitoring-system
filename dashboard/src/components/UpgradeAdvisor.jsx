import React, { useState, useEffect, useCallback } from 'react';
import { ArrowRight, BatteryCharging, Cpu, HardDrive, MemoryStick, RefreshCw, Search, ShieldAlert, CheckCircle2, Wrench } from 'lucide-react';
import { formatDeviceModel } from '../utils/formatters';

const PRIORITY_BADGE = {
  critical: { cls: 'wax-seal-critical', label: 'URGENT UPGRADE' },
  recommended: { cls: 'wax-seal-warning', label: 'RECOMMENDED' },
  healthy: { cls: 'wax-seal-normal', label: 'HEALTHY' },
};

export default function UpgradeAdvisor({ onSelectDevice }) {
  const [data, setData] = useState(null);
  const [days, setDays] = useState(30);
  const [filter, setFilter] = useState('all');
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);

  const loadInsights = useCallback(async (windowDays) => {
    setLoading(true);
    try {
      const res = await fetch(`/api/upgrade-insights?days=${windowDays}`);
      if (res.ok) {
        setData(await res.json());
      }
    } catch (err) {
      console.error('Failed to load upgrade insights:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    let active = true;
    fetch(`/api/upgrade-insights?days=${days}`)
      .then((res) => (res.ok ? res.json() : null))
      .then((json) => {
        if (active && json) setData(json);
      })
      .catch((err) => console.error('Failed to load upgrade insights:', err))
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => { active = false; };
  }, [days]);

  const devices = (data?.devices || []).filter((d) => {
    const matchesFilter = filter === 'all' || d.priority === filter;
    const q = search.trim().toLowerCase();
    const matchesSearch = !q ||
      d.server_id.toLowerCase().includes(q) ||
      (d.owner_name && d.owner_name.toLowerCase().includes(q)) ||
      (d.department && d.department.toLowerCase().includes(q)) ||
      (d.device_model && d.device_model.toLowerCase().includes(q)) ||
      (d.processor_model && d.processor_model.toLowerCase().includes(q));
    return matchesFilter && matchesSearch;
  });

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-4 pb-4 border-b border-ink-muted/30">
        <div>
          <h2 className="font-display text-2xl font-semibold text-ink m-0">Upgrade Advisor</h2>
          <p className="font-mono-ui text-[11px] text-ink-muted uppercase tracking-wider mt-1 m-0">
            Hardware lifecycle &amp; bottleneck evaluation ({days}-day window)
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <div className="flex items-center gap-1 bg-parchment-dark border border-ink-muted/30 p-1">
            {[7, 30].map((d) => (
              <button
                key={d}
                onClick={() => setDays(d)}
                className={`px-3 py-1.5 text-[10px] font-mono-ui uppercase tracking-wider transition rounded-sm ${
                  days === d ? 'bg-ink text-parchment' : 'text-ink-muted hover:text-ink'
                }`}
              >
                {d}D Window
              </button>
            ))}
          </div>
          <button
            onClick={() => loadInsights(days)}
            className="flex items-center gap-2 px-4 py-2 bg-parchment-dark border border-ink-muted/40 text-ink text-[11px] font-mono-ui uppercase tracking-wider hover:bg-ink/5 transition rounded-sm"
          >
            <RefreshCw className="w-4 h-4" strokeWidth={2} />
            Recalculate
          </button>
        </div>
      </div>

      {/* Summary Registers */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="ledger-card p-5">
          <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Total Assessed</p>
          <p className="font-display text-3xl font-semibold text-ink mt-1 m-0">{data?.total_devices || 0}</p>
          <p className="font-mono-ui text-[11px] text-ink-muted mt-2 m-0">Managed Holdings</p>
        </div>
        <div className="ledger-card p-5">
          <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Urgent Upgrade</p>
          <p className="font-display text-3xl font-semibold text-cardinal mt-1 m-0">{data?.critical_count || 0}</p>
          <p className="font-mono-ui text-[11px] text-ink-muted mt-2 m-0">HDD / Critical RAM / Disk</p>
        </div>
        <div className="ledger-card p-5">
          <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Recommended</p>
          <p className="font-display text-3xl font-semibold text-gold-dim mt-1 m-0">{data?.recommended_count || 0}</p>
          <p className="font-mono-ui text-[11px] text-ink-muted mt-2 m-0">Approaching Capacity / Age</p>
        </div>
        <div className="ledger-card p-5">
          <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Healthy</p>
          <p className="font-display text-3xl font-semibold text-moss mt-1 m-0">{data?.healthy_count || 0}</p>
          <p className="font-mono-ui text-[11px] text-ink-muted mt-2 m-0">No Upgrade Needed</p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-col md:flex-row gap-4 items-start justify-between bg-parchment-dark border border-ink-muted/30 p-4">
        <div className="relative w-full md:w-96">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-ink-muted" strokeWidth={2} />
          <input
            type="text"
            placeholder="Filter by device, owner, department, model, CPU..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full bg-parchment border border-ink-muted/40 pl-10 pr-4 py-2.5 text-sm text-ink focus:outline-none focus:border-gold rounded-sm"
          />
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {['all', 'critical', 'recommended', 'healthy'].map((p) => (
            <button
              key={p}
              onClick={() => setFilter(p)}
              className={`px-3 py-1.5 text-[10px] font-mono-ui uppercase tracking-wider transition rounded-sm ${
                filter === p ? 'bg-ink text-parchment' : 'text-ink-muted bg-parchment border border-ink-muted/30 hover:text-ink'
              }`}
            >
              {p === 'all' ? 'All' : p}
            </button>
          ))}
        </div>
      </div>

      {/* Device Upgrade Cards */}
      {loading ? (
        <div className="text-center py-16 text-ink-muted font-mono-ui">Evaluating upgrade readiness...</div>
      ) : devices.length === 0 ? (
        <div className="text-center py-16 text-ink-muted bg-parchment-dark border border-ink-muted/30">
          <Wrench className="w-12 h-12 mx-auto mb-3 opacity-30" strokeWidth={1.5} />
          <p className="font-display text-lg text-ink m-0">No matching holdings</p>
        </div>
      ) : (
        <div className="space-y-4">
          {devices.map((d) => {
            const badge = PRIORITY_BADGE[d.priority] || PRIORITY_BADGE.healthy;
            const freeSlots = d.ram_slots_total > 0 ? Math.max(0, d.ram_slots_total - d.ram_slots_used) : null;
            return (
              <article key={d.server_id} className="ledger-card-elevated p-6 space-y-4">
                <div className="flex flex-col md:flex-row md:items-center justify-between gap-3 pb-3 border-b border-ink-muted/20">
                  <div>
                    <div className="flex flex-wrap items-center gap-3">
                      <h3 className="font-display text-xl font-semibold text-ink m-0">{d.server_id}</h3>
                      <span className={`wax-seal ${badge.cls}`}>{badge.label}</span>
                      <span className="font-mono-ui text-[10px] uppercase px-2 py-0.5 bg-parchment-dark border border-ink-muted/30 text-ink-muted">
                        {d.device_type}
                      </span>
                    </div>
                    <p className="font-mono-ui text-[11px] text-ink-muted mt-1 m-0">
                      {formatDeviceModel(d.device_manufacturer, d.device_model) || d.hostname || 'Model belum dilaporkan'}
                      {d.device_age_years !== null && d.device_age_years !== undefined ? ` · Usia ~${d.device_age_years} thn (BIOS ${d.bios_release_date})` : ''}
                      {d.owner_name ? ` · Pemegang: ${d.owner_name}` : ''}
                      {d.department ? ` (${d.department})` : ''}
                    </p>
                  </div>
                  <button
                    onClick={() => onSelectDevice?.(d.server_id)}
                    className="flex items-center gap-1.5 px-3 py-2 text-[10px] font-mono-ui uppercase tracking-wider bg-parchment-dark border border-ink-muted/40 text-ink hover:border-gold transition rounded-sm self-start md:self-auto"
                  >
                    Inspect Holding <ArrowRight className="w-3.5 h-3.5" strokeWidth={2} />
                  </button>
                </div>

                {/* Spec & Telemetry Grid */}
                <div className="grid grid-cols-1 md:grid-cols-4 gap-4 font-mono-ui text-[11px]">
                  <div className="p-3 bg-parchment-dark border border-ink-muted/20">
                    <p className="text-[10px] text-ink-muted uppercase flex items-center gap-1.5 m-0 mb-1">
                      <Cpu className="w-3.5 h-3.5 text-moss" /> Processor &amp; GPU
                    </p>
                    <p className="font-medium text-ink m-0 truncate" title={d.processor_model}>{d.processor_model || '—'}</p>
                    <p className="text-[10px] text-ink-muted mt-1 m-0">
                      Avg {d.avg_cpu_percent.toFixed(1)}% · P95 {d.p95_cpu_percent.toFixed(1)}%
                    </p>
                    {d.gpu_model && <p className="text-[10px] text-ink-muted mt-0.5 m-0 truncate" title={d.gpu_model}>GPU: {d.gpu_model}</p>}
                  </div>

                  <div className="p-3 bg-parchment-dark border border-ink-muted/20">
                    <p className="text-[10px] text-ink-muted uppercase flex items-center gap-1.5 m-0 mb-1">
                      <MemoryStick className="w-3.5 h-3.5 text-gold-dim" /> Memory Upgrade Spec
                    </p>
                    <p className="font-medium text-ink m-0">
                      {d.ram_total_gb ? `${d.ram_total_gb.toFixed(0)} GB` : '—'} {d.ram_ddr_type || ''} {d.ram_speed_mhz > 0 ? `@ ${d.ram_speed_mhz} MHz` : ''}
                    </p>
                    <p className="text-[10px] text-ink-muted mt-1 m-0">
                      Slot: {d.ram_slots_used}/{d.ram_slots_total || '?'}
                      {freeSlots !== null ? ` (${freeSlots} kosong)` : ''}
                      {d.ram_form_factor ? ` · ${d.ram_form_factor}` : ''}
                      {d.ram_max_gb > 0 ? ` · Maks ${d.ram_max_gb.toFixed(0)} GB` : ''}
                    </p>
                    <p className="text-[10px] text-ink-muted mt-0.5 m-0">
                      Avg {d.avg_ram_percent.toFixed(1)}% · P95 {d.p95_ram_percent.toFixed(1)}%
                    </p>
                  </div>

                  <div className="p-3 bg-parchment-dark border border-ink-muted/20">
                    <p className="text-[10px] text-ink-muted uppercase flex items-center gap-1.5 m-0 mb-1">
                      <HardDrive className="w-3.5 h-3.5 text-ink" /> Storage &amp; Volumes
                    </p>
                    <p className="font-medium text-ink m-0">
                      {d.disk_type || '—'} · {d.disk_size_gb ? `${d.disk_size_gb.toFixed(0)} GB` : '—'}
                    </p>
                    <p className="text-[10px] text-ink-muted mt-1 m-0">
                      SMART: {d.disk_health_status || 'UNKNOWN'}
                      {d.disk_health_percent !== null && d.disk_health_percent !== undefined ? ` (${d.disk_health_percent}%)` : ''}
                    </p>
                    {d.disk_volumes?.length > 0 ? (
                      <p className="text-[10px] text-ink-muted mt-0.5 m-0 truncate">
                        {d.disk_volumes.map((v) => `${v.drive} ${v.used_percent.toFixed(0)}%`).join(' · ')}
                      </p>
                    ) : (
                      <p className="text-[10px] text-ink-muted mt-0.5 m-0">C: {d.latest_disk_percent.toFixed(1)}%</p>
                    )}
                  </div>

                  <div className="p-3 bg-parchment-dark border border-ink-muted/20">
                    <p className="text-[10px] text-ink-muted uppercase flex items-center gap-1.5 m-0 mb-1">
                      <BatteryCharging className="w-3.5 h-3.5 text-gold-dim" /> Power / Samples
                    </p>
                    <p className="font-medium text-ink m-0">
                      {d.battery_health_percent !== null && d.battery_health_percent !== undefined
                        ? `Baterai ${d.battery_health_percent.toFixed(0)}%`
                        : 'AC / Mains Power'}
                    </p>
                    <p className="text-[10px] text-ink-muted mt-1 m-0">
                      {d.metric_samples} sample dalam {days} hari
                    </p>
                  </div>
                </div>

                {/* Recommendations */}
                {d.recommendations?.length > 0 ? (
                  <div className="space-y-2 pt-1">
                    {d.recommendations.map((rec, idx) => (
                      <div
                        key={idx}
                        className={`p-3 border rounded-sm flex flex-col md:flex-row md:items-center justify-between gap-2 ${
                          rec.severity === 'critical'
                            ? 'bg-cardinal/5 border-cardinal/30'
                            : 'bg-gold-leaf/10 border-gold-leaf/40'
                        }`}
                      >
                        <div className="flex items-start gap-2.5">
                          <ShieldAlert
                            className={`w-4 h-4 mt-0.5 flex-shrink-0 ${
                              rec.severity === 'critical' ? 'text-cardinal' : 'text-gold-dim'
                            }`}
                          />
                          <div>
                            <p className="font-mono-ui text-xs font-semibold text-ink m-0">{rec.title}</p>
                            <p className="font-mono-ui text-[11px] text-ink-muted m-0">{rec.detail}</p>
                          </div>
                        </div>
                        <p className="font-mono-ui text-[11px] font-medium text-ink bg-parchment px-2.5 py-1 border border-ink-muted/30 rounded-sm m-0">
                          Saran: {rec.action}
                        </p>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="flex items-center gap-2 text-moss font-mono-ui text-[11px]">
                    <CheckCircle2 className="w-4 h-4" />
                    <span>Tidak ada indikasi bottleneck atau komponen aus pada rentang {days} hari terakhir.</span>
                  </div>
                )}
              </article>
            );
          })}
        </div>
      )}
    </div>
  );
}
