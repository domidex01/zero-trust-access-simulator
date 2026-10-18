package engine

import (
	"reflect"
	"testing"
)

func TestNew_SortsByPriorityThenID(t *testing.T) {
	tests := []struct {
		name string
		give []Policy
		want []string
	}{
		{
			name: "priority ascending",
			give: []Policy{
				{ID: "C", Priority: 30},
				{ID: "A", Priority: 10},
				{ID: "B", Priority: 20},
			},
			want: []string{"A", "B", "C"},
		},
		{
			name: "tie broken by ascending ID",
			give: []Policy{
				{ID: "B", Priority: 10},
				{ID: "A", Priority: 10},
				{ID: "C", Priority: 10},
			},
			want: []string{"A", "B", "C"},
		},
		{
			name: "mixed priorities and ties",
			give: []Policy{
				{ID: "C", Priority: 30},
				{ID: "B", Priority: 10},
				{ID: "A", Priority: 10},
				{ID: "D", Priority: 20},
			},
			want: []string{"A", "B", "D", "C"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := New(tc.give)
			got := make([]string, len(e.policies))
			for i, p := range e.policies {
				got[i] = p.ID
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("New sorted IDs = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNew_ClonesCallerSlice(t *testing.T) {
	give := []Policy{
		{ID: "C", Priority: 30},
		{ID: "A", Priority: 10},
	}
	want := []Policy{
		{ID: "C", Priority: 30},
		{ID: "A", Priority: 10},
	}

	New(give)

	if !reflect.DeepEqual(give, want) {
		t.Errorf("caller slice after New = %+v, want %+v (New must not sort in place)", give, want)
	}
}
