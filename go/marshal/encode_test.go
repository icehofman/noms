// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package marshal

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"testing"

	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/testify/assert"
)

func TestEncode(tt *testing.T) {
	t := func(exp types.Value, v interface{}) {
		actual, err := Marshal(v)
		assert.NoError(tt, err)
		assert.True(tt, exp.Equals(actual))

		// Encode again for fallthrough
		actual2, err := Marshal(actual)
		assert.NoError(tt, err)
		assert.True(tt, exp.Equals(actual2))
	}

	for _, n := range []float32{0, 42, 3.14159265359, math.MaxFloat32} {
		t(types.Number(n), n)
		t(types.Number(-n), -n)
	}

	for _, n := range []float64{0, 42, 3.14159265359, 9007199254740991, math.MaxFloat64} {
		t(types.Number(n), n)
		t(types.Number(-n), -n)
	}

	for _, n := range []int8{0, 42, math.MaxInt8} {
		t(types.Number(n), n)
		t(types.Number(-n), -n)
	}

	for _, n := range []int16{0, 42, math.MaxInt16} {
		t(types.Number(n), n)
		t(types.Number(-n), -n)
	}

	for _, n := range []int32{0, 42, math.MaxInt32} {
		t(types.Number(n), n)
		t(types.Number(-n), -n)
	}

	// int is at least int32
	for _, n := range []int{0, 42, math.MaxInt32} {
		t(types.Number(n), n)
		t(types.Number(-n), -n)
	}

	for _, n := range []int64{0, 42, math.MaxInt64} {
		t(types.Number(n), n)
		t(types.Number(-n), -n)
	}

	for _, n := range []uint8{0, 42, math.MaxUint8} {
		t(types.Number(n), n)
	}

	for _, n := range []uint16{0, 42, math.MaxUint16} {
		t(types.Number(n), n)
	}

	for _, n := range []uint32{0, 42, math.MaxUint32} {
		t(types.Number(n), n)
	}

	// uint is at least uint32
	for _, n := range []uint{0, 42, math.MaxUint32} {
		t(types.Number(n), n)
	}

	for _, n := range []uint64{0, 42, math.MaxUint64} {
		t(types.Number(n), n)
	}

	t(types.Bool(true), true)
	t(types.Bool(false), false)

	for _, s := range []string{"", "s", "hello", "💩"} {
		t(types.String(s), s)
	}

	t(types.NewList(types.Number(42)), types.NewList(types.Number(42)))
	t(types.NewMap(types.Number(42), types.String("hi")), types.NewMap(types.Number(42), types.String("hi")))
	t(types.NewSet(types.String("bye")), types.NewSet(types.String("bye")))
	t(types.NewBlob(bytes.NewBufferString("hello")), types.NewBlob(bytes.NewBufferString("hello")))

	type TestStruct struct {
		Str string
		Num float64
	}
	t(types.NewStruct("TestStruct", types.StructData{
		"num": types.Number(42),
		"str": types.String("Hello"),
	}), TestStruct{Str: "Hello", Num: 42})
	// Same again to test caching
	t(types.NewStruct("TestStruct", types.StructData{
		"num": types.Number(1),
		"str": types.String("Bye"),
	}), TestStruct{Str: "Bye", Num: 1})

	anonStruct := struct {
		B bool
	}{
		true,
	}
	t(types.NewStruct("", types.StructData{
		"b": types.Bool(true),
	}), anonStruct)

	type TestNestedStruct struct {
		A types.List
		B TestStruct
		C float64
	}
	t(types.NewStruct("TestNestedStruct", types.StructData{
		"a": types.NewList(types.String("hi")),
		"b": types.NewStruct("TestStruct", types.StructData{
			"str": types.String("bye"),
			"num": types.Number(5678),
		}),
		"c": types.Number(1234),
	}), TestNestedStruct{
		A: types.NewList(types.String("hi")),
		B: TestStruct{
			Str: "bye",
			Num: 5678,
		},
		C: 1234,
	})

	type testStruct struct {
		Str string
		Num float64
	}
	t(types.NewStruct("TestStruct", types.StructData{
		"num": types.Number(42),
		"str": types.String("Hello"),
	}), testStruct{Str: "Hello", Num: 42})
}

func assertEncodeErrorMessage(t *testing.T, v interface{}, expectedMessage string) {
	_, err := Marshal(v)
	assert.Error(t, err)
	assert.Equal(t, expectedMessage, err.Error())
}

func TestInvalidTypes(t *testing.T) {
	assertEncodeErrorMessage(t, make(chan int), "Type is not supported, type: chan int")
	x := 42
	assertEncodeErrorMessage(t, &x, "Type is not supported, type: *int")
}

func TestEncodeEmbeddedStruct(t *testing.T) {
	type EmbeddedStruct struct{}
	type TestStruct struct {
		EmbeddedStruct
	}
	assertEncodeErrorMessage(t, TestStruct{EmbeddedStruct{}}, "Embedded structs are not supported, type: marshal.TestStruct")
}

func TestEncodeNonExportedField(t *testing.T) {
	type TestStruct struct {
		x int
	}
	assertEncodeErrorMessage(t, TestStruct{1}, "Non exported fields are not supported, type: marshal.TestStruct")
}

func TestEncodeTaggingSkip(t *testing.T) {
	assert := assert.New(t)

	type S struct {
		Abc int `noms:"-"`
		Def bool
	}
	s := S{42, true}
	v, err := Marshal(s)
	assert.NoError(err)
	assert.True(types.NewStruct("S", types.StructData{
		"def": types.Bool(true),
	}).Equals(v))
}

func TestEncodeNamedFields(t *testing.T) {
	assert := assert.New(t)

	type S struct {
		Aaa int  `noms:"a"`
		Bbb bool `noms:"B"`
		Ccc string
	}
	s := S{42, true, "Hi"}
	v, err := Marshal(s)
	assert.NoError(err)
	assert.True(types.NewStruct("S", types.StructData{
		"a":   types.Number(42),
		"B":   types.Bool(true),
		"ccc": types.String("Hi"),
	}).Equals(v))
}

func TestEncodeInvalidNamedFields(t *testing.T) {
	type S struct {
		A int `noms:"1a"`
	}
	assertEncodeErrorMessage(t, S{42}, "Invalid struct field name: 1a")
}

func TestEncodeOmitEmpty(t *testing.T) {
	assert := assert.New(t)

	type S struct {
		String  string  `noms:",omitempty"`
		Bool    bool    `noms:",omitempty"`
		Int     int     `noms:",omitempty"`
		Int8    int8    `noms:",omitempty"`
		Int16   int16   `noms:",omitempty"`
		Int32   int32   `noms:",omitempty"`
		Int64   int64   `noms:",omitempty"`
		Uint    uint    `noms:",omitempty"`
		Uint8   uint8   `noms:",omitempty"`
		Uint16  uint16  `noms:",omitempty"`
		Uint32  uint32  `noms:",omitempty"`
		Uint64  uint64  `noms:",omitempty"`
		Float32 float32 `noms:",omitempty"`
		Float64 float64 `noms:",omitempty"`
	}
	s := S{
		String:  "s",
		Bool:    true,
		Int:     1,
		Int8:    1,
		Int16:   1,
		Int32:   1,
		Int64:   1,
		Uint:    1,
		Uint8:   1,
		Uint16:  1,
		Uint32:  1,
		Uint64:  1,
		Float32: 1,
		Float64: 1,
	}
	v, err := Marshal(s)
	assert.NoError(err)
	assert.True(types.NewStruct("S", types.StructData{
		"string":  types.String("s"),
		"bool":    types.Bool(true),
		"int":     types.Number(1),
		"int8":    types.Number(1),
		"int16":   types.Number(1),
		"int32":   types.Number(1),
		"int64":   types.Number(1),
		"uint":    types.Number(1),
		"uint8":   types.Number(1),
		"uint16":  types.Number(1),
		"uint32":  types.Number(1),
		"uint64":  types.Number(1),
		"float32": types.Number(1),
		"float64": types.Number(1),
	}).Equals(v))

	s2 := S{
		String:  "",
		Bool:    false,
		Int:     0,
		Int8:    0,
		Int16:   0,
		Int32:   0,
		Int64:   0,
		Uint:    0,
		Uint8:   0,
		Uint16:  0,
		Uint32:  0,
		Uint64:  0,
		Float32: 0,
		Float64: 0,
	}
	v2, err := Marshal(s2)
	assert.NoError(err)
	assert.True(types.NewStruct("S", types.StructData{}).Equals(v2))

	type S2 struct {
		Slice []int       `noms:",omitempty"`
		Map   map[int]int `noms:",omitempty"`
	}

	s3 := S2{
		Slice: []int{0},
		Map:   map[int]int{0: 0},
	}
	v3, err := Marshal(s3)
	assert.NoError(err)
	assert.True(types.NewStruct("S2", types.StructData{
		"slice": types.NewList(types.Number(0)),
		"map":   types.NewMap(types.Number(0), types.Number(0)),
	}).Equals(v3))

	s4 := S2{
		Slice: []int{},
		Map:   map[int]int{},
	}
	v4, err := Marshal(s4)
	assert.NoError(err)
	assert.True(types.NewStruct("S2", types.StructData{}).Equals(v4))

	s5 := S2{
		Slice: nil,
		Map:   nil,
	}
	v5, err := Marshal(s5)
	assert.NoError(err)
	assert.True(types.NewStruct("S2", types.StructData{}).Equals(v5))

	type S3 struct {
		List  types.List  `noms:",omitempty"`
		Value types.Value `noms:",omitempty"`
	}
	s6 := S3{
		List:  types.NewList(),
		Value: types.Number(0),
	}
	v6, err := Marshal(s6)
	assert.NoError(err)
	assert.True(types.NewStruct("S3", types.StructData{
		"list":  types.NewList(),
		"value": types.Number(0),
	}).Equals(v6))

	s7 := S3{
		List:  types.List{},
		Value: nil,
	}
	v7, err := Marshal(s7)
	assert.NoError(err)
	assert.True(types.NewStruct("S3", types.StructData{}).Equals(v7))

	// Both name and omitempty
	type S4 struct {
		X int `noms:"y,omitempty"`
	}
	s8 := S4{
		X: 1,
	}
	v8, err := Marshal(s8)
	assert.NoError(err)
	assert.True(types.NewStruct("S4", types.StructData{
		"y": types.Number(1),
	}).Equals(v8))

	s9 := S4{
		X: 0,
	}
	v9, err := Marshal(s9)
	assert.NoError(err)
	assert.True(types.NewStruct("S4", types.StructData{}).Equals(v9))
}

func ExampleMarshal() {
	type Person struct {
		Given string
		Male  bool
	}
	arya, err := Marshal(Person{"Arya", false})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Given: %s, Male: %t\n", arya.(types.Struct).Get("given").(types.String), arya.(types.Struct).Get("male").(types.Bool))
	// Output: Given: Arya, Male: false
}

func TestEncodeSlice(t *testing.T) {
	assert := assert.New(t)
	v, err := Marshal([]string{"a", "b", "c"})
	assert.NoError(err)
	assert.True(types.NewList(types.String("a"), types.String("b"), types.String("c")).Equals(v))
}

func TestEncodeArray(t *testing.T) {
	assert := assert.New(t)
	v, err := Marshal([3]int{1, 2, 3})
	assert.NoError(err)
	assert.True(types.NewList(types.Number(1), types.Number(2), types.Number(3)).Equals(v))
}

func TestEncodeStructWithSlice(t *testing.T) {
	assert := assert.New(t)

	type S struct {
		List []int
	}
	v, err := Marshal(S{[]int{1, 2, 3}})
	assert.NoError(err)
	assert.True(types.NewStruct("S", types.StructData{
		"list": types.NewList(types.Number(1), types.Number(2), types.Number(3)),
	}).Equals(v))
}

func TestEncodeStructWithArrayOfNomsValue(t *testing.T) {
	assert := assert.New(t)

	type S struct {
		List [1]types.Set
	}
	v, err := Marshal(S{[1]types.Set{types.NewSet(types.Bool(true))}})
	assert.NoError(err)
	assert.True(types.NewStruct("S", types.StructData{
		"list": types.NewList(types.NewSet(types.Bool(true))),
	}).Equals(v))
}

func TestEncodeNomsTypePtr(t *testing.T) {
	assert := assert.New(t)

	testMarshal := func(g interface{}, expected types.Value) {
		v, err := Marshal(g)
		assert.NoError(err)
		assert.Equal(expected, v)
	}

	type S struct {
		Type *types.Type
	}

	primitive := types.StringType
	testMarshal(S{primitive}, types.NewStruct("S", types.StructData{"type": primitive}))

	complex := types.MakeStructType("Complex",
		types.StructField{
			Name: "stuff",
			Type: types.StringType,
		},
	)
	testMarshal(S{complex}, types.NewStruct("S", types.StructData{"type": complex}))

	var empty *types.Type
	testMarshal(S{empty}, types.NewStruct("S", types.StructData{"type": empty}))
}

func TestEncodeRecursive(t *testing.T) {
	assert := assert.New(t)

	type Node struct {
		Value    int
		Children []Node
	}
	v, err := Marshal(Node{
		1, []Node{
			{2, []Node{}},
			{3, []Node(nil)},
		},
	})
	assert.NoError(err)

	typ := types.MakeStructType("Node",
		types.StructField{
			Name: "children",
			Type: types.MakeListType(types.MakeCycleType("Node")),
		},
		types.StructField{
			Name: "value",
			Type: types.NumberType,
		},
	)
	assert.True(typ.Equals(types.TypeOf(v)))

	assert.True(types.NewStruct("Node", types.StructData{
		"children": types.NewList(
			types.NewStruct("Node", types.StructData{
				"children": types.NewList(),
				"value":    types.Number(2),
			}),
			types.NewStruct("Node", types.StructData{
				"children": types.NewList(),
				"value":    types.Number(3),
			}),
		),
		"value": types.Number(1),
	}).Equals(v))
}

func TestEncodeMap(t *testing.T) {
	assert := assert.New(t)

	v, err := Marshal(map[string]int{"a": 1, "b": 2, "c": 3})
	assert.NoError(err)
	assert.True(types.NewMap(
		types.String("a"), types.Number(1),
		types.String("b"), types.Number(2),
		types.String("c"), types.Number(3)).Equals(v))

	type S struct {
		N string
	}
	v, err = Marshal(map[S]bool{S{"Yes"}: true, S{"No"}: false})
	assert.NoError(err)
	assert.True(types.NewMap(
		types.NewStruct("S", types.StructData{"n": types.String("Yes")}), types.Bool(true),
		types.NewStruct("S", types.StructData{"n": types.String("No")}), types.Bool(false)).Equals(v))

	v, err = Marshal(map[string]int(nil))
	assert.NoError(err)
	assert.True(types.NewMap().Equals(v))

	v, err = Marshal(map[string]int{})
	assert.NoError(err)
	assert.True(types.NewMap().Equals(v))
}

func TestEncodeInterface(t *testing.T) {
	assert := assert.New(t)

	var i interface{}
	i = []string{"a", "b"}
	v, err := Marshal(i)
	assert.NoError(err)
	assert.True(types.NewList(types.String("a"), types.String("b")).Equals(v))

	i = map[interface{}]interface{}{"a": true, struct{ Name string }{"b"}: 42}
	v, err = Marshal(i)
	assert.NoError(err)
	assert.True(types.NewMap(
		types.String("a"), types.Bool(true),
		types.NewStruct("", types.StructData{"name": types.String("b")}), types.Number(42),
	).Equals(v))
}

func TestEncodeSet(t *testing.T) {
	assert := assert.New(t)

	v, err := Marshal(struct {
		A map[int]struct{} `noms:",set"`
		B map[int]struct{}
		C map[int]string      `noms:",set"`
		D map[string]struct{} `noms:",set"`
		E map[string]struct{}
		F map[string]int `noms:",set"`
		G []int          `noms:",set"`
		H string         `noms:",set"`
	}{
		map[int]struct{}{0: {}, 1: {}, 2: {}},
		map[int]struct{}{3: {}, 4: {}, 5: {}},
		map[int]string{},
		map[string]struct{}{"A": {}, "B": {}, "C": {}},
		map[string]struct{}{"D": {}, "E": {}, "F": {}},
		map[string]int{},
		[]int{1, 2, 3},
		"",
	})
	assert.NoError(err)
	s, ok := v.(types.Struct)
	assert.True(ok)

	expect := map[string]types.NomsKind{
		"a": types.SetKind,
		"b": types.MapKind,
		"c": types.MapKind,
		"d": types.SetKind,
		"e": types.MapKind,
		"f": types.MapKind,
		"g": types.SetKind,
		"h": types.StringKind,
	}
	for fieldName, kind := range expect {
		assert.Equal(kind, s.Get(fieldName).Kind())
	}

	// Test both the Set values are correct, and that the equivalent typed Map
	// are correct in case the Set marshaling interferes with it.

	a := s.Get("a").(types.Set)
	assert.True(a.Has(types.Number(0)))
	assert.True(a.Has(types.Number(1)))
	assert.True(a.Has(types.Number(2)))

	b := s.Get("b").(types.Map)
	assert.True(b.Has(types.Number(3)))
	assert.True(b.Has(types.Number(4)))
	assert.True(b.Has(types.Number(5)))

	d := s.Get("d").(types.Set)
	assert.True(d.Has(types.String("A")))
	assert.True(d.Has(types.String("B")))
	assert.True(d.Has(types.String("C")))

	e := s.Get("e").(types.Map)
	assert.True(e.Has(types.String("D")))
	assert.True(e.Has(types.String("E")))
	assert.True(e.Has(types.String("F")))

	g := s.Get("g").(types.Set)
	assert.True(g.Has(types.Number(1)))
	assert.True(g.Has(types.Number(2)))
	assert.True(g.Has(types.Number(3)))
}

func TestEncodeSetWithTags(t *testing.T) {
	assert := assert.New(t)

	v, err := Marshal(struct {
		A map[int]struct{} `noms:"foo,set"`
		B map[int]struct{} `noms:",omitempty,set"`
		C map[int]struct{} `noms:"bar,omitempty,set"`
	}{
		A: map[int]struct{}{0: {}, 1: {}},
		C: map[int]struct{}{2: {}, 3: {}},
	})
	assert.NoError(err)
	s, ok := v.(types.Struct)
	assert.True(ok)

	_, ok = s.MaybeGet("a")
	assert.False(ok)
	_, ok = s.MaybeGet("b")
	assert.False(ok)
	_, ok = s.MaybeGet("c")
	assert.False(ok)

	foo, ok := s.Get("foo").(types.Set)
	assert.True(ok)
	assert.True(types.NewSet(types.Number(0), types.Number(1)).Equals(foo))

	bar, ok := s.Get("bar").(types.Set)
	assert.True(ok)
	assert.True(types.NewSet(types.Number(2), types.Number(3)).Equals(bar))
}

func TestInvalidTag(t *testing.T) {
	_, err := Marshal(struct {
		F string `noms:",omitEmpty"`
	}{"F"})
	assert.Error(t, err)
	assert.Equal(t, `Unrecognized tag: omitEmpty`, err.Error())
}

func TestEncodeCanSkipUnexportedField(t *testing.T) {
	assert := assert.New(t)

	type S struct {
		Abc         int
		notExported bool `noms:"-"`
	}
	s := S{42, true}
	v, err := Marshal(s)
	assert.NoError(err)
	assert.True(types.NewStruct("S", types.StructData{
		"abc": types.Number(42),
	}).Equals(v))
}

func TestEncodeOriginal(t *testing.T) {
	assert := assert.New(t)

	type S struct {
		Foo int          `noms:",omitempty"`
		Bar types.Struct `noms:",original"`
	}

	var s S
	var err error
	var orig types.Struct

	// New field value clobbers old field value
	orig = types.NewStruct("S", types.StructData{
		"foo": types.Number(42),
	})
	err = Unmarshal(orig, &s)
	assert.NoError(err)
	s.Foo = 43
	assert.True(MustMarshal(s).Equals(orig.Set("foo", types.Number(43))))

	// New field extends old struct
	orig = types.NewStruct("S", types.StructData{})
	err = Unmarshal(orig, &s)
	assert.NoError(err)
	s.Foo = 43
	assert.True(MustMarshal(s).Equals(orig.Set("foo", types.Number(43))))

	// Old struct name always used
	orig = types.NewStruct("Q", types.StructData{})
	err = Unmarshal(orig, &s)
	assert.NoError(err)
	s.Foo = 43
	assert.True(MustMarshal(s).Equals(orig.Set("foo", types.Number(43))))

	// Field type of base are preserved
	orig = types.NewStruct("S", types.StructData{
		"foo": types.Number(42),
	})
	err = Unmarshal(orig, &s)
	assert.NoError(err)
	s.Foo = 43
	out := MustMarshal(s)
	assert.True(out.Equals(orig.Set("foo", types.Number(43))))

	st2 := types.MakeStructTypeFromFields("S", types.FieldMap{
		"foo": types.NumberType,
	})
	assert.True(types.TypeOf(out).Equals(st2))

	// It's OK to have an empty original field
	s = S{
		Foo: 42,
	}
	assert.True(MustMarshal(s).Equals(
		types.NewStruct("S", types.StructData{"foo": types.Number(float64(42))})))
}

func TestNomsTypes(t *testing.T) {
	assert := assert.New(t)

	type S struct {
		Blob   types.Blob
		Bool   types.Bool
		Number types.Number
		String types.String
		Type   *types.Type
	}
	s := S{
		Blob:   types.NewBlob(),
		Bool:   types.Bool(true),
		Number: types.Number(42),
		String: types.String("hi"),
		Type:   types.NumberType,
	}
	assert.True(MustMarshal(s).Equals(
		types.NewStruct("S", types.StructData{
			"blob":   types.NewBlob(),
			"bool":   types.Bool(true),
			"number": types.Number(42),
			"string": types.String("hi"),
			"type":   types.NumberType,
		}),
	))
}

type primitiveType int

func (t primitiveType) MarshalNoms() (types.Value, error) {
	return types.Number(int(t) + 1), nil
}

func TestMarshalerPrimitiveType(t *testing.T) {
	assert := assert.New(t)

	u := primitiveType(42)
	v := MustMarshal(u)
	assert.Equal(types.Number(43), v)
}

type primitiveSliceType []string

func (u primitiveSliceType) MarshalNoms() (types.Value, error) {
	return types.String(strings.Join(u, ",")), nil
}

func TestMarshalerPrimitiveSliceType(t *testing.T) {
	assert := assert.New(t)

	u := primitiveSliceType([]string{"a", "b", "c"})
	v := MustMarshal(u)
	assert.Equal(types.String("a,b,c"), v)
}

type primitiveMapType map[string]string

func (u primitiveMapType) MarshalNoms() (types.Value, error) {
	var vals types.ValueSlice
	for k, v := range u {
		vals = append(vals, types.String(k+","+v))
	}
	return types.NewSet(vals...), nil
}

func TestMarshalerPrimitiveMapType(t *testing.T) {
	assert := assert.New(t)

	u := primitiveMapType(map[string]string{
		"a": "foo",
		"b": "bar",
	})
	v := MustMarshal(u)
	assert.True(types.NewSet(types.String("a,foo"), types.String("b,bar")).Equals(v))
}

type primitiveStructType struct {
	x, y int
}

func (u primitiveStructType) MarshalNoms() (types.Value, error) {
	return types.Number(u.x + u.y), nil
}

func TestMarshalerPrimitiveStructType(t *testing.T) {
	assert := assert.New(t)

	u := primitiveStructType{1, 2}
	v := MustMarshal(u)
	assert.Equal(types.Number(3), v)
}

type builtinType regexp.Regexp

func (u builtinType) MarshalNoms() (types.Value, error) {
	r := regexp.Regexp(u)
	return types.String(r.String()), nil
}

func TestMarshalerBuiltinType(t *testing.T) {
	assert := assert.New(t)

	s := "[a-z]+$"
	r := regexp.MustCompile(s)
	u := builtinType(*r)
	v := MustMarshal(u)
	assert.Equal(types.String(s), v)
}

type wrappedMarshalerType primitiveType

func (u wrappedMarshalerType) MarshalNoms() (types.Value, error) {
	return types.Number(int(u) + 2), nil
}

func TestMarshalerWrapperMarshalerType(t *testing.T) {
	assert := assert.New(t)

	u := wrappedMarshalerType(primitiveType(42))
	v := MustMarshal(u)
	assert.Equal(types.Number(44), v)
}

type TestComplexStructType struct {
	P       primitiveType
	Ps      []primitiveType
	Pm      map[string]primitiveType
	Pslice  primitiveSliceType
	Pmap    primitiveMapType
	Pstruct primitiveStructType
	B       builtinType
}

func TestMarshalerComplexStructType(t *testing.T) {
	assert := assert.New(t)

	s := "foo|bar"
	r := regexp.MustCompile(s)
	u := TestComplexStructType{
		P:  42,
		Ps: []primitiveType{1, 2},
		Pm: map[string]primitiveType{
			"x": 100,
			"y": 101,
		},
		Pslice: primitiveSliceType{"a", "b", "c"},
		Pmap: primitiveMapType{
			"c": "123",
			"d": "456",
		},
		Pstruct: primitiveStructType{10, 20},
		B:       builtinType(*r),
	}

	v := MustMarshal(u)

	assert.True(types.NewStruct("TestComplexStructType", types.StructData{
		"p":       types.Number(43),
		"ps":      types.NewList(types.Number(2), types.Number(3)),
		"pm":      types.NewMap(types.String("x"), types.Number(101), types.String("y"), types.Number(102)),
		"pslice":  types.String("a,b,c"),
		"pmap":    types.NewSet(types.String("c,123"), types.String("d,456")),
		"pstruct": types.Number(30),
		"b":       types.String(s),
	}).Equals(v))
}

type returnsMarshalerError struct {
	err error
}

func (u returnsMarshalerError) MarshalNoms() (types.Value, error) {
	return nil, u.err
}

type returnsMarshalerNil struct{}

func (u returnsMarshalerNil) MarshalNoms() (types.Value, error) {
	return nil, nil
}

type panicsMarshaler struct{}

func (u panicsMarshaler) MarshalNoms() (types.Value, error) {
	panic("panic")
}

func TestMarshalerErrors(t *testing.T) {
	assert := assert.New(t)

	expErr := errors.New("expected error")
	m1 := returnsMarshalerError{expErr}
	_, actErr := Marshal(m1)
	assert.Equal(expErr, actErr)

	m2 := returnsMarshalerNil{}
	assert.Panics(func() { Marshal(m2) })

	m3 := panicsMarshaler{}
	assert.Panics(func() { Marshal(m3) })
}
