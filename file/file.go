package file

import (
	"bufio"
	"io"
)

func ReadLine(read io.Reader) <-chan string {
	lineCh := make(chan string)
	rd := bufio.NewReader(read)

	go func() {
		for {
			line, err := rd.ReadString('\n')

			//when error close lineCh include err == io.EOF
			if err != nil {
				close(lineCh)
				return
			}

			// line include \n
			lineCh <- line
		}
	}()

	return lineCh
}
