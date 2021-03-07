package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"tartanhackathon/recommend"
	"tartanhackathon/utils"
	"tartanhackathon/wiki"
)

func setupResponse(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func main() {
	ctx := context.Background()
	firestoreClient := utils.CreateFirestoreClient(ctx)
	languageHandlerChn := recommend.CreateLanguageHandler(ctx)
	defer firestoreClient.Close()
	defer close(languageHandlerChn)

	apis := []recommend.API{wiki.WikiRecommender()}

	// get list of recommendations
	http.HandleFunc("/recommend", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/recommend" {
			http.NotFound(w, r)
			return
		}
		setupResponse(&w, r)

		switch r.Method {
		// handle updating content and getting unique id
		case "POST":
			query := r.URL.Query()
			requestJSON, errJSON := utils.IOToJson(r.Body)
			nRecommendations, errRecommendations := strconv.ParseInt(query.Get("num"), 0, 64)
			texts, textsOk := requestJSON["texts"].([]interface{})

			// check for bugs
			if errJSON != nil {
				errMsg := "Unable to read request body as JSON"
				http.Error(w, errMsg, 400)
				log.Println(errMsg)
				return
			} else if errRecommendations != nil {
				errMsg := fmt.Sprintf("Unable to parse the number of recommendations needed: %s", query.Get("num"))
				http.Error(w, errMsg, 400)
				log.Println(errMsg)
				return
			} else if !textsOk {
				errMsg := fmt.Sprintf("JSON not properly constructed. Missing `texts` field that should contain a list: %s", requestJSON)
				http.Error(w, errMsg, 400)
				log.Println(errMsg)
				return
			}

			nTexts := len(texts)
			resChn := make(chan []recommend.EntityResult)

			// send all texts for language processing
			for _, text := range texts {
				textAsMap, textOk := text.(map[string]interface{})
				if !textOk {
					errMsg := fmt.Sprintf("JSON not properly constructed. Each element in the `texts` field should be a JSON object: %s", requestJSON)
					http.Error(w, errMsg, 400)
					log.Println(errMsg)
					return
				}

				body, bodyOk := textAsMap["body"].(string)
				if !bodyOk {
					errMsg := fmt.Sprintf("JSON not properly constructed. Missing `body` field per text snippet: %s", textAsMap)
					http.Error(w, errMsg, 400)
					log.Println(errMsg)
					return
				}
				languageHandlerChn <- recommend.TextQuery{body, resChn} // request sent for language processing
			}
			// retrieve new sentiment information
			toRecommendWith := make(map[string][]float32)
			fetchTimeout := time.After(recommend.SentimentFetchTimeout * time.Second)
		EntitySentimentLoop:
			for i := 0; i < nTexts; i++ {
				select {
				case <-fetchTimeout:
					log.Println("Fetching results of sentiment analysis timed out")
					break EntitySentimentLoop
				case results := <-resChn:
					for _, res := range results {
						toRecommendWith[res.Label] = append(toRecommendWith[res.Label], res.Sentiment)
					}
				}
			}

			type SentLabelPair struct {
				label string
				mean  float32
			}

			// calculate means
			meanChn := make(chan SentLabelPair)
			nLabels := len(toRecommendWith)
			defer close(meanChn)
			for label, sentiments := range toRecommendWith {
				go func(l string, s []float32) {
					var sum float32
					for _, sentiment := range s {
						sum += sentiment
					}
					meanChn <- SentLabelPair{l, sum / float32(len(s))}
				}(label, sentiments)
			}
			means := make([]SentLabelPair, 0, nLabels)
			for i := 0; i < nLabels; i++ {
				means = append(means, <-meanChn)
			}
			rand.Shuffle(nLabels, func(i, j int) { means[i], means[j] = means[j], means[i] })

			// recommend enough
			recommendations := make(chan []*url.URL)

			if int64(nLabels) < nRecommendations {
				nRecommendations = int64(nLabels)
			}
			log.Printf("Number of labels to perform recommendations on: %d\n", nRecommendations)

			defer close(recommendations)
			for i := int64(0); i < nRecommendations && i < int64(nLabels); i++ {
				go func(label string, mean float32) {
					recommendations <- recommend.Recommend(label, mean, apis)
				}(means[i].label, means[i].mean)
			}
			asList := make([]string, 0)
			for i := int64(0); i < nRecommendations && i < int64(nLabels); i++ {
				for _, recommendation := range <-recommendations {
					asList = append(asList, recommendation.String())
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string][]string{"urls": asList})
		case "OPTIONS":
			return
		default:
			http.Error(w, "Unsupported request method.", 405)
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
