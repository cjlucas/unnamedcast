package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/cjlucas/unnamedcast/worker/rss"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: rssparse <url>")
		return
	}

	input := os.Args[1]

	var data []byte

	if input == "-" {
		data, _ = ioutil.ReadAll(os.Stdin)

	} else {
		resp, err := http.Get(input)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		data, _ = ioutil.ReadAll(resp.Body)
	}

	doc, err := rss.ParseFeed(bytes.NewReader(data))
	if err != nil {
		panic(err)
	}

	if buf, err := json.Marshal(doc); err != nil {
		panic(err)
	} else {
		var out bytes.Buffer
		json.Indent(&out, buf, "", "  ")
		fmt.Println(out.String())
	}
}
