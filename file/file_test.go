package file

import (
	"bytes"
	"testing"
)

func Test_ReadLine(t *testing.T) {
	lines := []string{
		"aaaaaa\n",
		"bbbbbb\n",
		"cccccc\n",
	}

	buf := bytes.NewBuffer([]byte{})
	for _, l := range lines {
		if _, err := buf.WriteString(l); err != nil {
			t.Error(err)
			return
		}
	}

	i := 0
	for line := range ReadLine(buf) {
		if line != lines[i] {
			t.Errorf("readline err: expect:%s, actual:%s", lines[i], line)
			return
		}
		i++
	}
}
