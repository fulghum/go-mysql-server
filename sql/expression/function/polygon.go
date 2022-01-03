// Copyright 2020-2021 Dolthub, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package function

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/dolthub/go-mysql-server/sql"
)

// Polygon is a function that returns a polygon type containing values Y and Y.
type Polygon struct {
	args []sql.Expression
}

var _ sql.FunctionExpression = (*Polygon)(nil)

// NewPolygon creates a new polygon expression.
func NewPolygon(args ...sql.Expression) (sql.Expression, error) {
	if len(args) < 1 {
		return nil, sql.ErrInvalidArgumentNumber.New("Polygon", "1 or more", len(args))
	}
	return &Polygon{args}, nil
}

// FunctionName implements sql.FunctionExpression
func (l *Polygon) FunctionName() string {
	return "polygon"
}

// Description implements sql.FunctionExpression
func (l *Polygon) Description() string {
	return "returns a new polygon."
}

// Children implements the sql.Expression interface.
func (l *Polygon) Children() []sql.Expression {
	return l.args
}

// Resolved implements the sql.Expression interface.
func (l *Polygon) Resolved() bool {
	for _, arg := range l.args {
		if !arg.Resolved() {
			return false
		}
	}
	return true
}

// IsNullable implements the sql.Expression interface.
func (l *Polygon) IsNullable() bool {
	for _, arg := range l.args {
		if arg.IsNullable() {
			return true
		}
	}
	return false
}

// Type implements the sql.Expression interface.
func (l *Polygon) Type() sql.Type {
	return sql.PolygonType{}
}

func (l *Polygon) String() string {
	var args = make([]string, len(l.args))
	for i, arg := range l.args {
		args[i] = arg.String()
	}
	return fmt.Sprintf("POLYGON(%s)", strings.Join(args, ","))
}

// WithChildren implements the Expression interface.
func (l *Polygon) WithChildren(children ...sql.Expression) (sql.Expression, error) {
	return NewPolygon(children...)
}

// TODO: https://www.geeksforgeeks.org/orientation-3-ordered-points/
func pointOrientation(p1, p2, p3 sql.Point) int {
	// compare slopes of line(p1, p2) and line(p2, p3)
	val := (p2.Y-p1.Y)*(p3.X-p2.X) - (p3.Y-p2.Y)*(p2.X-p1.X)
	// check orientation
	if val == 0 {
		return 0 // collinear or both on axis and perpendicular
	} else if val > 0 {
		return 1 // clockwise
	} else {
		return 2 // counter-clockwise
	}
}

// Check if point c is in line segment ab
func onSegment(a, b, c sql.Point) bool {
	return c.X > math.Min(a.X, b.X) && c.X < math.Max(a.X, b.X) && c.Y > math.Min(a.Y, b.Y) && c.Y < math.Max(a.Y, b.Y)
}

// TODO: https://www.geeksforgeeks.org/check-if-two-given-line-segments-intersect/
func lineSegmentsIntersect(a, b, c, d sql.Point) bool {
	abc := pointOrientation(a, b, c)
	abd := pointOrientation(a, b, d)
	cda := pointOrientation(c, d, a)
	cdb := pointOrientation(c, d, b)

	// different orientations mean they intersect
	if (abc != abd) && (cda != cdb) {
		return true
	}

	// if orientation is collinear, check if point is inside segment
	if abc == 0 && onSegment(a, b, c) {
		return true
	}
	if abd == 0 && onSegment(a, b, d) {
		return true
	}
	if cda == 0 && onSegment(c, d, a) {
		return true
	}
	if cdb == 0 && onSegment(c, d, b) {
		return true
	}

	// no intersection
	return false
}

// TODO: should go in line?
func isLinearRing(line sql.Linestring) bool {
	// Get number of points
	numPoints := len(line.Points)
	// Check length of Linestring (must be 0 or 4+) points
	if numPoints != 0 && numPoints < 4 {
		return false
	}
	// Check if it is closed (first and last point are the same)
	if line.Points[0] != line.Points[numPoints-1] {
		return false
	}
	return true // TODO: MySQL appears to not check this, and there are issues so return true for now
	// TODO: how to deal with same point?
	// TODO: easy, but slow O(n^2) solution; apparently O(nlogn) exists
	// Check each segment for intersections
	for i := 0; i < numPoints-1; i++ {
		for j := i + 1; j < numPoints; j++ {
			if lineSegmentsIntersect(line.Points[i], line.Points[i+1], line.Points[j], line.Points[j+1]) {
				return false
			}
		}
	}
	return true
}

// Eval implements the sql.Expression interface.
func (l *Polygon) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	// Allocate array of lines
	var lines = make([]sql.Linestring, len(l.args))

	// Go through each argument
	for i, arg := range l.args {
		// Evaluate argument
		val, err := arg.Eval(ctx, row)
		if err != nil {
			return nil, err
		}
		// Must be of type point, throw error otherwise
		switch v := val.(type) {
		case sql.Linestring:
			// Check that line is a linear ring
			if isLinearRing(v) {
				lines[i] = v
			} else {
				return nil, errors.New("polygon constructor encountered a non-linearring")
			}
		default:
			return nil, errors.New("polygon constructor encountered a non-linestring")
		}
	}

	return sql.Polygon{Lines: lines}, nil
}