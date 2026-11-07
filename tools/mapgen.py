#!/usr/bin/env python3
"""Draw level maps from road waypoints and pads, check them and print Go map lines."""
import sys, json

def build(w, h, pts, base, pads):
    g = [['.'] * w for _ in range(h)]
    road = []
    for (x0, y0), (x1, y1) in zip(pts, pts[1:]):
        assert x0 == x1 or y0 == y1, ('diagonal', (x0, y0), (x1, y1))
        dx = (x1 > x0) - (x1 < x0); dy = (y1 > y0) - (y1 < y0)
        x, y = x0, y0
        if not road:
            road.append((x, y))
        while (x, y) != (x1, y1):
            x += dx; y += dy
            road.append((x, y))
    assert len(set(road)) == len(road), 'road crosses itself'
    idx = {c: i for i, c in enumerate(road)}
    for i, (x, y) in enumerate(road):
        for nx, ny in ((x+1, y), (x-1, y), (x, y+1), (x, y-1)):
            j = idx.get((nx, ny))
            if j is not None and abs(j - i) > 1:
                raise SystemExit(f'road touches itself at {x},{y} and {nx},{ny}')
    bx, by = base
    for (x, y) in road:
        assert x < bx, f'road east of base at {x},{y}'
        g[y][x] = '#'
    sx, sy = road[0]
    assert sx == 0
    g[sy][sx] = 'S'
    assert road[-1] == (bx - 1, by + 1), f'road must end at the gate {(bx-1, by+1)}, ends {road[-1]}'
    for i, (x, y) in enumerate(road[:-1]):
        for nx, ny in ((x+1, y), (x-1, y), (x, y+1), (x, y-1)):
            if bx <= nx < bx + 3 and by <= ny < by + 3:
                raise SystemExit(f'road touches the base early at {x},{y}')
    for yy in range(by, by + 3):
        for xx in range(bx, bx + 3):
            g[yy][xx] = 'B'
    for (x, y) in pads:
        assert g[y][x] == '.', f'pad on {g[y][x]} at {x},{y}'
        g[y][x] = 'o'
    return g, road

def show(name, w, h, pts, base, pads):
    g, road = build(w, h, pts, base, pads)
    print(f'{name}: road cells {len(road)}, pads {len(pads)}')
    print('   ' + ''.join(str(x % 10) for x in range(w)))
    for y, row in enumerate(g):
        print(f'{y:2d} ' + ''.join(row))
    print('\t\tMap: []string{')
    for row in g:
        print(f'\t\t\t"{"".join(row)}",')
    print('\t\t},')

if __name__ == '__main__':
    spec = json.load(open(sys.argv[1]))
    for m in spec:
        show(m['name'], m['w'], m['h'], [tuple(p) for p in m['pts']], tuple(m['base']), [tuple(p) for p in m['pads']])
