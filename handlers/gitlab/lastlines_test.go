package gitlab

import (
	"strings"
	"testing"
)

func TestReadLastLines(t *testing.T) {
	cases := []struct {
		name string
		in   string
		n    int
		want string
	}{
		{"fewer than n", "1\n2\n3\n", 5, "1\n2\n3"},
		{"exact n", "1\n2\n3\n", 3, "1\n2\n3"},
		{"more than n", "1\n2\n3\n4\n5\n", 3, "3\n4\n5"},
		{"short lines", "aaa\nbbb\nccc\nddd\n", 3, "bbb\nccc\nddd"},
		{"empty reader", "", 3, ""},
		{"no trailing newline", "1\n2\n3", 3, "1\n2\n3"},
		{"n zero", "1\n2\n3\n", 0, ""},
		{"long line", strings.Repeat("x", 2048) + "\n" + "tail\n", 1, "tail"},
		{"crlf", "1\r\n2\r\n3\r\n", 3, "1\n2\n3"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := readLastLines(strings.NewReader(c.in), c.n)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}
