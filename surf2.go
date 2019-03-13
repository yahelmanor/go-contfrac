package main

import (
	"math"
	"time"
)

type growsurf struct {
	ptr     map[wcord]int
	pts     []dot
	data    []growsurfData
	maxcord cord
	eps     float64
}

var _ surf = &growsurf{}

type growsurfData struct {
	p plane
	c cord
}

//wcord is a warper for cord
type wcord int

//simply add to the map and array
func (sf *growsurf) add(c cord, pt dot, p plane) {
	sf.data = append(sf.data, growsurfData{p: p, c: c})
	sf.pts = append(sf.pts, pt)
	sf.ptr[wcord(sf.maxcord.ctoi(c))] = len(sf.pts) - 1
}

//get from the map
func (sf *growsurf) getI(c cord) (i int, ok bool) {
	i, ok = sf.ptr[wcord(sf.maxcord.ctoi(c))]
	return
}

//take the dot, the place to put and the tangent plane at the point
//calc the right cs and put it in the sf
func (sf *growsurf) addO(ast cord, c cord, pt dot, p plane) {
	astI, _ := sf.getI(ast)
	cs2 := make([][]float64, len(p.cs))
	cs2[0] = mul(p.cs[0], 1.0)
	for j, c1 := range ones(c) {
		v := sf.data[astI].p.cs.getD(c1)
		i, flip := findCorl(v, p.cs[1:])
		if flip {
			cs2[i+1] = mul(p.cs[j+1], -1.0)
		} else {
			cs2[i+1] = mul(p.cs[j+1], 1.0)
		}
		// log.Println("put", cs2[i], "into cs", i)
	}
	// log.Println("cs:", p.cs)
	p.cs = cs2
	// log.Println("cs2:", cs2)
	sf.add(c, pt, p)
}

func (sf *growsurf) copy() surf {
	g := growsurf{}
	g.data = make([]growsurfData, len(sf.data))
	copy(g.data, sf.data)
	g.pts = make([]dot, len(sf.pts))
	for i, v := range sf.pts {
		g.pts[i] = make(dot, len(v))
		copy(g.pts[i], v)
	}
	g.ptr = make(map[wcord]int)
	for k, v := range sf.ptr {
		g.ptr[k] = v
	}
	return &g
}

func (sf *growsurf) dots() []dot {
	return sf.pts
}

func (sf *growsurf) get(i int) dot {
	return sf.pts[i]
}

func (sf *growsurf) set(i int, d dot) {
	sf.pts[i] = d
}

func (sf *growsurf) len() int {
	return len(sf.pts)
}

func (sf *growsurf) neighbors(c int) []int {
	return nil
}

//return the dot from point and cord
//get c when len(c) = len(cs)-1
func (cs cs) getD(c cord) dot {
	sum := make(dot, len(cs))
	for i, v := range cs {
		if i == 0 {
			continue
		}

		sum = add(sum, mul(v, float64(c[i-1])))
	}
	return sum
}

func findCorl(v vec, c cs) (i int, flip bool) {
	max := 0.0
	i = -1
	for j, v2 := range c {
		val := ddot(v, v2)
		val2 := math.Abs(val)
		if max < val2 {
			max = val2
			i = j
			flip = val < 0.0
		}
	}
	return
}

func ones(c cord) []cord {
	a := make([]cord, len(c))
	for i := range c {
		a[i] = make(cord, len(c))
		a[i][i] = 1
	}
	return a
}

func (sf *growsurf) addIn(ast, c cord, e errfunc) {
	astI, _ := sf.getI(ast)
	v := sf.data[astI].p.cs.getD(minusC(c, ast))
	d := add(sf.pts[astI], mul(v, sf.eps))
	for i := 0; i < 1000; i++ {
		// fall2()
		e, _ := fall2(d, errf, 0.001)
		if e < 0.0001 {
			break
		}
	}
	time.Sleep(time.Millisecond * 50)
	p := getZplane(d, e)
	// log.Println("cs0", p.cs)
	sf.addO(ast, c, d, p)
}

func minusC(x, y cord) (r cord) {
	r = make(cord, len(x))
	for i, v := range x {
		r[i] = v - y[i]
	}
	return
}
