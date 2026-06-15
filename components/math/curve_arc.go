package math

type ArcCurve[T Float] struct {
	EllipseCurve[T]
}

func NewArcCurve[T Float](aX, aY, aRadius, aStartAngle, aEndAngle T, aClockwise bool) *ArcCurve[T] {
	return &ArcCurve[T]{
		EllipseCurve: *NewEllipseCurve(
			aX, aY,
			aRadius, aRadius,
			aStartAngle, aEndAngle,
			0, aClockwise,
		),
	}
}
