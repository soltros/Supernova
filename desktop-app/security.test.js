const test = require('node:test');
const assert = require('node:assert/strict');
const { normalizeInstanceUrl, sameOrigin, isLastFmAuthUrl } = require('./security');
test('server addresses allow HTTP(S) and reject local or active schemes', () => {
  assert.equal(normalizeInstanceUrl('localhost:5174/'), 'http://localhost:5174');
  assert.equal(normalizeInstanceUrl('https://music.example/'), 'https://music.example');
  for (const value of ['', null, 'file:///etc/passwd', 'javascript:alert(1)', 'data:text/html,test', 'https://user:secret@host/']) assert.throws(() => normalizeInstanceUrl(value));
});
test('navigation is bound to the exact configured origin', () => {
  assert.equal(sameOrigin('https://music.example/album/1', 'https://music.example'), true);
  assert.equal(sameOrigin('https://music.example.evil/', 'https://music.example'), false);
  assert.equal(sameOrigin('http://music.example/', 'https://music.example'), false);
  assert.equal(sameOrigin('file:///tmp/a', 'file:///tmp/b'), false);
});

test('Last.fm OAuth is restricted to the official HTTPS authorization endpoint', () => {
  assert.equal(isLastFmAuthUrl('https://www.last.fm/api/auth/?api_key=abc&cb=https%3A%2F%2Fmusic.example%2Fsettings'), true);
  assert.equal(isLastFmAuthUrl('https://last.fm/api/auth'), true);
  for (const value of [
    'http://www.last.fm/api/auth',
    'https://last.fm.evil.example/api/auth',
    'https://www.last.fm/music',
    'https://user:pass@www.last.fm/api/auth',
    'javascript:alert(1)'
  ]) assert.equal(isLastFmAuthUrl(value), false);
});
