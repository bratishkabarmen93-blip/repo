// Screens of the game shell (menu, level select, cards, pause, guide, results).
const { chromium } = require('playwright');

const W = +(process.env.W || 1280), H = +(process.env.H || 720);
const OUT = process.env.OUT || '/data/td/shots';
const script = process.argv[2] || 'menu';

(async () => {
  const browser = await chromium.launch({
    executablePath: '/usr/local/bin/chromium', headless: true,
    args: ['--use-angle=swiftshader', '--enable-unsafe-swiftshader', '--ignore-gpu-blocklist', '--no-sandbox'],
  });
  const page = await browser.newPage({ viewport: { width: W, height: H } });
  const logs = [];
  page.on('console', m => logs.push(m.type() + ': ' + m.text()));
  page.on('pageerror', e => logs.push('pageerror: ' + e.message));
  const state = async () => JSON.parse(await page.evaluate(() => window.tdState || 'null'));
  const brief = async () => { const s = await state(); if (!s) return null; const { slots, buttons, enemyPos, strip, saved, ...rest } = s; return rest; };
  const shot = async (name) => { await page.screenshot({ path: `${OUT}/${name}.png` }); console.log('shot', name, JSON.stringify(await brief())); };
  const wait = ms => page.waitForTimeout(ms);
  const key = async k => { await page.keyboard.down(k); await wait(90); await page.keyboard.up(k); await wait(250); };
  const waitFor = async (pred, timeout = 120000) => {
    const t0 = Date.now();
    while (Date.now() - t0 < timeout) { const s = await state(); if (s && pred(s)) return s; await wait(250); }
    throw new Error('timeout waiting');
  };
  const clickAt = async (x, y) => { await page.mouse.move(x, y); await wait(150); await page.mouse.down(); await wait(90); await page.mouse.up(); await wait(350); };
  const clickBtn = async (id) => { const s = await state(); const b = s.buttons[id]; if (!b) throw new Error('no button ' + id + ' in ' + Object.keys(s.buttons)); await clickAt(b[0] + b[2] / 2, b[1] + b[3] / 2); };
  const hoverBtn = async (id) => { const s = await state(); const b = s.buttons[id]; await page.mouse.move(b[0] + b[2] / 2, b[1] + b[3] / 2); await wait(500); };
  const clickSlot = async (i) => { const s = await state(); const p = s.slots[i]; await clickAt(p.X, p.Y); };
  const build = async (i, k = 1) => { const s = await state(); const p = s.slots[i]; await page.mouse.move(p.X, p.Y); await wait(150); await key('Digit' + k); };
  const idle = () => waitFor(s => s.slots.every(p => !p.Busy), 90000);
  try {
    const q = process.env.Q || '';
    await page.goto(`http://localhost:8080/index.html?${q}`);
    await waitFor(s => s.sw > 0, 90000);
    await wait(1500);
    if (script === 'menu') {
      await shot('m01_menu');
      await key('Enter'); await wait(700);
      await clickBtn('ch:0'); await wait(300);
      await shot('m02_chapters');
      await clickBtn('open'); await wait(600);
      await shot('m03_levels');
      await clickBtn('lv:1'); await wait(300);
      await shot('m04_level');
      await clickBtn('lv:6'); await wait(300);
      await shot('m05_boss');
      await key('Escape'); await wait(400); await key('Escape'); await wait(400);
      await clickBtn('settings'); await wait(400);
      await shot('m06_settings');
    } else if (script === 'chapters') {
      await shot('c01_chapters');
      await clickBtn('ch:0'); await wait(300);
      await shot('c02_ch1');
      await clickBtn('ch:2'); await wait(300);
      await shot('c03_skirmish');
      await clickBtn('ch:1'); await wait(300);
      await shot('c04_ch2');
      await clickBtn('open'); await wait(600);
      await shot('c05_levels2');
      await clickBtn('lv:1'); await wait(300);
      await shot('c06_level22');
      await key('Escape'); await wait(500);
      await shot('c07_back');
      await key('Escape'); await wait(500);
      await shot('c08_menu');
    } else if (script === 'twin') {
      await wait(800);
      await shot('t01_lesson');
      await key('Enter'); await wait(600);
      await shot('t02_strip');
      const s = await state();
      if (s.strip && s.strip.length) { const r = s.strip[s.strip.length - 1]; await page.mouse.move(r[0] + r[2] / 2, r[1] + r[3] / 2); await wait(500); }
      await shot('t03_tip');
      if (process.env.GO) { await key('Space'); await wait(+process.env.GO); await shot('t04_wave'); }
    } else if (script === 'peek') {
      await wait(+(process.env.PEEKWAIT || 3000));
      await shot('k01_' + (process.env.TAG || 'x'));
      const s = await state();
      if (s.strip && s.strip.length) { const r = s.strip[s.strip.length - 1]; await page.mouse.move(r[0] + r[2] / 2, r[1] + r[3] / 2); await wait(500); }
      await shot('k02_' + (process.env.TAG || 'x'));
      await page.mouse.move(W / 2, H - 5);
      if (process.env.GO) { await key('Space'); await wait(+process.env.GO); await shot('k03_' + (process.env.TAG || 'x')); }
    } else if (script === 'progress') {
      await shot('l01_progress');
      await key('ArrowRight'); await key('ArrowRight');
      await shot('l02_moved');
    } else if (script === 'lesson') {
      await wait(800);
      await shot('g01_lesson');
      await key('Enter'); await wait(500);
      await clickSlot(0); await wait(600);
      await shot('g02_menu2');
      await key('Escape');
      await wait(400);
      await key('Space');
      await waitFor(s => s.cards > 0, 60000); await wait(800);
      await shot('g03_enemy_card');
    } else if (script === 'pause') {
      await key('Enter');
      await key('Escape'); await wait(300);
      await shot('p01_pause');
      await key('KeyI'); await wait(300);
      await shot('p02_guide_controls');
      await clickBtn('tab:1'); await wait(300);
      await shot('p03_guide_turrets');
      await clickBtn('tab:2'); await wait(300);
      await shot('p04_guide_enemies');
      await clickBtn('entry:' + (process.env.ENTRY || 0)); await wait(300);
      await shot('p05_guide_crawler');
      await key('Escape'); await wait(300);
      await clickBtn('settings'); await wait(300);
      await shot('p06_settings');
      await key('Escape'); await key('Escape'); await wait(300);
      await shot('p07_resumed');
    } else if (script === 'win') {
      await key('Enter');
      const s0 = await state();
      for (let i = 0; i < s0.slots.length; i++) await build(i, 1);
      await idle();
      for (let l = 0; l < 2; l++) {
        for (let i = 0; i < s0.slots.length; i++) { await clickSlot(i); await key('KeyU'); }
        await page.mouse.move(W / 2, H - 5);
        await idle();
      }
      await key('Escape');
      for (let k = 0; k < 600; k++) { const s = await state(); if (s.phase >= 2) break; if (s.phase === 0) await key('Space'); await wait(500); }
      await waitFor(s => s.endT > 1.4, 60000);
      await shot('r01_victory_pop');
      await waitFor(s => s.endT > 3, 60000);
      await shot('r02_victory');
    } else if (script === 'ring') {
      await key('Enter'); await wait(400);
      const s0 = await state();
      const i = +(process.env.SLOT || 2);
      await clickSlot(i); await wait(600);
      await shot('b01_ring_' + (process.env.TAG || 'x'));
    } else if (script === 'split') {
      // Turrets on the given pads (pad:hotkey), then watch the first wave.
      const pads = (process.env.PADS || '4:3,2:1,1:1,5:1').split(',').map(x => x.split(':').map(Number));
      for (const [i, k] of pads) await build(i, k);
      await page.mouse.move(W / 2, H - 5);
      await idle();
      if ((await state()).selected) await key('Escape');
      await key('Space');
      await waitFor(s => s.cards > 0, 90000); await wait(600);
      await shot('s01_card');
      await key('Enter');
      const n = +(process.env.N || 8);
      for (let i = 0; i < n; i++) { await wait(+(process.env.STEP || 1000)); await shot('s' + String(i + 2).padStart(2, '0') + '_play'); }
    } else if (script === 'wyrm') {
      // Turrets on the given pads, then shots of the boss at the given path distances.
      const pads = (process.env.PADS || '').split(',').filter(x => x).map(x => x.split(':').map(Number));
      for (const [i, k] of pads) await build(i, k);
      await page.mouse.move(W / 2, H - 5);
      await idle();
      for (let l = 0; l < +(process.env.UPG || 0); l++) {
        for (const [i] of pads) { await clickSlot(i); await key('KeyU'); }
        await page.mouse.move(W / 2, H - 5);
        await idle();
      }
      if ((await state()).selected) await key('Escape');
      await key('Space');
      const dists = (process.env.DISTS || '3,5,6,8,10.2,11').split(',').filter(x => x).map(Number);
      let n = 0;
      for (const d of dists) {
        for (let k = 0; k < 2400; k++) {
          const s = await state();
          if (s.cards > 0 && !process.env.KEEP) { await key('Enter'); continue; }
          if (s.boss && s.bossDist >= d) break;
          if (s.phase >= 2) break;
          await wait(60);
        }
        n++;
        await shot('w' + String(n).padStart(2, '0') + '_d' + d);
      }
      if (process.env.KILL) {
        // Wait for the boss to be nearly dead, then film its death.
        const low = +(process.env.KILL);
        for (let k = 0; k < 4000; k++) {
          const s = await state();
          if (s.cards > 0) { await key('Enter'); continue; }
          if (!s.boss || s.bossHP < low || s.phase >= 2) break;
          await wait(40);
        }
        for (let i = 0; i < +(process.env.N || 8); i++) { await shot('x' + String(i + 1).padStart(2, '0') + '_kill'); await wait(+(process.env.STEP || 100)); }
      }
    } else if (script === 'cards') {
      // Start the wave and page through the cards it brings.
      await key('Space');
      await waitFor(s => s.cards > 0, 90000);
      await wait(+(process.env.WAIT || 6000));
      for (let i = 0; i < +(process.env.N || 3); i++) { await shot('n0' + (i + 1) + '_card'); await key('Enter'); await wait(500); }
    } else if (script === 'leak') {
      for (let k = 0; k < 4; k++) { const s = await state(); if (!s.cards && !s.lesson) break; await key('Enter'); await wait(300); }
      await key('Space');
      const s1 = await waitFor(s => s.hp < 100, 120000);
      await wait(+(process.env.LEAKWAIT || 200));
      await shot('q01_leak');
      await waitFor(s => s.hp < s1.hp - 12, 120000);
      await wait(+(process.env.LEAKWAIT || 200));
      await shot('q02_leak2');
    } else if (script === 'bridge') {
      // The bridge: overview, pedestals, jars, the holo-table, keys.
      await wait(+(process.env.CWAIT || 1500));
      let s = await state();
      if (!s.bridge) { await clickBtn('loc:0'); await wait(600); s = await state(); }
      await wait(+(process.env.BWAIT || 2500));
      await shot('b01_bridge');
      const P = +(process.env.PED || 0), J = +(process.env.JAR || 0);
      s = await state();
      await page.mouse.move(s.pedestals[P][0], s.pedestals[P][1]); await wait(500);
      await shot('b02_hover');
      await clickAt(s.pedestals[P][0], s.pedestals[P][1]); await wait(250);
      await shot('b03_reveal');
      await wait(2500);
      await shot('b04_selected');
      s = await state();
      await page.mouse.move(s.jars[J][0], s.jars[J][1]); await wait(500);
      await shot('b05_jar_hover');
      await clickAt(s.jars[J][0], s.jars[J][1]); await wait(100);
      await shot('b06a_jar_reveal');
      await wait(2400);
      await shot('b06_jar');
      await key('Escape'); await wait(300);
      await hoverBtn('tsel:' + (process.env.ROW || 5)); await wait(300);
      await shot('b07_row_hover');
      await clickBtn('tsel:' + (process.env.ROW || 5)); await wait(2500);
      await shot('b08_row');
      await key('Escape'); await wait(300);
      s = await state();
      await page.mouse.move(s.table[0], s.table[1]); await wait(2500);
      await shot('b09_table');
      await page.mouse.move(5, 5); await wait(+(process.env.IDLE || 8000));
      await shot('b10_idle');
      await key('Tab'); await wait(1200);
      await shot('b11_tab');
    } else if (script === 'colony') {
      // The colony screen: overview, hover, select, build, the bridge tab.
      const at = async id => (await state()).lots[id];
      await wait(+(process.env.CWAIT || 1500));
      await shot('o01_colony');
      const id = 'mars.' + (process.env.LOT || 'habitat');
      let p = await at(id);
      await page.mouse.move(p[0], p[1]); await wait(500);
      await shot('o02_hover');
      await clickAt(p[0], p[1]); await wait(400);
      await shot('o03_detail');
      await key('Enter'); await wait(900);
      await shot('o04_rising');
      await wait(2200);
      await shot('o05_built');
      await key('Escape'); await wait(300);
      await hoverBtn('csel:5'); await wait(200);
      await shot('o06_overview');
      await hoverBtn('loc:0'); await wait(300);
      await shot('o07_soon');
      if (process.env.ZOOM) {
        p = await at(id);
        await page.mouse.move(p[0], p[1]);
        for (let i = 0; i < +process.env.ZOOM; i++) { await page.mouse.wheel(0, -100); await wait(120); }
        await wait(800);
        await shot('o08_zoom');
      }
    } else if (script === 'lose') {
      await key('Space');
      for (let k = 0; k < 600; k++) { const s = await state(); if (s.phase >= 2) break; if (s.phase === 0) await key('Space'); await wait(500); }
      await waitFor(s => s.endT > 2.5, 60000);
      await shot('r03_defeat');
    }
  } catch (e) {
    console.log('ERROR', e.message);
    await page.screenshot({ path: `${OUT}/error.png` });
  }
  const errs = logs.filter(l => /error|panic/i.test(l));
  if (errs.length) console.log(errs.slice(0, 20).join('\n'));
  await browser.close();
})();
