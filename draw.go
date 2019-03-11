package main

import (
	"image"
	"image/color"
	"log"
	"math"
	"sync"
	"time"
)

type gen64 uint64

type ptrStat uint8

var genstat [256]stat

func drowLine(i *image.RGBA, x0, x1, y0, y1 int, c color.Color) {
	dx, dy := x1-x0, y1-y0
	if dy < 0 {
		dy = -dy
	}
	if dx < 0 {
		dx = -dx
	}
	l := dx + dy
	if l == 0 {
		l = 1
	}
	dx, dy = x1-x0, y1-y0
	for j := 0; j <= l; j++ {
		i.Set(x0+(j*dx)/l, y0+(j*dy)/l, c)
	}
}

func drowCirc(img *image.RGBA, x, y, r int, c color.Color) {
	for i := -r; i < r; i++ {
		for j := -r; j < r; j++ {
			if i*i+j*j < r*r {
				img.Set(x+i, y+j, c)
			}
		}
	}
}

//(0,0)--> x
//|1|0
//|---
//|2|3
//V
//y
type qtree struct {
	p     *qtree
	ch    *[4]qtree
	f     float32
	r     rectf
	depth uint8
	gen   ptrStat
}

//rectf = x0, y0, x1, y1 float64
type rectf [4]float32

func qdrow(gen gen64, q *qtree, img *image.RGBA, cf func(f float32) color.Color) {
	_qdrow(gen, q, img, cf, q.r)
}

func _qdrow(gen gen64, q *qtree, img *image.RGBA, cf func(f float32) color.Color, r rectf) {
	rect := img.Bounds()
	mx, my := float32(rect.Max.X-rect.Min.X), float32(rect.Max.Y-rect.Min.Y)
	if q.ch == nil || mx*my*(q.ch[0].r[2]-q.ch[0].r[0])*(q.ch[0].r[3]-q.ch[0].r[1]) < 1.0 {
		xi, yi := int(mx*(q.r[0]-r[0])/(r[2]-r[0])), int(my*(q.r[1]-r[1])/(r[3]-r[1]))
		xf, yf := int(mx*(q.r[2]-r[0])/(r[2]-r[0]))+1, int(my*(q.r[3]-r[1])/(r[3]-r[1]))+1
		cl := cf(_qdrow2(q, gen))
		//log.Println("set!", cl, xf-xi, yf-yi, "R =", q.r)
		yi0 := yi
		for ; xi < xf; xi++ {
			yi = yi0
			for ; yi < yf; yi++ {
				img.Set(xi, yi, cl)
			}
		}
		return
	}

	for i := range q.ch {
		_qdrow(gen, &q.ch[i], img, cf, r)
	}
}

func _qdrow2(q *qtree, gen gen64) float32 {
	if q.ch == nil {
		return q.f
	}
	avg := float32(0.0)
	k := 0
	for i := range q.ch {
		if genstat[q.ch[i].gen].fgen == gen {
			avg += _qdrow2(&q.ch[i], gen)
			k++
		}
	}
	if k == 0 {
		return float32(math.NaN())
	}
	return avg / float32(k)
}

type qruner struct {
	q []req

	mind, maxd uint8
	//f          func(x, y float64, parm int) float64

	maxDepth int
	// chk      func(x, y float64, parm int) float64

	c calc

	mux   sync.Locker
	state state

	//maxsubgen is a pointer to the end of genstat
	maxsubgen ptrStat
	//max64gen is the maximal fgen <req.fgen>
	max64gen gen64
	ptr      map[[2]int]ptrStat
}

type req struct {
	h *qtree
	d uint8
	//gen of type fgen
	gen ptrStat
	dep int
}

type stat struct {
	fgen  gen64
	depth int
}

func (q *qruner) get(s stat) ptrStat {
	a := [2]int{int(s.fgen), s.depth}
	v, ok := q.ptr[a]
	if !ok || genstat[v] != s {
		log.Println("not ok")
		genstat[q.maxsubgen] = s
		q.maxsubgen++
		q.ptr[a] = q.maxsubgen - 1
		log.Println("|", q.maxsubgen-1, " -> ", genstat[q.maxsubgen-1])
		return q.maxsubgen - 1
	}
	// log.Println("ok", v, " -> ", genstat[v])
	return v
}

func (q *qruner) push(r req, upt bool) {
	if upt {
		q.max64gen++
		log.Println("pushed fgen", q.max64gen)
		r.gen = q.get(stat{
			depth: r.dep,
			fgen:  q.max64gen,
		})
		log.Printf("pushed gen:%d|max gen = %d\n", r.gen, q.max64gen)
	}
	// log.Printf("pushed %#v", r)
	q.q = append(q.q, r)
}

func (q *qruner) pushUpt(r req, s stat) {
	q.max64gen++
	s.fgen = q.max64gen
	r.gen = q.get(s)
	log.Printf("<pushUpt> pushed gen:%d|max gen = %d\n", r.gen, q.max64gen)
	log.Printf("pushed %#v", r)
	q.q = append(q.q, r)
}

func (q *qruner) pop() (r req) {
	r = q.q[0]
	q.q = q.q[1:]
	return
}

func (q *qruner) empty() bool {
	return len(q.q) == 0
}

func (q *qruner) init() {
	q.mux = &sync.Mutex{}
	q.q = make([]req, 0)
	q.state = state{l: &sync.Mutex{}}
}

func (q *qruner) wake() {
	q.state.l.Lock()
	if !q.state.running {
		go q.state.run(q)
	}
	q.state.l.Unlock()
}

type state struct {
	l       sync.Locker
	running bool
}

func (s *state) run(q *qruner) {
	if &q.state != s {
		log.Println("running on the run state")
		return
	}
	t0 := time.Now()
	s.running = true
	for !q.empty() {
		// log.Println("loop the loop!")
		q.mux.Lock()
		r := q.pop()
		if r.h == nil {
			log.Println("h - del")
			continue
		}
		if genstat[r.gen].fgen != q.max64gen {
			log.Println("del", r, "-> s = ", genstat[r.gen], "|max64gen = ", q.max64gen)
			q.mux.Unlock()
			continue
		}
		//log.Println("DO SOMTHING")
		x, y := r.h.r.mid()
		S := genstat[r.gen]

		q.c.at(x, y, S.depth)
		if r.h.f == 0.0 || r.h.f != r.h.f {
			// log.Println("calc1")
			r.h.f = float32(q.c.getf())
		}

		if S.depth < q.maxDepth && q.c.chk() > eps2/4.0 {

			r.gen = q.get(stat{
				depth: S.depth + 10,
				fgen:  S.fgen,
			})
			q.push(r, false)
			// log.Println("0", q.c.chk(), "dep,S", S.depth, S)
			// log.Println("r.gen -> s", genstat[r.gen])
			q.mux.Unlock()
			continue
		}

		if r.h.ch == nil || r.h.gen != r.gen {
			if r.d < q.maxd {
				V := [4]float32{}
				for i := range r.h.ch {
					xi, yi := subi(i, r.h.r).mid()
					d := S.depth
					q.c.at(xi, yi, d)
					V[i] = float32(q.c.getf())
					// log.Println("calc2", V[i], r.h.r, "stat=", d, S.depth, xi, yi)
				}
				if r.d < q.mind || dev(V) > eps2 {
					//log.Println("other")
					bNil := r.h.ch == nil
					if bNil {
						r.h.ch = &[4]qtree{}
					}
					for i := range r.h.ch {
						if bNil {
							r.h.ch[i] = qtree{
								p:     r.h,
								depth: r.d + 1,
								r:     subi(i, r.h.r),
							}
						} else {
							// log.Println("ELSE")
						}
						r.h.ch[i].f = V[i]
						// log.Println("data:", r.h.ch[i].f, r.h.r)
						r.h.ch[i].gen = r.gen
						q.push(req{
							h:   &r.h.ch[i],
							d:   r.d + 1,
							gen: r.gen,
						}, false)
					}
				}
			}
		} else {
			//log.Println("child")
			for i := range r.h.ch {
				q.push(req{
					h:   &r.h.ch[i],
					d:   r.d + 1,
					gen: r.gen,
				}, false)
			}
		}
		q.mux.Unlock()
	}
	s.running = false
	log.Println("END RUN!", time.Now().Sub(t0))
}

func create(mind, maxd int, f func(x, y float64) float64, r rectf) *qtree {
	if r[0] > r[2] {
		r[0], r[2] = r[2], r[0]
	}
	if r[1] > r[3] {
		r[1], r[3] = r[3], r[1]
	}
	return ncreate(0, mind, maxd, f, r, float32(math.NaN()))
}

func ncreate(d, mind, maxd int, f func(x, y float64) float64, r rectf, v float32) *qtree {
	q := &qtree{}
	q.r = r
	//if v is NaN {...
	if v != v {
		q.f = float32(f(r.mid()))
	} else {
		q.f = v
	}

	if d > maxd {
		return q
	}

	V := [4]float32{}
	for i := range q.ch {
		V[i] = float32(f(subi(i, r).mid()))
	}

	if d < mind || dev(V) > eps2 {
		for i := range q.ch {
			//log.Println("subi(i, r) on", i, r, " => ", subi(i, r))
			q.ch[i] = *ncreate(d+1, mind, maxd, f, subi(i, r), V[i])
		}
	}
	// now cheak if we need to create sub childes
	return q
}

func subi(i int, r rectf) rectf {
	xm64, ym64 := r.mid()
	xm, ym := float32(xm64), float32(ym64)
	switch i {
	case 0:
		return rectf([4]float32{xm, r[1], r[2], ym})
	case 1:
		return rectf([4]float32{r[0], r[1], xm, ym})
	case 2:
		return rectf([4]float32{r[0], ym, xm, r[3]})
	}
	return rectf([4]float32{xm, ym, r[2], r[3]})
}

func (r rectf) mid() (x, y float64) {
	return float64((r[0] + r[2]) / 2.0), float64((r[1] + r[3]) / 2.0)
}

func dev(V [4]float32) float32 {
	d := (V[0]+V[2]-V[1]-V[3])*(V[0]+V[2]-V[1]-V[3]) + (V[0]+V[1]-V[2]-V[3])*(V[0]+V[1]-V[2]-V[3])
	s := 0.0
	for i := range V {
		s += math.Abs(float64(V[i]))
	}
	return (d * 4) / float32(s)
}

func depthdrow(gen gen64, q *qtree, img *image.RGBA, cf func(f float32) color.Color) {
	_depthdrow(gen, q, img, cf, q.r)
}

func _depthdrow(gen gen64, q *qtree, img *image.RGBA, cf func(f float32) color.Color, r rectf) {
	rect := img.Bounds()
	mx, my := float32(rect.Max.X-rect.Min.X), float32(rect.Max.Y-rect.Min.Y)
	if q.ch == nil || mx*my*(q.ch[0].r[2]-q.ch[0].r[0])*(q.ch[0].r[3]-q.ch[0].r[1]) < 1.0 {
		xi, yi := int(mx*(q.r[0]-r[0])/(r[2]-r[0])), int(my*(q.r[1]-r[1])/(r[3]-r[1]))
		xf, yf := int(mx*(q.r[2]-r[0])/(r[2]-r[0]))+1, int(my*(q.r[3]-r[1])/(r[3]-r[1]))+1
		cl := cf(_depthdrow2(q, gen))
		//log.Println("set!", cl, xf-xi, yf-yi, "R =", q.r)
		yi0 := yi
		for ; xi < xf; xi++ {
			yi = yi0
			for ; yi < yf; yi++ {
				img.Set(xi, yi, cl)
			}
		}
		return
	}

	for i := range q.ch {
		_depthdrow(gen, &q.ch[i], img, cf, r)
	}
}

func _depthdrow2(q *qtree, gen gen64) float32 {
	if q.ch == nil {
		return float32(genstat[q.gen].depth)
	}
	avg := float32(0.0)
	k := 0
	for i := range q.ch {
		if genstat[q.ch[i].gen].fgen == gen {
			avg += _depthdrow2(&q.ch[i], gen)
			k++
		}
	}
	if k == 0 {
		return float32(math.NaN())
	}
	return avg / float32(k)
}

func find(root *qtree, x, y float64, d int) *qtree {
	if root.ch == nil || d == 0 {
		return root
	}
	i := 1
	x0, y0 := root.r.mid()
	if x > x0 {
		i = 3
	}

	if y < y0 {
		i = -i
	}

	i += 3
	i = i / 2

	return find(&root.ch[i], x, y, d-1)
}

//(-Inf,Inf)
func cfBtoR(f float64) (c color.RGBA) {
	c.R = uint8(math.Tanh(f/5)*127.5 + 127.5)
	c.B = uint8(math.Tanh(-f/5)*127.5 + 127.5)
	return
}
