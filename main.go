package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/fatih/set"
	"github.com/golang/glog"
	_ "github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

type Response struct {
	Photos struct {
		Page  int
		Pages int
		//Perpage int
		Total string
		Photo []struct {
			Id string
			//Owner string
			//Secret string
			//Server string
			//Farm int
			//Title string
			//Ispublic int
			//Isfriend int
			//Isfamily int
			Url_o string // original
			//Height_o string : there is a bug that flickr API returns int instead of str
			//Width_o string
			Url_k string // 2048
			//Height_k string
			//Width_k string
			Url_h string // 1600
			//Height_h string
			//Width_h string
		}
	}
	Stat string
}

func main() {
	flag.Parse()
	// ctx := context.Background()
	// ctx, cancel := context.WithCancel(ctx)
	// sigint_ch := make(chan os.Signal, 1)
	// signal.Notify(sigint_ch, os.Interrupt)
	// defer func() {
	// 	signal.Stop(sigint_ch)
	// 	cancel()
	// }()

	throttle := time.Tick(time.Millisecond * 1050)
	wg := sync.WaitGroup{}

	urls := set.New()
	min_date := time.Date(2014, 1, 1, 0, 0, 0, 0, time.UTC)
	til := time.Date(2017, 12, 31, 0, 0, 0, 0, time.UTC)
	// til := time.Date(2017, 12, 31, 0, 0, 0, 0, time.UTC)
	// num_interval := 12 * 4 // 4 years // 365 * 7 / 5

	for {
		max_date := min_date.AddDate(0, 0, 1)
		if max_date.After(til) {
			break
		}

		sum := 0

		for page := 1; page <= 4000; page++ { // flickr returns up to 4000 results for the same query parameter => 500*8=400

			// select {
			// case <-sigint_ch: // ctrl+C -> cancel all downloads
			// 	glog.Warning("Pressed ctrl+C. Cancelling ..")
			// 	cancel()
			// 	break main_loop
			// default:
			req, err := http.NewRequest("GET", "https://api.flickr.com/services/rest/", nil)
			if err != nil {
				glog.Fatal("Error in http.NewRequest")
				panic(err)
			}
			q := req.URL.Query()
			q.Add("method", "flickr.photos.search")
			q.Add("api_key", "2a778c6655c1cf48f1feaf0fac3fc764")
			q.Add("format", "json")
			q.Add("text", "people -blackandwhite -monochrome")
			q.Add("woe_id", "23424868") // geo query -> 250
			q.Add("nojsoncallback", "1")
			q.Add("extras", "url_h,url_k,url_o")
			q.Add("min_upload_date", strconv.FormatInt(min_date.Unix(), 10))
			q.Add("max_upload_date", strconv.FormatInt(max_date.Unix()-1, 10))
			q.Add("page", strconv.Itoa(page))
			q.Add("per_page", "500")
			req.URL.RawQuery = q.Encode()

			<-throttle
			wg.Add(1)

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				glog.Fatal("Error in response")
				panic(err)
			}

			obj := Response{}
			json.NewDecoder(res.Body).Decode(&obj)
			res.Body.Close()

			fmt.Printf("[%s - %s] page %d/%d - %d\n", min_date.Format("2006-01-02"), max_date.Format("2006-01-02"), page, obj.Photos.Pages, len(obj.Photos.Photo))

			for _, photo := range obj.Photos.Photo {
				url := ""
				if photo.Url_o != "" {
					url = photo.Url_o
				} else if photo.Url_k != "" {
					url = photo.Url_k
				} else if photo.Url_h != "" {
					url = photo.Url_h
				}
				if url != "" {
					urls.Add(url)
				}
			}

			wg.Done()
			sum += len(obj.Photos.Photo)

			if sum >= 4000 || page >= obj.Photos.Pages {
				break
			}
			// }
		}
		min_date = max_date
	}
	wg.Wait()
	glog.Info("Crawling done. Writing to urls.txt ..")
	{
		f, err := os.Create("urls.txt")
		if err != nil {
			glog.Fatal("cannot open urls.txt")
		}
		defer f.Close()
		w := bufio.NewWriter(f)
		defer w.Flush()

		urls := urls.List()
		for _, url := range urls {
			w.WriteString(url.(string))
			w.WriteByte('\n')
		}
	}
}
