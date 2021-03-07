package wiki

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"tartanhackathon/utils"
	"time"
)

const apiURL = "https://en.wikipedia.org/w/api.php"
const queryTimeout = 5
const numNeededRes = 1000

func WikiRecommender() func(string, float32, float32, chan<- []*url.URL) {
	return func(label string, mean float32, dip float32, urlChn chan<- []*url.URL) {
		timeout := time.After(queryTimeout * time.Second)
		u, err := url.Parse(apiURL)
		if err != nil {
			log.Fatalln("Failed to parse wikipedia API URL")
		}
		// query for label in wikipedia
		vals := u.Query()
		vals.Set("action", "query")
		vals.Set("list", "search")
		vals.Set("srsearch", label)
		vals.Set("format", "json")
		u.RawQuery = vals.Encode()
		log.Printf("Requesting url %s\n", u.String())
		resp, reqErr := http.Get(u.String())
		if reqErr != nil {
			log.Println("Failed to parse body of request as JSON")
			return
		}
		// parse query
		json, jsonErr := utils.IOToJson(resp.Body)
		if jsonErr != nil {
			log.Printf("Failed to receive query from wikipedia: %s\n", json)
			return
		}
		query, queryOk := json["query"].(map[string]interface{})
		if !queryOk {
			log.Printf("Failed to access query for Wikipedia. json: %s", json)
			return
		}

		search := query["search"].([]interface{})
		if len(search) != 0 {
			top := search[0].(map[string]interface{})
			pageid := int64(top["pageid"].(float64))
			u, urlErr := url.Parse(fmt.Sprintf("https://en.wikipedia.org/?curid=%d", pageid))
			if urlErr != nil {
				log.Println("Failed to create URL for id of label")
				return
			}
			select {
			case <-timeout:
				log.Println("Timed out on sending wikipedia link")
				return
			case urlChn <- []*url.URL{u}:
			}
		}
	}
}
