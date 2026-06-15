package geo

import "testing"

func TestHelpers(t *testing.T) {
	nyc := Point{X: -73.9857, Y: 40.7484}
	sydney := Point{X: 151.2093, Y: -33.8688}
	if DistanceHaversine(nyc, sydney) <= 0 {
		t.Fatal("DistanceHaversine returned non-positive distance")
	}

	bound := NewBoundAroundPoint(nyc, 1000)
	if !Contains(bound, nyc) {
		t.Fatal("bound does not contain center")
	}
	if !Intersects(bound, bound) {
		t.Fatal("bound does not intersect itself")
	}

	polygon := Polygon{{
		{0, 0},
		{1, 0},
		{1, 1},
		{0, 0},
	}}
	if !PolygonContains(polygon, Point{X: 0.5, Y: 0.25}) {
		t.Fatal("polygon should contain point")
	}
}
