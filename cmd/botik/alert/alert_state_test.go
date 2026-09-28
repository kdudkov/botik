package alert

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStateUpdates(t *testing.T) {
	for _, active := range []bool{false, true} {
		t.Run(map[bool]string{false: "resolved", true: "firing"}[active], func(t *testing.T) {
			a := testAlert(active)
			st := &AlertState{id: a.Key()}
			assert.True(t, st.Update(a))
			assert.Equal(t, uint64(1), st.DTO().Revision)
			assert.Equal(t, active, st.DTO().Active)
			assert.False(t, st.Update(testAlert(active)))
			a.Annotations["description"] = "updated"
			a.EndsAt = a.EndsAt.Add(time.Second)
			assert.False(t, st.Update(a))
			assert.Equal(t, "updated", st.DTO().Annotations["description"])
			next := *a
			next.StartsAt = next.StartsAt.Add(time.Minute)
			assert.True(t, st.Update(&next))
			assert.Equal(t, uint64(2), st.DTO().Revision)
			assert.True(t, st.Update(testAlert(!active)))
			assert.Equal(t, uint64(3), st.DTO().Revision)
			assert.False(t, st.Update(nil))
		})
	}
}
