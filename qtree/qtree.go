package qtree

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func NewPoint(x, y float64) *Point {
	return &Point{X: x, Y: y}
}

type Rectangle struct {
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
	HW float64 `json:"hw"`
	HH float64 `json:"hh"`
}

func NewRectangle(x, y, hw, hh float64) *Rectangle {
	return &Rectangle{
		X:  x,
		Y:  y,
		HW: hw,
		HH: hh,
	}
}

func (r *Rectangle) Contains(p *Point) bool {
	return p.X >= r.X-r.HW &&
		p.X <= r.X+r.HW &&
		p.Y >= r.Y-r.HH &&
		p.Y <= r.Y+r.HH
}

func (r *Rectangle) Intersects(other *Rectangle) bool {
	return !(other.X-other.HW > r.X+r.HW ||
		other.X+other.HW < r.X-r.HW ||
		other.Y-other.HH > r.Y+r.HH ||
		other.Y+other.HH < r.Y-r.HH)
}

type Qtree struct {
	Boundary *Rectangle `json:"boundary"`
	Capacity int        `json:"capacity"`
	Points   []*Point   `json:"points"`

	NorthEast *Qtree `json:"northEast"`
	NorthWest *Qtree `json:"northWest"`
	SouthEast *Qtree `json:"southEast"`
	SouthWest *Qtree `json:"southWest"`

	Divided bool `json:"Divided"`
}

func NewQtree(Boundary *Rectangle, Capacity int) *Qtree {
	return &Qtree{
		Boundary: Boundary,
		Capacity: Capacity,
		Points:   make([]*Point, 0, Capacity),
	}
}

func (q *Qtree) Subdivide() {
	x := q.Boundary.X
	y := q.Boundary.Y
	hw := q.Boundary.HW / 2
	hh := q.Boundary.HH / 2

	q.NorthEast = NewQtree(
		NewRectangle(x+hw, y-hh, hw, hh),
		q.Capacity,
	)

	q.NorthWest = NewQtree(
		NewRectangle(x-hw, y-hh, hw, hh),
		q.Capacity,
	)

	q.SouthEast = NewQtree(
		NewRectangle(x+hw, y+hh, hw, hh),
		q.Capacity,
	)

	q.SouthWest = NewQtree(
		NewRectangle(x-hw, y+hh, hw, hh),
		q.Capacity,
	)

	q.Divided = true
}

func (q *Qtree) Insert(point Point) bool {

	//check the Boundary if not within Boundary then return
	if !q.Boundary.Contains(&point) {
		return false
	}

	//if within the Boundary then check for
	// available space , if not then subdevide and
	//call insert on those Divided parts to place the
	//point perfectly
	if len(q.Points) < q.Capacity {
		q.Points = append(q.Points, &point)
		return true
	}

	//because not enough space then subdevide and
	if !q.Divided {
		q.Subdivide()
	}

	//try to put that point recursively
	return q.NorthEast.Insert(point) ||
		q.NorthWest.Insert(point) ||
		q.SouthEast.Insert(point) ||
		q.SouthWest.Insert(point)

}

/*
func (q *Qtree) Query(rangeRect *Rectangle, found *[]*Point) {
	if !q.Boundary.Intersects(rangeRect) {
		return
	}

	for _, p := range q.Points {
		if rangeRect.Contains(p) {
			*found = append(*found, p)
		}
	}

	if q.Divided {
		q.NorthEast.Query(rangeRect, found)
		q.NorthWest.Query(rangeRect, found)
		q.SouthEast.Query(rangeRect, found)
		q.SouthWest.Query(rangeRect, found)
	}
}

Usage example:

found := []*Point{}
qt.Query(searchArea, &found)
*/
