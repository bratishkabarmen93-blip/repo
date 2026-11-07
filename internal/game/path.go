package game

import (
	"math"
	"sort"
)

// Vec is a 2D vector measured in map cells.
type Vec struct{ X, Y float64 }

func (a Vec) Add(b Vec) Vec             { return Vec{a.X + b.X, a.Y + b.Y} }
func (a Vec) Sub(b Vec) Vec             { return Vec{a.X - b.X, a.Y - b.Y} }
func (a Vec) Mul(s float64) Vec         { return Vec{a.X * s, a.Y * s} }
func (a Vec) Len() float64              { return math.Hypot(a.X, a.Y) }
func (a Vec) Dist(b Vec) float64        { return math.Hypot(a.X-b.X, a.Y-b.Y) }
func (a Vec) Dist2(b Vec) float64       { dx, dy := a.X-b.X, a.Y-b.Y; return dx*dx + dy*dy }
func (a Vec) Angle() float64            { return math.Atan2(a.Y, a.X) }
func (a Vec) Perp() Vec                 { return Vec{-a.Y, a.X} }
func FromAngle(rad float64) Vec         { return Vec{math.Cos(rad), math.Sin(rad)} }
func (a Vec) Lerp(b Vec, t float64) Vec { return a.Add(b.Sub(a).Mul(t)) }

func (a Vec) Norm() Vec {
	l := a.Len()
	if l == 0 {
		return Vec{}
	}
	return Vec{a.X / l, a.Y / l}
}

// SegDist is the distance from p to the segment a-b.
func SegDist(p, a, b Vec) float64 {
	ab := b.Sub(a)
	l2 := ab.X*ab.X + ab.Y*ab.Y
	if l2 < 1e-12 {
		return p.Dist(a)
	}
	t := Clamp(((p.X-a.X)*ab.X+(p.Y-a.Y)*ab.Y)/l2, 0, 1)
	return p.Dist(a.Add(ab.Mul(t)))
}

func Clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func Smoothstep(a, b, x float64) float64 {
	t := Clamp((x-a)/(b-a), 0, 1)
	return t * t * (3 - 2*t)
}

// WrapAngle maps an angle into [-pi, pi].
func WrapAngle(a float64) float64 {
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	for a < -math.Pi {
		a += 2 * math.Pi
	}
	return a
}

// AngleDiff returns the signed shortest rotation from b to a.
func AngleDiff(a, b float64) float64 { return WrapAngle(a - b) }

// Path is a polyline with cumulative distances, used for enemy movement.
type Path struct {
	Pts []Vec
	Cum []float64
	Len float64
}

func NewPath(pts []Vec) *Path {
	p := &Path{Pts: pts, Cum: make([]float64, len(pts))}
	for i := 1; i < len(pts); i++ {
		p.Cum[i] = p.Cum[i-1] + pts[i].Dist(pts[i-1])
	}
	p.Len = p.Cum[len(pts)-1]
	return p
}

// At returns the position and the unit direction at distance d along the path.
func (p *Path) At(d float64) (Vec, Vec) {
	n := len(p.Pts)
	if d <= 0 {
		dir := p.Pts[1].Sub(p.Pts[0]).Norm()
		return p.Pts[0].Add(dir.Mul(d)), dir
	}
	if d >= p.Len {
		dir := p.Pts[n-1].Sub(p.Pts[n-2]).Norm()
		return p.Pts[n-1], dir
	}
	i := sort.SearchFloat64s(p.Cum, d)
	if i <= 0 {
		i = 1
	}
	a, b := p.Pts[i-1], p.Pts[i]
	seg := p.Cum[i] - p.Cum[i-1]
	t := 0.0
	if seg > 0 {
		t = (d - p.Cum[i-1]) / seg
	}
	return a.Lerp(b, t), b.Sub(a).Norm()
}

// Closest returns the distance from q to the path and the path distance of the closest point.
func (p *Path) Closest(q Vec) (dist, along float64) {
	dist = math.Inf(1)
	for i := 1; i < len(p.Pts); i++ {
		a, b := p.Pts[i-1], p.Pts[i]
		ab := b.Sub(a)
		l2 := ab.X*ab.X + ab.Y*ab.Y
		t := 0.0
		if l2 > 0 {
			t = Clamp(((q.X-a.X)*ab.X+(q.Y-a.Y)*ab.Y)/l2, 0, 1)
		}
		c := a.Add(ab.Mul(t))
		if d := c.Dist(q); d < dist {
			dist = d
			along = p.Cum[i-1] + t*math.Sqrt(l2)
		}
	}
	return dist, along
}

// Coverage returns how much path length lies within radius r of q.
func (p *Path) Coverage(q Vec, r float64) float64 {
	const step = 0.05
	total := 0.0
	for d := 0.0; d < p.Len; d += step {
		pos, _ := p.At(d)
		if pos.Dist2(q) <= r*r {
			total += step
		}
	}
	return total
}

// SmoothCorners replaces sharp corners with quadratic curves of radius r.
func SmoothCorners(pts []Vec, r float64, samples int) []Vec {
	if len(pts) < 3 {
		return pts
	}
	out := []Vec{pts[0]}
	for i := 1; i < len(pts)-1; i++ {
		p0, p1, p2 := pts[i-1], pts[i], pts[i+1]
		din, dout := p1.Sub(p0), p2.Sub(p1)
		rr := math.Min(r, math.Min(din.Len()/2, dout.Len()/2))
		a := p1.Sub(din.Norm().Mul(rr))
		b := p1.Add(dout.Norm().Mul(rr))
		out = append(out, a)
		for s := 1; s < samples; s++ {
			t := float64(s) / float64(samples)
			u := 1 - t
			out = append(out, a.Mul(u*u).Add(p1.Mul(2*u*t)).Add(b.Mul(t*t)))
		}
		out = append(out, b)
	}
	return append(out, pts[len(pts)-1])
}
