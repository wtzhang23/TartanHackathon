package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
)

const test = "https://tartanhackathon.uc.r.appspot.com/recommend"
const beescript = `
My point is that writing a new operating system that is closely tied to any
particular piece of hardware, especially a weird one like the Intel line,
is basically wrong.  An OS itself should be easily portable to new hardware
platforms.  When OS/360 was written in assembler for the IBM 360
25 years ago, they probably could be excused.  When MS-DOS was written
specifically for the 8088 ten years ago, this was less than brilliant, as
IBM and Microsoft now only too painfully realize. Writing a new OS only for the
386 in 1991 gets you your second 'F' for this term.  But if you do real well
on the final exam, you can still pass the course.
`

func main() {
	query := map[string]interface{}{
		"texts": []interface{}{
			map[string]interface{}{
				"body": beescript,
			},
		},
	}
	u, err := url.Parse(test)
	if err != nil {
		log.Fatalln("Failed to create URL")
	}

	newQuery := u.Query()
	newQuery.Set("num", "10")
	u.RawQuery = newQuery.Encode()
	asBytes, err := json.Marshal(query)
	if err != nil {
		log.Fatalln("Failed to create json query")
	}

	log.Printf("%s\n", u.String())
	resp, err := http.Post(u.String(), "application/json", bytes.NewReader(asBytes))
	if err != nil {
		log.Fatalln(err.Error())
	} else {
		log.Println(resp.Status)
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		log.Println(result)
	}
}
