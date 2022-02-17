package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStateBuilder(t *testing.T) {
	s := New("", nil)
	s.Set(0, "1", []byte("val1"))
	s.Set(1, "1", []byte("val2"))
	s.Set(3, "1", []byte("val3"))
	s.Flush()
	s.Set(0, "1", []byte("val4"))
	s.Set(1, "1", []byte("val5"))
	s.Set(3, "1", []byte("val6"))
	s.Del(4, "1")
	s.Set(5, "1", []byte("val7"))

	val, found := s.GetFirst("1")
	assert.Equal(t, string("val3"), string(val))
	assert.True(t, found)

	val, found = s.GetAt(0, "1")
	assert.Equal(t, string("val4"), string(val))
	assert.True(t, found)

	val, found = s.GetAt(1, "1")
	assert.Equal(t, string("val5"), string(val))
	assert.True(t, found)

	val, found = s.GetAt(3, "1")
	assert.Equal(t, string("val6"), string(val))
	assert.True(t, found)

	val, found = s.GetAt(4, "1")
	assert.Nil(t, val)
	assert.False(t, found)

	val, found = s.GetAt(5, "1")
	assert.Equal(t, string("val7"), string(val))
	assert.True(t, found)

	val, found = s.GetLast("1")
	assert.Equal(t, string("val7"), string(val))
	assert.True(t, found)
}
