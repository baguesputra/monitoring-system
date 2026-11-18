import React, { useState, useEffect, useCallback } from 'react';
import { AlertTriangle, ArrowRight, CheckCircle2, KeyRound, Plus, RefreshCw, Search, ShieldAlert, ShieldCheck, Trash2 } from 'lucide-react';

const COMPLIANCE_BADGE = {
  violation: { cls: 'wax-seal-critical', label: 'VIOLATION' },
  warning: { cls: 'wax-seal-warning', label: 'WARNING' },
  compliant: { cls: 'wax-seal-normal', label: 'COMPLIANT' },
};

const POLICY_TYPE_STYLE = {
  forbidden: 'bg-cardinal/10 text-cardinal border-cardinal/30',
  required: 'bg-moss/10 text-moss border-moss/30',
  licensed: 'bg-gold-leaf/15 text-gold-dim border-gold-leaf/40',
};

export default function PolicyAudit({ onSelectDevice }) {
  const [audit, setAudit] = useState(null);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('all');
  const [search, setSearch] = useState('');
  const [form, setForm] = useState({ pattern: '', policy_type: 'forbidden', category: '', licensed_seats: 0, notes: '' });
  const [saving, setSaving] = useState(false);
  const [formMsg, setFormMsg] = useState(null);

  const fetchAudit = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/policy-audit');
      if (res.ok) {
        setAudit(await res.json());
      }
    } catch (err) {
      console.error('Failed to load policy audit:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    let active = true;
    fetch('/api/policy-audit')
      .then((res) => (res.ok ? res.json() : null))
      .then((json) => {
        if (active && json) setAudit(json);
      })
      .catch((err) => console.error('Failed to load policy audit:', err))
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => { active = false; };
  }, []);

  const handleAddPolicy = async (e) => {
    e.preventDefault();
    if (!form.pattern.trim()) return;
    setSaving(true);
    setFormMsg(null);
    try {
      const res = await fetch('/api/policies', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...form,
          pattern: form.pattern.trim(),
          licensed_seats: Number(form.licensed_seats) || 0,
        }),
      });
      if (res.ok) {
        setForm({ pattern: '', policy_type: 'forbidden', category: '', licensed_seats: 0, notes: '' });
        setFormMsg({ ok: true, text: 'Aturan policy ditambahkan' });
        fetchAudit();
      } else {
        const msg = await res.text();
        setFormMsg({ ok: false, text: msg || 'Gagal menambahkan policy' });
      }
    } catch {
      setFormMsg({ ok: false, text: 'Gagal terhubung ke collector' });
    } finally {
      setSaving(false);
    }
  };

  const handleDeletePolicy = async (id) => {
    try {
      const res = await fetch(`/api/policies/${id}`, { method: 'DELETE' });
      if (res.ok) fetchAudit();
    } catch (err) {
      console.error('Failed to delete policy:', err);
    }
  };

  const devices = (audit?.devices || []).filter((d) => {
    const matchFilter = filter === 'all' || d.compliance_status === filter;
    const q = search.trim().toLowerCase();
    const matchSearch = !q ||
      d.server_id.toLowerCase().includes(q) ||
      (d.owner_name && d.owner_name.toLowerCase().includes(q)) ||
      (d.department && d.department.toLowerCase().includes(q));
    return matchFilter && matchSearch;
  });

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-4 pb-4 border-b border-ink-muted/30">
        <div>
          <h2 className="font-display text-2xl font-semibold text-ink m-0">Policy &amp; License Audit</h2>
          <p className="font-mono-ui text-[11px] text-ink-muted uppercase tracking-wider mt-1 m-0">
            Software compliance, pirated/KMS activation checks &amp; endpoint security
          </p>
        </div>
        <button
          onClick={fetchAudit}
          className="flex items-center gap-2 px-4 py-2 bg-parchment-dark border border-ink-muted/40 text-ink text-[11px] font-mono-ui uppercase tracking-wider hover:bg-ink/5 transition rounded-sm self-start md:self-auto"
        >
          <RefreshCw className="w-4 h-4" strokeWidth={2} />
          Run Audit
        </button>
      </div>

      {/* Summary Registers */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="ledger-card p-5">
          <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Policy Violations</p>
          <p className="font-display text-3xl font-semibold text-cardinal mt-1 m-0">{audit?.violation_devices || 0}</p>
          <p className="font-mono-ui text-[11px] text-ink-muted mt-2 m-0">{audit?.total_forbidden_hits || 0} forbidden hits</p>
        </div>
        <div className="ledger-card p-5">
          <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Suspect License / KMS</p>
          <p className="font-display text-3xl font-semibold text-gold-dim mt-1 m-0">{audit?.suspect_license_devices || 0}</p>
          <p className="font-mono-ui text-[11px] text-ink-muted mt-2 m-0">KMS / Unlicensed OS &amp; Office</p>
        </div>
        <div className="ledger-card p-5">
          <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Weak Endpoint Security</p>
          <p className="font-display text-3xl font-semibold text-cardinal mt-1 m-0">{audit?.weak_security_devices || 0}</p>
          <p className="font-mono-ui text-[11px] text-ink-muted mt-2 m-0">AV / Realtime / Firewall Off</p>
        </div>
        <div className="ledger-card p-5">
          <p className="font-mono-ui text-[10px] text-ink-muted uppercase tracking-wider m-0">Compliant</p>
          <p className="font-display text-3xl font-semibold text-moss mt-1 m-0">{audit?.compliant_devices || 0}</p>
          <p className="font-mono-ui text-[11px] text-ink-muted mt-2 m-0">Of {audit?.total_devices || 0} devices</p>
        </div>
      </div>

      {/* Licensed Seat Summary */}
      {audit?.license_summaries?.length > 0 && (
        <section className="ledger-card-elevated p-6">
          <h3 className="font-display text-lg font-semibold text-ink m-0 mb-4 flex items-center gap-2">
            <KeyRound className="w-5 h-5 text-gold-dim" />
            Paid Software Seat Audit
          </h3>
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse font-mono-ui text-[11px]">
              <thead>
                <tr className="border-b border-ink-muted/30 text-ink-muted uppercase tracking-wider">
                  <th className="py-2 pr-4">Software Pattern</th>
                  <th className="py-2 pr-4">Category</th>
                  <th className="py-2 pr-4">Installed Count</th>
                  <th className="py-2 pr-4">Licensed Seats</th>
                  <th className="py-2">Devices</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-ink-muted/15">
                {audit.license_summaries.map((ls) => (
                  <tr key={ls.pattern}>
                    <td className="py-2.5 pr-4 font-semibold text-ink">{ls.pattern}</td>
                    <td className="py-2.5 pr-4 text-ink-muted">{ls.category || '—'}</td>
                    <td className={`py-2.5 pr-4 font-bold ${ls.over_allocated ? 'text-cardinal' : 'text-ink'}`}>
                      {ls.installed_on.length} {ls.over_allocated ? '(OVER SEATS!)' : ''}
                    </td>
                    <td className="py-2.5 pr-4 text-ink">{ls.licensed_seats || 'Unlimited / Track Only'}</td>
                    <td className="py-2.5 text-ink-muted">{ls.installed_on.join(', ') || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}

      {/* Device Compliance List */}
      <section className="space-y-4">
        <div className="flex flex-col md:flex-row gap-4 items-start justify-between bg-parchment-dark border border-ink-muted/30 p-4">
          <div className="relative w-full md:w-96">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-ink-muted" strokeWidth={2} />
            <input
              type="text"
              placeholder="Search device, owner, department..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full bg-parchment border border-ink-muted/40 pl-10 pr-4 py-2.5 text-sm text-ink focus:outline-none focus:border-gold rounded-sm"
            />
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {['all', 'violation', 'warning', 'compliant'].map((st) => (
              <button
                key={st}
                onClick={() => setFilter(st)}
                className={`px-3 py-1.5 text-[10px] font-mono-ui uppercase tracking-wider transition rounded-sm ${
                  filter === st ? 'bg-ink text-parchment' : 'text-ink-muted bg-parchment border border-ink-muted/30 hover:text-ink'
                }`}
              >
                {st}
              </button>
            ))}
          </div>
        </div>

        {loading ? (
          <div className="text-center py-12 text-ink-muted font-mono-ui">Auditing software &amp; security posture...</div>
        ) : devices.length === 0 ? (
          <div className="text-center py-12 text-ink-muted bg-parchment-dark border border-ink-muted/30 font-mono-ui text-xs">
            Tidak ada perangkat yang cocok dengan filter.
          </div>
        ) : (
          <div className="space-y-4">
            {devices.map((d) => {
              const badge = COMPLIANCE_BADGE[d.compliance_status] || COMPLIANCE_BADGE.compliant;
              const avNames = d.security_status?.antivirus_names || [];
              return (
                <article key={d.server_id} className="ledger-card-elevated p-5 space-y-4">
                  <div className="flex flex-col md:flex-row md:items-center justify-between gap-3 pb-3 border-b border-ink-muted/20">
                    <div>
                      <div className="flex flex-wrap items-center gap-3">
                        <h3 className="font-display text-lg font-semibold text-ink m-0">{d.server_id}</h3>
                        <span className={`wax-seal ${badge.cls}`}>{badge.label}</span>
                        <span className="font-mono-ui text-[10px] text-ink-muted">
                          {d.installed_apps_count} apps ({d.user_scope_apps_count} user-scope) · {d.running_processes_count} processes
                        </span>
                      </div>
                      <p className="font-mono-ui text-[11px] text-ink-muted mt-1 m-0">
                        {d.owner_name ? `Owner: ${d.owner_name}` : 'Owner belum diisi'}
                        {d.department ? ` · ${d.department}` : ''}
                        {d.admin_users?.length > 0 ? ` · Admin lokal: ${d.admin_users.join(', ')}` : ''}
                      </p>
                    </div>
                    <button
                      onClick={() => onSelectDevice?.(d.server_id)}
                      className="flex items-center gap-1.5 px-3 py-1.5 text-[10px] font-mono-ui uppercase tracking-wider bg-parchment-dark border border-ink-muted/40 text-ink hover:border-gold rounded-sm self-start md:self-auto"
                    >
                      Detail <ArrowRight className="w-3.5 h-3.5" />
                    </button>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4 font-mono-ui text-[11px]">
                    {/* Forbidden & Missing */}
                    <div className="p-3 bg-parchment-dark border border-ink-muted/20 space-y-2">
                      <p className="text-[10px] text-ink-muted uppercase m-0">Policy Findings</p>
                      {d.forbidden_hits?.length > 0 ? (
                        <div className="space-y-1.5">
                          {d.forbidden_hits.map((h, idx) => (
                            <div key={idx} className="p-2 bg-cardinal/10 border border-cardinal/30 rounded-sm">
                              <p className="text-cardinal font-semibold m-0 flex items-center gap-1">
                                <ShieldAlert className="w-3.5 h-3.5" /> {h.matched_name}
                              </p>
                              <p className="text-[10px] text-ink-muted m-0">
                                Rule: {h.policy_pattern} ({h.category || 'Forbidden'}) · {h.source === 'running_process' ? 'Running Process' : `Installed (${h.install_scope})`}
                              </p>
                            </div>
                          ))}
                        </div>
                      ) : d.missing_required?.length > 0 ? null : (
                        <p className="text-moss flex items-center gap-1.5 m-0">
                          <CheckCircle2 className="w-3.5 h-3.5" /> Tidak ada aplikasi terlarang terdeteksi
                        </p>
                      )}
                      {d.missing_required?.length > 0 && (
                        <div className="p-2 bg-gold-leaf/15 border border-gold-leaf/40 rounded-sm">
                          <p className="text-gold-dim font-semibold m-0">Wajib belum terinstal:</p>
                          <p className="text-[10px] text-ink m-0">{d.missing_required.join(', ')}</p>
                        </div>
                      )}
                    </div>

                    {/* Licenses */}
                    <div className="p-3 bg-parchment-dark border border-ink-muted/20 space-y-2">
                      <p className="text-[10px] text-ink-muted uppercase m-0">OS &amp; Office Activation</p>
                      {d.license_findings?.length > 0 ? (
                        <div className="space-y-1.5">
                          {d.license_findings.map((lf, idx) => (
                            <div
                              key={idx}
                              className={`p-2 border rounded-sm ${
                                lf.flag === 'ok'
                                  ? 'bg-moss/5 border-moss/30'
                                  : lf.flag === 'unlicensed'
                                    ? 'bg-cardinal/10 border-cardinal/30'
                                    : 'bg-gold-leaf/15 border-gold-leaf/40'
                              }`}
                            >
                              <p className="font-semibold text-ink m-0 truncate" title={lf.product_name}>
                                {lf.family}: {lf.license_status}
                              </p>
                              <p className="text-[10px] text-ink-muted m-0 truncate" title={lf.channel}>
                                Channel: {lf.channel || 'Unknown'}
                              </p>
                              {lf.reason && <p className="text-[10px] text-cardinal m-0 mt-0.5">{lf.reason}</p>}
                            </div>
                          ))}
                        </div>
                      ) : (
                        <p className="text-ink-muted m-0">Belum ada data lisensi (tunggu agent v2 sinkron)</p>
                      )}
                    </div>

                    {/* Security */}
                    <div className="p-3 bg-parchment-dark border border-ink-muted/20 space-y-2">
                      <p className="text-[10px] text-ink-muted uppercase m-0">Endpoint Security</p>
                      {d.security_status?.reported ? (
                        <div className="space-y-1.5">
                          <p className={`m-0 flex items-center gap-1.5 ${d.security_status.antivirus_enabled && d.security_status.realtime_protection ? 'text-moss' : 'text-cardinal font-semibold'}`}>
                            <ShieldCheck className="w-3.5 h-3.5" />
                            AV: {avNames.length > 0 ? avNames.join(', ') : 'None'} ({d.security_status.realtime_protection ? 'Realtime ON' : 'Realtime OFF'})
                          </p>
                          <p className={`m-0 ${d.security_status.firewall_enabled ? 'text-moss' : 'text-cardinal font-semibold'}`}>
                            Firewall: {d.security_status.firewall_enabled ? 'Enabled' : 'Disabled'}
                          </p>
                          {d.security_warnings?.map((w, idx) => (
                            <p key={idx} className="text-cardinal text-[10px] m-0 flex items-center gap-1">
                              <AlertTriangle className="w-3 h-3" /> {w}
                            </p>
                          ))}
                          {d.admin_users?.length > 1 && (
                            <p className="text-gold-dim text-[10px] m-0">
                              Catatan: {d.admin_users.length} akun punya hak Administrator ({d.admin_users.join(', ')})
                            </p>
                          )}
                        </div>
                      ) : (
                        <p className="text-ink-muted m-0">Belum ada data security (tunggu agent v2 sinkron)</p>
                      )}
                    </div>
                  </div>
                </article>
              );
            })}
          </div>
        )}
      </section>

      {/* Policy Rules Manager */}
      <section className="ledger-card-elevated p-6 space-y-5">
        <div>
          <h3 className="font-display text-lg font-semibold text-ink m-0">Company Software Policy Rules</h3>
          <p className="font-mono-ui text-[11px] text-ink-muted mt-1 m-0">
            Aturan pencocokan substring (case-insensitive) terhadap daftar aplikasi terinstal dan proses berjalan.
          </p>
        </div>

        <form onSubmit={handleAddPolicy} className="grid grid-cols-1 md:grid-cols-6 gap-3 bg-parchment-dark border border-ink-muted/30 p-4">
          <input
            type="text"
            placeholder="Keyword app / .exe (mis. AnyDesk)"
            value={form.pattern}
            onChange={(e) => setForm({ ...form, pattern: e.target.value })}
            required
            className="md:col-span-2 bg-parchment border border-ink-muted/40 px-3 py-2 text-xs text-ink rounded-sm"
          />
          <select
            value={form.policy_type}
            onChange={(e) => setForm({ ...form, policy_type: e.target.value })}
            className="bg-parchment border border-ink-muted/40 px-3 py-2 text-xs text-ink rounded-sm"
          >
            <option value="forbidden">Forbidden (Terlarang)</option>
            <option value="required">Required (Wajib)</option>
            <option value="licensed">Licensed (Hitung Kursi)</option>
          </select>
          <input
            type="text"
            placeholder="Kategori (mis. Remote)"
            value={form.category}
            onChange={(e) => setForm({ ...form, category: e.target.value })}
            className="bg-parchment border border-ink-muted/40 px-3 py-2 text-xs text-ink rounded-sm"
          />
          <input
            type="number"
            min="0"
            placeholder="Seats (khusus licensed)"
            value={form.licensed_seats}
            onChange={(e) => setForm({ ...form, licensed_seats: e.target.value })}
            disabled={form.policy_type !== 'licensed'}
            className="bg-parchment border border-ink-muted/40 px-3 py-2 text-xs text-ink rounded-sm disabled:opacity-40"
          />
          <button
            type="submit"
            disabled={saving}
            className="flex items-center justify-center gap-1.5 px-4 py-2 bg-ink text-parchment text-[11px] font-mono-ui uppercase tracking-wider hover:bg-ink/90 rounded-sm disabled:opacity-50"
          >
            <Plus className="w-4 h-4" /> Add Rule
          </button>
          <input
            type="text"
            placeholder="Catatan alasan policy (opsional)"
            value={form.notes}
            onChange={(e) => setForm({ ...form, notes: e.target.value })}
            className="md:col-span-6 bg-parchment border border-ink-muted/40 px-3 py-2 text-xs text-ink rounded-sm"
          />
        </form>

        {formMsg && (
          <p className={`font-mono-ui text-[11px] m-0 ${formMsg.ok ? 'text-moss' : 'text-cardinal'}`}>
            {formMsg.text}
          </p>
        )}

        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse font-mono-ui text-[11px]">
            <thead>
              <tr className="border-b border-ink-muted/30 text-ink-muted uppercase tracking-wider">
                <th className="py-2 pr-4">Pattern</th>
                <th className="py-2 pr-4">Type</th>
                <th className="py-2 pr-4">Category</th>
                <th className="py-2 pr-4">Seats</th>
                <th className="py-2 pr-4">Notes</th>
                <th className="py-2 text-right">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-ink-muted/15">
              {(audit?.policies || []).map((pol) => (
                <tr key={pol.id}>
                  <td className="py-2.5 pr-4 font-semibold text-ink">{pol.pattern}</td>
                  <td className="py-2.5 pr-4">
                    <span className={`px-2 py-0.5 border text-[10px] uppercase rounded-sm ${POLICY_TYPE_STYLE[pol.policy_type] || ''}`}>
                      {pol.policy_type}
                    </span>
                  </td>
                  <td className="py-2.5 pr-4 text-ink-muted">{pol.category || '—'}</td>
                  <td className="py-2.5 pr-4 text-ink-muted">{pol.policy_type === 'licensed' ? pol.licensed_seats : '—'}</td>
                  <td className="py-2.5 pr-4 text-ink-muted">{pol.notes || '—'}</td>
                  <td className="py-2.5 text-right">
                    <button
                      onClick={() => handleDeletePolicy(pol.id)}
                      className="p-1.5 text-ink-muted hover:text-cardinal transition"
                      aria-label={`Delete ${pol.pattern}`}
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
