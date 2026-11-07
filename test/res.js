const { chromium } = require('playwright');
(async () => {
  const browser = await chromium.launch({ executablePath: '/usr/local/bin/chromium', headless: true, args: ['--use-angle=swiftshader', '--enable-unsafe-swiftshader', '--ignore-gpu-blocklist', '--no-sandbox'] });
  const page = await browser.newPage({ viewport: { width: 1280, height: 720 } });
  const state = async () => JSON.parse(await page.evaluate(() => window.tdState || 'null'));
  await page.goto('http://localhost:8080/index.html?' + process.env.Q);
  for (let i = 0; i < 400; i++) { const s = await state(); if (s && s.sw > 0 && s.endT > 3.5) break; await page.waitForTimeout(250); }
  await page.screenshot({ path: process.env.OUT });
  await browser.close();
  process.exit(0);
})();
