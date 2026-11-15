// Run after frontend build. Install Playwright separately, then set PLAYWRIGHT_MODULE
// and PLAYWRIGHT_BROWSERS_PATH if it is outside this repository.
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const http = require('node:http');
const fs = require('node:fs/promises');
const path = require('node:path');
const assert = require('node:assert/strict');
const root = path.resolve(__dirname, '../frontend/dist');
const user = { id: 'soak-user', username: 'profiler', is_admin: true };
const server = http.createServer(async (req, res) => {
  try {
    const pathname = new URL(req.url, 'http://localhost').pathname;
    const file = path.resolve(root, '.' + pathname);
    if (!file.startsWith(root + path.sep) && file !== root) { res.writeHead(403).end(); return; }
    let bytes;
    let target = file;
    try { bytes = await fs.readFile(file); } catch { target = path.join(root, 'index.html'); bytes = await fs.readFile(target); }
    const types = { '.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css', '.svg': 'image/svg+xml' };
    res.setHeader('Content-Type', types[path.extname(target)] || 'application/octet-stream'); res.end(bytes);
  } catch { res.writeHead(500).end(); }
});
(async () => {
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  const browser = await chromium.launch({ headless: true });
  try {
    const page = await browser.newPage({ viewport: { width: 1280, height: 900 } });
    const errors = [];
    page.on('pageerror', error => errors.push(error.message));
    await page.addInitScript(user => {
      localStorage.setItem('sn_token', 'synthetic-test-token'); localStorage.setItem('sn_user', JSON.stringify(user));
    }, user);
    await page.route('**/api/**', async route => {
      const url = new URL(route.request().url());
      const values = {
        '/api/auth/me': user, '/api/dashboard': {}, '/api/discovery': {},
        '/api/hearts/details': { tracks: [], albums: [], artists: [], playlists: [], radio_stations: [], podcasts: [] },
        '/api/scan/progress': { status: 'idle', files_scanned: 0 },
      };
      await route.fulfill({ contentType: 'application/json', body: JSON.stringify(values[url.pathname] || []) });
    });
    await page.goto(`http://127.0.0.1:${server.address().port}`);
    const paths = ['/artists', '/albums', '/hearts', '/playlists', '/radio', '/podcasts', '/settings', '/'];
    const cycle = async () => {
      for (const href of paths) {
        await page.locator(`.nav-menu > a[href="${href}"]`).click();
        await page.waitForURL(url => url.pathname === href);
        await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => resolve())));
      }
    };
    const session = await page.context().newCDPSession(page);
    await session.send('Performance.enable');
    const snapshot = async () => {
      await session.send('HeapProfiler.collectGarbage');
      const metrics = await session.send('Performance.getMetrics');
      const dom = await session.send('Memory.getDOMCounters');
      return { heap: metrics.metrics.find(m => m.name === 'JSHeapUsedSize').value, ...dom };
    };
    for (let i = 0; i < 3; i++) await cycle();
    const before = await snapshot();
    const cycles = Number(process.env.SUPERNOVA_BROWSER_CYCLES || 25);
    for (let i = 0; i < cycles; i++) await cycle();
    const after = await snapshot();
    assert.equal(errors.length, 0, errors.join('\n'));
    assert.ok(after.heap < before.heap + 8 * 1024 * 1024, 'retained browser heap grew by more than 8 MiB');
    assert.ok(after.jsEventListeners < before.jsEventListeners + 100, 'event listeners accumulated');
    assert.ok(after.nodes < before.nodes + 3000, 'DOM nodes accumulated');
    const summary = { navigations: cycles * paths.length, before, after, page_errors: errors };
    await fs.writeFile(process.env.SUPERNOVA_BROWSER_OUTPUT || '/tmp/supernova-browser-results.json', JSON.stringify(summary, null, 2));
    console.log(JSON.stringify(summary, null, 2));
  } finally { await browser.close(); server.close(); }
})().catch(error => { console.error(error); server.close(); process.exitCode = 1; });
