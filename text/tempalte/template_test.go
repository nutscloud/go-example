package tempalte

import (
	"fmt"
	"testing"
)

func TestExecuteTemplate(t *testing.T) {
	// Define a template.
	var (
		letter = `
{{$data := json . -}}
Dear {{$data.Name}},`
		d = map[string]string{"Name": "xiaojian"}
	)

	ret, err := ExecuteTempalte(letter, d)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(ret)
}
