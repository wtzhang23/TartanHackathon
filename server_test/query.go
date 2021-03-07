package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
)

const test = "http://localhost:8080/recommend"
const beescript = `
According to all known laws of aviation, 
there is no way a bee should be able to fly.  
Its wings are too small to get
its fat little body off the ground.  
The bee, of course, flies anyway  
because bees don't care
what humans think is impossible.
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
