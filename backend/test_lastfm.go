package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load("../.env")
	key := os.Getenv("LASTFM_API_KEY")
	url := fmt.Sprintf("http://ws.audioscrobbler.com/2.0/?method=artist.getsimilar&artist=cher&api_key=%s&format=json&limit=3", key)
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(b))
}
