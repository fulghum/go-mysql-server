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

package sql

import (
	"errors"

	"github.com/dolthub/vitess/go/sqltypes"
	"github.com/dolthub/vitess/go/vt/proto/query"
)

// Represents the Point type.
// https://dev.mysql.com/doc/refman/8.0/en/gis-class-point.html
type Point struct {
	X float64
	Y float64
}

type PointType struct{}

var _ Type = PointType{}

// Compare implements Type interface.
func (t PointType) Compare(a interface{}, b interface{}) (int, error) {
	// Compare nulls
	if hasNulls, res := compareNulls(a, b); hasNulls {
		return res, nil
	}

	// Expect to receive a Point, throw error otherwise
	_a, ok := a.(Point)
	if !ok {
		return 0, errors.New("received a non-Point type") // TODO: turn this into a const error
	}
	_b, ok := b.(Point)
	if !ok {
		return 0, errors.New("received a non-Point type")
	}

	// Compare X values
	if _a.X > _b.X {
		return 1, nil
	}
	if _a.X < _b.X {
		return -1, nil
	}

	// Compare Y values
	if _a.Y > _b.Y {
		return 1, nil
	}
	if _a.Y < _b.Y {
		return -1, nil
	}

	// Points must be the same
	return 0, nil
}

// Convert implements Type interface.
func (t PointType) Convert(v interface{}) (interface{}, error) {
	// Must be a Point, fail otherwise
	if v, ok := v.(Point); ok {
		return v, nil
	}

	return nil, errors.New("can't convert to Point")
}

// Promote implements the Type interface.
func (t PointType) Promote() Type {
	return t
}

// SQL implements Type interface.
func (t PointType) SQL(v interface{}) (sqltypes.Value, error) {
	if v == nil {
		return sqltypes.NULL, nil
	}

	pv, err := t.Convert(v)
	if err != nil {
		return sqltypes.Value{}, nil
	}

	return sqltypes.MakeTrusted(sqltypes.Geometry, []byte(pv.(string))), nil
}

// String implements Type interface.
func (t PointType) String() string {
	// TODO: this is what prints on describe table
	return "POINT"
}

// Type implements Type interface.
func (t PointType) Type() query.Type {
	return sqltypes.Geometry
}

// Zero implements Type interface.
func (t PointType) Zero() interface{} {
	return nil
}