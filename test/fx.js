const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch({ executablePath: '/usr/local/bin/chromium', headless: true, args: ['--use-angle=swiftshader', '--enable-unsafe-swiftshader', '--ignore-gpu-blocklist', '--no-sandbox'] });
  const page = await browser.newPage({ viewport: { width: 1280, height: 720 } });
  await page.goto('http://localhost:8080/index.html?fx=1');
  for (let i = 0; i < 100; i++) { const s = await page.evaluate(() => window.tdState || null); if (s) break; await page.waitForTimeout(200); }
  await page.waitForTimeout(1700);
  for (let i = 0; i < 3; i++) { await page.screenshot({ path: `/data/td/shots/30_fx${i}.png`, clip: { x: 100, y: 200, width: 1000, height: 300 } }); await page.waitForTimeout(150); }
  await page.close(); await browser.close(); process.exit(0);
})();
