package main

import (
	"image"
	"image/color"
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"outpost/internal/art"
	"outpost/internal/game"
)

const (
	headAngles  = 64
	spinAngles  = 16 // heads that only spin repeat every quarter turn
	enemyAngles = 32
	boltAngles  = 32
	radarAngles = 64
	towerLevels = art.TurretLevels
)

// towerLook is everything the renderer needs to show one turret type.
type towerLook struct {
	def    *game.TowerDef
	base   [towerLevels]sprite
	head   [towerLevels][]sprite
	period float64 // the head sprites cover this much rotation
	tipFwd [towerLevels]float64
	tipZ   [towerLevels]float64 // above the pad top
	topZ   [towerLevels]float64 // highest point above the pad top (measured)
	icon   sprite               // build menu portrait
	card   sprite               // big portrait for cards and the guide
	accent color.RGBA
	glow   string
}

// headAt returns the head sprite of level lv turned to angle a.
func (l *towerLook) headAt(lv int, a float64) sprite {
	n := len(l.head[lv])
	i := int(math.Round(a / l.period * float64(n)))
	i %= n
	if i < 0 {
		i += n
	}
	return l.head[lv][i]
}

// tip is the world point (pixels, height above the pad top) where shots leave a turret.
func (l *towerLook) tip(t *game.Tower, lv int) (float64, float64, float64) {
	P := float64(art.CellPx)
	f := l.tipFwd[lv]
	return t.Pos.X*P + math.Cos(t.Angle)*f, t.Pos.Y*P + math.Sin(t.Angle)*f, l.tipZ[lv]
}

var towerAccents = map[string]struct {
	col  color.RGBA
	glow string
}{
	"blaster":  {art.Cyan, "cyan"},
	"cryo":     {art.Cyan, "ice"},
	"flamer":   {art.Orange, "orange"},
	"railgun":  {art.Lilac, "violet"},
	"skyguard": {art.Red, "red"},
	"beacon":   {art.Amber, "amber"},
}

// enemyLook is everything the renderer needs to show one enemy type.
type enemyLook struct {
	def     *game.EnemyDef
	spr     [enemyAngles][]sprite
	frames  int
	perCell float64 // animation frames per cell travelled
	fps     float64 // time-based animation instead (rotors)
	bodyZ   float64 // where bolts hit and effects start
	topZ    float64 // highest point of the model (measured)
	barW    float64 // health bar width (world px)
	hover   float64 // flight height
	size    float64 // effect size factor
	robot   bool
	goo     []color.RGBA // death burst colours
	glow    string       // glow colour of the death burst
	splats  []*ebiten.Image
	icon    sprite // small portrait for the wave preview
	card    sprite // big portrait for the "new enemy" card
}

type lookSpec struct {
	id            string
	frames        int
	perCell, fps  float64
	bodyZ, barW   float64
	hover, size   float64
	robot         bool
	goo           []color.RGBA
	glow          string
	dark, mid, hi color.RGBA
}

var lookSpecs = []lookSpec{
	{id: "crawler", frames: 6, perCell: 10, bodyZ: 4.5, barW: 14, size: 1,
		goo: []color.RGBA{art.Lime, art.Green, art.Yellow, art.Forest}, glow: "green", dark: art.Forest, mid: art.Green, hi: art.Lime},
	{id: "runner", frames: 6, perCell: 7.5, bodyZ: 4.5, barW: 12, size: 0.9,
		goo: []color.RGBA{art.Amber, art.Ochre, art.Yellow, art.Rust}, glow: "amber", dark: art.Rust, mid: art.Ochre, hi: art.Amber},
	{id: "swarmling", frames: 4, perCell: 12, bodyZ: 2.6, barW: 9, size: 0.6,
		goo: []color.RGBA{art.Pink, art.Magenta, art.Peach, art.Purple}, glow: "purple", dark: art.Plum, mid: art.Purple, hi: art.Magenta},
	{id: "walker", frames: 8, perCell: 17.6, bodyZ: 14, barW: 18, size: 1.4, robot: true},
	{id: "drone", frames: 4, fps: 20, bodyZ: 4, barW: 14, hover: 17, size: 1, robot: true},
	{id: "queen", frames: 8, perCell: 22.2, bodyZ: 12, barW: 30, size: 2.6,
		goo: []color.RGBA{art.Magenta, art.Purple, art.Lime, art.Pink, art.Plum}, glow: "purple", dark: art.Purple, mid: art.Magenta, hi: art.Lime},
	{id: "beetle", frames: 6, perCell: 9, bodyZ: 5.5, barW: 16, size: 1.15,
		goo: []color.RGBA{art.Cyan, art.Blue, art.White, art.DeepBlue}, glow: "ice", dark: art.DeepBlue, mid: art.Blue, hi: art.Cyan},
	{id: "splitter", frames: 6, perCell: 9.5, bodyZ: 5, barW: 15, size: 1.1,
		goo: []color.RGBA{art.Violet, art.Lilac, art.Purple, art.White}, glow: "violet", dark: art.Purple, mid: art.Violet, hi: art.Lilac},
	{id: "shard", frames: 4, perCell: 11, bodyZ: 3, barW: 9, size: 0.7,
		goo: []color.RGBA{art.Lilac, art.Violet, art.White}, glow: "violet", dark: art.Purple, mid: art.Violet, hi: art.Lilac},
	{id: "stinger", frames: 4, fps: 26, bodyZ: 4, barW: 11, hover: 13, size: 0.8,
		goo: []color.RGBA{art.Amber, art.Yellow, art.Ochre, art.Cyan}, glow: "amber", dark: art.Brown, mid: art.Ochre, hi: art.Amber},
	{id: "wyrm", frames: 8, fps: 5, bodyZ: 10, barW: 30, size: 2.4,
		goo: []color.RGBA{art.Cyan, art.Blue, art.White, art.DeepBlue}, glow: "ice", dark: art.DeepBlue, mid: art.Blue, hi: art.Cyan},
	{id: "bloater", frames: 6, perCell: 14, bodyZ: 7.5, barW: 18, size: 1.35,
		goo: []color.RGBA{art.Amber, art.Ochre, art.Yellow, art.Tan}, glow: "amber", dark: art.Rust, mid: art.Ochre, hi: art.Amber},
	{id: "stalker", frames: 6, perCell: 8, bodyZ: 5.5, barW: 13, size: 1,
		goo: []color.RGBA{art.Cyan, art.Silver, art.White, art.DeepBlue}, glow: "cyan", dark: art.DeepBlue, mid: art.Steel, hi: art.Cyan},
	{id: "kraken", frames: 8, perCell: 20, bodyZ: 9, barW: 30, size: 2.8,
		goo: []color.RGBA{art.Magenta, art.Purple, art.Pink, art.Amber, art.Plum}, glow: "purple", dark: art.Purple, mid: art.Magenta, hi: art.Pink},
}

type glyph struct {
	img *ebiten.Image
	adv int
}

// sprite is a baked 2.5D model: the image, its white silhouette (for flashes),
// the anchor (image pixel of the model origin) and the ground shadow mask.
type sprite struct {
	img, sil *ebiten.Image
	ox, oy   float64
	sh       *ebiten.Image
	sx, sy   float64
}

// mark is a flat overlay with an anchor.
type mark struct {
	img    *ebiten.Image
	ox, oy float64
}

// terrainSet is the baked ground of one level.
type terrainSet struct {
	img *ebiten.Image
	ti  *art.TerrainInfo
}

// Assets holds every texture, generated procedurally at start-up.
type Assets struct {
	Terrain   *ebiten.Image // ground of the level on screen
	TI        *art.TerrainInfo
	Terrains  map[string]terrainSet // by level ID
	PadMarker mark
	Towers    map[*game.TowerDef]*towerLook
	Muzzle    *ebiten.Image
	Bolt      [3][boltAngles]*ebiten.Image
	Missile   [boltAngles]*ebiten.Image
	Flake     *ebiten.Image
	Glow      map[string]*ebiten.Image
	SoftGlow  *ebiten.Image // smooth amber halo for the result stars
	Looks     map[*game.EnemyDef]*enemyLook
	WyrmParts [art.WyrmKinds][enemyAngles]sprite // the Ice Wyrm's body behind its head
	Scorch    [3]*ebiten.Image
	Plus      *ebiten.Image
	Shield    *ebiten.Image
	Base      sprite
	BaseEast  sprite // with a second gate on the east side
	Radar     [radarAngles]sprite
	Icons     map[string]*ebiten.Image
	Glyphs    map[rune]glyph
	White     *ebiten.Image
	rings     map[ringKey]mark
	Colonies  map[*game.Location]*colonyLook
	Colonists [2][colonistAngles][colonistFrames]sprite // by suit colour
	Shuttle   sprite
	Bridge    *bridgeLook
}

// bridgeLook is the bridge of the Vanguard baked for the colony screen.
type bridgeLook struct {
	ter       terrainSet
	holo      [][pedAngles]sprite // per trophy: its projection on the pedestal as it turns
	ghost     []sprite            // per trophy: the dim outline of one not won yet
	jar       [2]jarLook          // empty, full
	specimens []sprite            // per specimen: the enemy in its jar (anchor at the bottom)
	crew      [2]sprite
}

// jarLook is a specimen jar: the liquid behind the specimen and the glass
// in front of it, both anchored at the bottom of the jar.
type jarLook struct {
	back, front *ebiten.Image
	ox, oy      float64
}

// colonyLook is a colony location baked for the colony screen.
type colonyLook struct {
	ter    terrainSet
	stages map[*game.Building][]sprite         // by stage - 1
	fx     map[*game.Building][]art.BuildingFX // by stage - 1
	holo   map[*game.Building]*ebiten.Image    // projection of the first stage (its anchor)
}

type ringKey struct {
	r   int
	col color.RGBA
}

func toImage(c *art.Canvas) *ebiten.Image { return ebiten.NewImageFromImage(c.Img) }

func toSprite(s *art.Sprite) sprite {
	sp := sprite{img: toImage(s.C), sil: toImage(s.C.Silhouette()), ox: float64(s.OX), oy: float64(s.OY)}
	if s.Sh != nil {
		sp.sh, sp.sx, sp.sy = toImage(s.Sh), float64(s.SX), float64(s.SY)
	}
	return sp
}

func angleOf(i, n int) float64 { return float64(i) / float64(n) * 2 * math.Pi }

// angleIndex maps an angle in radians to the nearest of n pre-rendered rotations.
func angleIndex(a float64, n int) int {
	i := int(math.Round(a / (2 * math.Pi) * float64(n)))
	i %= n
	if i < 0 {
		i += n
	}
	return i
}

// UseTerrain switches the ground to the level with the given ID.
func (a *Assets) UseTerrain(id string) {
	t := a.Terrains[id]
	a.Terrain, a.TI = t.img, t.ti
}

// NewAssets bakes every model and the ground of every level.
func NewAssets(levels []*game.Level) *Assets {
	pm, px, py := art.PadMarker()
	a := &Assets{
		Terrains:  map[string]terrainSet{},
		PadMarker: mark{toImage(pm), float64(px), float64(py)},
		Muzzle:    toImage(art.Muzzle()),
		Glow:      map[string]*ebiten.Image{},
		Icons:     map[string]*ebiten.Image{},
		Glyphs:    map[rune]glyph{},
		rings:     map[ringKey]mark{},
	}
	// Bake all models in parallel (the slowest first), then upload them.
	nt := len(art.TurretIDs)
	var (
		bases   = make([][towerLevels]*art.Sprite, nt)
		heads   = make([][towerLevels][]*art.Sprite, nt)
		ticons  = make([]*art.Sprite, nt)
		tcards  = make([]*art.Sprite, nt)
		grounds = make([]*art.Canvas, len(levels))
		tinfos  = make([]*art.TerrainInfo, len(levels))
		radars  [radarAngles]*art.Sprite
		base    *art.Sprite
		baseE   *art.Sprite
		jobs    []func()
	)
	for i, l := range levels {
		jobs = append(jobs, func() { grounds[i], tinfos[i] = art.Terrain(l) })
	}
	baked := make([][enemyAngles][]*art.Sprite, len(lookSpecs))
	icons := make([][2]*art.Sprite, len(lookSpecs))
	// The slowest bakes (kraken, wyrm, queen, bloater, walker, beetle, splitter) start first.
	var wyrm [art.WyrmKinds][enemyAngles]*art.Sprite
	for k := 0; k < art.WyrmKinds; k++ {
		for h := 0; h < enemyAngles; h++ {
			jobs = append(jobs, func() { wyrm[k][h] = art.WyrmPart(k, angleOf(h, enemyAngles)) })
		}
	}
	var order []int
	for _, id := range []string{"kraken", "wyrm", "queen", "bloater", "walker", "beetle", "splitter"} {
		for li, ls := range lookSpecs {
			if ls.id == id {
				order = append(order, li)
			}
		}
	}
	for li := range lookSpecs {
		if !slices.Contains(order, li) {
			order = append(order, li)
		}
	}
	for _, li := range order {
		ls := lookSpecs[li]
		for h := 0; h < enemyAngles; h++ {
			baked[li][h] = make([]*art.Sprite, ls.frames)
			for f := 0; f < ls.frames; f++ {
				jobs = append(jobs, func() {
					baked[li][h][f] = art.EnemySprite(ls.id, angleOf(h, enemyAngles), f, ls.frames)
				})
			}
		}
		jobs = append(jobs, func() {
			icons[li] = [2]*art.Sprite{art.EnemyPortrait(ls.id, 18, 14), art.EnemyPortrait(ls.id, 44, 34)}
		})
	}
	for ti, id := range art.TurretIDs {
		n, per := headAngles, art.TurretSpin(id)
		if per < 2*math.Pi-1e-9 {
			n = spinAngles
		}
		for lv := 0; lv < towerLevels; lv++ {
			heads[ti][lv] = make([]*art.Sprite, n)
			jobs = append(jobs, func() { bases[ti][lv] = art.TurretBase(id, lv) })
			for i := 0; i < n; i++ {
				jobs = append(jobs, func() { heads[ti][lv][i] = art.TurretHead(id, lv, float64(i)/float64(n)*per) })
			}
		}
		jobs = append(jobs, func() { ticons[ti] = art.TurretIcon(id, 0, 24, 22, 1.35) })
		jobs = append(jobs, func() { tcards[ti] = art.TurretIcon(id, 0, 44, 36, 1.2) })
	}
	for i := 0; i < radarAngles; i++ {
		jobs = append(jobs, func() { radars[i] = art.Radar(angleOf(i, radarAngles)) })
	}
	jobs = append(jobs, func() { base = art.Base(false) }, func() { baseE = art.Base(true) })
	// The colonies: ground, every stage of every building, settlers, shuttle.
	type colonyBake struct {
		ground *art.Canvas
		ti     *art.TerrainInfo
		stages map[*game.Building][]*art.Sprite
	}
	colBakes := map[*game.Location]*colonyBake{}
	for _, loc := range game.Colonies {
		if loc.Soon {
			continue
		}
		cb := &colonyBake{stages: map[*game.Building][]*art.Sprite{}}
		colBakes[loc] = cb
		jobs = append(jobs, func() { cb.ground, cb.ti = art.ColonyTerrain(loc) })
		for _, b := range loc.Buildings {
			sp := make([]*art.Sprite, len(b.Stages))
			cb.stages[b] = sp
			for st := range sp {
				jobs = append(jobs, func() { sp[st] = art.Building(loc, b, st+1) })
			}
		}
	}
	var settlers [2][colonistAngles][colonistFrames]*art.Sprite
	for v := range settlers {
		for h := range settlers[v] {
			for f := range settlers[v][h] {
				jobs = append(jobs, func() {
					settlers[v][h][f] = art.ColonistScene(v, angleOf(h, colonistAngles), f, colonistFrames).Bake(art.Opts{Shadows: true}, true, 0, 0)
				})
			}
		}
	}
	var shuttle *art.Sprite
	jobs = append(jobs, func() { shuttle = art.ShuttleScene(math.Pi).Bake(art.Opts{Inner: 3, Shadows: true}, true, 0, 0) })
	// The bridge: the room, the holograms on the pedestals, specimens, crew.
	var (
		bridgeC  *art.Canvas
		bridgeTI *art.TerrainInfo
		pedHolo  = make([][pedAngles]*art.Canvas, len(game.Trophies))
		pedGhost = make([]*art.Canvas, len(game.Trophies))
		pedAnch  = make([][pedAngles][2]int, len(game.Trophies))
		specs    = make([]*art.Sprite, len(game.Specimens))
		crew     [2]*art.Sprite
	)
	jobs = append(jobs, func() { bridgeC, bridgeTI = art.BridgeTerrain() })
	for i, t := range game.Trophies {
		jobs = append(jobs, func() {
			k := art.TrophyFit(t.Model, 30, 34)
			for h := 0; h < pedAngles; h++ {
				sp := art.TrophyModel(t.Model, k, angleOf(h, pedAngles)+0.5)
				pedHolo[i][h] = art.Holo(sp.C, 0.8)
				pedAnch[i][h] = [2]int{sp.OX, sp.OY}
				if h == 0 {
					pedGhost[i] = art.Ghost(sp.C)
				}
			}
		})
	}
	for i, t := range game.Specimens {
		jobs = append(jobs, func() { specs[i] = art.EnemyPortrait(t.Enemy.ID, 17, 19) })
	}
	for v := range crew {
		jobs = append(jobs, func() {
			crew[v] = art.ColonistScene(v, art.CrewHeading[v], 0, colonistFrames).BakeScaled(art.Opts{Inner: 1.5, Shadows: true}, 1.9)
		})
	}
	art.Parallel(len(jobs), func(i int) { jobs[i]() })
	a.Towers = map[*game.TowerDef]*towerLook{}
	for ti, id := range art.TurretIDs {
		def := game.TowerDefs[id]
		ac := towerAccents[id]
		lk := &towerLook{def: def, period: art.TurretSpin(id), icon: toSprite(ticons[ti]), card: toSprite(tcards[ti]), accent: ac.col, glow: ac.glow}
		for lv := 0; lv < towerLevels; lv++ {
			lk.base[lv] = toSprite(bases[ti][lv])
			top := bases[ti][lv].OY
			for _, h := range heads[ti][lv] {
				lk.head[lv] = append(lk.head[lv], toSprite(h))
				top = max(top, h.OY)
			}
			lk.topZ[lv] = float64(top) / art.HZ
			lk.tipFwd[lv], lk.tipZ[lv] = art.TurretTip(id, lv)
		}
		a.Towers[def] = lk
	}
	for lv := 0; lv < 3; lv++ {
		for i := 0; i < boltAngles; i++ {
			a.Bolt[lv][i] = toImage(art.Bolt(lv, angleOf(i, boltAngles)))
		}
	}
	for i := 0; i < boltAngles; i++ {
		a.Missile[i] = toImage(art.Missile(angleOf(i, boltAngles)))
	}
	a.Flake = toImage(art.Flake())
	a.Looks = map[*game.EnemyDef]*enemyLook{}
	for li, ls := range lookSpecs {
		def := game.EnemyDefs[ls.id]
		lk := &enemyLook{
			def: def, frames: ls.frames, perCell: ls.perCell, fps: ls.fps, bodyZ: ls.bodyZ, barW: ls.barW,
			hover: ls.hover, size: ls.size, robot: ls.robot, goo: ls.goo, glow: ls.glow,
			icon: toSprite(icons[li][0]), card: toSprite(icons[li][1]),
		}
		top := 0
		for h := 0; h < enemyAngles; h++ {
			lk.spr[h] = make([]sprite, ls.frames)
			for f := 0; f < ls.frames; f++ {
				lk.spr[h][f] = toSprite(baked[li][h][f])
				top = max(top, baked[li][h][f].OY)
			}
		}
		lk.topZ = float64(top) / art.HZ
		if !ls.robot {
			for v := 0; v < 4; v++ {
				lk.splats = append(lk.splats, toImage(art.GooSplat(v, ls.dark, ls.mid, ls.hi)))
			}
		}
		a.Looks[def] = lk
	}
	for k := range wyrm {
		for h := range wyrm[k] {
			a.WyrmParts[k][h] = toSprite(wyrm[k][h])
		}
	}
	for i := range radars {
		a.Radar[i] = toSprite(radars[i])
	}
	a.Base = toSprite(base)
	a.BaseEast = toSprite(baseE)
	for i := range a.Scorch {
		a.Scorch[i] = toImage(art.Scorch(i))
	}
	a.Plus = toImage(art.Plus(art.White))
	a.Shield = toImage(art.Shield())
	for name, col := range map[string]color.RGBA{
		"cyan": art.Cyan, "red": art.Red, "green": art.Lime, "purple": art.Magenta,
		"amber": art.Amber, "white": art.White, "orange": art.Orange,
		"ice": art.Hex(0x9fe8ff), "violet": art.Violet, "fire": art.Red,
	} {
		a.Glow[name] = toImage(art.Glow(col))
	}
	a.SoftGlow = softGlow(64, art.Amber)
	for i, l := range levels {
		a.Terrains[l.Spec.ID] = terrainSet{toImage(grounds[i]), tinfos[i]}
	}
	for _, n := range []string{"hp", "cr", "wave", "time", "star", "star0", "lock", "info", "target", "super", "entrances", "crystal", "up", "check", "trophy", "fog", "kraken"} {
		a.Icons[n] = toImage(art.Icon(n))
	}
	for r, g := range art.Glyphs() {
		a.Glyphs[r] = glyph{img: toImage(g.C), adv: g.Advance}
	}
	a.Colonies = map[*game.Location]*colonyLook{}
	for loc, cb := range colBakes {
		cl := &colonyLook{
			ter:    terrainSet{toImage(cb.ground), cb.ti},
			stages: map[*game.Building][]sprite{},
			fx:     map[*game.Building][]art.BuildingFX{},
			holo:   map[*game.Building]*ebiten.Image{},
		}
		for b, sps := range cb.stages {
			for st, sp := range sps {
				cl.stages[b] = append(cl.stages[b], toSprite(sp))
				cl.fx[b] = append(cl.fx[b], art.Effects(loc, b, st+1))
			}
			cl.holo[b] = toImage(art.Hologram(sps[0].C))
		}
		a.Colonies[loc] = cl
	}
	for v := range settlers {
		for h := range settlers[v] {
			for f := range settlers[v][h] {
				a.Colonists[v][h][f] = toSprite(settlers[v][h][f])
			}
		}
	}
	a.Shuttle = toSprite(shuttle)
	bl := &bridgeLook{ter: terrainSet{toImage(bridgeC), bridgeTI}}
	for i := range game.Trophies {
		var hs [pedAngles]sprite
		for h := range hs {
			hs[h] = sprite{img: toImage(pedHolo[i][h]), ox: float64(pedAnch[i][h][0]), oy: float64(pedAnch[i][h][1])}
		}
		bl.holo = append(bl.holo, hs)
		bl.ghost = append(bl.ghost, sprite{img: toImage(pedGhost[i]), ox: hs[0].ox, oy: hs[0].oy})
	}
	for k, full := range []bool{false, true} {
		j := art.JarLayers(full)
		bl.jar[k] = jarLook{toImage(j.Back), toImage(j.Front), float64(j.OX), float64(j.OY)}
	}
	for _, sp := range specs {
		bl.specimens = append(bl.specimens, sprite{img: toImage(sp.C), ox: float64(sp.C.W) / 2, oy: float64(sp.C.H)})
	}
	for v := range crew {
		bl.crew[v] = toSprite(crew[v])
	}
	a.Bridge = bl
	w := ebiten.NewImage(3, 3)
	w.Fill(color.White)
	a.White = w.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
	return a
}

// Ring returns a cached range ellipse with radius r world pixels.
func (a *Assets) Ring(r int, col color.RGBA) mark {
	k := ringKey{r, col}
	if m, ok := a.rings[k]; ok {
		return m
	}
	c, ox, oy := art.RangeRing(r, col)
	m := mark{toImage(c), float64(ox), float64(oy)}
	a.rings[k] = m
	return m
}

// WindowIcons renders the application icon in a few sizes.
func WindowIcons() []image.Image {
	c := art.IconArt()
	n := max(c.W, c.H) + 2
	sq := art.NewCanvas(n, n)
	sq.Stamp(c, (n-c.W)/2, (n-c.H)/2)
	var out []image.Image
	for _, s := range []int{16, 32, 48, 64} {
		img := image.NewRGBA(image.Rect(0, 0, s, s))
		for y := 0; y < s; y++ {
			for x := 0; x < s; x++ {
				img.Set(x, y, sq.Img.At(x*n/s, y*n/s))
			}
		}
		out = append(out, img)
	}
	return out
}

// softGlow is an n×n radial halo (premultiplied) that fades out smoothly.
func softGlow(n int, col color.RGBA) *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	c := float64(n) / 2
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			d := math.Hypot(float64(x)+0.5-c, float64(y)+0.5-c) / c
			if d >= 1 {
				continue
			}
			k := (1 - d) * (1 - d)
			img.SetRGBA(x, y, color.RGBA{uint8(float64(col.R) * k), uint8(float64(col.G) * k), uint8(float64(col.B) * k), uint8(255 * k)})
		}
	}
	return ebiten.NewImageFromImage(img)
}

// baseOf is the command center sprite of a level (with an east gate when a
// road comes in there).
func (a *Assets) baseOf(l *game.Level) sprite {
	if l.EastGate() {
		return a.BaseEast
	}
	return a.Base
}
