package main

import (
	"log"
	"math"
	"sync"

	"gonum.org/v1/gonum/blas/blas64"
	"gonum.org/v1/gonum/mat"
)

type dot = []float64

type errfunc func(dot) float64

const drveps = 0.000000001

const (
	maxPush  = 7.07
	maxPush2 = maxPush * maxPush
)

func falla(da []dot, err errfunc, step float64) {
	for _, d := range da {
		fall(d, err, step)
	}
}

func fall(d dot, err errfunc, step float64) {
	e0 := err(d)
	el := make([]float64, len(d))
	d2 := make([]float64, len(d))
	for i := range d {
		copy(d2, d)
		d2[i] += drveps
		el[i] = (err(d2) - e0) / drveps
	}
	size := 0.0
	for i := range el {
		size += el[i] * el[i]
	}

	//if size < step*step {
	//	size = step
	//} else {
	size = math.Sqrt(size)
	//}
	for i := range d {
		d[i] -= el[i] / size * step
	}
}

func fall2(d dot, err errfunc, step float64) (e, drv float64) {
	e0 := err(d)
	el := make([]float64, len(d))
	d2 := make([]float64, len(d))
	for i := range d {
		copy(d2, d)
		d2[i] += drveps
		el[i] = (err(d2) - e0) / drveps
	}
	size := 0.0
	for i := range el {
		size += el[i] * el[i]
	}

	//if size < step*step {
	//	size = step
	//} else {
	size = math.Sqrt(size)
	//}
	for i := range d {
		d[i] -= el[i] / size * step
	}
	return e0, size
}

func godrv(eps1, step float64, d []float64, err errfunc) []float64 {
	e0 := err(d)
	el := make([]float64, len(d))
	d2 := make([]float64, len(d))
	for i := range d {
		copy(d2, d)
		d2[i] += eps1
		el[i] = (err(d2) - e0) / eps1
	}
	size := 0.0
	for i := range el {
		size += el[i] * el[i]
	}

	if size < step*step {
		size = step
	} else {
		size = math.Sqrt(size)
	}
	for i := range d {
		d[i] -= el[i] / size * step
	}
	return d
}

func spread(da []dot, step float64) {
	if len(da) == 0 {
		return
	}
	da2 := make([][]float64, len(da))

	for i, r := range da {
		da2[i] = make([]float64, len(r))
		copy(da2[i], r)
	}

	v1 := blas64.Vector{N: len(da[0]), Inc: 1}
	v2 := v1
	v3 := blas64.Vector{N: len(da[0]), Data: make([]float64, len(da[0])), Inc: 1}
	for i, r1 := range da {
		v1.Data = r1
		for j, r2 := range da {
			if i == j {
				continue
			}
			v2.Data = r2
			//PUSH r1` by r2
			blas64.Copy(v2, v3)
			//v3 = v2
			blas64.Axpy(-1.0, v1, v3)
			//v3 = v2-v1
			d := 1 / blas64.Dot(v3, v3)
			//d = 1/|dv|^2
			if d > maxPush2 {
				d = d / math.Sqrt(d) * maxPush
			}
			d *= step
			v2.Data = da2[i]
			blas64.Axpy(-d, v3, v2)
			//v1` = v1 - d * v3 = v1 - d*(dv[2-1]) = v1 * (1+d) + v2 * (-d)
			//|d*(dv[2-1])| = 1/|dv|^2 * |dv| = 1/|dv|
		}
	}

	for i, r := range da {
		copy(r, da2[i])
	}
}

type surf interface {
	dots() []dot
	len() int
	get(int) dot
	set(int, dot)
	neighbors(int) []int
	//MTS ~ Max Triangle size
	// upt(MTS float64)
	copy() surf
}

//surface is n-dimonationl surface
type surface struct {
	v []dot
	s cord
	n uint
	l sync.Locker
}

type cord []int

func (s surface) get(i cord) dot {
	if uint(len(i)) != s.n {
		return nil
	}
	mul := 1
	sum := 0
	for k, j := range s.s {
		if i[k] < 0 || i[k] >= j {
			return nil
		}
		sum += i[k] * mul
		mul *= j
	}
	// log.Println("sum =", sum)
	return s.v[sum]
}

func news(d dot, eps float64, c cord) surface {
	s := surface{}
	s.s = c
	s.n = uint(len(c))
	mul := 1
	for _, i := range c {
		mul *= i
	}
	log.Println("size of sf", mul)
	s.v = make([]dot, mul)
	//end initing
	//start warkning with the ematy surface
	for i := 0; i < mul; i++ {
		d2 := make(dot, len(d))
		copy(d2, d)
		for n, m := range s.itoc(i) {
			d2[n] += float64(m) * eps
		}
		s.v[i] = d2
	}
	return s
}

func news2(d dot, c cord, f func(cord) dot) surface {
	s := surface{}
	s.s = c
	s.n = uint(len(c))
	mul := 1
	for _, i := range c {
		mul *= i
	}
	log.Println("size of sf", mul)
	s.v = make([]dot, mul)
	//end initing
	//start warkning with the ematy surface
	for i := 0; i < mul; i++ {
		d2 := make(dot, len(d))
		copy(d2, d)
		add := f(s.itoc(i))
		for j, v := range add {
			d2[j] += v
		}
		s.v[i] = d2
	}
	return s
}

func nearst(i cord) []cord {
	c := make([]cord, len(i))
	for j := range i {
		c1 := make(cord, len(i))
		copy(c1, i)
		c1[j]++
		c[j] = c1
	}
	return c
}

func nearst2(i cord) []cord {
	c := make([]cord, 2*len(i))
	for j := range i {
		c1 := make(cord, len(i))
		copy(c1, i)
		c1[j]++
		c2 := make(cord, len(i))
		copy(c2, i)
		c2[j]--
		c[2*j] = c1
		c[2*j+1] = c2
	}
	//log.Printf("nearst to %v is %v", i, c)
	return c
}

func zero(c cord) cord {
	return make(cord, len(c))
}

func (s surface) copy() surface {
	r := surface{}
	r.n = s.n
	r.s = make(cord, len(s.s))
	copy(r.s, s.s)
	r.v = make([]dot, len(s.v))
	copy(r.v, s.v)
	return r
}

func (s surface) itoc(i int) cord {
	c := make(cord, s.n)
	for k, j := range s.s {
		c[k] = i % j
		i /= j
	}
	return c
}

func (max cord) ctoi(c cord) (sum int) {
	mul := 1
	for i, v := range c {
		sum += v * mul
		mul *= max[i]
	}
	return
}

func push(s surface, step float64) {
	v1 := blas64.Vector{N: int(s.n), Inc: 1}
	v2 := v1
	v3 := blas64.Vector{N: int(s.n), Data: make([]float64, s.n), Inc: 1}
	s2 := s.copy()
	for i, v := range s.v {
		c := s.itoc(i)
		v1.Data = v
		for _, c2 := range nearst2(c) {
			v2.Data = s.get(c2)
			if v2.Data == nil {
				continue
			}
			//PUSH r1` by r2

			//log.Println(c, "<", i, ">", v2, c2, v1)

			blas64.Copy(v2, v3)
			//v3 = v2
			blas64.Axpy(-1.0, v1, v3)
			//v3 = v2-v1

			// log.Println("C_{1,2} = ", c, c2)
			// log.Println("V_{1,2,3} = ", v1, v2, v3)

			d := 1 / blas64.Dot(v3, v3)
			if d != d {
				log.Println("NaN!!")
			}
			//d = 1/|dv|^2
			if d > maxPush2 {
				d = d / math.Sqrt(d) * maxPush
			}
			d *= step
			v2.Data = s2.get(c)
			if v2.Data == nil {
				log.Println("ERROR <- 255")
			}
			// log.Println("V_{1,3} = ", v1, v3)
			blas64.Axpy(-d, v3, v2)
		}
	}
}

func pushs(s surf, step float64) surf {
	v1 := blas64.Vector{N: len(s.get(0)), Inc: 1}
	v2 := v1
	v3 := blas64.Vector{N: len(s.get(0)), Data: make([]float64, len(s.get(0))), Inc: 1}
	s2 := s.copy()
	for i := 0; i < s.len(); i++ {
		v := s.get(i)
		v1.Data = v
		for _, i2 := range s.neighbors(i) {
			v2.Data = s.get(i2)
			if v2.Data == nil {
				continue
			}
			//PUSH r1` by r2

			//log.Println(c, "<", i, ">", v2, c2, v1)

			blas64.Copy(v2, v3)
			//v3 = v2
			blas64.Axpy(-1.0, v1, v3)
			//v3 = v2-v1

			// log.Println("C_{1,2} = ", c, c2)
			// log.Println("V_{1,2,3} = ", v1, v2, v3)

			d := 1 / blas64.Dot(v3, v3)
			if d != d {
				log.Println("NaN!!")
			}
			//d = 1/|dv|^2
			if d > maxPush2 {
				d = d / math.Sqrt(d) * maxPush
			}
			d *= step
			v2.Data = s2.get(i)
			if v2.Data == nil {
				log.Println("ERROR <- 255")
			}
			// log.Println("V_{1,3} = ", v1, v3)
			blas64.Axpy(-d, v3, v2)
		}
	}
	return s2
}

//get the plane of zeros near by
func getZplane(d dot, e errfunc) plane {
	d2 := make(dot, len(d))
	copy(d2, d)
	val := make(dot, len(d))
	v0 := math.Sqrt(e(d))
	for i, v := range d {
		if i != 0 {
			d2[i-1] = d[i-1]
		}
		d2[i] = v + pstep
		val[i] = math.Sqrt(e(d2)) - v0
	}
	return plane{grmSchCS(val), val, ddot(val, d)}
}

func ddot(x, y dot) (f float64) {
	for i, v := range x {
		f += v * y[i]
	}
	return
}

//dot is the closest point to the origen
type plane struct {
	cs cs
	v  dot
	f  float64
}

func (p plane) vec(c cord, step float64) dot {

	return dot{}
}

//cs is cordinate system
//cs need to work like cordinate shifer so it will be a matrix
//cs can also get simgle vector and return all the other
type cs [][]float64

type vec = dot

func grmSchCS(v vec) cs {
	gcs := make([]vec, len(v))
	for i := range gcs {
		gcs[i] = make(vec, len(v))
		if v[i] >= 0 {
			gcs[i][i] = 1
		} else {
			gcs[i][i] = -1
		}

	}
	for i := 0; i < len(v); i++ {
		if v[i] != 0.0 {
			gcs[i] = v
			goto after
		}
	}
	panic("ERORR")
after:
	e := make([]vec, 0)
	for _, a := range gcs {
		for _, b := range e {
			a = add(a, mul(b, -ddot(a, b)))
		}
		e = append(e, norm(a))
	}
	return e
}

func add(x, y vec) (z vec) {
	z = make(vec, len(x))
	for i, v := range x {
		z[i] = y[i] + v
	}
	return
}

func mul(x vec, f float64) (y vec) {
	y = make(vec, len(x))
	for i, v := range x {
		y[i] = v * f
	}
	return
}

func norm(v vec) (r vec) {
	r = make(vec, len(v))
	f := math.Sqrt(ddot(v, v))
	for i, x := range v {
		r[i] = x / f
	}
	return
}

func ort(c cs) cs {
	//ort(cs[])
	det := mat.Det(c)
	if det < 0.0 { // == -1.0

	}
	return nil
}

func (c cs) Dims() (x, y int) {
	return len(c), len(c)
}

func (c cs) At(i, j int) float64 {
	return c[i][j]
}

func (c cs) T() mat.Matrix {
	return mat.Transpose{Matrix: c}
}
