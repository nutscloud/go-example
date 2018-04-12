package tempalte

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"path"
	"strings"
	"text/template"
	"time"
)

//
func ExecuteTempalte(letter string, data interface{}) (ret string, err error) {
	buf := bytes.NewBuffer([]byte{})
	// Create a new template and parse the letter into it.
	t := template.Must(template.New("letter").Funcs(newFuncMap()).Parse(letter))

	defer func() {
		ret = buf.String()
	}()

	dataMarshal, err := json.Marshal(data)
	if err != nil {
		log.Println("Marshal error:", err)
		return
	}
	// Execute the template for each recipient.
	err = t.Execute(buf, dataMarshal)
	if err != nil {
		log.Println("executing template:", err)
		return
	}

	return
}

func newFuncMap() map[string]interface{} {
	m := make(map[string]interface{})
	m["base"] = path.Base
	m["split"] = strings.Split
	m["json"] = UnmarshalJsonObject
	m["dir"] = path.Dir
	m["getenv"] = os.Getenv
	m["join"] = strings.Join
	m["datetime"] = time.Now
	m["toUpper"] = strings.ToUpper
	m["toLower"] = strings.ToLower
	m["contains"] = strings.Contains
	m["replace"] = strings.Replace
	return m
}

func UnmarshalJsonObject(data []byte) (map[string]interface{}, error) {
	var ret map[string]interface{}
	err := json.Unmarshal(data, &ret)
	return ret, err
}
