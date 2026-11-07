package game

import (
	"errors"
	"fmt"
	"math"
)

type Tile uint8

const (
	TileGround Tile = iota
	TileRoad
	TileSlot
	TileBase
	TileSpawn
)

type Cell struct{ X, Y int }

func (c Cell) Center() Vec { return Vec{float64(c.X) + 0.5, float64(c.Y) + 0.5} }

// Route is the way enemies walk from one entrance to the base.
type Route struct {
	Spawn   Cell
	Cells   []Cell // road cells from the spawn to the base gate
	Corners []Vec
	Path    *Path
	EndDir  Cell // direction from the last cell into the base
	Join    int  // cells walked before the route merges into an earlier one (len(Cells) if never)
	// Ice lists the stretches of the path (from, to in path distance) between
	// paired ice holes, where a diving enemy swims under the surface.
	Ice [][2]float64
}

// UnderIce reports whether path distance d lies between a pair of ice holes.
func (r *Route) UnderIce(d float64) bool {
	for _, s := range r.Ice {
		if d >= s[0] && d < s[1] {
			return true
		}
	}
	return false
}

// Level is a parsed, ready-to-play map.
type Level struct {
	Spec     *LevelSpec
	W, H     int
	Tiles    [][]Tile
	Routes   []*Route // one per entrance, in entrance order
	BaseMin  Cell
	BaseSize int
	Slots    []Cell
	Holes    []Cell // ice holes in the road ('H' on the map)

	// The first route, kept for single-entrance code paths.
	Spawn   Cell
	Road    []Cell
	Corners []Vec
	Path    *Path
}

func (l *Level) In(c Cell) bool { return c.X >= 0 && c.Y >= 0 && c.X < l.W && c.Y < l.H }

func (l *Level) BaseCenter() Vec {
	s := float64(l.BaseSize) / 2
	return Vec{float64(l.BaseMin.X) + s, float64(l.BaseMin.Y) + s}
}

// SlotAt returns the slot index at the cell, or -1.
func (l *Level) SlotAt(x, y int) int {
	for i, s := range l.Slots {
		if s.X == x && s.Y == y {
			return i
		}
	}
	return -1
}

// NearestRoad returns the point of any route closest to q.
func (l *Level) NearestRoad(q Vec) Vec {
	best, bd := q, math.Inf(1)
	for _, r := range l.Routes {
		d, along := r.Path.Closest(q)
		if d < bd {
			bd = d
			best, _ = r.Path.At(along)
		}
	}
	return best
}

// Coverage is the route length within r of q, summed over all routes (a road
// shared by two routes counts twice: it sees the enemies of both).
func (l *Level) Coverage(q Vec, r float64) float64 {
	total := 0.0
	for _, rt := range l.Routes {
		total += rt.Path.Coverage(q, r)
	}
	return total
}

// EastGate reports whether a road enters the base from the east (the base
// then has a second gate on that side).
func (l *Level) EastGate() bool {
	for _, r := range l.Routes {
		if r.EndDir.X < 0 {
			return true
		}
	}
	return false
}

// Entrances is the number of enemy entrances of the map.
func (l *Level) Entrances() int { return len(l.Routes) }

var dirs4 = []Cell{{1, 0}, {0, 1}, {-1, 0}, {0, -1}}

// spawnIndex maps a map symbol to an entrance index: 'S' or '1' is the first
// entrance, '2' the second and so on.
func spawnIndex(ch rune) int {
	switch {
	case ch == 'S':
		return 0
	case ch >= '1' && ch <= '9':
		return int(ch - '1')
	}
	return -1
}

func ParseLevel(spec *LevelSpec) (*Level, error) {
	h := len(spec.Map)
	if h == 0 {
		return nil, errors.New("empty map")
	}
	w := len(spec.Map[0])
	l := &Level{Spec: spec, W: w, H: h, Tiles: make([][]Tile, h)}
	minX, minY, maxX, maxY := w, h, -1, -1
	spawns := map[int]Cell{}
	for y, row := range spec.Map {
		if len(row) != w {
			return nil, fmt.Errorf("row %d has width %d, want %d", y, len(row), w)
		}
		l.Tiles[y] = make([]Tile, w)
		for x, ch := range row {
			switch {
			case ch == '.':
			case ch == '#':
				l.Tiles[y][x] = TileRoad
			case ch == 'H':
				l.Tiles[y][x] = TileRoad
				l.Holes = append(l.Holes, Cell{x, y})
			case spawnIndex(ch) >= 0:
				i := spawnIndex(ch)
				if _, dup := spawns[i]; dup {
					return nil, fmt.Errorf("entrance %c appears twice", ch)
				}
				l.Tiles[y][x] = TileSpawn
				spawns[i] = Cell{x, y}
			case ch == 'o':
				l.Tiles[y][x] = TileSlot
				l.Slots = append(l.Slots, Cell{x, y})
			case ch == 'B':
				l.Tiles[y][x] = TileBase
				minX, minY = min(minX, x), min(minY, y)
				maxX, maxY = max(maxX, x), max(maxY, y)
			default:
				return nil, fmt.Errorf("unknown map symbol %q at %d,%d", ch, x, y)
			}
		}
	}
	if len(spawns) == 0 || maxX < 0 {
		return nil, errors.New("map needs a spawn and a base")
	}
	for i := range spawns {
		if _, ok := spawns[i]; !ok || i >= len(spawns) {
			return nil, errors.New("entrances must be numbered 1..n without gaps")
		}
	}
	if maxX-minX != maxY-minY {
		return nil, errors.New("base must be square")
	}
	l.BaseMin, l.BaseSize = Cell{minX, minY}, maxX-minX+1

	// Distance to the base over road cells: enemies always step to the
	// neighbour one cell closer, so merging roads are fine but forks are not.
	dist := make([][]int, h)
	for y := range dist {
		dist[y] = make([]int, w)
		for x := range dist[y] {
			dist[y][x] = -1
		}
	}
	tile := func(c Cell) Tile {
		if !l.In(c) {
			return TileGround
		}
		return l.Tiles[c.Y][c.X]
	}
	var queue []Cell
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := Cell{x, y}
			if tile(c) != TileRoad {
				continue
			}
			for _, d := range dirs4 {
				if tile(Cell{x + d.X, y + d.Y}) == TileBase {
					dist[y][x] = 0
					queue = append(queue, c)
					break
				}
			}
		}
	}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		for _, d := range dirs4 {
			n := Cell{c.X + d.X, c.Y + d.Y}
			if tile(n) == TileRoad && dist[n.Y][n.X] < 0 {
				dist[n.Y][n.X] = dist[c.Y][c.X] + 1
				queue = append(queue, n)
			}
		}
	}

	used := map[Cell]bool{}
	for i := 0; i < len(spawns); i++ {
		sp := spawns[i]
		cur := sp
		cells := []Cell{cur}
		curD := math.MaxInt
		var endDir Cell
		join := -1
		for {
			if curD == 0 {
				for _, d := range dirs4 {
					if tile(Cell{cur.X + d.X, cur.Y + d.Y}) == TileBase {
						endDir = d
					}
				}
				break
			}
			var next Cell
			best, count := math.MaxInt, 0
			for _, d := range dirs4 {
				n := Cell{cur.X + d.X, cur.Y + d.Y}
				if tile(n) != TileRoad || dist[n.Y][n.X] < 0 {
					continue
				}
				switch v := dist[n.Y][n.X]; {
				case v < best:
					next, best, count = n, v, 1
				case v == best:
					count++
				}
			}
			if count == 0 || (curD != math.MaxInt && best != curD-1) {
				return nil, fmt.Errorf("road from entrance %d does not reach the base (stuck at %d,%d)", i+1, cur.X, cur.Y)
			}
			if count > 1 {
				return nil, fmt.Errorf("road forks at %d,%d", cur.X, cur.Y)
			}
			cur, curD = next, best
			if join < 0 && used[cur] {
				join = len(cells)
			}
			cells = append(cells, cur)
		}
		if join < 0 {
			join = len(cells)
		}
		for _, c := range cells[1:] {
			used[c] = true
		}
		rt := buildRoute(sp, cells, endDir, join)
		if err := rt.findIce(l.Holes, i); err != nil {
			return nil, err
		}
		l.Routes = append(l.Routes, rt)
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if l.Tiles[y][x] == TileRoad && !used[Cell{x, y}] {
				return nil, fmt.Errorf("road cell %d,%d is not on any route", x, y)
			}
		}
	}
	r0 := l.Routes[0]
	l.Spawn, l.Road, l.Corners, l.Path = r0.Spawn, r0.Cells, r0.Corners, r0.Path
	return l, nil
}

// findIce pairs the ice holes along the route: the first pair of holes it
// passes bounds the first stretch under the ice, and so on.
func (r *Route) findIce(holes []Cell, idx int) error {
	var ds []float64
	for _, c := range r.Cells {
		for _, h := range holes {
			if c == h {
				_, along := r.Path.Closest(c.Center())
				ds = append(ds, along)
			}
		}
	}
	if len(ds)%2 != 0 {
		return fmt.Errorf("road from entrance %d passes %d ice holes: they must come in pairs", idx+1, len(ds))
	}
	for i := 0; i+1 < len(ds); i += 2 {
		r.Ice = append(r.Ice, [2]float64{ds[i], ds[i+1]})
	}
	return nil
}

func buildRoute(sp Cell, cells []Cell, endDir Cell, join int) *Route {
	pts := []Vec{cells[0].Center()}
	for i := 1; i < len(cells)-1; i++ {
		a, b, c := cells[i-1], cells[i], cells[i+1]
		if b.X-a.X != c.X-b.X || b.Y-a.Y != c.Y-b.Y {
			pts = append(pts, b.Center())
		}
	}
	last := cells[len(cells)-1].Center()
	pts = append(pts, last, last.Add(Vec{float64(endDir.X), float64(endDir.Y)}.Mul(0.75)))
	return &Route{Spawn: sp, Cells: cells, Corners: pts, Path: NewPath(SmoothCorners(pts, 0.5, 8)), EndDir: endDir, Join: join}
}
