export const STALE_AFTER_MS = 120_000;

export const STATUS_LABELS = {
  normal: 'Normal', warning: 'Peringatan', critical: 'Kritis',
  stale: 'Data terlambat', unknown: 'Belum tersedia',
};

export function metricTone(name, value) {
  if (typeof value !== 'number' || !Number.isFinite(value)) return 'unknown';
  const [warning, critical] = name === 'cpu' ? [75, 90] : [85, 95];
  return value >= critical ? 'critical' : value >= warning ? 'warning' : 'normal';
}

export function deviceStatus(status, now = Date.now(), failed = false) {
  if (failed) return { tone: 'unknown', label: 'Gagal diperbarui', reason: 'Tidak dapat mengambil status terbaru' };
  const timestamp = Date.parse(status?.last_seen_at);
  if (!Number.isFinite(timestamp) || timestamp <= 0) {
    return { tone: 'unknown', label: STATUS_LABELS.unknown, reason: 'Belum menerima data perangkat' };
  }
  if (now - timestamp > STALE_AFTER_MS) {
    return { tone: 'stale', label: STATUS_LABELS.stale, reason: 'Tidak ada laporan selama lebih dari 2 menit' };
  }
  const metrics = [['cpu', 'CPU'], ['ram', 'RAM'], ['disk', 'Disk']];
  for (const tone of ['critical', 'warning']) {
    const affected = metrics.filter(([key]) => metricTone(key, status[`${key}_percent`]) === tone);
    if (affected.length) {
      return { tone, label: STATUS_LABELS[tone], reason: affected.map(([key, label]) => `${label} ${status[`${key}_percent`].toFixed(1)}%`).join(' · ') };
    }
  }
  if (metrics.some(([key]) => metricTone(key, status[`${key}_percent`]) === 'unknown')) {
    return { tone: 'unknown', label: 'Data belum lengkap', reason: 'Sebagian metrik belum tersedia' };
  }
  return { tone: 'normal', label: STATUS_LABELS.normal, reason: 'Pemakaian dalam batas normal' };
}

export function timeAgo(value, now = Date.now()) {
  const timestamp = Date.parse(value);
  if (!Number.isFinite(timestamp) || timestamp <= 0) return 'Belum ada laporan';
  const seconds = Math.max(0, Math.floor((now - timestamp) / 1000));
  if (seconds < 60) return `${seconds} detik lalu`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)} menit lalu`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)} jam lalu`;
  return `${Math.floor(seconds / 86400)} hari lalu`;
}
