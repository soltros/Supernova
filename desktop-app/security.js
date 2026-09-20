function normalizeInstanceUrl(value) {
  if (typeof value !== 'string' || !value.trim()) throw new Error('Enter a server address.');
  value = value.trim();
  if (!/^https?:\/\//i.test(value)) {
    if (/^[a-z][a-z0-9+.-]*:/i.test(value) && !/^[^/:]+:\d+(\/|$)/.test(value)) throw new Error('Use an HTTP or HTTPS address.');
    value = 'http://' + value;
  }
  const url = new URL(value);
  if (!['http:', 'https:'].includes(url.protocol) || !url.hostname || url.username || url.password) throw new Error('Invalid server address.');
  url.hash = ''; url.search = '';
  return url.toString().replace(/\/$/, '');
}
function sameOrigin(value, instance) {
  try { return new URL(value).origin === new URL(instance).origin && /^https?:$/.test(new URL(value).protocol); }
  catch { return false; }
}
module.exports = { normalizeInstanceUrl, sameOrigin };
