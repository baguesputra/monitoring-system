export function formatDeviceModel(manufacturer, model) {
  const mfr = (manufacturer || '').trim();
  const mdl = (model || '').trim();
  if (!mfr && !mdl) return '';
  if (!mfr) return mdl;
  if (!mdl) return mfr;
  if (mdl.toLowerCase().startsWith(mfr.toLowerCase())) return mdl;
  return `${mfr} ${mdl}`;
}
