package news

import (
	"log"
	"net/http"
	"net/url"
	"strconv"
	"tartanhackathon/recommend"
	"tartanhackathon/utils"
	"time"
)

const apiKey = "009de51effee49ada26b0ed9282b28bf"
const apiURL = "https://newsapi.org/v2/everything"
const queryTimeout = 5
const numNeededRes = 1000

type urlContentPair struct {
	u       *url.URL
	content string
}

func NewsRecommender(textQueryChn chan<- recommend.TextQuery) func(string, float32, float32, chan<- []*url.URL) {
	return func(label string, mean float32, dip float32, urlChn chan<- []*url.URL) {
		timeout := time.After(queryTimeout * time.Second)
		contentChn := make(chan urlContentPair)
		go func() {
			query(label, numNeededRes, timeout, contentChn)
		}()
		for {
			select {
			case <-timeout:
				log.Println("Timed out on getting list of news URL")
				return
			case content := <-contentChn:
				go func() {
					resChn := make(chan []recommend.EntityResult)
					query := recommend.TextQuery{content.content, resChn}
					select {
					case <-timeout:
						log.Println("Timed out on enqueuing task for sentiment analysis of a news article")
						return
					case textQueryChn <- query:
						select {
						case <-timeout:
							log.Println("Timed out on getting results of sentiment analysis of news article")
							return
						case entities := <-resChn:
							toSend := make([]*url.URL, 0)
							for _, entity := range entities {
								if entity.Label == label &&
									(mean < 0 && entity.Sentiment >= mean+dip || mean > 0 && entity.Sentiment <= mean-dip) {
									toSend = append(toSend, content.u)
								}
							}
							select {
							case <-timeout:
								log.Println("Timed out on sending URL of news articles that passed filters")
								return
							case urlChn <- toSend:
								return
							}
						}
					}
				}()
			}
		}
	}
}

func query(label string, nRes int, timeout <-chan time.Time, contentChn chan<- urlContentPair) {
	u, err := url.Parse(apiURL)
	if err == nil {
		page := 1
		numRes := 0
		for numRes < nRes {
			vals := u.Query()
			vals.Set("apiKey", apiKey)
			vals.Set("q", label)
			vals.Set("sortBy", "relevancy")
			vals.Set("page", strconv.Itoa(page))
			u.RawQuery = vals.Encode()
			resp, reqErr := http.Get(u.String())
			if reqErr == nil {
				json, jsonErr := utils.IOToJson(resp.Body)
				if jsonErr == nil {
					log.Printf("%s\n", json)
					status := json["status"].(string)
					if status != "ok" {
						return
					}
					articles := json["articles"].([]interface{})
					for _, article := range articles {
						asMap := article.(map[string]interface{})
						u, urlErr := url.Parse(asMap["url"].(string))
						if urlErr != nil {
							content := asMap["content"].(string)
							select {
							case contentChn <- urlContentPair{u, content}:
								numRes++
							case <-timeout:
								return
							}
						}
					}
				} else {
					log.Printf("Failed to parse response body as JSON")
				}
			} else {
				log.Printf("Failed to get request from news API. Response: %s", reqErr)
			}
			page++
		}
	}
}
