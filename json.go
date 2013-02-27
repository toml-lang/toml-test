package main

import (
	"strconv"
)

// compareJson consumes the recursive structure of both `expected` and `test`
// simultaneously. If anything is unequal, the result has failed and
// comparison stops.
//
// N.B. `reflect.DeepEqual` could work here, but it won't tell us how the
// two structures are different.
func (r result) cmpJson(expected, test interface{}) result {
	e, ok := expected.(map[string]interface{})
	if !ok {
		return r.failedf("Key '%s' in expected output should be a map, but "+
			"it's a %T.", r.key, expected)
	}
	return r.cmpMaps(e, test)
}

func (r result) cmpMaps(
	e map[string]interface{}, test interface{}) result {

	t, ok := test.(map[string]interface{})
	if !ok {
		return r.mismatch("hash map", t)
	}

	// Check to make sure both or neither are values.
	if isValue(e) && !isValue(t) {
		return r.failedf("Key '%s' is supposed to be a value, but the "+
			"parser reports it as a hash.", r.key)
	}
	if !isValue(e) && isValue(t) {
		return r.failedf("Key '%s' is supposed to be a hash, but the "+
			"parser reports it as a value.", r.key)
	}
	if isValue(e) && isValue(t) {
		return r.cmpValues(e, t)
	}

	// Check that the keys of each map are equivalent.
	for k, _ := range e {
		if _, ok := t[k]; !ok {
			bunk := r.kjoin(k)
			return bunk.failedf("Could not find key '%s' in parser output.",
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
		if sub := r.kjoin(k).cmpJson(e[k], t[k]); sub.failed() {
			return sub
		}
	}
	return r
}

func (r result) cmpArray(e, t interface{}) result {
	ea, ok := e.([]interface{})
	if !ok {
		return r.failedf("BUG in test case. 'value' should be a JSON array "+
			"when 'type' indicates 'array', but it is a %T.", e)
	}

	ta, ok := t.([]interface{})
	if !ok {
		return r.failedf("Malformed parser output. 'value' should be a "+
			"JSON array when 'type' indicates 'array', but it is a %T.", t)
	}

	if len(ea) != len(ta) {
		return r.failedf("Array lengths differ for key '%s'. Expected a "+
			"length of %d but got %d.", r.key, len(ea), len(ta))
	}

	for i := 0; i < len(ea); i++ {
		if sub := r.cmpJson(ea[i], ta[i]); sub.failed() {
			return sub
		}
	}

	return r
}

func (r result) cmpValues(e, t map[string]interface{}) result {
	etype, ok := e["type"].(string)
	if !ok {
		return r.failedf("BUG in test case. 'type' should be a string, "+
			"but it is a %T.", e["type"])
	}

	ttype, ok := t["type"].(string)
	if !ok {
		return r.failedf("Malformed parser output. 'type' should be a string, "+
			"but it is a %T.", t["type"])
	}

	if etype != ttype {
		return r.valMismatch(etype, ttype)
	}

	// If this is an array, then we've got to do some work to check
	// equality.
	if etype == "array" {
		return r.cmpArray(e["value"], t["value"])
	}

	// Floats need special attention too. Not every language can
	// represent the same floats, and sometimes the string version of
	// a float can be wonky with extra zeroes and what not.
	if etype == "float" {
		return r.cmpFloat(e["value"].(string), t["value"].(string))
	}

	// Otherwise, we can do simple string equality.
	if e["value"] != t["value"] {
		return r.failedf("Values for key '%s' don't match. Expected a "+
			"value of '%s' but got '%s'.", r.key, e["value"], t["value"])
	}

	return r
}

func (r result) cmpFloat(e, t string) result {
	ef, err := strconv.ParseFloat(e, 64)
	if err != nil {
		return r.failedf("BUG in test case. Could not read '%s' as a float "+
			"value for key '%s'.", e, r.key)
	}

	tf, err := strconv.ParseFloat(t, 64)
	if err != nil {
		return r.failedf("Malformed parser output. Could not read '%s' as "+
			"a float value for key '%s'.", t, r.key)
	}

	if ef != tf {
		return r.failedf("Values for key '%s' don't match. Expected a "+
			"value of '%v' but got '%v'.", r.key, ef, tf)
	}
	return r
}

func isValue(m map[string]interface{}) bool {
	if len(m) != 2 {
		return false
	}
	if _, ok := m["type"]; !ok {
		return false
	}
	if _, ok := m["value"]; !ok {
		return false
	}
	return true
}
