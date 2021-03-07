package main

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	language "cloud.google.com/go/language/apiv1"
	languagepb "google.golang.org/genproto/googleapis/cloud/language/v1"
)

const MaxRecordsPerLabel = 1000
const SalienceThreshold = .8
const LanguageHandlerTimeout = 1
const SentimentFetchTimeout = 5
const RecommendationTimeout = 5
const dip = 0.1

type API = func(string, float32, float32, chan<- url.URL)

func createFirestoreClient(ctx context.Context) *firestore.Client {
	projectID := "TartanHackathon"

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	return client
}

func main() {
	ctx := context.Background()
	firestoreClient := createFirestoreClient(ctx)
	languageHandlerChn := createLanguageHandler(ctx)
	defer firestoreClient.Close()
	defer close(languageHandlerChn)

	apis := []API{NewsRecommender(languageHandlerChn)}

	// get list of recommendations
	http.HandleFunc("/recommend", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/recommend" {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		// handle updating content and getting unique id
		case "POST":
			query := r.URL.Query()
			responseJSON, errJSON := IOToJson(r.Body)
			nRecommendations, errRecommendations := strconv.ParseInt(query.Get("num"), 0, 64)
			if errJSON != nil {
				http.Error(w, "Unable to read request body as JSON", 400)
			} else if errRecommendations != nil {
				http.Error(w, "Unable to parse number of recommendations needed", 400)
			} else {
				texts := responseJSON["texts"].([]interface{})
				nTexts := len(texts)
				resChn := make(chan []EntityResult)

				// send all texts for language processing
				for _, text := range texts {
					textAsMap := text.(map[string]interface{})
					body := textAsMap["body"].(string)
					select {
					case languageHandlerChn <- TextQuery{body, resChn}: // request sent for language processing
					case <-time.After(LanguageHandlerTimeout):
						http.Error(w, "Timeout in sentiment detection", 408)
					}
				}
				// retrieve new sentiment information
				toRecommendWith := make(map[string][]float32)
				fetchTimeout := time.After(SentimentFetchTimeout)
			EntitySentimentLoop:
				for i := 0; i < nTexts; i++ {
					select {
					case <-fetchTimeout:
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
				for label, sentiments := range toRecommendWith {
					go func(l string, s []float32) {
						var sum float32
						for _, sentiment := range s {
							sum += sentiment
						}
						meanChn <- SentLabelPair{l, sum / float32(len(s))}
					}(label, sentiments)
				}
				means := make([]SentLabelPair, 0, len(toRecommendWith))
				for pair := range meanChn {
					means = append(means, pair)
				}
				rand.Shuffle(len(means), func(i, j int) { means[i], means[j] = means[j], means[i] })

				// recommend enough
				recommendations := make(chan url.URL)
				var i int64
				for i = 0; i < nRecommendations; i++ {
					go func(label string, mean float32) {
						for _, recommendation := range recommend(label, mean, apis) {
							recommendations <- recommendation
						}
					}(means[i].label, means[i].mean)
				}

				asList := make([]string, 0)
				for recommendation := range recommendations {
					asList = append(asList, recommendation.String())
				}

				// encode recommendations
				json.NewEncoder(w).Encode(map[string][]string{"urls": asList})
			}
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

type TextQuery struct {
	Text string
	Res  chan<- []EntityResult
}

type EntityResult struct {
	Sentiment float32
	Label     string
}

func IOToJson(body io.ReadCloser) (map[string]interface{}, error) {
	var asMap map[string]interface{}
	asBytes, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	} else {
		json.Unmarshal(asBytes, &asMap)
		return asMap, nil
	}
}

func createLanguageHandler(ctx context.Context) chan<- TextQuery {
	queryChn := make(chan TextQuery)
	client, err := language.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	} else {
		go func() {
			for query := range queryChn {
				go func() {
					sendTimeout := time.After(SentimentFetchTimeout)

					// get sentiments of all entities
					sentimentResult, err := client.AnalyzeEntitySentiment(ctx,
						&languagepb.AnalyzeEntitySentimentRequest{
							Document: &languagepb.Document{
								Source: &languagepb.Document_Content{
									Content: query.Text,
								},
								Type: languagepb.Document_PLAIN_TEXT,
							},
							EncodingType: languagepb.EncodingType_UTF8,
						})
					if err == nil {
						res := make([]EntityResult, 0)

						// string together entities and their sentiments
						for _, entity := range sentimentResult.GetEntities() {
							salience := entity.GetSalience()
							sentiment := entity.GetSentiment().Score
							name := entity.GetName()
							if salience >= SalienceThreshold {
								res = append(res, EntityResult{sentiment, name})
							}
						}
						select {
						case query.Res <- res:
						case <-sendTimeout:
							log.Println("Timed out sending sentiment analysis")
						}
					}
				}()
			}
		}()
	}
	return queryChn
}

func recommend(label string, mean float32, apis []API) []url.URL {
	urls := make(chan url.URL)
	defer close(urls)

	for _, api := range apis {
		go func(api API) {
			api(label, mean, dip, urls)
		}(api)
	}

	asSlice := make([]url.URL, 0)
	recommendTimeout := time.After(RecommendationTimeout)
RecommendLoop:
	for {
		select {
		case url, ok := <-urls:
			if !ok {
				break RecommendLoop
			}
			asSlice = append(asSlice, url)
		case <-recommendTimeout:
			break RecommendLoop
		}
	}
	return asSlice
}
