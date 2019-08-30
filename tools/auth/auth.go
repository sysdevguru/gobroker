package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/alpacahq/gopaca/auth"
)

var (
	key = flag.String("key", "", "key to get")
)

func init() {
	flag.Parse()
}

func main() {
	a, err := auth.Get(*key)
	if err != nil {
		panic(err)
	}

	buf, err := json.MarshalIndent(a, "", "    ")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(buf))
}
