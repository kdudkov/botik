package answer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWords(t *testing.T) {

	var q = Q{}
	var w []string

	q.Msg = " message,  number one!   5,25\n"
	w = q.Words()

	assert.Equal(t, "message", w[0])
	assert.Equal(t, "number", w[1])
	assert.Equal(t, "one", w[2])
	assert.Equal(t, "5,25", w[3])
}
