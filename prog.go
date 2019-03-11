package main

import (
	"fmt"
	"log"
	"math"
	"sort"
	"time"
)

const cns = math.Pi / 4.0

func errf(d dot) float64 {
	v := K([]float64{d[0], d[1], 2.0}, []float64{d[2], d[3], d[4]}, 20).Actz(0.0)
	return (v - cns) * (v - cns)
}
func intgerf(d dot) float64 {
	e := 0.0
	for _, v := range d {
		_, f := math.Modf(math.Abs(v))
		if f > 0.5 {
			f = 1.0 - f
		}
		e += f * f
	}
	return e
}

func newsfProg() surf {
	return newgs2(dot{0.0, 0.0, 0.0, 0.0, 0.0}, 3, cord{10, 10, 10}, func(c cord) dot {
		d := make(dot, 5)
		d[0] = -10.0
		for i := 1; i < 3; i++ {
			d[i] = 5.0 - float64(c[i-1])*0.4
		}
		return d
	})
}

const (
	pathLong    = 750
	pathLong1_5 = 750
	pathLong2   = 1000
	pathLong3   = 200

	/*
		pathLong    = 30
		pathLong1_5 = 30
		pathLong2   = 50
		pathLong3   = 10
	*/
)

func prog(sf *surf) {

	fmt.Printf("\n------------------------------------------------\n\t\tPHAZE 1 - start\n------------------------------------------------\n")

	t0 := time.Now()
	for j := 0; j < pathLong1_5; j++ {
		if j&7 == 0 {
			dt := time.Now().Sub(t0)
			rt := int(dt) * (pathLong1_5 - j) / (j + 1)
			ops := float64(j) * float64(time.Second) / float64(dt)
			fmt.Printf("1 > (j = %4.d) pass %.1f%% exp time is %v        \b\b\b\b\b\b\b\b\t|        \b\b\b\b\b\b\b\b\t%e op/s \r", j, float64(j)/float64(pathLong1_5)*100, time.Duration(rt), ops)
		}
		falla((*sf).dots(), errf, 0.03)
	}

	fmt.Printf("\n------------------------------------------------\n\t\tPHAZE 2 - start\n------------------------------------------------\n\n")

	t0 = time.Now()
	for j := 0; j < pathLong2; j++ {
		if j&15 == 0 {
			dt := time.Now().Sub(t0)
			rt := int(dt) * (pathLong2 - j) / (j + 1)
			ops := float64(j) * float64(time.Second) / float64(dt)
			fmt.Printf("2 > (j = %4.d) pass %.1f%% exp time is %10.10v        \b\b\b\b\b\b\b\b\t|        \b\b\b\b\b\b\b\b\t%g op/s \r", j, float64(j)/float64(pathLong2)*100, time.Duration(rt), ops)
		}
		f := float64(j) * float64(j) / (float64(pathLong2) * float64(pathLong2))
		switch j % fallRatio {
		case 0:
			(*sf) = pushs((*sf), 1.0*f+0.005)
		default:
			falla((*sf).dots(), errf, 0.2*f+0.05)
		}
	}

	fmt.Printf("\n------------------------------------------------\n\t\tPHAZE 3 - start\n------------------------------------------------\n")

	//for i := 0; i < duptimes; i++ {
	for j := 0; j < pathLong3; j++ {
		//time.Sleep(time.Millisecond * 10)
		switch j % 2 {
		case 0:
			falla((*sf).dots(), intgerf, 0.007)
		default:
			falla((*sf).dots(), errf, 0.02)
		}
	}
	//}

	fmt.Printf("\n------------------------------------------------\n\t\t      END      \n------------------------------------------------\n")
	fmt.Printf("Good News List:\n\n")
	ir := make(dot, len((*sf).dots()[0]))
	for _, r := range (*sf).dots() {
		if errf(r) < 0.0001 {
			for i, v := range r {
				ir[i] = math.Round(v)
			}
			if math.Sqrt(errf(ir)) < 0.0001 {
				log.Println(math.Sqrt(errf(ir)), ir)
			}
		}
	}

	fmt.Printf("\t----\nEnd of list.\n")
	ierrf := func(d dot) float64 {
		id := make(dot, len(d))
		for i, v := range d {
			id[i] = math.Round(v)
		}
		return math.Sqrt(errf(id))
	}
	d := newdotptr(*sf, ierrf)
	sort.Sort(d)
	log.Println(d.a)
	for i, j := range d.a[0:50] {
		d0 := (*sf).dots()[j]
		d1 := (*sf).dots()[d.a[i+1]]
		if ierrf(d0) != ierrf(d1) {
			log.Println(ierrf(d0), d0)
		}
	}
	fmt.Printf("\n\n>> end.\n\n\n")
}

type dotptr struct {
	a    []int
	sf   surf
	errf errfunc
}

func (d dotptr) Less(i, j int) bool {
	ii, jj := d.a[i], d.a[j]
	return d.errf(d.sf.dots()[ii]) < d.errf(d.sf.dots()[jj])
}

func (d dotptr) Swap(i, j int) {
	d.a[i], d.a[j] = d.a[j], d.a[i]
}

func (d dotptr) Len() int {
	return len(d.a)
}

func newdotptr(sf surf, errf errfunc) dotptr {
	d := dotptr{}
	d.a = make([]int, 0)
	for i := range sf.dots() {
		d.a = append(d.a, i)
	}
	log.Println(d.a)
	d.sf = sf
	d.errf = errf
	return d
}
