package main

import (
	"fmt"
	"strings"
	"testing"
)

type MockInflux struct {
	result string
}

func (i *MockInflux) Send(q string) error {
	i.result = q
	return nil
}

func (i *MockInflux) GetData(q string, fn convert) error {
	return nil
}

func TestInfluxSendWeight(t *testing.T) {
	mock := &MockInflux{}

	l := &Influx{api: mock}

	var q *Q
	var words []string

	q = l.Check("user", "Вес 90.1")
	words = q.words()

	if !(q.Matched && q.Cmd == WEIGHT) {
		t.Fail()
	}

	if words[1] != "90.1" {
		t.Fail()
	}

	l.Process(q)
	fmt.Println(mock.result)

	if !strings.HasPrefix(mock.result, "weight,name=user weight=90.100000 ") {
		t.Errorf("bad send %s", mock.result)
	}

	// second

	mock.result = ""

	q = l.Check("user", "Вес 90,2")
	words = q.words()

	if !(q.Matched && q.Cmd == WEIGHT) {
		t.Fail()
	}

	if words[1] != "90,2" {
		t.Fail()
	}

	l.Process(q)
	fmt.Println(mock.result)

	if !strings.HasPrefix(mock.result, "weight,name=user weight=90.200000 ") {
		t.Errorf("bad send %s", mock.result)
	}
}

func TestInfluxSendBPt(t *testing.T) {
	mock := &MockInflux{}

	l := &Influx{api: mock}

	var q *Q

	q = l.Check("user", "Давление 120 70")

	if !(q.Matched && q.Cmd == BP) {
		t.Fail()
	}

	l.Process(q)
	fmt.Println(mock.result)

	if !strings.HasPrefix(mock.result, "pressure,name=user sys=120,dia=70 ") {
		t.Errorf("bad send %s", mock.result)
	}
}

func TestInfluxSendBPtNote(t *testing.T) {
	mock := &MockInflux{}

	l := &Influx{api: mock}

	var q *Q

	q = l.Check("user", "Давление 120 70 и всё круто")

	if !(q.Matched && q.Cmd == BP) {
		t.Fail()
	}

	l.Process(q)
	fmt.Println(mock.result)

	if !strings.HasPrefix(mock.result, "pressure,name=user sys=120,dia=70,note=\"всё круто\" ") {
		t.Errorf("bad send %s", mock.result)
	}
}
