package gcbench

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLastLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		n    int
		want string
	}{
		{
			name: "last two of three",
			in:   "a\nb\nc",
			n:    2,
			want: "b\nc",
		},
		{
			name: "trailing newline trimmed",
			in:   "a\nb\nc\n",
			n:    2,
			want: "b\nc",
		},
		{
			name: "fewer lines than n",
			in:   "only",
			n:    5,
			want: "only",
		},
		{
			name: "empty input",
			in:   "",
			n:    3,
			want: "",
		},
		{
			name: "last one of three",
			in:   "x\ny\nz",
			n:    1,
			want: "z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, lastLines(tt.in, tt.n))
		})
	}
}
