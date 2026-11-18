import { test } from 'node:test';
import assert from 'node:assert/strict';
import { deviceStatus, metricTone, STALE_AFTER_MS } from './deviceStatus.js';

test('status distinguishes missing, expired, failed and healthy telemetry', () => {
  const now = Date.parse('2026-11-17T10:00:00Z');
  const status = { last_seen_at: new Date(now).toISOString(), cpu_percent: 20, ram_percent: 30, disk_percent: 40 };
  assert.equal(deviceStatus(undefined, now).tone, 'unknown');
  assert.equal(deviceStatus({}, now).tone, 'unknown');
  assert.equal(deviceStatus(status, now).tone, 'normal');
  assert.equal(deviceStatus(status, now, true).label, 'Gagal diperbarui');
  assert.equal(deviceStatus(status, now + STALE_AFTER_MS + 1).tone, 'stale');
  assert.equal(deviceStatus({ ...status, ram_percent: null }, now).tone, 'unknown');
  assert.equal(deviceStatus({ ...status, cpu_percent: 75 }, now).tone, 'warning');
  assert.equal(deviceStatus({ ...status, cpu_percent: 90, ram_percent: 85 }, now).tone, 'critical');
});

test('metric boundaries match overall severity for CPU, RAM and disk', () => {
  for (const [metric, warning, critical] of [['cpu', 75, 90], ['ram', 85, 95], ['disk', 85, 95]]) {
    assert.equal(metricTone(metric, warning - 0.1), 'normal');
    assert.equal(metricTone(metric, warning), 'warning');
    assert.equal(metricTone(metric, critical), 'critical');
    assert.equal(metricTone(metric, NaN), 'unknown');
    assert.equal(metricTone(metric, undefined), 'unknown');
    assert.equal(metricTone(metric, 0), 'normal');
  }
});
