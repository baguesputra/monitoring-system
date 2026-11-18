import React, { useState, useEffect } from 'react';
import { ArrowLeft, Cpu, Activity, Server, CheckCircle, XCircle, RefreshCw, Save, X, UserCog, KeyRound, UserX, Pencil, Crown, HardDrive, ShieldCheck, Terminal } from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid, Legend } from 'recharts';
import { formatDeviceModel } from '../utils/formatters';
import { deviceStatus, metricTone } from '../utils/deviceStatus';

const CHART_COLORS = {
  cpu: '#4A5D4E',
  ram: '#364238',
  sent: '#C5A059',
  recv: '#8B1E1E',
};

const PRIVILEGE_META = {
  administrator: {
    label: 'Administrator',
    icon: Crown,
    chip: 'bg-gold-leaf/15 text-gold-dim border-gold-leaf/40',
  },
  standard_user: {
    label: 'Standard User',
    icon: UserCog,
    chip: 'bg-moss/10 text-moss border-moss/30',
  },
  limited: {
    label: 'Guest',
    icon: UserX,
    chip: 'bg-ink-muted/10 text-ink-muted border-ink-muted/30',
  },
};

function MetricBar({ label, value }) {
  const tone = metricTone(label.toLowerCase(), value);
  const critical = tone === 'critical';
  const warning = tone === 'warning';
  const known = tone !== 'unknown';
  return (
    <div>
      <div className="flex justify-between text-[11px] mb-1 font-mono-ui">
        <span className="text-ink-muted">{label}</span>
        <span className={critical ? 'text-cardinal font-bold' : warning ? 'text-gold-dim font-bold' : 'text-ink'}>
            {known ? `${value.toFixed(1)}%` : '—'}
        </span>
      </div>
      <div className="w-full bg-parchment-dark h-1.5 rounded-sm overflow-hidden border border-ink-muted/20">
        <div
          className={`h-full rounded-sm transition-all duration-500 ${critical ? 'bg-cardinal' : warning ? 'bg-gold-dim' : 'bg-moss'}`}
          style={{ width: `${known ? Math.max(0, Math.min(value, 100)) : 0}%` }}
        />
      </div>
    </div>
  );
}

const axisProps = {
  stroke: '#2C2621', strokeOpacity: 0.4, fontSize: 10,
  fontFamily: "'IBM Plex Mono', monospace", tickLine: false,
  tick: { fill: '#5C5348' },
};

const CustomTooltip = ({ active, payload, label }) => {
  if (active && payload?.length) {
    return (
      <div className="bg-ink text-parchment p-3 border border-gold-leaf rounded-sm font-mono-ui text-[11px] shadow-seal">
        <p className="font-display text-gold-leaf mb-1">{label}</p>
        {payload.map((entry, i) => (
          <p key={i} style={{ color: entry.color }}>{entry.name}: {entry.value}</p>
        ))}
      </div>
    );
  }
  return null;
};

export default function DeviceDetail({ serverId, onBack, onRenamed }) {
  const [metrics, setMetrics] = useState([]);
  const [status, setStatus] = useState(null);
  const [statusFailed, setStatusFailed] = useState(false);
  const [statusTime, setStatusTime] = useState(() => Date.now());
  const [asset, setAsset] = useState(null);
  const [users, setUsers] = useState([]);
  const [meta, setMeta] = useState(null);
  const [range, setRange] = useState('1h');
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState({ owner_name: '', department: '', location: '', notes: '', device_type: '' });
  const [saving, setSaving] = useState(false);
  const [saveMsg, setSaveMsg] = useState(null);
  const [renaming, setRenaming] = useState(false);
  const [renameDraft, setRenameDraft] = useState('');
  const [renameMsg, setRenameMsg] = useState(null);
  const [procSearch, setProcSearch] = useState('');

  const fetchDetail = async () => {
    try {
      const [mRes, sRes, aRes, uRes, srvRes] = await Promise.all([
        fetch(`/api/servers/${serverId}/metrics?range=${range}`).catch(() => null),
        fetch(`/api/servers/${serverId}/status`).catch(() => null),
        fetch(`/api/servers/${serverId}/asset-info`).catch(() => null),
        fetch(`/api/servers/${serverId}/users`).catch(() => null),
        fetch('/api/servers').catch(() => null),
      ]);

      if (mRes?.ok) {
        const mData = await mRes.json();
        const isMultiDay = range.endsWith('d');
        setMetrics((mData || []).map(item => ({
          ...item,
          time: isMultiDay
            ? new Date(item.recorded_at).toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
            : new Date(item.recorded_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
          network_sent_kb: (item.network_sent_bps / 1024).toFixed(1),
          network_recv_kb: (item.network_recv_bps / 1024).toFixed(1),
        })));
      }

      setStatusFailed(!sRes?.ok);
      if (sRes?.ok) setStatus(await sRes.json());
      if (aRes?.ok) setAsset(await aRes.json());
      if (uRes?.ok) setUsers(await uRes.json());

      if (srvRes?.ok) {
        const servers = await srvRes.json();
        const found = (servers || []).find(s => s.server_id === serverId);
        if (found) {
          setMeta(found);
          setDraft({
            owner_name: found.owner_name || '',
            department: found.department || '',
            location: found.location || '',
            notes: found.notes || '',
            device_type: found.device_type || 'server',
          });
        }
      }
    } catch (err) {
      setStatusFailed(true);
      console.error('Failed to fetch holding detail:', err);
    } finally {
      setStatusTime(Date.now());
      setLoading(false);
    }
  };

  useEffect(() => {
    setEditing(false);
    setSaveMsg(null);
    fetchDetail();
    const interval = setInterval(fetchDetail, 10000);
    return () => clearInterval(interval);
  }, [serverId, range]);

  const renameServer = async () => {
    setRenaming(true);
    setRenameMsg(null);
    try {
      const res = await fetch(`/api/servers/${serverId}/rename`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ new_server_id: renameDraft.trim() }),
      });
      const data = await res.json();
      if (res.ok) {
        setRenameMsg({ ok: true, text: `Nama diganti menjadi "${data.new_server_id}"` });
        setRenaming(false);
        setRenameDraft('');
        onRenamed(data.new_server_id);
      } else {
        setRenameMsg({ ok: false, text: (data.detail || data.text || 'Gagal ganti nama').replace(/^.*?\n/, '') });
      }
    } catch {
      setRenameMsg({ ok: false, text: 'Gagal terhubung ke collector' });
    } finally {
      setRenaming(false);
    }
  };

  const saveMetadata = async () => {
    setSaving(true);
    setSaveMsg(null);
    try {
      const res = await fetch(`/api/servers/${serverId}/metadata`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(draft),
      });
      if (res.ok) {
        setSaveMsg({ ok: true, text: 'Metadata disimpan' });
        setEditing(false);
        fetchDetail();
      } else {
        const err = await res.text();
        setSaveMsg({ ok: false, text: err || 'Gagal menyimpan metadata' });
      }
    } catch {
      setSaveMsg({ ok: false, text: 'Gagal terhubung ke collector' });
    } finally {
      setSaving(false);
      setTimeout(() => setSaveMsg(null), 4000);
    }
  };

  const cpu = status?.cpu_percent;
  const ram = status?.ram_percent;
  const disk = status?.disk_percent;
  const seal = deviceStatus(status, statusTime, statusFailed);
  const admins = users.filter(u => u.privilege === 'administrator').length;

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 pb-4 border-b border-ink-muted/30">
        <button
          onClick={onBack}
          className="flex items-center gap-2 px-4 py-2 bg-parchment-dark border border-ink-muted/40 text-ink text-[11px] font-mono-ui uppercase tracking-wider hover:bg-ink/5 transition focus-ring rounded-sm self-start"
        >
          <ArrowLeft className="w-4 h-4" strokeWidth={2} />
          Kembali ke daftar
        </button>

        <div className="flex flex-1 flex-col md:flex-row md:items-center justify-between gap-4 md:pl-4">
          <div className="flex items-center gap-4">
            <div className="p-2.5 bg-ink rounded-sm border border-ink">
              <Server className="w-6 h-6 text-gold-leaf" strokeWidth={1.5} />
            </div>
            <div>
              <h2 className="font-display text-2xl font-semibold text-ink m-0">{serverId}</h2>
              <p className="font-mono-ui text-[11px] text-ink-muted m-0">
                {formatDeviceModel(asset?.device_manufacturer, asset?.device_model) || asset?.os_version || 'Windows Estate Node'} · IP: {status?.ip_address || asset?.ip_address || '—'}
              </p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <span className="status-badge" data-tone={seal.tone} title={seal.reason}>{seal.label}</span>
            <div className="flex flex-wrap items-center gap-1.5 bg-parchment-dark border border-ink-muted/30 p-1">
              {['30m', '1h', '6h', '24h', '7d', '30d'].map(r => (
                <button
                  key={r}
                  onClick={() => setRange(r)}
                  className={`px-3 py-1.5 text-[10px] font-mono-ui uppercase tracking-wider transition focus-ring rounded-sm ${
                    range === r ? 'bg-ink text-parchment' : 'text-ink-muted hover:text-ink hover:bg-ink/5'
                  }`}
                >
                  {r}
                </button>
              ))}
            </div>
            <button onClick={fetchDetail} className="p-2 bg-parchment-dark border border-ink-muted/40 text-ink hover:bg-ink/5 transition focus-ring rounded-sm" aria-label="Refresh data">
              <RefreshCw className="w-4 h-4" strokeWidth={2} />
            </button>
          </div>
        </div>
      </div>

      {loading ? (
        <div className="text-center py-16 text-ink-muted font-mono-ui">Consulting the archives...</div>
      ) : (
        <>
          {/* Device Assignment Panel */}
          <section className="ledger-card-elevated p-6">
            <div className="flex items-center justify-between mb-5 pb-3 border-b border-ink-muted/20">
              <h3 className="font-display text-lg font-semibold text-ink flex items-center gap-2 m-0">
                <Crown className="w-5 h-5 text-gold-dim" strokeWidth={1.5} />
                Device Assignment
              </h3>
              {!editing && (
                <button
                  onClick={() => { setEditing(true); setSaveMsg(null); }}
                  className="flex items-center gap-1.5 px-3 py-1.5 text-[10px] font-mono-ui uppercase tracking-wider text-ink-muted border border-ink-muted/40 hover:text-ink hover:border-ink/60 transition focus-ring rounded-sm"
                >
                  <Pencil className="w-3 h-3" strokeWidth={2} />
                  Edit
                </button>
              )}
            </div>

            <div className="mb-6 pb-6 border-b border-ink-muted/20">
              <div className="flex flex-wrap items-end gap-3">
                <div className="flex-1 min-w-[240px]">
                  <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0 mb-1.5">
                    Device Name (server_id)
                  </p>
                  {renaming ? (
                    <input
                      type="text"
                      value={renameDraft}
                      onChange={(e) => setRenameDraft(e.target.value)}
                      onKeyDown={(e) => e.key === 'Enter' && renameDraft.trim() && renameServer()}
                      placeholder={serverId}
                      maxLength={50}
                      autoFocus
                      className="w-full bg-parchment border border-gold px-3 py-2 font-mono-ui text-sm text-ink focus:outline-none focus:border-gold-dim transition focus-ring rounded-sm"
                    />
                  ) : (
                    <p className="font-display text-2xl font-semibold text-ink m-0">{serverId}</p>
                  )}
                  <p className="font-mono-ui text-[10px] text-ink-muted mt-1.5 m-0">
                    Huruf, angka, titik, underscore, dan tanda hubung. Maksimal 50 karakter.
                  </p>
                </div>

                {renaming ? (
                  <div className="flex items-center gap-2">
                    <button
                      onClick={renameServer}
                      disabled={!renameDraft.trim() || renameDraft.trim() === serverId}
                      className="flex items-center gap-2 px-4 py-2 bg-ink text-parchment text-[11px] font-mono-ui uppercase tracking-wider hover:bg-ink/90 transition focus-ring rounded-sm disabled:opacity-40 disabled:cursor-not-allowed"
                    >
                      <Save className="w-4 h-4" strokeWidth={2} />
                      {renaming ? 'Saving...' : 'Rename'}
                    </button>
                    <button
                      onClick={() => { setRenaming(false); setRenameDraft(''); setRenameMsg(null); }}
                      className="flex items-center gap-2 px-4 py-2 text-[11px] font-mono-ui uppercase tracking-wider text-ink-muted border border-ink-muted/40 hover:text-ink transition focus-ring rounded-sm"
                    >
                      <X className="w-4 h-4" strokeWidth={2} />
                      Cancel
                    </button>
                  </div>
                ) : (
                  <button
                    onClick={() => { setRenaming(true); setRenameDraft(serverId); setRenameMsg(null); }}
                    className="flex items-center gap-1.5 px-3 py-2 text-[10px] font-mono-ui uppercase tracking-wider text-ink-muted border border-ink-muted/40 hover:text-ink hover:border-ink/60 transition focus-ring rounded-sm"
                  >
                    <Pencil className="w-3 h-3" strokeWidth={2} />
                    Rename
                  </button>
                )}
              </div>

              {renameMsg && (
                <p className={`font-mono-ui text-[11px] mt-3 mb-0 ${renameMsg.ok ? 'text-moss' : 'text-cardinal'}`}>
                  {renameMsg.text}
                </p>
              )}
            </div>

              {editing ? (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <label className="block">
                    <span className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider">Assigned To</span>
                    <input
                      type="text"
                      value={draft.owner_name}
                      onChange={(e) => setDraft({ ...draft, owner_name: e.target.value })}
                      placeholder="Nama pemegang perangkat"
                      maxLength={100}
                      className="mt-1.5 w-full bg-parchment border border-ink-muted/40 px-3 py-2 text-sm text-ink focus:outline-none focus:border-gold transition focus-ring rounded-sm"
                    />
                  </label>
                  <label className="block">
                    <span className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider">Department</span>
                  <input
                    type="text"
                    value={draft.department}
                    onChange={(e) => setDraft({ ...draft, department: e.target.value })}
                    placeholder="Divisi / departemen"
                    maxLength={100}
                    className="mt-1.5 w-full bg-parchment border border-ink-muted/40 px-3 py-2 text-sm text-ink focus:outline-none focus:border-gold transition focus-ring rounded-sm"
                  />
                </label>
                <label className="block">
                  <span className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider">Location</span>
                  <input
                    type="text"
                    value={draft.location}
                    onChange={(e) => setDraft({ ...draft, location: e.target.value })}
                    placeholder="Ruangan / gedung"
                    maxLength={100}
                    className="mt-1.5 w-full bg-parchment border border-ink-muted/40 px-3 py-2 text-sm text-ink focus:outline-none focus:border-gold transition focus-ring rounded-sm"
                  />
                </label>
                <label className="block">
                  <span className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider">Device Type</span>
                  <select
                    value={draft.device_type}
                    onChange={(e) => setDraft({ ...draft, device_type: e.target.value })}
                    className="mt-1.5 w-full bg-parchment border border-ink-muted/40 px-3 py-2 text-sm text-ink focus:outline-none focus:border-gold transition focus-ring rounded-sm"
                  >
                    <option value="server">Server</option>
                    <option value="pc">PC</option>
                    <option value="laptop">Laptop</option>
                  </select>
                </label>
                <label className="block md:col-span-2">
                  <span className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider">Notes</span>
                  <textarea
                    value={draft.notes}
                    onChange={(e) => setDraft({ ...draft, notes: e.target.value })}
                    placeholder="Catatan steward (opsional)"
                    rows={2}
                    maxLength={255}
                    className="mt-1.5 w-full bg-parchment border border-ink-muted/40 px-3 py-2 text-sm text-ink focus:outline-none focus:border-gold transition focus-ring rounded-sm resize-none"
                  />
                </label>

                <div className="md:col-span-2 flex items-center gap-3 pt-2">
                  <button
                    onClick={saveMetadata}
                    disabled={saving}
                    className="flex items-center gap-2 px-4 py-2 bg-ink text-parchment text-[11px] font-mono-ui uppercase tracking-wider hover:bg-ink/90 transition focus-ring rounded-sm disabled:opacity-50"
                  >
                    <Save className="w-4 h-4" strokeWidth={2} />
                    {saving ? 'Saving...' : 'Save changes'}
                  </button>
                  <button
                    onClick={() => { setEditing(false); setSaveMsg(null); }}
                    className="flex items-center gap-2 px-4 py-2 text-[11px] font-mono-ui uppercase tracking-wider text-ink-muted border border-ink-muted/40 hover:text-ink transition focus-ring rounded-sm"
                  >
                    <X className="w-4 h-4" strokeWidth={2} />
                    Cancel
                  </button>
                  {saveMsg && (
                    <span className={`font-mono-ui text-[11px] ${saveMsg.ok ? 'text-moss' : 'text-cardinal'}`}>
                      {saveMsg.text}
                    </span>
                  )}
                </div>
              </div>
            ) : (
              <div className="grid grid-cols-2 md:grid-cols-4 gap-5">
                <div>
                  <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Assigned To</p>
                  <p className="font-display text-lg font-semibold text-ink mt-1 m-0">{meta?.owner_name || '—'}</p>
                </div>
                <div>
                  <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Department</p>
                  <p className="font-display text-lg font-semibold text-ink mt-1 m-0">{meta?.department || '—'}</p>
                </div>
                <div>
                  <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Location</p>
                  <p className="font-display text-lg font-semibold text-ink mt-1 m-0">{meta?.location || '—'}</p>
                </div>
                <div>
                  <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Device Type</p>
                  <p className="font-display text-lg font-semibold text-ink mt-1 m-0 capitalize">{meta?.device_type || 'server'}</p>
                </div>
                {meta?.notes && (
                  <div className="col-span-2 md:col-span-4 pt-3 border-t border-ink-muted/20">
                    <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Notes</p>
                    <p className="font-mono-ui text-sm text-ink mt-1 m-0">{meta.notes}</p>
                  </div>
                )}
              </div>
            )}
          </section>

          {/* Current Metrics */}
          <section className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="ledger-card p-5">
              <div className="flex items-center gap-2 mb-3">
                <Cpu className="w-4 h-4 text-moss" strokeWidth={1.5} />
                <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Processor Load</p>
              </div>
              <MetricBar label="CPU" value={cpu} />
            </div>
            <div className="ledger-card p-5">
              <div className="flex items-center gap-2 mb-3">
                <Cpu className="w-4 h-4 text-blue-600" strokeWidth={1.5} />
                <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Memory</p>
              </div>
              <MetricBar label="RAM" value={ram} />
            </div>
            <div className="ledger-card p-5">
              <div className="flex items-center gap-2 mb-3">
                <Activity className="w-4 h-4 text-emerald-600" strokeWidth={1.5} />
                <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Storage</p>
              </div>
              <MetricBar label="Disk" value={disk} />
            </div>
          </section>

          {/* Charts */}
          <section className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="ledger-card-elevated p-6">
              <h3 className="font-display text-lg font-semibold text-ink flex items-center gap-2 m-0 mb-5">
                <Cpu className="w-5 h-5 text-moss" strokeWidth={1.5} />
                Resource Utilisation
              </h3>
              <div className="h-72 w-full">
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={metrics} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
                    <CartesianGrid strokeDasharray="2 4" stroke="#2C2621" strokeOpacity={0.1} vertical={false} />
                    <XAxis dataKey="time" {...axisProps} dy={8} />
                    <YAxis {...axisProps} domain={[0, 100]} dx={-8} tickFormatter={v => `${v}%`} />
                    <Tooltip content={<CustomTooltip />} />
                    <Legend wrapperStyle={{ fontSize: '10px', paddingTop: '4px', fontFamily: "'IBM Plex Mono', monospace" }} />
                    <Line type="monotone" dataKey="cpu_percent" name="CPU %" stroke={CHART_COLORS.cpu} strokeWidth={2} dot={false} />
                    <Line type="monotone" dataKey="ram_percent" name="RAM %" stroke={CHART_COLORS.ram} strokeWidth={2} dot={false} />
                  </LineChart>
                </ResponsiveContainer>
              </div>
            </div>

            <div className="ledger-card-elevated p-6">
              <h3 className="font-display text-lg font-semibold text-ink flex items-center gap-2 m-0 mb-5">
                <Activity className="w-5 h-5 text-gold-dim" strokeWidth={1.5} />
                Network Throughput
              </h3>
              <div className="h-72 w-full">
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={metrics} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
                    <CartesianGrid strokeDasharray="2 4" stroke="#2C2621" strokeOpacity={0.1} vertical={false} />
                    <XAxis dataKey="time" {...axisProps} dy={8} />
                    <YAxis {...axisProps} dx={-8} tickFormatter={v => `${v} KB/s`} />
                    <Tooltip content={<CustomTooltip />} />
                    <Legend wrapperStyle={{ fontSize: '10px', paddingTop: '4px', fontFamily: "'IBM Plex Mono', monospace" }} />
                    <Line type="monotone" dataKey="network_sent_kb" name="Sent (KB/s)" stroke={CHART_COLORS.sent} strokeWidth={2} dot={false} />
                    <Line type="monotone" dataKey="network_recv_kb" name="Recv (KB/s)" stroke={CHART_COLORS.recv} strokeWidth={2} dot={false} />
                  </LineChart>
                </ResponsiveContainer>
              </div>
            </div>
          </section>

          {/* Hardware Upgrade Specs & Disk Volumes */}
          {asset && (
            <section className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <div className="ledger-card-elevated p-6 space-y-4">
                <h3 className="font-display text-lg font-semibold text-ink flex items-center gap-2 m-0 pb-3 border-b border-ink-muted/20">
                  <HardDrive className="w-5 h-5 text-gold-dim" strokeWidth={1.5} />
                  Hardware Upgrade Specs &amp; Volumes
                </h3>
                <div className="grid grid-cols-2 gap-4 font-mono-ui text-[11px]">
                  <div>
                    <p className="text-[10px] text-ink-muted uppercase m-0">Model &amp; BIOS</p>
                    <p className="text-ink font-medium mt-0.5 m-0">{formatDeviceModel(asset.device_manufacturer, asset.device_model) || '—'}</p>
                    <p className="text-[10px] text-ink-muted m-0">BIOS: {asset.bios_release_date || '—'} · S/N: {asset.serial_number || '—'}</p>
                  </div>
                  <div>
                    <p className="text-[10px] text-ink-muted uppercase m-0">Processor &amp; GPU</p>
                    <p className="text-ink font-medium mt-0.5 m-0">{asset.processor_model || '—'}</p>
                    <p className="text-[10px] text-ink-muted m-0">GPU: {asset.gpu_model || '—'}</p>
                  </div>
                  <div>
                    <p className="text-[10px] text-ink-muted uppercase m-0">Memory Upgrade Info</p>
                    <p className="text-ink font-medium mt-0.5 m-0">
                      {asset.ram_total_gb?.toFixed(0) || 0} GB {asset.ram_ddr_type || ''} {asset.ram_speed_mhz > 0 ? `@ ${asset.ram_speed_mhz} MHz` : ''}
                    </p>
                    <p className="text-[10px] text-ink-muted m-0">
                      {asset.ram_slots_used}/{asset.ram_slots_total || '?'} slots {asset.ram_form_factor || ''} {asset.ram_max_gb > 0 ? `(Maks ${asset.ram_max_gb.toFixed(0)} GB)` : ''}
                    </p>
                  </div>
                  <div>
                    <p className="text-[10px] text-ink-muted uppercase m-0">Primary Disk</p>
                    <p className="text-ink font-medium mt-0.5 m-0">{asset.disk_type || '—'} · {asset.disk_size_gb?.toFixed(0) || 0} GB</p>
                    <p className="text-[10px] text-ink-muted m-0">
                      SMART: {asset.disk_health_status || 'UNKNOWN'}
                      {asset.disk_health_percent !== null && asset.disk_health_percent !== undefined ? ` (${asset.disk_health_percent}%)` : ''}
                    </p>
                  </div>
                </div>

                {asset.disk_volumes?.length > 0 && (
                  <div className="pt-3 border-t border-ink-muted/20 space-y-2.5">
                    <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Logical Volumes</p>
                    {asset.disk_volumes.map((v) => (
                      <div key={v.drive}>
                        <div className="flex justify-between font-mono-ui text-[11px] mb-1">
                          <span className="text-ink font-medium">{v.drive} ({v.file_system || 'NTFS'})</span>
                          <span className="text-ink-muted">{v.used_gb.toFixed(1)} / {v.total_gb.toFixed(1)} GB ({v.used_percent.toFixed(1)}%)</span>
                        </div>
                        <div className="w-full bg-parchment-dark h-1.5 rounded-sm overflow-hidden border border-ink-muted/20">
                          <div
                            className={`h-full rounded-sm ${v.used_percent >= 90 ? 'bg-cardinal' : v.used_percent >= 80 ? 'bg-gold-dim' : 'bg-moss'}`}
                            style={{ width: `${Math.min(v.used_percent, 100)}%` }}
                          />
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="ledger-card-elevated p-6 space-y-4">
                <h3 className="font-display text-lg font-semibold text-ink flex items-center gap-2 m-0 pb-3 border-b border-ink-muted/20">
                  <ShieldCheck className="w-5 h-5 text-moss" strokeWidth={1.5} />
                  Licensing &amp; Endpoint Security
                </h3>

                <div className="p-3 bg-parchment-dark border border-ink-muted/20 font-mono-ui text-[11px] space-y-1">
                  <p className="text-[10px] text-ink-muted uppercase m-0">Security Posture</p>
                  {asset.security_status?.reported ? (
                    <>
                      <p className="m-0 text-ink">
                        Antivirus: <span className="font-semibold">{(asset.security_status.antivirus_names || []).join(', ') || 'None'}</span>{' '}
                        ({asset.security_status.antivirus_enabled ? 'Enabled' : 'Disabled'} / Realtime {asset.security_status.realtime_protection ? 'ON' : 'OFF'})
                      </p>
                      <p className="m-0 text-ink">
                        Windows Firewall: <span className={asset.security_status.firewall_enabled ? 'text-moss font-semibold' : 'text-cardinal font-semibold'}>
                          {asset.security_status.firewall_enabled ? 'Enabled' : 'Disabled'}
                        </span>
                      </p>
                    </>
                  ) : (
                    <p className="m-0 text-ink-muted">Menunggu laporan agent terbaru</p>
                  )}
                </div>

                <div className="space-y-2">
                  <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">OS &amp; Office Activation</p>
                  {asset.licenses?.length > 0 ? (
                    asset.licenses.map((lic, idx) => {
                      const suspect = (lic.channel || '').toUpperCase().includes('KMS') || lic.license_status !== 'Licensed';
                      return (
                        <div key={idx} className={`p-2.5 border rounded-sm font-mono-ui text-[11px] ${suspect ? 'bg-gold-leaf/15 border-gold-leaf/40' : 'bg-parchment-dark border-ink-muted/20'}`}>
                          <div className="flex justify-between gap-2">
                            <span className="font-semibold text-ink truncate" title={lic.product_name}>{lic.product_name}</span>
                            <span className={lic.license_status === 'Licensed' ? 'text-moss font-semibold' : 'text-cardinal font-semibold'}>
                              {lic.license_status}
                            </span>
                          </div>
                          <p className="text-[10px] text-ink-muted m-0 mt-0.5">
                            Channel: {lic.channel || 'Unknown'} {lic.partial_key ? `· Key: ...-${lic.partial_key}` : ''} {lic.kms_machine ? `· KMS: ${lic.kms_machine}` : ''}
                          </p>
                        </div>
                      );
                    })
                  ) : (
                    <p className="font-mono-ui text-[11px] text-ink-muted m-0">Belum ada data lisensi</p>
                  )}
                </div>
              </div>
            </section>
          )}

          {/* Observed Running Processes */}
          {asset?.running_processes?.length > 0 && (
            <section className="ledger-card-elevated p-6 space-y-4">
              <div className="flex flex-col md:flex-row md:items-center justify-between gap-3 pb-3 border-b border-ink-muted/20">
                <h3 className="font-display text-lg font-semibold text-ink flex items-center gap-2 m-0">
                  <Terminal className="w-5 h-5 text-gold-dim" strokeWidth={1.5} />
                  Observed Running Processes ({asset.running_processes.length})
                </h3>
                <input
                  type="text"
                  placeholder="Cari proses .exe (mis. kmspico, torrent)..."
                  value={procSearch}
                  onChange={(e) => setProcSearch(e.target.value)}
                  className="bg-parchment border border-ink-muted/40 px-3 py-1.5 text-xs font-mono-ui text-ink rounded-sm md:w-72"
                />
              </div>
              <div className="flex flex-wrap gap-1.5 max-h-48 overflow-y-auto">
                {asset.running_processes
                  .filter((p) => !procSearch.trim() || p.toLowerCase().includes(procSearch.toLowerCase()))
                  .map((p) => (
                    <span key={p} className="px-2 py-1 bg-parchment-dark border border-ink-muted/25 font-mono-ui text-[10px] text-ink rounded-sm">
                      {p}
                    </span>
                  ))}
              </div>
            </section>
          )}

          {/* User Accounts & Privileges */}
          <section className="ledger-card-elevated p-6">
            <div className="flex items-center justify-between mb-5 pb-3 border-b border-ink-muted/20">
              <h3 className="font-display text-lg font-semibold text-ink flex items-center gap-2 m-0">
                <KeyRound className="w-5 h-5 text-gold-dim" strokeWidth={1.5} />
                Local User Accounts &amp; Privileges
              </h3>
              <div className="flex items-center gap-3 font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider">
                <span>{users.length} accounts</span>
                {admins > 0 && (
                  <span className="flex items-center gap-1 text-gold-dim">
                    <Crown className="w-3 h-3" strokeWidth={2} />
                    {admins} admin
                  </span>
                )}
              </div>
            </div>

            {users.length === 0 ? (
              <div className="text-center py-8 border border-ink-muted/20 border-dashed">
                <UserX className="w-10 h-10 mx-auto mb-3 opacity-30 text-ink-muted" strokeWidth={1.5} />
                <p className="font-mono-ui text-[11px] text-ink-muted m-0">No local accounts reported</p>
                <p className="font-mono-ui text-[10px] text-ink-muted mt-1 m-0">Agents report accounts every 24 hours</p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse font-mono-ui text-[11px]">
                  <thead>
                    <tr className="border-b border-ink-muted/20 text-ink-muted uppercase tracking-wider">
                      <th className="pb-3 pr-4 text-left font-medium">Username</th>
                      <th className="pb-3 pr-4 text-left font-medium">Full Name</th>
                      <th className="pb-3 pr-4 text-left font-medium">Privilege</th>
                      <th className="pb-3 pr-4 text-left font-medium">State</th>
                      <th className="pb-3 text-left font-medium">Session</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-ink-muted/15">
                    {users.map(u => {
                      const pm = PRIVILEGE_META[u.privilege] || PRIVILEGE_META.limited;
                      const PIcon = pm.icon;
                      return (
                        <tr key={u.username} className="hover:bg-ink/3 transition">
                          <td className="py-3 pr-4 font-medium text-ink">{u.username}</td>
                          <td className="py-3 pr-4 text-ink-muted">{u.full_name || '—'}</td>
                          <td className="py-3 pr-4">
                            <span className={`inline-flex items-center gap-1.5 px-2 py-1 border text-[10px] uppercase tracking-wider rounded-sm ${pm.chip}`}>
                              <PIcon className="w-3 h-3" strokeWidth={2} />
                              {pm.label}
                            </span>
                          </td>
                          <td className="py-3 pr-4">
                            {u.is_disabled ? (
                              <span className="flex items-center gap-1.5 text-ink-muted text-[10px] uppercase tracking-wider">
                                <XCircle className="w-3.5 h-3.5" strokeWidth={2} /> Disabled
                              </span>
                            ) : (
                              <span className="flex items-center gap-1.5 text-moss text-[10px] uppercase tracking-wider">
                                <CheckCircle className="w-3.5 h-3.5" strokeWidth={2} /> Enabled
                              </span>
                            )}
                          </td>
                          <td className="py-3">
                            {u.is_logged_in ? (
                              <span className="inline-flex items-center gap-1.5 px-2 py-1 bg-moss/10 text-moss border border-moss/30 text-[10px] uppercase tracking-wider rounded-sm">
                                <span className="w-1.5 h-1.5 rounded-full bg-moss animate-pulse" />
                                Active
                              </span>
                            ) : (
                              <span className="text-ink-muted text-[10px] uppercase tracking-wider">—</span>
                            )}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </section>

        </>
      )}
    </div>
  );
}
