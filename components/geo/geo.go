package geo

import (
	"encoding/json"
	"fmt"
	stdmath "math"

	ronmath "github.com/starfederation/ron-go/components/math"
)

const EarthRadius = 6378137.0

// Use local math types for public #geo helpers.
type (
	Point = ronmath.Vector2[float64]
	Bound = ronmath.Box2[float64]
)

// Geometry is any local geo geometry value.
type Geometry = any

// Local geometry collection types.
type (
	Collection      []Geometry
	LineString      []Point
	MultiLineString []LineString
	MultiPoint      []Point
	MultiPolygon    []Polygon
	Polygon         []Ring
	Ring            []Point
)

// Value is a GeoJSON geometry, feature, or feature collection value.
type Value struct {
	Data any
}

// Distance returns the geodesic distance in meters between lon/lat points.
func Distance(p1, p2 Point) float64 {
	dLat := deg2rad(p1.Y - p2.Y)
	dLon := stdmath.Abs(deg2rad(p1.X - p2.X))
	if dLon > stdmath.Pi {
		dLon = 2*stdmath.Pi - dLon
	}
	x := dLon * stdmath.Cos(deg2rad((p1.Y+p2.Y)/2))

	return stdmath.Sqrt(dLat*dLat+x*x) * EarthRadius
}

// DistanceHaversine returns the haversine distance in meters between lon/lat points.
func DistanceHaversine(p1, p2 Point) float64 {
	dLat := deg2rad(p1.Y - p2.Y)
	dLon := deg2rad(p1.X - p2.X)
	dLat2Sin := stdmath.Sin(dLat / 2)
	dLon2Sin := stdmath.Sin(dLon / 2)
	a := dLat2Sin*dLat2Sin + stdmath.Cos(deg2rad(p2.Y))*stdmath.Cos(deg2rad(p1.Y))*dLon2Sin*dLon2Sin

	return 2 * EarthRadius * stdmath.Atan2(stdmath.Sqrt(a), stdmath.Sqrt(1-a))
}

// Bearing returns the bearing in degrees from one lon/lat point to another.
func Bearing(from, to Point) float64 {
	dLon := deg2rad(to.X - from.X)
	fromLatRad := deg2rad(from.Y)
	toLatRad := deg2rad(to.Y)
	y := stdmath.Sin(dLon) * stdmath.Cos(toLatRad)
	x := stdmath.Cos(fromLatRad)*stdmath.Sin(toLatRad) - stdmath.Sin(fromLatRad)*stdmath.Cos(toLatRad)*stdmath.Cos(dLon)

	return rad2deg(stdmath.Atan2(y, x))
}

// Midpoint returns the geodesic midpoint between lon/lat points.
func Midpoint(p1, p2 Point) Point {
	dLon := deg2rad(p2.X - p1.X)
	aLatRad := deg2rad(p1.Y)
	bLatRad := deg2rad(p2.Y)
	x := stdmath.Cos(bLatRad) * stdmath.Cos(dLon)
	y := stdmath.Cos(bLatRad) * stdmath.Sin(dLon)

	return Point{
		X: rad2deg(deg2rad(p1.X) + stdmath.Atan2(y, stdmath.Cos(aLatRad)+x)),
		Y: rad2deg(stdmath.Atan2(
			stdmath.Sin(aLatRad)+stdmath.Sin(bLatRad),
			stdmath.Sqrt((stdmath.Cos(aLatRad)+x)*(stdmath.Cos(aLatRad)+x)+y*y),
		)),
	}
}

// PointAtBearingAndDistance returns a point from p at bearing and distance in meters.
func PointAtBearingAndDistance(p Point, bearing, distance float64) Point {
	aLat := deg2rad(p.Y)
	aLon := deg2rad(p.X)
	bearingRadians := deg2rad(bearing)
	distanceRatio := distance / EarthRadius
	bLat := stdmath.Asin(stdmath.Sin(aLat)*stdmath.Cos(distanceRatio) + stdmath.Cos(aLat)*stdmath.Sin(distanceRatio)*stdmath.Cos(bearingRadians))
	bLon := aLon + stdmath.Atan2(
		stdmath.Sin(bearingRadians)*stdmath.Sin(distanceRatio)*stdmath.Cos(aLat),
		stdmath.Cos(distanceRatio)-stdmath.Sin(aLat)*stdmath.Sin(bLat),
	)

	return Point{
		X: rad2deg(bLon),
		Y: rad2deg(bLat),
	}
}

// NewBoundAroundPoint returns a lon/lat bound around center with radius in meters.
func NewBoundAroundPoint(center Point, distance float64) Bound {
	radDist := distance / EarthRadius
	radLat := deg2rad(center.Y)
	radLon := deg2rad(center.X)
	minLat := radLat - radDist
	maxLat := radLat + radDist

	var minLon, maxLon float64
	if minLat > minLatitude && maxLat < maxLatitude {
		deltaLon := stdmath.Asin(stdmath.Sin(radDist) / stdmath.Cos(radLat))
		minLon = radLon - deltaLon
		if minLon < minLongitude {
			minLon += 2 * stdmath.Pi
		}
		maxLon = radLon + deltaLon
		if maxLon > maxLongitude {
			maxLon -= 2 * stdmath.Pi
		}
	} else {
		minLat = stdmath.Max(minLat, minLatitude)
		maxLat = stdmath.Min(maxLat, maxLatitude)
		minLon = minLongitude
		maxLon = maxLongitude
	}

	return Bound{
		Min: Point{
			X: rad2deg(minLon),
			Y: rad2deg(minLat),
		},
		Max: Point{
			X: rad2deg(maxLon),
			Y: rad2deg(maxLat),
		},
	}
}

// BoundWidth returns the geodesic width of a lon/lat bound in meters.
func BoundWidth(bound Bound) float64 {
	centerLat := (bound.Min.Y + bound.Max.Y) / 2

	return Distance(
		Point{
			X: bound.Min.X,
			Y: centerLat,
		},
		Point{
			X: bound.Max.X,
			Y: centerLat,
		},
	)
}

// BoundHeight returns the geodesic height of a lon/lat bound in meters.
func BoundHeight(bound Bound) float64 {
	return 111131.75 * (bound.Max.Y - bound.Min.Y)
}

// BoundPad pads a lon/lat bound by meters.
func BoundPad(bound Bound, meters float64) Bound {
	dy := meters / 111131.75
	dx := dy / stdmath.Cos(deg2rad(bound.Max.Y))
	dx = stdmath.Max(dx, dy/stdmath.Cos(deg2rad(bound.Min.Y)))
	bound.Min.X = stdmath.Max(bound.Min.X-dx, -180)
	bound.Min.Y = stdmath.Max(bound.Min.Y-dy, -90)
	bound.Max.X = stdmath.Min(bound.Max.X+dx, 180)
	bound.Max.Y = stdmath.Min(bound.Max.Y+dy, 90)

	return bound
}

// Contains reports whether bound contains point.
func Contains(bound Bound, point Point) bool {
	return point.X >= bound.Min.X && point.X <= bound.Max.X && point.Y >= bound.Min.Y && point.Y <= bound.Max.Y
}

// Intersects reports whether two bounds intersect.
func Intersects(a, b Bound) bool {
	return !(b.Max.X < a.Min.X || b.Min.X > a.Max.X || b.Max.Y < a.Min.Y || b.Min.Y > a.Max.Y)
}

// Area returns geodesic area in square meters for lon/lat geometry.
func Area(g Geometry) float64 {
	if g == nil {
		return 0
	}

	switch g := g.(type) {
	case Point, MultiPoint, LineString, MultiLineString:
		return 0
	case Ring:
		return stdmath.Abs(ringArea(g))
	case Polygon:
		return polygonArea(g)
	case MultiPolygon:
		sum := 0.0
		for _, polygon := range g {
			sum += polygonArea(polygon)
		}
		return sum
	case Collection:
		sum := 0.0
		for _, child := range g {
			sum += Area(child)
		}
		return sum
	case Bound:
		return Area(boundToRing(g))
	default:
		panic(fmt.Sprintf("geometry type not supported: %T", g))
	}
}

// Length returns geodesic length in meters for lon/lat geometry.
func Length(g Geometry) float64 {
	if g == nil {
		return 0
	}

	switch g := g.(type) {
	case Point, MultiPoint:
		return 0
	case LineString:
		return lineStringLength(g, Distance)
	case MultiLineString:
		sum := 0.0
		for _, line := range g {
			sum += lineStringLength(line, Distance)
		}
		return sum
	case Ring:
		return lineStringLength(LineString(g), Distance)
	case Polygon:
		sum := 0.0
		for _, ring := range g {
			sum += lineStringLength(LineString(ring), Distance)
		}
		return sum
	case MultiPolygon:
		sum := 0.0
		for _, polygon := range g {
			for _, ring := range polygon {
				sum += lineStringLength(LineString(ring), Distance)
			}
		}
		return sum
	case Collection:
		sum := 0.0
		for _, child := range g {
			sum += Length(child)
		}
		return sum
	case Bound:
		return Length(boundToRing(g))
	default:
		panic(fmt.Sprintf("geometry type not supported: %T", g))
	}
}

// PlanarDistance returns Euclidean distance between points.
func PlanarDistance(p1, p2 Point) float64 {
	return stdmath.Hypot(p1.X-p2.X, p1.Y-p2.Y)
}

// PlanarLength returns Euclidean length for geometry.
func PlanarLength(g Geometry) float64 {
	if g == nil {
		return 0
	}

	switch g := g.(type) {
	case Point, MultiPoint:
		return 0
	case LineString:
		return lineStringLength(g, PlanarDistance)
	case MultiLineString:
		sum := 0.0
		for _, line := range g {
			sum += lineStringLength(line, PlanarDistance)
		}
		return sum
	case Ring:
		return lineStringLength(LineString(g), PlanarDistance)
	case Polygon:
		sum := 0.0
		for _, ring := range g {
			sum += lineStringLength(LineString(ring), PlanarDistance)
		}
		return sum
	case MultiPolygon:
		sum := 0.0
		for _, polygon := range g {
			for _, ring := range polygon {
				sum += lineStringLength(LineString(ring), PlanarDistance)
			}
		}
		return sum
	case Collection:
		sum := 0.0
		for _, child := range g {
			sum += PlanarLength(child)
		}
		return sum
	case Bound:
		return PlanarLength(boundToRing(g))
	default:
		panic(fmt.Sprintf("geometry type not supported: %T", g))
	}
}

// PlanarArea returns Euclidean area for geometry.
func PlanarArea(g Geometry) float64 {
	if g == nil {
		return 0
	}

	switch g := g.(type) {
	case Point, MultiPoint, LineString, MultiLineString:
		return 0
	case Ring:
		return stdmath.Abs(planarRingArea(g))
	case Polygon:
		return planarPolygonArea(g)
	case MultiPolygon:
		sum := 0.0
		for _, polygon := range g {
			sum += planarPolygonArea(polygon)
		}
		return sum
	case Collection:
		sum := 0.0
		for _, child := range g {
			sum += PlanarArea(child)
		}
		return sum
	case Bound:
		return PlanarArea(boundToRing(g))
	default:
		panic(fmt.Sprintf("geometry type not supported: %T", g))
	}
}

// PolygonContains reports whether polygon contains point in planar coordinates.
func PolygonContains(polygon Polygon, point Point) bool {
	if len(polygon) == 0 || !RingContains(polygon[0], point) {
		return false
	}
	for _, hole := range polygon[1:] {
		if RingContains(hole, point) {
			return false
		}
	}

	return true
}

// RingContains reports whether ring contains point in planar coordinates.
func RingContains(ring Ring, point Point) bool {
	contains := false
	j := len(ring) - 1
	for i := range ring {
		if ((ring[i].Y > point.Y) != (ring[j].Y > point.Y)) &&
			(point.X < (ring[j].X-ring[i].X)*(point.Y-ring[i].Y)/(ring[j].Y-ring[i].Y)+ring[i].X) {
			contains = !contains
		}
		j = i
	}

	return contains
}

// MultiPolygonContains reports whether multipolygon contains point in planar coordinates.
func MultiPolygonContains(multiPolygon MultiPolygon, point Point) bool {
	for _, polygon := range multiPolygon {
		if PolygonContains(polygon, point) {
			return true
		}
	}

	return false
}

func lineStringLength(line LineString, distance func(Point, Point) float64) float64 {
	sum := 0.0
	for i := 1; i < len(line); i++ {
		sum += distance(line[i], line[i-1])
	}

	return sum
}

func ringArea(ring Ring) float64 {
	if len(ring) < 3 {
		return 0
	}

	var lo, mi, hi int
	length := len(ring)
	if ring[0] != ring[len(ring)-1] {
		length++
	}
	area := 0.0
	for i := 0; i < length; i++ {
		switch i {
		case length - 3:
			lo = length - 3
			mi = length - 2
			hi = 0
		case length - 2:
			lo = length - 2
			mi = 0
			hi = 0
		case length - 1:
			lo = 0
			mi = 0
			hi = 1
		default:
			lo = i
			mi = i + 1
			hi = i + 2
		}
		area += (deg2rad(ring[hi].X) - deg2rad(ring[lo].X)) * stdmath.Sin(deg2rad(ring[mi].Y))
	}

	return -area * EarthRadius * EarthRadius / 2
}

func polygonArea(polygon Polygon) float64 {
	if len(polygon) == 0 {
		return 0
	}
	sum := stdmath.Abs(ringArea(polygon[0]))
	for i := 1; i < len(polygon); i++ {
		sum -= stdmath.Abs(ringArea(polygon[i]))
	}

	return sum
}

func planarRingArea(ring Ring) float64 {
	if len(ring) < 3 {
		return 0
	}
	sum := 0.0
	previous := ring[len(ring)-1]
	for _, point := range ring {
		sum += previous.X*point.Y - point.X*previous.Y
		previous = point
	}

	return sum / 2
}

func planarPolygonArea(polygon Polygon) float64 {
	if len(polygon) == 0 {
		return 0
	}
	sum := stdmath.Abs(planarRingArea(polygon[0]))
	for i := 1; i < len(polygon); i++ {
		sum -= stdmath.Abs(planarRingArea(polygon[i]))
	}

	return sum
}

func boundToRing(bound Bound) Ring {
	return Ring{
		{
			X: bound.Min.X,
			Y: bound.Min.Y,
		},
		{
			X: bound.Max.X,
			Y: bound.Min.Y,
		},
		{
			X: bound.Max.X,
			Y: bound.Max.Y,
		},
		{
			X: bound.Min.X,
			Y: bound.Max.Y,
		},
		{
			X: bound.Min.X,
			Y: bound.Min.Y,
		},
	}
}

func deg2rad(degrees float64) float64 {
	return degrees * stdmath.Pi / 180
}

func rad2deg(radians float64) float64 {
	return 180 * radians / stdmath.Pi
}

var (
	minLatitude  = deg2rad(-90)
	maxLatitude  = deg2rad(90)
	minLongitude = deg2rad(-180)
	maxLongitude = deg2rad(180)
)

// Valid reports whether value has a supported RFC 7946 GeoJSON shape.
func Valid(value any) bool {
	object, ok := value.(map[string]any)
	if !ok {
		return false
	}
	geoType, ok := object["type"].(string)
	if !ok {
		return false
	}
	switch geoType {
	case "Point":
		return validPosition(object["coordinates"])
	case "MultiPoint", "LineString":
		return validNestedPositions(object["coordinates"], 1)
	case "MultiLineString", "Polygon":
		return validNestedPositions(object["coordinates"], 2)
	case "MultiPolygon":
		return validNestedPositions(object["coordinates"], 3)
	case "GeometryCollection":
		geometries, ok := object["geometries"].([]any)
		if !ok {
			return false
		}
		for _, geometry := range geometries {
			if !Valid(geometry) {
				return false
			}
		}
		return true
	case "Feature":
		if geometry := object["geometry"]; geometry != nil && !Valid(geometry) {
			return false
		}
		_, ok := object["properties"]
		return ok
	case "FeatureCollection":
		features, ok := object["features"].([]any)
		if !ok {
			return false
		}
		for _, feature := range features {
			if !Valid(feature) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func validNestedPositions(value any, depth int) bool {
	values, ok := value.([]any)
	if !ok || len(values) == 0 {
		return false
	}
	for _, value := range values {
		if depth == 1 {
			if !validPosition(value) {
				return false
			}
		} else if !validNestedPositions(value, depth-1) {
			return false
		}
	}
	return true
}

func validPosition(value any) bool {
	values, ok := value.([]any)
	if !ok || len(values) < 2 || len(values) > 3 {
		return false
	}
	for _, value := range values {
		switch value.(type) {
		case json.Number, float64, int64, uint64:
		default:
			return false
		}
	}
	return true
}
