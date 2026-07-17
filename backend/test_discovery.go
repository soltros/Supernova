package main
import (
	"fmt"
	"io/ioutil"
	"net/http"
)
func main() {
	url := "http://ws.audioscrobbler.com/2.0/?method=artist.getsimilar&artist=cher&api_key=2618991ccfc560ee1f67f082eecfb848&format=json&limit=3"
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(b))
}
