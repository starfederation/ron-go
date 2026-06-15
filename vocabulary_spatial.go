package ron

import (
	"github.com/paulmach/orb"
	ronmath "github.com/starfederation/ron-go/components/math"
)

const (
	// VocabularySpatialV1 is the RON spatial typed vocabulary URI.
	VocabularySpatialV1 = "https://ron.dev/vocab/spatial/v1"
)

// LngLatAlt is a spatial vocabulary #lla value.
type LngLatAlt struct {
	Point    orb.Point
	Altitude float64
}

// Spherical is a spatial vocabulary #sph value.
type Spherical = ronmath.Spherical[float64]

// Cylindrical is a spatial vocabulary #cyl value.
type Cylindrical = ronmath.Cylindrical[float64]

// Box2 is a spatial vocabulary #bx2 value.
type Box2 = orb.Bound

// Box3 is a spatial vocabulary #bx3 value.
type Box3 = ronmath.Box3[float64]

// Sphere is a spatial vocabulary #spr value.
type Sphere = ronmath.Sphere[float64]

// Plane is a spatial vocabulary #pln value.
type Plane = ronmath.Plane[float64]

// Ray is a spatial vocabulary #ray value.
type Ray = ronmath.Ray[float64]

// Line2 is a spatial vocabulary #ln2 value.
type Line2 struct {
	Line orb.LineString
}

// Line3 is a spatial vocabulary #ln3 value.
type Line3 = ronmath.Line3[float64]

// Triangle is a spatial vocabulary #tri value.
type Triangle = ronmath.Triangle[float64]

// Frustum is a spatial vocabulary #fru value.
type Frustum [6]Plane

// SphericalHarmonics3 is a spatial vocabulary #sh3 value.
type SphericalHarmonics3 = ronmath.SphericalHarmonics3[float64]

// VoxelSet is a spatial vocabulary #vox value.
type VoxelSet struct {
	Dimensions int64
	Origin     VectorN
	CellSize   VectorN
	Cells      []VoxelCell
}

// VoxelCell is one sparse #vox coordinate/value pair.
type VoxelCell struct {
	Coordinate []int64
	Value      any
}

func (opts optionState) isSpatialTag(tag string) bool {
	if _, ok := opts.vocabularies[VocabularySpatialV1]; !ok {
		return false
	}
	switch tag {
	case "#lla", "#sph", "#cyl", "#bx2", "#bx3", "#spr", "#pln", "#ray", "#ln2", "#ln3", "#tri", "#fru", "#sh3", "#vox":
		return true
	default:
		return false
	}
}

func (opts optionState) parseSpatialPayload(tag string, payload any) (any, error) {
	switch tag {
	case "#lla":
		values, err := parseSpatialFloatTuple(payload, 3, "invalid #lla payload")
		if err != nil {
			return nil, err
		}
		return LngLatAlt{Point: orb.Point{values[0], values[1]}, Altitude: values[2]}, nil
	case "#sph":
		values, err := parseSpatialFloatTuple(payload, 3, "invalid #sph payload")
		if err != nil {
			return nil, err
		}
		return Spherical{Radius: values[0], Phi: values[1], Theta: values[2]}, nil
	case "#cyl":
		values, err := parseSpatialFloatTuple(payload, 3, "invalid #cyl payload")
		if err != nil {
			return nil, err
		}
		return Cylindrical{Radius: values[0], Theta: values[1], Y: values[2]}, nil
	case "#bx2":
		vectors, err := parseSpatialFloatTuples(payload, 2, 2, "invalid #bx2 payload")
		if err != nil {
			return nil, err
		}
		return Box2{Min: orb.Point{vectors[0][0], vectors[0][1]}, Max: orb.Point{vectors[1][0], vectors[1][1]}}, nil
	case "#bx3":
		vectors, err := parseSpatialFloatTuples(payload, 2, 3, "invalid #bx3 payload")
		if err != nil {
			return nil, err
		}
		return Box3{Min: vector3FromSlice(vectors[0]), Max: vector3FromSlice(vectors[1])}, nil
	case "#spr":
		values, ok := payload.([]any)
		if !ok || len(values) != 2 {
			return nil, newError("invalid #spr payload")
		}
		center, err := parseSpatialFloatTuple(values[0], 3, "invalid #spr payload")
		if err != nil {
			return nil, err
		}
		radius, ok := numberAsFloat64(values[1])
		if !ok {
			return nil, newError("invalid #spr payload")
		}
		return Sphere{Center: vector3FromSlice(center), Radius: radius}, nil
	case "#pln":
		return parsePlane(payload, "invalid #pln payload")
	case "#ray":
		vectors, err := parseSpatialFloatTuples(payload, 2, 3, "invalid #ray payload")
		if err != nil {
			return nil, err
		}
		return Ray{Origin: vector3FromSlice(vectors[0]), Dir: vector3FromSlice(vectors[1])}, nil
	case "#ln2":
		vectors, err := parseSpatialFloatTuples(payload, 2, 2, "invalid #ln2 payload")
		if err != nil {
			return nil, err
		}
		return Line2{Line: orb.LineString{{vectors[0][0], vectors[0][1]}, {vectors[1][0], vectors[1][1]}}}, nil
	case "#ln3":
		vectors, err := parseSpatialFloatTuples(payload, 2, 3, "invalid #ln3 payload")
		if err != nil {
			return nil, err
		}
		return Line3{Start: vector3FromSlice(vectors[0]), End: vector3FromSlice(vectors[1])}, nil
	case "#tri":
		vectors, err := parseSpatialFloatTuples(payload, 3, 3, "invalid #tri payload")
		if err != nil {
			return nil, err
		}
		return Triangle{A: vector3FromSlice(vectors[0]), B: vector3FromSlice(vectors[1]), C: vector3FromSlice(vectors[2])}, nil
	case "#fru":
		values, ok := payload.([]any)
		if !ok || len(values) != 6 {
			return nil, newError("invalid #fru payload")
		}
		var frustum Frustum
		for i, value := range values {
			plane, err := parsePlane(value, "invalid #fru payload")
			if err != nil {
				return nil, err
			}
			frustum[i] = plane
		}
		return frustum, nil
	case "#sh3":
		vectors, err := parseSpatialFloatTuples(payload, 9, 3, "invalid #sh3 payload")
		if err != nil {
			return nil, err
		}
		var harmonics SphericalHarmonics3
		for i := range vectors {
			harmonics[i] = vector3FromSlice(vectors[i])
		}
		return harmonics, nil
	case "#vox":
		return opts.parseVoxelSet(payload)
	default:
		return nil, newError("unsupported spatial tag")
	}
}

func spatialTaggedMember(value any) (objectMember, bool) {
	switch value := value.(type) {
	case LngLatAlt:
		return objectMember{Key: "#lla", Value: []any{value.Point[0], value.Point[1], value.Altitude}}, true
	case Spherical:
		return objectMember{Key: "#sph", Value: []any{value.Radius, value.Phi, value.Theta}}, true
	case Cylindrical:
		return objectMember{Key: "#cyl", Value: []any{value.Radius, value.Theta, value.Y}}, true
	case Box2:
		return objectMember{Key: "#bx2", Value: []any{[]any{value.Min[0], value.Min[1]}, []any{value.Max[0], value.Max[1]}}}, true
	case Box3:
		return objectMember{Key: "#bx3", Value: []any{anyVector3(value.Min), anyVector3(value.Max)}}, true
	case Sphere:
		return objectMember{Key: "#spr", Value: []any{anyVector3(value.Center), value.Radius}}, true
	case Plane:
		return objectMember{Key: "#pln", Value: []any{anyVector3(value.Normal), value.Constant}}, true
	case Ray:
		return objectMember{Key: "#ray", Value: []any{anyVector3(value.Origin), anyVector3(value.Dir)}}, true
	case Line2:
		if len(value.Line) != 2 {
			return objectMember{}, false
		}
		return objectMember{Key: "#ln2", Value: []any{[]any{value.Line[0][0], value.Line[0][1]}, []any{value.Line[1][0], value.Line[1][1]}}}, true
	case Line3:
		return objectMember{Key: "#ln3", Value: []any{anyVector3(value.Start), anyVector3(value.End)}}, true
	case Triangle:
		return objectMember{Key: "#tri", Value: []any{anyVector3(value.A), anyVector3(value.B), anyVector3(value.C)}}, true
	case Frustum:
		planes := make([]any, len(value))
		for i, plane := range value {
			planes[i] = []any{anyVector3(plane.Normal), plane.Constant}
		}
		return objectMember{Key: "#fru", Value: planes}, true
	case SphericalHarmonics3:
		vectors := make([]any, len(value))
		for i, vector := range value {
			vectors[i] = anyVector3(vector)
		}
		return objectMember{Key: "#sh3", Value: vectors}, true
	case VoxelSet:
		return objectMember{Key: "#vox", Value: renderVoxelSet(value)}, true
	default:
		return objectMember{}, false
	}
}

func parseSpatialFloatTuple(value any, size int, message string) ([]float64, error) {
	values, ok := value.([]any)
	if !ok || len(values) != size {
		return nil, newError(message)
	}
	parsed, err := parseFloatVector(values)
	if err != nil {
		return nil, newError(message)
	}
	return parsed, nil
}

func parseSpatialFloatTuples(value any, count, size int, message string) ([][]float64, error) {
	values, ok := value.([]any)
	if !ok || len(values) != count {
		return nil, newError(message)
	}
	parsed := make([][]float64, count)
	for i, value := range values {
		vector, err := parseSpatialFloatTuple(value, size, message)
		if err != nil {
			return nil, err
		}
		parsed[i] = vector
	}
	return parsed, nil
}

func parsePlane(value any, message string) (Plane, error) {
	values, ok := value.([]any)
	if !ok || len(values) != 2 {
		return Plane{}, newError(message)
	}
	normal, err := parseSpatialFloatTuple(values[0], 3, message)
	if err != nil {
		return Plane{}, err
	}
	constant, ok := numberAsFloat64(values[1])
	if !ok {
		return Plane{}, newError(message)
	}
	return Plane{Normal: vector3FromSlice(normal), Constant: constant}, nil
}

func (opts optionState) parseVoxelSet(payload any) (VoxelSet, error) {
	object, ok := payload.(orderedObject)
	if !ok {
		if objectMap, mapOK := payload.(map[string]any); mapOK {
			object = orderedObject{Members: objectMembers(objectMap, false)}
		} else {
			return VoxelSet{}, newError("invalid #vox payload")
		}
	}
	var voxel VoxelSet
	seen := map[string]bool{}
	for _, member := range object.Members {
		seen[member.Key] = true
		switch member.Key {
		case "dimensions":
			dimensions, ok := numberAsInt64(member.Value)
			if !ok || dimensions <= 0 {
				return VoxelSet{}, newError("invalid #vox payload")
			}
			voxel.Dimensions = dimensions
		case "origin":
			parsed, err := opts.parseVocabularyValue(member.Value)
			if err != nil {
				return VoxelSet{}, err
			}
			origin, ok := parsed.(VectorN)
			if !ok || int64(len(origin)) != voxel.Dimensions && voxel.Dimensions != 0 {
				return VoxelSet{}, newError("invalid #vox payload")
			}
			voxel.Origin = origin
		case "cellSize":
			parsed, err := opts.parseVocabularyValue(member.Value)
			if err != nil {
				return VoxelSet{}, err
			}
			cellSize, ok := parsed.(VectorN)
			if !ok || int64(len(cellSize)) != voxel.Dimensions && voxel.Dimensions != 0 {
				return VoxelSet{}, newError("invalid #vox payload")
			}
			voxel.CellSize = cellSize
		case "cells":
			cells, ok := member.Value.([]any)
			if !ok {
				return VoxelSet{}, newError("invalid #vox payload")
			}
			voxel.Cells = make([]VoxelCell, len(cells))
			for i, cell := range cells {
				pair, ok := cell.([]any)
				if !ok || len(pair) != 2 {
					return VoxelSet{}, newError("invalid #vox payload")
				}
				coordinatesValue, ok := pair[0].([]any)
				if !ok {
					return VoxelSet{}, newError("invalid #vox payload")
				}
				coordinates, err := parseIntVector(coordinatesValue)
				if err != nil || int64(len(coordinates)) != voxel.Dimensions && voxel.Dimensions != 0 {
					return VoxelSet{}, newError("invalid #vox payload")
				}
				parsed, err := opts.parseVocabularyValue(pair[1])
				if err != nil {
					return VoxelSet{}, err
				}
				voxel.Cells[i] = VoxelCell{Coordinate: coordinates, Value: parsed}
			}
		}
	}
	if !seen["dimensions"] || !seen["origin"] || !seen["cellSize"] || !seen["cells"] {
		return VoxelSet{}, newError("invalid #vox payload")
	}
	if int64(len(voxel.Origin)) != voxel.Dimensions || int64(len(voxel.CellSize)) != voxel.Dimensions {
		return VoxelSet{}, newError("invalid #vox payload")
	}
	return voxel, nil
}

func renderVoxelSet(value VoxelSet) orderedObject {
	var object orderedObject
	object.Set("dimensions", value.Dimensions)
	object.Set("origin", VectorN(value.Origin))
	object.Set("cellSize", VectorN(value.CellSize))
	cells := make([]any, len(value.Cells))
	for i, cell := range value.Cells {
		cells[i] = []any{intSliceToAny(cell.Coordinate), cell.Value}
	}
	if len(cells) > 0 {
		object.Set("cells", multilineArray(cells))
	} else {
		object.Set("cells", cells)
	}
	return object
}

func vector3FromSlice(values []float64) Vector3 {
	return Vector3{X: values[0], Y: values[1], Z: values[2]}
}

func anyVector3(value Vector3) []any {
	return []any{value.X, value.Y, value.Z}
}
