const test = require('node:test');
const assert = require('node:assert/strict');
const { normalizeInstanceUrl, sameOrigin } = require('./security');
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
