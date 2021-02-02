package kubernetes

import "testing"

func TestStringMapsAreEqual(t *testing.T) {
	m1 := map[string]string{
		"a": "A",
		"b": "B",
	}

	m2 := map[string]string{
		"a": "A",
		"b": "B",
	}

	res := stringMapsAreEqual(m1, m2)

	if res == false {
		t.Fatalf("String maps should be equal: received %v, expected %v", res, true)
	}

	m1 = map[string]string{
		"a": "A",
		"b": "B",
	}

	m2 = map[string]string{
		"a": "A",
		"b": "B",
		"c": "C",
	}

	res = stringMapsAreEqual(m1, m2)

	if res == true {
		t.Fatalf("String maps should be equal: received %v, expected %v", res, false)
	}

	m1 = map[string]string{
		"a": "A",
		"b": "B",
		"c": "c",
	}

	m2 = map[string]string{
		"a": "A",
		"b": "B",
		"c": "C",
	}

	res = stringMapsAreEqual(m1, m2)

	if res == true {
		t.Fatalf("String maps should be equal: received %v, expected %v", res, false)
	}

}
