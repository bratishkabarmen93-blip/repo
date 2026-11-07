// Drives the wasm build in headless Chromium and saves screenshots.
const { chromium } = require('playwright');
const fs = require('fs');

const W = +(process.env.W || 1600), H = +(process.env.H || 900), DPR = +(process.env.DPR || 1);
const SPEED = process.env.SPEED || '1';
const OUT = process.env.OUT || '/data/td/shots';
const script = process.argv[2] || 'basic';

(async () => {
  const browser = await chromium.launch({
    executablePath: '/usr/local/bin/chromium', headless: true,
    args: ['--use-angle=swiftshader', '--enable-unsafe-swiftshader', '--ignore-gpu-blocklist', '--no-sandbox'],
  });
  const page = await browser.newPage({ viewport: { width: W, height: H }, deviceScaleFactor: DPR });
  const logs = [];
  page.on('console', m => logs.push(m.type() + ': ' + m.text()));
  page.on('pageerror', e => logs.push('pageerror: ' + e.message));
  const state = async () => JSON.parse(await page.evaluate(() => window.tdState || 'null'));
  const shot = async (name) => { await page.screenshot({ path: `${OUT}/${name}.png` }); console.log('shot', name, JSON.stringify(await brief())); };
  const brief = async () => { const s = await state(); if (!s) return null; const { slots, buttons, ...rest } = s; return rest; };
  const wait = ms => page.waitForTimeout(ms);
  const key = async k => { await page.keyboard.down(k); await wait(70); await page.keyboard.up(k); };
  const waitFor = async (pred, timeout = 120000) => {
    const t0 = Date.now();
    while (Date.now() - t0 < timeout) { const s = await state(); if (s && pred(s)) return s; await wait(250); }
    throw new Error('timeout waiting');
  };
  const clickSlot = async (i) => { const s = await state(); const p = s.slots[i]; await page.mouse.move(p.X, p.Y); await wait(120); await page.mouse.down(); await wait(60); await page.mouse.up(); await wait(150); };
  const build = async (i, k = 1) => { const s = await state(); const p = s.slots[i]; if (!p) return; await page.mouse.move(p.X, p.Y); await wait(120); await key('Digit' + k); await wait(120); };
  const idle = () => waitFor(s => s.slots.every(p => !p.Busy), 90000);
  const clickBtn = async (id) => { const s = await state(); const b = s.buttons[id]; if (!b) throw new Error('no button ' + id); await page.mouse.move(b[0] + b[2] / 2, b[1] + b[3] / 2); await wait(120); await page.mouse.down(); await wait(60); await page.mouse.up(); await wait(150); };
  try {
    const extra = process.env.Q ? '&' + process.env.Q : '';
    await page.goto(`http://localhost:8080/index.html?speed=${SPEED}${extra}`);
    await waitFor(s => s.sw > 0, 60000);
    await wait(800);
    if (script === 'basic') {
      await shot('01_start');
      const s = await state();
      await page.mouse.move(s.slots[1].X, s.slots[1].Y); await wait(400);
      await shot('02_hover_pad');
      await build(1);
      await build(3);
      await wait(300);
      await idle();
      await clickSlot(3);
      await shot('03_built_selected');
      const s2 = await state();
      await page.mouse.move(s2.buttons.upgrade[0] + 5, s2.buttons.upgrade[1] + 5); await wait(300);
      await shot('04_hover_upgrade');
      await page.mouse.move(W / 2, H / 2);
      await key('Space');
      await wait(9000);
      await shot('05_wave');
      await page.mouse.move(W * 0.3, H * 0.45);
      for (let i = 0; i < 4; i++) { await page.mouse.wheel(0, -120); await wait(80); }
      await wait(900);
      await shot('06_zoomed');
      await page.mouse.move(W * 0.5, H * 0.5); await page.mouse.down(); 
      for (let i = 1; i <= 10; i++) { await page.mouse.move(W * 0.5 - i * 30, H * 0.5 - i * 12); await wait(30); }
      await page.mouse.up(); await wait(300);
      await shot('07_panned');
      await key('Escape'); await wait(200);
      await key('Escape'); await wait(300);
      await shot('08_paused');
    } else if (script === 'lose') {
      await key('Space');
      await wait(700);
      await shot('20_banner');
      await waitFor(s => s.phase === 0, 120000);
      await wait(300);
      await shot('21_cleared');
      for (let k = 0; k < 300; k++) { const s = await state(); if (s.phase >= 2) break; if (s.phase === 0) await key('Space'); await wait(400); }
      await wait(1500);
      await shot('22_lost_early');
      await wait(5000);
      await shot('23_lost');
      await page.setViewportSize({ width: 1100, height: 700 }); await wait(1500);
      await shot('24_resized');
    } else if (script === 'clear') {
      for (const i of [1, 3, 6]) await build(i);
      await page.mouse.move(W / 2, H - 5);
      await key('Space');
      await waitFor(s => s.phase === 0 && s.wave === 1, 240000);
      await wait(300);
      await shot('g1_cleared');
      await wait(1500);
      await shot('g2_cleared_later');
    } else if (script === 'wave') {
      // Q="wave=N&credits=C": build everywhere, upgrade, start, take timed shots.
      const P = process.env.P || 'w';
      await shot(P + '0_prep');
      const s0 = await state();
      if (s0.strip && s0.strip.length) {
        const it = s0.strip[s0.strip.length - 1];
        await page.mouse.move(it[0] + it[2] / 2, it[1] + it[3] / 2); await wait(400);
        await shot(P + '1_strip_tip');
      }
      // TYPES: build menu keys per pad (1 Blaster, 2 Cryo, 3 Flamer, 4 Railgun, 5 Skyguard).
      const LV = +(process.env.LV || 2);
      const types = (process.env.TYPES || '1,1,1,1,1,1,1,1,1,1').split(',').map(Number);
      for (let i = 0; i < 10; i++) if (types[i]) await build(i, types[i]);
      await idle();
      for (let l = 0; l < LV; l++) {
        for (let i = 0; i < 10; i++) { if (!types[i]) continue; await clickSlot(i); await key('KeyU'); }
        await page.mouse.move(W / 2, H - 5);
        await idle();
      }
      await page.mouse.move(W / 2, H - 5); await wait(200);
      await key('Escape');
      await key('Space');
      const times = (process.env.T || '6,14,24').split(',').map(Number);
      let t0 = Date.now(), n = 2;
      for (const t of times) {
        while (Date.now() - t0 < t * 1000) await wait(200);
        await shot(P + (n++) + '_t' + t);
        if (process.env.ENTER && n === 3) await key('Enter');
      }
      if (process.env.ZOOM) {
        const s = await state();
        let ep = (process.env.ZOOM === 'fly' && s.flyPos) ? [s.flyPos] : s.bossPos ? [s.bossPos] : (s.enemyPos || []);
        if (process.env.ZOOM.startsWith('slot:')) { const p = s.slots[+process.env.ZOOM.slice(5)]; ep = [[p.X, p.Y]]; }
        if (ep.length) {
          const e = ep[Math.floor(ep.length / 2)];
          await page.mouse.move(e[0], e[1]);
          for (let i = 0; i < 5; i++) { await page.mouse.wheel(0, -120); await wait(60); }
          await wait(700);
          await shot(P + (n++) + '_zoom');
          for (const dt of (process.env.ZT || '').split(',').filter(Boolean).map(Number)) {
            await wait(dt);
            await shot(P + (n++) + '_zoom');
          }
        }
      }
    } else if (script === 'turrets') {
      // Q="credits=3000": build menu, construction, upgrades, supers, sell and a wave.
      const hoverBtn = async (id) => { const s = await state(); const b = s.buttons[id]; if (!b) throw new Error('no button ' + id); await page.mouse.move(b[0] + b[2] / 2, b[1] + b[3] / 2); await wait(350); };
      const s0 = await state();
      await page.mouse.move(s0.slots[1].X, s0.slots[1].Y); await wait(400);
      await shot('t01_hover_pad');
      await clickSlot(1); await wait(300);
      await shot('t02_menu');
      await hoverBtn('build:blaster');
      await shot('t03_menu_blaster');
      await hoverBtn('build:railgun');
      await shot('t04_menu_railgun');
      await hoverBtn('build:skyguard');
      await shot('t05_menu_skyguard');
      await clickBtn('build:blaster'); await wait(500);
      await shot('t06_building');
      await hoverBtn('cancel');
      await shot('t07_hover_cancel');
      // The rest via hotkeys on hovered pads.
      const plan = [[3, 2], [6, 3], [7, 4], [2, 5], [5, 1], [4, 2], [8, 3], [0, 4], [9, 5]];
      for (const [slot, k] of plan) {
        const s = await state(); const p = s.slots[slot]; if (!p) continue;
        await page.mouse.move(p.X, p.Y); await wait(150);
        await key('Digit' + k); await wait(150);
      }
      await page.mouse.move(W / 2, H - 5); await wait(1200);
      await shot('t08_many_building');
      await waitFor(s => s.slots.every(p => !p.Busy), 60000);
      await wait(500);
      await clickSlot(1);
      await hoverBtn('upgrade');
      await shot('t09_hover_upgrade');
      await clickBtn('upgrade'); await wait(400);
      await shot('t10_upgrading');
      await waitFor(s => !s.slots[1].Busy, 30000);
      await clickBtn('upgrade');
      await waitFor(s => !s.slots[1].Busy, 30000);
      await hoverBtn('upgrade');
      await shot('t11_hover_super');
      await clickBtn('upgrade'); await wait(1500);
      await shot('t12_installing_super');
      await waitFor(s => !s.slots[1].Busy, 30000); await wait(300);
      await page.mouse.move(W / 2, H - 5); await wait(300);
      await shot('t13_super_selected');
      await hoverBtn('target');
      await shot('t14_hover_target');
      await clickBtn('target');
      await hoverBtn('sell');
      await shot('t15_hover_sell');
      await clickBtn('sell'); await wait(200);
      await shot('t16_sell_armed');
      // Upgrade everything to its super.
      for (let round = 0; round < 3; round++) {
        const s = await state();
        for (let i = 0; i < s.slots.length; i++) {
          if (s.slots[i].Tower < 0 || s.slots[i].Busy || s.slots[i].Tower >= 3) continue;
          await clickSlot(i); await key('KeyU');
        }
        await waitFor(s => s.slots.every(p => !p.Busy), 60000);
      }
      await key('Escape'); await page.mouse.move(W / 2, H - 5); await wait(300);
      await shot('t17_all_super');
      await key('Space');
      const times = (process.env.T || '8,16').split(',').map(Number);
      let t0 = Date.now(), n = 18;
      for (const t of times) {
        while (Date.now() - t0 < t * 1000) await wait(200);
        await shot('t' + (n++) + '_wave_t' + t);
        const s = await state(); if (s.cards > 0) await key('Enter');
      }
      const s9 = await state();
      const ep = s9.enemyPos || [];
      if (ep.length) {
        const e = ep[Math.floor(ep.length / 2)];
        await page.mouse.move(e[0], e[1]);
        for (let i = 0; i < 5; i++) { await page.mouse.wheel(0, -120); await wait(60); }
        await wait(700);
        await shot('t' + (n++) + '_zoom');
        await wait(600);
        await shot('t' + (n++) + '_zoom');
      }
    } else if (script === 'ui') {
      // Q="credits=65": affordable/unaffordable options, zoomed and clamped menus, pause screen.
      const hoverBtn = async (id) => { const s = await state(); const b = s.buttons[id]; if (!b) throw new Error('no button ' + id); await page.mouse.move(b[0] + b[2] / 2, b[1] + b[3] / 2); await wait(350); };
      await clickSlot(1); await wait(300);
      await hoverBtn('build:railgun');
      await shot('u01_menu_poor');
      await clickBtn('build:railgun'); await wait(300);
      await shot('u02_not_enough');
      await key('Escape'); await wait(150);
      const s1 = await state();
      await page.mouse.move(s1.slots[1].X, s1.slots[1].Y);
      for (let i = 0; i < 6; i++) { await page.mouse.wheel(0, -120); await wait(60); }
      await wait(900);
      await clickSlot(1); await wait(300);
      await hoverBtn('build:cryo');
      await shot('u03_menu_zoomed');
      // Drag the map so the pad sits at the top edge.
      await key('Escape');
      const s2 = await state();
      const p = s2.slots[1];
      await page.mouse.move(p.X + 40, p.Y + 40); await page.mouse.down();
      for (let i = 1; i <= 10; i++) { await page.mouse.move(p.X + 40, p.Y + 40 - i * (p.Y - 70) / 10); await wait(30); }
      await page.mouse.up(); await wait(400);
      await clickSlot(1); await wait(300);
      await hoverBtn('build:blaster');
      await shot('u04_menu_edge');
      await key('Escape'); await wait(150);
      await key('Home'); await wait(1200);
      await build(3, 1); await wait(300);
      await shot('u05_hotkey_build');
      await key('Escape'); await wait(150);
      await key('Escape'); await wait(400);
      await shot('u06_pause');
    } else if (script === 'sell') {
      // Cancel refunds everything, selling needs a confirm and returns 70%.
      const log = async (tag) => { await wait(600); const s = await state(); console.log(tag, 'credits', s.credits, 'towers', s.towers, 'sel', s.selected, 'arm', s.sellArm); };
      await log('start');
      await build(1, 4); await log('built railgun');
      await clickBtn('cancel'); await log('cancelled');
      await build(1, 1); await idle(); await log('blaster done');
      await clickSlot(1); await key('KeyX'); await log('X once');
      await wait(3300); await log('after 3.3s');
      await key('KeyX'); await key('KeyX'); await log('X twice');
      await build(2, 2); await idle(); await clickSlot(2);
      await clickBtn('sell'); await clickBtn('sell'); await log('sold cryo via buttons');
      await build(3, 3); await wait(300); await key('Escape'); await log('esc deselect');
      await clickSlot(3); await key('KeyX'); await log('X on busy = cancel');
      await shot('s01_end');
    } else if (script === 'full') {
      // Build on several pads, upgrade, play all waves at high speed.
      for (const i of [1, 3]) await build(i);
      await page.mouse.move(W / 2, H - 5);
      await key('Space');
      const order = [6, 7, 2, 5, 4, 8, 0, 9];
      let lastWave = 0;
      for (let k = 0; k < 1500; k++) {
        const s = await state();
        if (s.phase >= 2) break;
        if (s.wave !== lastWave) { lastWave = s.wave; await shot('30_wave' + s.wave); }
        if (s.cards > 0) await key('Enter');
        if (s.credits >= 60 && order.length) { const i = order.shift(); await build(i); }
        else if (s.credits >= 110) {
          const cand = s.slots.map((p, i) => ({ p, i })).filter(o => o.p.Tower >= 0 && o.p.Tower < 2);
          if (cand.length) { await clickSlot(cand[0].i); await key('KeyU'); await page.mouse.move(W / 2, H - 5); }
        }
        if (s.phase === 0) await key('Space');
        await wait(400);
      }
      await wait(5000);
      await shot('31_end');
    }
  } catch (e) {
    console.log('ERROR', e.message);
    await page.screenshot({ path: `${OUT}/error.png` });
  }
  fs.writeFileSync(`${OUT}/console.txt`, logs.join('\n'));
  console.log('console lines:', logs.length, logs.slice(0, 10).join(' | '));
  await page.close();
  await browser.close();
  process.exit(0);
})();
