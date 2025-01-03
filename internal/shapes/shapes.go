package shapes

type Size struct {
	Width  int
	Height int
}

type Point struct {
	X int
	Y int
}

type Rectangle struct {
	Point
	Size
}
