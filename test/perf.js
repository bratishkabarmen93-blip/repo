// Measures FPS/TPS of the wasm build during a wave.
const { chromium } = require('playwright');
const W = +(process.env.W || 1600), H = +(process.env.H || 900);
(async () => {
  const browser = await chromium.launch({
    executablePath: '/usr/local/bin/chromium', headless: true,
    args: ['--use-angle=swiftshader', '--enable-unsafe-swiftshader', '--ignore-gpu-blocklist', '--no-sandbox'],
  });
  const page = await browser.newPage({ viewport: { width: W, height: H } });
  const state = async () => JSON.parse(await page.evaluate(() => window.tdState || 'null'));
  const wait = ms => page.waitForTimeout(ms);
  const t0 = Date.now();
  await page.goto(`http://localhost:8080/index.html?speed=${process.env.SPEED || 1}`);
  while (true) { const s = await state(); if (s && s.sw > 0) break; await wait(100); }
  console.log('loaded in', Date.now() - t0, 'ms');
  await wait(3000);
  let s = await state(); console.log('idle fps', s.fps.toFixed(1), 'tps', s.tps.toFixed(1));
  await page.keyboard.down('Space'); await wait(70); await page.keyboard.up('Space');
  await wait(8000);
  s = await state(); console.log('wave fps', s.fps.toFixed(1), 'tps', s.tps.toFixed(1), 'enemies', s.enemies);
  await page.close(); await browser.close(); process.exit(0);
})();
