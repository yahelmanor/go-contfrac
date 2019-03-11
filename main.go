package main

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"math"
	"os"
	"strconv"
	"sync"
	"time"

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
)

//Mob is mobius transformation
type Mob [][]float64

//Actf is activion on other Mobius
func (m Mob) Actf(a Mob) Mob {
	return [][]float64{
		{m[0][0]*a[1][0] + a[0][0]*m[0][1], m[0][0]*a[1][1] + m[0][1]*a[0][1]},
		{m[1][1]*a[0][0] + m[1][0]*a[1][0], m[1][1]*a[0][1] + a[1][1]*m[1][0]},
	}
}

//Actz is activion on float velue
func (m Mob) Actz(z float64) float64 {
	return (m[0][0] + z*m[0][1]) / (m[1][0] + z*m[1][1])
}

func (m Mob) simply() Mob {
	a, b, c, d := m[0][0], m[0][1], m[1][0], m[1][1]
	if v := math.Min(math.Min(a, b), math.Min(c, d)); v > 2.0 {
		a, b, c, d = a/v, b/v, c/v, d/v
	}
	return [][]float64{{a, b}, {c, d}}
}
func p(p []float64, x float64) float64 {
	x2, sum := float64(1), float64(0)
	for _, i := range p {
		sum += x2 * i
		x2 *= x
	}
	return sum
}

//K sum of gauss
func K(a, b []float64, N int) Mob {
	Mob := Mob([][]float64{
		{0, 1},
		{1, 0},
	})
	for i := 0; i < N; i++ {
		Mob = Mob.Actf([][]float64{
			{p(b, float64(i)), 0},
			{p(a, float64(i)), 1},
		})
		Mob = Mob.simply()
	}
	return Mob
}

const (
	X0, X1, Y0, Y1 = -40.0, 20.0, -40.0, 20.0
	Depth          = 3
	fallRatio      = 7
	pstep          = 0.01
	eps2           = 0.001
	minD, maxD     = 3, 9
	refrashRate    = 50

	dotSize = 5
)

type contfrac struct {
	m    Mob
	x, y float64
	deg  int
	d    dataptr
}

type dataptr struct{}

var mainMetadata metadata

func (dataptr) get() *metadata {
	return &mainMetadata
}

type metadata struct {
	proj func(x, y float64) (a, b []float64)
}

func (c *contfrac) at(x, y float64, d int) {
	a, b := c.d.get().proj(x, y)
	c.m = K(a, b, d)
	c.deg = d
	c.x = x
	c.y = y
}

func (c *contfrac) getf() float64 {
	v := c.m.Actz(0.0) - cns
	return math.Log(v * v)
}

func (c *contfrac) chk() float64 {
	a, b := c.d.get().proj(c.x, c.y)
	v0 := c.m.Actz(0.0)
	//K sum of gauss
	for i := c.deg; i < c.deg+10; i++ {
		c.m = c.m.Actf([][]float64{
			{p(b, float64(i)), 0},
			{p(a, float64(i)), 1},
		})
		c.m = c.m.simply()
	}
	v1 := c.m.Actz(0.0)
	if v1 != v1 {
		return 0.0
	}
	//log.Println("chk", x, y, C*(v1-v0), "=", C, v1, v0, "|", d)
	return (v1/v0 + v0/v1 - 2.0) / 10.0
	// return 0.0
}

var _ calc = &contfrac{}

var C = 1.0

type calc interface {
	at(x, y float64, d int)
	getf() float64
	chk() float64
}

func main() {
	log.Println(grmSchCS(vec{3.0, 4.0}))
	log.Println(grmSchCS(vec{3.0, -4.0}))
	log.Println(grmSchCS(vec{-3.0, 4.0}))
	log.Println(grmSchCS(vec{-3.0, -4.0}))

	log.Println(grmSchCS(vec{+1, 0}))
	log.Println(grmSchCS(vec{-1, 0}))
	log.Println(grmSchCS(vec{0, +1}))
	log.Println(grmSchCS(vec{0, -1}))

	return
	/*k := 1000
	maxN := 10
	r := 20000
	s := 20.0
	arr := make([]float64, k*maxN)
	t0 := time.Now()
	for i := 0; i < r; i++ {
		p := int(1000 * float64(i) / float64(r))
		sec := time.Duration(((r - i) * int(time.Now().Sub(t0))) / (i + 1))
		fmt.Printf("%2.1f%%|%v|\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b", float64(p)/10.0, sec)
		for j := 0; j < r; j++ {
			f := K([]float64{float64(i)/s - float64(r)/s/2, 1.0}, []float64{float64(j)/s - float64(r)/s/2, 1.0}, Depth).Actz(0.0)
			f = math.Abs(f)
			if f != f {
				continue
			}
			if f >= float64(maxN) {
				f = float64(maxN) - 0.01
			}
			idx := int(math.Abs(f) * float64(k))
			// log.Println(idx, b)
			arr[idx]++
		}
	}
	for i, v := range arr {
		arr[i] = v * float64(k) / float64(r*r)
		fmt.Print(arr[i], ",")
	}
	return*/
	d := make(dot, dotSize)
	plotP := []dot{}
	list := [][]float64{}
	proj := stdProj(0, 1, X0, X1, Y0, Y1)
	refrash := make(chan struct{}, 1)
	/*
	 * THE REAL ALGO
	 *
	 * here we done the real work in two steps
	 * 1) find local min
	 * 2) spred in the valy
	 */
	//STEP 1
	for j := 0; j < pathLong; j++ {
		/*if j%(pathLong/20) == 0 {
			fmt.Println(d, math.Sqrt(errf(d)))
		}*/
		fall(d, errf, 0.01)
		d2 := make([]float64, len(d))
		copy(d2, d)
		list = append(list, d2)
	}

	//fmt.Println(d, math.Sqrt(errf(d)))

	fmt.Printf(" > start(Y/n)?\n > ")
	l, _, _ := bufio.NewReader(os.Stdin).ReadLine()
	if len(l) < 1 || l[0] == 'n' || l[0] == 'N' {
		return
	}

	sf := newsfProg()
	go prog(&sf)

	/*sf := newgs2(dot{0.0, 0.0}, 1, cord{1}, func(c cord) dot {
		return make(dot, 2)
	})*/

	//
	//----------END OF START------------
	//

	/*painf := func(x, y float64) float64 {
		m := K([]float64{x, y, 1.0}, []float64{1.0, 1.0, 1.0}, Depth)
		m2 := m.Actf(Mob{
			{p([]float64{1.0, 1.0, 1.0}, float64(Depth)), 0},
			{p([]float64{x, y, 1.0}, float64(Depth)), 1},
		})
		m2 = m2.simply()
		// log.Println(m.Actz(0.0)-m2.Actz(0.0), m, m2)
		return math.Abs((m.Actz(0.0) - m2.Actz(0.0)))
		//return m2.Actz(0.0)
	}*/

	/*painf := func(x, y float64, d int) float64 {
		v := errf(dot{x, y})
		//log.Println(C, v, v*C, math.Log(C*v))
		return math.Log(v)
	}*/

	/*cf := func(f float64) color.Color {
		if f != f {
			return color.RGBA{255, 50, 100, 0}
		}
		return color.RGBA{uint8(math.Tanh(f/10.0)*127.0 + 128), uint8(math.Tanh(f)*127.0 + 128), uint8(math.Tanh(f*10.0)*127.0 + 128), 0}
	}*/

	cf := func(f float32) color.Color {
		if f != f {
			return color.RGBA{255, 50, 100, 0}
		}
		f64 := float64(f) / C
		speed := 10.0
		b := 0.3
		return color.RGBA{uint8(math.Tanh(f64/speed-b)*79.0 + 80), uint8(math.Tanh(-f64/speed+b)*127.0 + 128), uint8(255), 0}
	}

	/*cf := func(f float64) color.Color {
		if f != f {
			return color.RGBA{255, 50, 100, 0}
		}

		return color.RGBA{}
	}*/

	pmod := 0
	q := &qtree{r: rectf([4]float32{X0, Y0, X1, Y1})}
	c := &contfrac{}
	c.d.get().proj = func(x, y float64) (a, b []float64) {
		return []float64{-10.0, y, 1.0}, []float64{x, 0.0, 1.0}
	}
	runer := &qruner{mind: minD, maxd: maxD, maxDepth: 100, c: c, ptr: make(map[[2]int]ptrStat)}
	runer.init()
	runer.pushUpt(req{h: q, d: 0, dep: Depth}, stat{
		depth: Depth,
	})
	runer.wake()

	go func() {
		for {
			time.Sleep(time.Millisecond * refrashRate)
			refrash <- struct{}{}
		}
	}()

	go func() {
		in := bufio.NewReader(os.Stdin)
		for {
			l, _, _ := in.ReadLine()
			if string(l[0:3]) == "go " {
				switch c := l[len(l)-1]; c {
				case '1', '2', '3', '4':
					q = &q.ch[c-'1']
					runer.push(req{h: q, d: 0, gen: q.gen}, false)
					runer.wake()
				case 'u':
					if q.p != nil {
						q = q.p
					}
				}
			} else if string(l[0:4]) == "set " {
				f, err := strconv.ParseFloat(string(l[4:]), 64)
				if err != nil {
					log.Println(err)
				}
				runer.mux.Lock()
				C = f
				// q.f = float32(math.NaN())
				//runer.push(req{h: q, d: 0}, true)
				// runer.wake()
				fmt.Println("C is equal", C)
				runer.mux.Unlock()
			} else if string(l[0:4]) == "rect" {
				log.Println("rect => ", q.r)
			} else if string(l[0:4]) == "res " {
				if len(l) < 6 {
					log.Println("res [1/2][+/-]")
				}
				pi := &runer.mind
				if l[4] == '2' {
					pi = &runer.maxd
				}
				if l[5] == '-' {
					*pi = *pi - 1
				} else {
					*pi = *pi + 1
					runer.push(req{q, 0, q.gen, Depth}, false)
					runer.wake()
				}
				log.Println("res =", runer.mind, runer.maxd)
			} else if string(l[0:5]) == "pmod " {
				if len(l) < 7 {
					log.Println("pmod [-d] [-n]")
				}
				if string(l[0:7]) == "pmod -d" {
					pmod = 1
				}
				if string(l[0:7]) == "pmod -n" {
					pmod = 0
				}
			} else if string(l[0:4]) == "mine" {
				pmod = 2
			}
			//refrash <- struct{}{}
		}
	}()

	driver.Main(func(s screen.Screen) {
		w, err := s.NewWindow(&screen.NewWindowOptions{
			Title: "~gallery~",
		})
		if err != nil {
			log.Fatal(err)
		}
		defer w.Release()

		quit := make(chan struct{}, 1)
		endWg := &sync.WaitGroup{}
		sz := size.Event{}
		run := false
		//eLoop:
		for {
			e := w.NextEvent()
			switch e := e.(type) {
			case lifecycle.Event:
				if e.To == lifecycle.StageDead {
					quit <- struct{}{}
					endWg.Wait()
					return
				}
			case paint.Event:
				if run {
					continue
				}
				run = true
				go func(sz *size.Event) {
					log.Println("start2")
				start:
					log.Println("start")
					szPt := image.Point{sz.WidthPx, sz.HeightPx}
					b0, err := s.NewBuffer(szPt)
					if err != nil {
						log.Fatal(err)
					}
					defer b0.Release()
					t0, err := s.NewTexture(szPt)
					if err != nil {
						log.Fatal(err)
					}
					defer t0.Release()
					t0.Upload(image.Point{}, b0, b0.Bounds())
					//drowing into bufer
				paint:
					//log.Println("paint")
					proj = stdProj(0, 1, float64(q.r[0]), float64(q.r[2]), float64(q.r[1]), float64(q.r[3]))
					img := b0.RGBA()
					draw.Draw(img, img.Bounds(), &image.Uniform{color.Black}, image.ZP, draw.Src)

					runer.mux.Lock()
					switch pmod {
					case 1:
						depthdrow(runer.max64gen, q, img, cf)
					default:
						qdrow(runer.max64gen, q, img, cf)
					}
					runer.mux.Unlock()

					for i, d := range list {
						if i == 0 {
							continue
						}
						pt := proj(d)
						pt2 := proj(list[i-1])
						x0, y0 := int(pt[0]*float64(szPt.X)), int(pt[1]*float64(szPt.Y))
						x1, y1 := int(pt2[0]*float64(szPt.X)), int(pt2[1]*float64(szPt.Y))
						drowLine(img, x0, x1, y0, y1, color.RGBA{128 + uint8((127*i)/pathLong), 0, 0, 0})

					}

					for _, d := range sf.dots() {
						pt := proj(d)
						x, y := int(pt[0]*float64(szPt.X)), int(pt[1]*float64(szPt.Y))
						drowCirc(img, x, y, 5, cfBtoR(math.Log(errf(d))))
					}

					for _, d := range plotP {
						pt := proj(d)
						x, y := int(pt[0]*float64(szPt.X)), int(pt[1]*float64(szPt.Y))
						drowCirc(img, x, y, 5, color.RGBA{255, 50, 0, 0})
					}

					w.Upload(image.Point{0, 0}, b0, b0.Bounds())
					w.Publish()
					szPt = image.Point{sz.WidthPx, sz.HeightPx}
					if len(refrash) > 0 {
						<-refrash
						if img.Bounds().Max == szPt {
							goto paint
						}
						goto start
					}
					select {
					case <-refrash:
						if img.Bounds().Max == szPt {
							goto paint
						}
						goto start
					case q := <-quit:
						quit <- q
						log.Println("quit")
					}
					run = false
				}(&sz)

			case size.Event:
				sz = e
				refrash <- struct{}{}
			case mouse.Event:
				if e.Direction != mouse.DirPress {
					continue
				}
				log.Println("click!")
				if q == nil {
					log.Println(q)
					continue
				}
				x1, y1 := e.X/float32(sz.Size().X), e.Y/float32(sz.Size().Y)
				x, y := x1*(q.r[2]-q.r[0])+q.r[0], y1*(q.r[3]-q.r[1])+q.r[1]
				f := 0.0
				switch pmod {
				case 1:
					ptQ := find(q, float64(x), float64(y), -1)
					f = float64(genstat[ptQ.gen].depth)
					log.Printf("ptQ(%.2f,%.2f) -> %f", x, y, f)
				case 2:
					log.Println("mine - used")
					ptQ := find(q, float64(x), float64(y), 2)
					ptQ.gen = runer.get(stat{
						fgen:  genstat[ptQ.gen].fgen,
						depth: genstat[ptQ.gen].depth + 40,
					})
					runer.push(req{
						h:   ptQ,
						d:   0,
						gen: ptQ.gen,
					}, false)
					runer.wake()
					pmod = 1
				default:
					ptQ := find(q, float64(x), float64(y), -1)
					f = float64(ptQ.f)
					log.Printf("ptQ(%.2f,%.2f) -> %f", x, y, f)
				}

			case error:
				log.Print(e)
			}
		}
	})

}

func stdProj(i, j int, x0, x1, y0, y1 float64) func([]float64) []float64 {
	return func(d []float64) []float64 {
		return []float64{(d[i] - x0) / (x1 - x0), (d[j] - y0) / (y1 - y0)}
	}
}
