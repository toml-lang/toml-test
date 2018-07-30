package main

import (
	"reflect"
)

// cmpToml consumes the recursive structure of both `expected` and `test`
// simultaneously. If anything is unequal, the result has failed and
// comparison stops.
//
// N.B. `reflect.DeepEqual` could work here, but it won't tell us how the
// two structures are different. (Although we do use it here on primitive
// values.)
func (r result) cmpToml(expected, test interface{}) result {
	if isTomlValue(expected) {
		if !isTomlValue(test) {
			return r.failedf("Key '%s' in expected output is a primitive "+
				"TOML value (not table or array), but the encoder provided "+
				"a %T.", r.key, test)
		}
		if !reflect.DeepEqual(expected, test) {
			return r.failedf("Values for key '%s' differ. Expected value is "+
				"%v (%T), but your encoder produced %v (%T).",
				r.key, expected, expected, test, test)
		}
		return r
	}
	switch e := expected.(type) {
	case map[string]interface{}:
		return r.cmpTomlMaps(e, test)
	case []interface{}:
		return r.cmpTomlArrays(e, test)
	default:
		return r.failedf("Unrecognized TOML structure: %T", expected)
	}
	panic("unreachable")
}

func (r result) cmpTomlMaps(
	e map[string]interface{},
	test interface{},
) result {
	t, ok := test.(map[string]interface{})
	if !ok {
		return r.mismatch("table", t)
	}

	// Check that the keys of each map are equivalent.
	for k, _ := range e {
		if _, ok := t[k]; !ok {
			bunk := r.kjoin(k)
			return bunk.failedf("Could not find key '%s' in encoder output.",
				bunk.key)
		}
	}
	for k, _ := range t {
		if _, ok := e[k]; !ok {
			bunk := r.kjoin(k)
			return bunk.failedf("Could not find key '%s' in expected output.",
				bunk.key)
		}
	}

	// Okay, now make sure that each value is equivalent.
	for k, _ := range e {
		if sub := r.kjoin(k).cmpToml(e[k], t[k]); sub.failed() {
			return sub
		}
	}
	return r
}

func (r result) cmpTomlArrays(ea []interface{}, t interface{}) result {
	ta, ok := t.([]interface{})
	if !ok {
		return r.mismatch("array", t)
	}
	if len(ea) != len(ta) {
		return r.failedf("Array lengths differ for key '%s'. Expected a "+
			"length of %d but got %d.", r.key, len(ea), len(ta))
	}
	for i := 0; i < len(ea); i++ {
		if sub := r.cmpToml(ea[i], ta[i]); sub.failed() {
			return sub
		}
	}
	return r
}

func isTomlValue(v interface{}) bool {
	switch v.(type) {
	case map[string]interface{}, []interface{}:
		return false
	}
	return true
}
