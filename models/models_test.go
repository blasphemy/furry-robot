package models

import "testing"

func TestAddSlash1(t *testing.T) {
	f := File{}
	f.Id = "test"
	test := f.GetUrl("test")
	expected := "test/test"
	if test != expected {
		t.Fail()
	}
}

func TestNotAddSlash(t *testing.T) {
	f := File{}
	f.Id = "test"
	test := f.GetUrl("test/")
	expected := "test/test"
	if test != expected {
		t.Fail()
	}
}
