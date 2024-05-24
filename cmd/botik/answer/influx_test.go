package answer

import (
	"botik/internal/api"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockInflux struct {
	result string
}

func (i *MockInflux) Send(db, q string) error {
	i.result = q
	return nil
}

func (i *MockInflux) GetData(db, q string) (api.InfluxAnswer, error) {
	return api.InfluxAnswer{}, nil
}

func (i *MockInflux) GetSingleSeries(db string, q string) ([]map[string]interface{}, error) {
	return nil, nil
}

var (
	mock = &MockInflux{}
	l    = &Influx{api: mock}
)

func TestInfluxSendWeight(t *testing.T) {
	var q *Q
	var words []string

	q = l.Check("user", "Вес 90.1")
	words = q.Words()

	assert.True(t, q.Matched)
	assert.Equal(t, WEIGHT, q.Cmd)
	assert.Equal(t, "90.1", words[1])

	l.Process(q)
	fmt.Println(mock.result)

	if !strings.HasPrefix(mock.result, "weight,name=user weight=90.100000 ") {
		t.Errorf("bad send %s", mock.result)
	}

	// second

	mock.result = ""

	q = l.Check("user", "Вес 90,2")
	words = q.Words()

	assert.True(t, q.Matched)
	assert.Equal(t, WEIGHT, q.Cmd)
	assert.Equal(t, "90,2", words[1])

	l.Process(q)
	fmt.Println(mock.result)

	if !strings.HasPrefix(mock.result, "weight,name=user weight=90.200000 ") {
		t.Errorf("bad send %s", mock.result)
	}
}

func TestInfluxSendBPt(t *testing.T) {
	var q *Q

	q = l.Check("user", "Давление 120 70")

	assert.True(t, q.Matched)
	assert.Equal(t, BP, q.Cmd)

	l.Process(q)
	fmt.Println(mock.result)

	if !strings.HasPrefix(mock.result, "pressure,name=user sys=120,dia=70 ") {
		t.Errorf("bad send %s", mock.result)
	}
}

func TestInfluxSendBPtNote(t *testing.T) {
	var q *Q

	q = l.Check("user", "Давление 120 70 и всё круто")

	assert.True(t, q.Matched)
	assert.Equal(t, BP, q.Cmd)

	l.Process(q)
	fmt.Println(mock.result)

	if !strings.HasPrefix(mock.result, "pressure,name=user sys=120,dia=70,note=\"и всё круто\" ") {
		t.Errorf("bad send %s", mock.result)
	}
}
