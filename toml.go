package main

import (
	"math"
	"reflect"
)

// cmpToml consumes the recursive structure of both `expected` and `test`
// simultaneously. If anything is unequal, the result has failed and
// comparison stops.
//
// N.B. `reflect.DeepEqual` could work here, but it won't tell us how the
// two structures are different. (Although we do use it here on primitive
// values.)
func (r result) cmpTOML(want, have interface{}) result {
	if isTomlValue(want) {
		if !isTomlValue(have) {
			return r.failedf("Type for key '%s' differs:\n"+
				"  Expected:     %[2]v (%[2]T)\n"+
				"  Your encoder: %[3]v (%[3]T)",
				r.key, want, have)
		}

		if !deepEqual(want, have) {
			return r.failedf("Values for key '%s' differ:\n"+
				"  Expected:     %[2]v (%[2]T)\n"+
				"  Your encoder: %[3]v (%[3]T)",
				r.key, want, have)
		}
		return r
	}

	switch w := want.(type) {
	case map[string]interface{}:
		return r.cmpTOMLMap(w, have)
	case []interface{}:
		return r.cmpTOMLArrays(w, have)
	default:
		return r.failedf("Unrecognized TOML structure: %T", want)
	}
}

func (r result) cmpTOMLMap(want map[string]interface{}, have interface{}) result {
	haveMap, ok := have.(map[string]interface{})
	if !ok {
		return r.mismatch("table", want, haveMap)
	}

	// Check that the keys of each map are equivalent.
	for k := range want {
		if _, ok := haveMap[k]; !ok {
			bunk := r.kjoin(k)
			return bunk.failedf("Could not find key '%s' in encoder output", bunk.key)
		}
	}
	for k := range haveMap {
		if _, ok := want[k]; !ok {
			bunk := r.kjoin(k)
			return bunk.failedf("Could not find key '%s' in expected output", bunk.key)
		}
	}

	// Okay, now make sure that each value is equivalent.
	for k := range want {
		if sub := r.kjoin(k).cmpTOML(want[k], haveMap[k]); sub.failed() {
			return sub
		}
	}
	return r
}

func (r result) cmpTOMLArrays(want []interface{}, have interface{}) result {
	// Slice can be decoded to []interface{} for an array of primitives, or
	// []map[string]interface{} for an array of tables.
	//
	// TODO: it would be nicer if it could always decode to []interface{}?
	haveSlice, ok := have.([]interface{})
	if !ok {
		tblArray, ok := have.([]map[string]interface{})
		if !ok {
			return r.mismatch("array", want, have)
		}

		haveSlice = make([]interface{}, len(tblArray))
		for i := range tblArray {
			haveSlice[i] = tblArray[i]
		}
	}

	if len(want) != len(haveSlice) {
		return r.failedf("Array lengths differ for key '%s'"+
			"  Expected:     %[2]v (len=%[4]d)\n"+
			"  Your encoder: %[3]v (len=%[5]d)",
			r.key, want, haveSlice, len(want), len(haveSlice))
	}
	for i := 0; i < len(want); i++ {
		if sub := r.cmpTOML(want[i], haveSlice[i]); sub.failed() {
			return sub
		}
	}
	return r
}

// reflect.DeepEqual() that deals with NaN != NaN
func deepEqual(want, have interface{}) bool {
	var wantF, haveF float64
	switch f := want.(type) {
	case float32:
		wantF = float64(f)
	case float64:
		wantF = f
	}
	switch f := have.(type) {
	case float32:
		haveF = float64(f)
	case float64:
		haveF = f
	}
	if math.IsNaN(wantF) && math.IsNaN(haveF) {
		return true
	}

	return reflect.DeepEqual(want, have)
}

func isTomlValue(v interface{}) bool {
	switch v.(type) {
	case map[string]interface{}, []interface{}:
		return false
	}
	return true
}
