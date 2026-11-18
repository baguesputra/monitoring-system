import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Server, Laptop, Monitor, Cpu, MemoryStick, HardDrive, Search, RefreshCw, ArrowUpRight, ArrowRight, X, CheckCircle2, AlertTriangle, CircleAlert, Clock3, Layers, User, WifiOff } from 'lucide-react';
import { deviceStatus, metricTone, timeAgo, STATUS_LABELS } from '../utils/deviceStatus';

const priority = { critical: 0, warning: 1, stale: 2, unknown: 3, normal: 4 };
const summaries = [
  { tone: 'all', label: 'Total perangkat', caption: 'Terdaftar dalam inventaris', icon: Layers },
  { tone: 'normal', label: 'Normal', caption: 'Pemakaian dalam batas normal', icon: CheckCircle2 },
  { tone: 'warning', label: 'Peringatan', caption: 'Perlu pemeriksaan lebih lanjut', icon: AlertTriangle },
  { tone: 'critical', label: 'Kritis', caption: 'Prioritaskan pemeriksaan', icon: CircleAlert },
];
const metrics = [['cpu', 'CPU', Cpu], ['ram', 'RAM', MemoryStick], ['disk', 'Disk', HardDrive]];

async function getJSON(url, signal) {
  const response = await fetch(url, { signal: AbortSignal.any([signal, AbortSignal.timeout(15000)]) });
  if (!response.ok) throw new Error(`HTTP ${response.status}`);
  return response.json();
}

export default function Overview({ onSelectDevice, active = true }) {
  const [servers, setServers] = useState([]);
  const [statuses, setStatuses] = useState({});
  const [failures, setFailures] = useState(new Set());
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState('');
  const [lastUpdated, setLastUpdated] = useState(null);
  const [now, setNow] = useState(() => Date.now());
  const [search, setSearch] = useState('');
  const [typeFilter, setTypeFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const [sort, setSort] = useState('priority');
  const request = useRef(null);

  const fetchData = useCallback(async () => {
    if (request.current) return;
    const controller = new AbortController();
    request.current = controller;
    await Promise.resolve();
    if (controller.signal.aborted) return;
    setRefreshing(true);
    try {
      const data = await getJSON('/api/servers', controller.signal);
      if (!Array.isArray(data)) throw new Error('Invalid server list');
      setServers(data);
      const nextStatuses = {};
      const failed = new Set();
      let nextIndex = 0;
      await Promise.all(Array.from({ length: Math.min(6, data.length) }, async () => {
        while (nextIndex < data.length && !controller.signal.aborted) {
          const server = data[nextIndex++];
          try {
            const status = await getJSON(`/api/servers/${encodeURIComponent(server.server_id)}/status`, controller.signal);
            if (!status || typeof status !== 'object' || Array.isArray(status)) throw new Error('Invalid status');
            nextStatuses[server.server_id] = status;
          } catch {
            failed.add(server.server_id);
          }
        }
      }));
      if (controller.signal.aborted) return;
      setStatuses(previous => Object.fromEntries(data.map(server => [server.server_id, nextStatuses[server.server_id] ?? previous[server.server_id]])));
      setFailures(failed);
      setError(failed.size ? `${failed.size} perangkat gagal diperbarui. Data terakhir yang tersedia tetap ditampilkan.` : '');
      if (!failed.size) setLastUpdated(new Date());
    } catch {
      if (!controller.signal.aborted) setError('Tidak dapat terhubung ke layanan monitoring. Periksa koneksi lalu coba lagi.');
    } finally {
      if (request.current === controller) {
        request.current = null;
        if (!controller.signal.aborted) {
          setLoading(false);
          setRefreshing(false);
          setNow(Date.now());
        }
      }
    }
  }, []);

  useEffect(() => {
    if (!active) return;
    let disposed = false;
    queueMicrotask(() => { if (!disposed) fetchData(); });
    const interval = setInterval(() => {
      setNow(Date.now());
      if (!document.hidden) fetchData();
    }, 10000);
    return () => {
      disposed = true;
      clearInterval(interval);
      request.current?.abort();
      request.current = null;
    };
  }, [active, fetchData]);

  const devices = servers.map(server => ({ ...server, status: statuses[server.server_id], health: deviceStatus(statuses[server.server_id], now, failures.has(server.server_id)) }));
  const counts = devices.reduce((result, device) => { result[device.health.tone]++; return result; }, { all: devices.length, normal: 0, warning: 0, critical: 0, stale: 0, unknown: 0 });
  const comparePriority = (a, b) => priority[a.health.tone] - priority[b.health.tone] || a.server_id.localeCompare(b.server_id);
  const attention = devices.filter(device => device.health.tone !== 'normal').sort(comparePriority);
  const query = search.trim().toLowerCase();
  const filtered = devices.filter(device => {
    const matchSearch = [device.server_id, device.hostname, device.ip_address, device.owner_name, device.department].some(value => String(value || '').toLowerCase().includes(query));
    return matchSearch && (typeFilter === 'all' || device.device_type?.toLowerCase() === typeFilter) && (statusFilter === 'all' || (statusFilter === 'attention' ? device.health.tone !== 'normal' : device.health.tone === statusFilter));
  }).sort((a, b) => {
    if (sort === 'name') return a.server_id.localeCompare(b.server_id);
    if (sort !== 'priority') return (b.status?.[`${sort}_percent`] ?? -1) - (a.status?.[`${sort}_percent`] ?? -1) || comparePriority(a, b);
    return comparePriority(a, b);
  });
  const hasFilters = search !== '' || typeFilter !== 'all' || statusFilter !== 'all';
  const reset = () => { setSearch(''); setTypeFilter('all'); setStatusFilter('all'); };
  const filterAttention = () => {
    setSearch(''); setTypeFilter('all'); setStatusFilter('attention');
    document.getElementById('device-list-heading')?.focus({ preventScroll: true });
    document.getElementById('device-list-heading')?.scrollIntoView({ block: 'start' });
  };

  return (
    <div className="overview">
      <div className="page-heading">
        <div><p className="eyebrow">MONITORING PERANGKAT</p><h1>Ringkasan infrastruktur</h1><p className="page-description">Pantau kondisi perangkat dan temukan yang perlu ditangani.</p></div>
        <button className="button button-primary" onClick={fetchData} disabled={refreshing}><RefreshCw size={17} className={`refresh-icon ${refreshing ? 'is-spinning' : ''}`} />{refreshing ? 'Memperbarui…' : 'Perbarui data'}</button>
      </div>
      <div className="freshness-row"><span><Clock3 size={14} />{lastUpdated ? `Pembaruan lengkap ${lastUpdated.toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit', second: '2-digit' })}` : 'Menunggu pembaruan lengkap'}</span><span>Otomatis setiap 10 detik</span></div>
      {error && <div className="notice notice-error" role="alert"><AlertTriangle size={19} /><span>{error}</span><button className="reset-button" onClick={fetchData} disabled={refreshing}>Coba lagi</button></div>}
      <section className="summary-grid" aria-label="Ringkasan status perangkat" aria-busy={loading}>
        {summaries.map(({ tone, label, caption, icon: Icon }) => <button key={tone} className={`summary-card ${statusFilter === tone ? 'is-selected' : ''}`} data-tone={tone} onClick={() => setStatusFilter(tone)} aria-pressed={statusFilter === tone}>
          <span className="summary-label">{label}<span className="summary-icon"><Icon size={19} /></span></span><strong className="summary-value">{loading ? '—' : counts[tone]}</strong><span className="summary-caption">{caption}<ArrowUpRight size={15} /></span>
        </button>)}
      </section>
      <div className="freshness-row">
        <div className="filter-controls">{['stale', 'unknown'].map(tone => <button key={tone} className={`filter-chip ${statusFilter === tone ? 'is-active' : ''}`} data-tone={tone} onClick={() => setStatusFilter(statusFilter === tone ? 'all' : tone)} aria-pressed={statusFilter === tone}>{tone === 'stale' ? <Clock3 size={14} /> : <WifiOff size={14} />}{tone === 'unknown' ? 'Data belum lengkap / gagal' : STATUS_LABELS[tone]}<strong>{loading ? '—' : counts[tone]}</strong></button>)}</div>
        <span>Data terlambat: tidak ada laporan &gt; 2 menit</span>
      </div>
      {!loading && <section className="attention-panel" aria-labelledby="attention-heading">
        <div className="section-heading"><h2 id="attention-heading"><AlertTriangle size={18} />Perlu perhatian <span>{attention.length}</span></h2>{attention.length > 0 && <button className="reset-button" onClick={filterAttention}>Lihat semua <ArrowRight size={15} /></button>}</div>
        {attention.length ? <div className="attention-list">{attention.slice(0, 3).map(device => <button className="attention-item" key={device.server_id} onClick={() => onSelectDevice(device.server_id)}>
          <span className="attention-icon" data-tone={device.health.tone}>{device.health.tone === 'critical' ? <CircleAlert size={19} /> : <AlertTriangle size={19} />}</span><span className="attention-copy"><span className="attention-title">{device.server_id}</span><span className="attention-reason">{device.health.reason}</span></span><ArrowUpRight size={17} />
        </button>)}</div> : <div className="attention-empty"><CheckCircle2 size={20} /><span>{error ? 'Kondisi perangkat belum dapat dipastikan. Coba perbarui data.' : servers.length ? 'Semua perangkat melaporkan pemakaian dalam batas normal.' : 'Belum ada perangkat untuk dipantau.'}</span></div>}
      </section>}
      <section aria-labelledby="device-list-heading">
        <div className="section-heading"><h2 id="device-list-heading" tabIndex={-1}>Daftar perangkat</h2><span>CPU · RAM · Penyimpanan</span></div>
        <div className="device-toolbar">
          <div className="search-field"><Search size={19} /><label className="sr-only" htmlFor="device-search">Cari nama, IP, pemilik, atau departemen</label><input id="device-search" type="search" placeholder="Cari nama, IP, pemilik…" value={search} onChange={event => setSearch(event.target.value)} />{search && <button onClick={() => setSearch('')} aria-label="Hapus pencarian"><X size={16} /></button>}</div>
          <div className="filter-controls">
            <label>Jenis<select value={typeFilter} onChange={event => setTypeFilter(event.target.value)}><option value="all">Semua jenis</option><option value="server">Server</option><option value="pc">PC</option><option value="laptop">Laptop</option></select></label>
            <label>Status<select value={statusFilter} onChange={event => setStatusFilter(event.target.value)}><option value="all">Semua status</option><option value="attention">Perlu perhatian</option>{Object.entries(STATUS_LABELS).map(([value, label]) => <option key={value} value={value}>{value === 'unknown' ? 'Belum lengkap / gagal' : label}</option>)}</select></label>
            <label>Urutkan<select value={sort} onChange={event => setSort(event.target.value)}><option value="priority">Prioritas</option><option value="name">Nama A–Z</option><option value="cpu">CPU tertinggi</option><option value="ram">RAM tertinggi</option><option value="disk">Disk tertinggi</option></select></label>
          </div>
        </div>
        <div className="results-bar"><span role="status">{loading ? 'Memuat perangkat…' : `${filtered.length} dari ${servers.length} perangkat`}{statusFilter !== 'all' && ` · ${statusFilter === 'attention' ? 'Perlu perhatian' : STATUS_LABELS[statusFilter]}`}</span>{hasFilters && <button className="reset-button" onClick={reset}><X size={14} />Reset filter</button>}</div>
        {loading ? <div className="loading-grid" role="status" aria-label="Memuat kondisi perangkat">{Array.from({ length: 6 }, (_, index) => <div className="skeleton-card" key={index} aria-hidden="true"><div className="skeleton-line" /><div className="skeleton-line" /><div className="skeleton-bar" /><div className="skeleton-bar" /><div className="skeleton-bar" /></div>)}</div>
          : filtered.length === 0 ? <div className="empty-state"><Server size={36} /><h3>{hasFilters ? 'Tidak ada perangkat yang cocok' : error ? 'Data perangkat belum dapat dimuat' : 'Belum ada perangkat terdaftar'}</h3><p>{hasFilters ? 'Coba kata kunci lain atau hapus filter untuk melihat semua perangkat.' : error ? 'Periksa koneksi ke layanan monitoring, lalu coba lagi.' : 'Perangkat akan muncul setelah agent mengirimkan laporan pertama.'}</p>{hasFilters ? <button className="button" onClick={reset}>Reset filter</button> : error && <button className="button" onClick={fetchData} disabled={refreshing}>Coba lagi</button>}</div>
            : <div className="device-grid">{filtered.map(device => {
              const DeviceIcon = device.device_type?.toLowerCase() === 'laptop' ? Laptop : device.device_type?.toLowerCase() === 'pc' ? Monitor : Server;
              return <article key={device.server_id} className="device-card">
                <div className="device-header"><span className="device-icon"><DeviceIcon size={21} /></span><div className="device-identity"><h3><button type="button" className="device-title-button" onClick={() => onSelectDevice(device.server_id)} aria-label={`Lihat detail ${device.server_id}`}>{device.server_id}</button></h3><p className="device-subtitle">{device.hostname || 'Hostname belum tersedia'}</p></div></div>
                <span className="status-badge" data-tone={device.health.tone} title={device.health.reason}>{device.health.tone === 'normal' ? <CheckCircle2 size={13} /> : device.health.tone === 'stale' ? <Clock3 size={13} /> : <CircleAlert size={13} />}{device.health.label}</span>
                <p className="device-owner"><User size={14} />{[device.owner_name, device.department].filter(Boolean).join(' · ') || 'Pemilik belum ditetapkan'}</p>
                <div className="metric-list">{metrics.map(([key, label, Icon]) => {
                  const value = device.status?.[`${key}_percent`];
                  const tone = metricTone(key, value);
                  const known = tone !== 'unknown';
                  return <div key={key}><div className="metric-row"><span className="metric-label"><Icon size={14} />{label}</span><span className="metric-value" data-tone={tone}>{known ? `${value.toFixed(1)}%` : '—'}</span></div><div className="metric-track" role="meter" aria-label={`${label} ${device.server_id}${known ? '' : ': belum tersedia'}`} aria-valuemin={0} aria-valuemax={100} aria-valuenow={known ? Math.max(0, Math.min(100, value)) : undefined} aria-valuetext={known ? `${value.toFixed(1)} persen` : 'Belum tersedia'}><div className="metric-fill" data-tone={device.health.tone === 'stale' || failures.has(device.server_id) ? 'unknown' : tone} style={{ width: `${known ? Math.max(0, Math.min(100, value)) : 0}%` }} /></div></div>;
                })}</div>
                <div className="device-footer"><span className="device-address">{device.ip_address || 'IP belum tersedia'}</span><span className="device-link" aria-hidden="true"><ArrowUpRight size={15} /></span></div>
                <p className="last-seen"><Clock3 size={12} />{timeAgo(device.status?.last_seen_at, now)}</p>
              </article>;
            })}</div>}
      </section>
    </div>
  );
}
