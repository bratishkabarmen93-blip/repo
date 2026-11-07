const {chromium}=require('playwright');
const assert=require('assert');
(async()=>{
 const browser=await chromium.launch({executablePath:'/usr/local/bin/chromium',headless:true,args:['--use-angle=swiftshader','--enable-unsafe-swiftshader','--ignore-gpu-blocklist','--no-sandbox']});
 const page=await browser.newPage({viewport:{width:1280,height:720}}); const errors=[];page.on('pageerror',e=>errors.push(e.message));
 const state=async()=>JSON.parse(await page.evaluate(()=>window.tdState||'null'));
 const wait=async(f,ms=20000)=>{let t=Date.now();while(Date.now()-t<ms){let s=await state();if(s&&f(s))return s;await page.waitForTimeout(150)}throw Error('timeout '+JSON.stringify(await state()))};
 const click=async(id)=>{let s=await state(),b=s.buttons[id];assert(b,'missing button '+id);await page.mouse.click(b[0]+b[2]/2,b[1]+b[3]/2);await page.waitForTimeout(500)};
 try {
 const q='screen=colony&stars=3,3,3,3,3,3,3,3,3,3,3,3,3,3&crystals=500&colony=europa.habitat:3,europa.greenhouse:2,europa.lab:2,europa.thermal:2,europa.pad:2,europa.monument:1,mars.habitat:1';
 await page.goto('http://localhost:8080/index.html?'+q);await wait(s=>s.sw>0,150000); await wait(s=>s.screen===4,30000);
 await click('loc:2');let s=await wait(s=>s.colTab===2);assert(!s.colLocked);assert.equal(s.colLocation,'europa');await page.waitForTimeout(2000);await page.screenshot({path:'shots/europa/full.png'});console.log('EUROPA',JSON.stringify({tab:s.colTab,locked:s.colLocked,walkers:s.walkers,colony:s.colony,crystals:s.crystals}));
 await click('csel:1');s=await wait(s=>s.colSel==='europa.drill');const before=s.crystals; await click('cbuild');s=await wait(s=>s.colony['europa.drill']===1);assert.equal(s.crystals,before-30);console.log('BOUGHT',s.crystals,'anims',s.anims);
 await click('loc:1');await wait(s=>s.colTab===1);await click('loc:2');s=await wait(s=>s.colTab===2);assert.equal(s.colony['europa.drill'],1);assert.equal(s.colony['mars.habitat'],1);assert.equal(s.crystals,before-30);assert.equal(s.anims,0);await page.waitForTimeout(1500);await page.screenshot({path:'shots/europa/drill.png'});console.log('SWITCH_OK',JSON.stringify({mars:s.colony['mars.habitat'],drill:s.colony['europa.drill'],crystals:s.crystals,anims:s.anims}));
 await page.keyboard.press('Escape');s=await wait(s=>s.screen===0);console.log('MENU',JSON.stringify({fit:s.fit,zoom:s.zoom}));assert(Math.abs(s.zoom-s.fit*1.3)<0.001);assert.deepEqual(errors,[]);console.log('PASS');
 }catch(e){console.error('FAIL',e.stack); console.error('page errors',errors);process.exitCode=1}finally{await browser.close()}
})();
