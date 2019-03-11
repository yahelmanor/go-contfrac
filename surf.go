package main

import (
	"log"
)

type graphsurf struct {
	v []dot
	e [][]int
}

var _ surf = graphsurf{}

func (gs graphsurf) len() int {
	return len(gs.v)
}

func (gs graphsurf) get(i int) dot {
	return gs.v[i]
}

func (gs graphsurf) set(i int, d dot) {
	if i >= len(gs.v) || i < 0 {
		log.Println("i,len(v),gs = ", i, len(gs.v), gs)
	}
	gs.v[i] = d
}

func (gs graphsurf) neighbors(i int) []int {
	return gs.e[i]
}

func (gs graphsurf) upt(float64) {

}

func (gs graphsurf) dots() []dot {
	return gs.v
}

func (gs graphsurf) copy() surf {
	gs2 := graphsurf{}
	gs2.v = make([]dot, len(gs.v))
	gs2.e = make([][]int, len(gs.e))
	for i, v := range gs.v {
		gs2.v[i] = make(dot, len(v))
		copy(gs2.v[i], v)
	}
	copy(gs2.e, gs.e)
	return gs2
}

func newgs2(d dot, dist int, c cord, f func(cord) dot) surf {
	s := graphsurf{}
	mul := cordL(c)
	log.Println("size of sf", mul)
	s.v = make([]dot, mul)
	s.e = make([][]int, mul)
	//end initing
	//start warkning with the ematy surface
	for i := 0; i < mul; i++ {
		d2 := make(dot, len(d))
		copy(d2, d)
		add := f(c.itoc(i))
		for j, v := range add {
			d2[j] += v
		}
		s.v[i] = d2
		e2 := []int{}
		iS, iF := i-dist, i+dist+1
		if iS < 0 {
			iS = 0
		}
		if iF > mul {
			iF = mul
		}
		for j := iS; j < iF; j++ {
			if j == i {
				continue
			}
			e2 = append(e2, j)
		}
		s.e[i] = e2
	}
	return s
}

func cordL(c cord) int {
	mul := 1
	for _, i := range c {
		mul *= i
	}
	return mul
}

func (c cord) itoc(i int) cord {
	c2 := make(cord, len(c))
	for k, j := range c {
		c2[k] = i % j
		i /= j
	}
	return c2
}
