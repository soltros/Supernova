package main
import (
	"fmt"
	"io/ioutil"
	"net/http"
)
func main() {
	url := "https://itunes.apple.com/search?term=taylor+swift&entity=album&limit=3&sort=recent"
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(b))
}
