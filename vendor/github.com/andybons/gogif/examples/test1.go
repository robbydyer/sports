package main

import (
	"bytes"
	"gogif"
	"image/gif"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	f, err := os.Open("testdata/scape.gif")
	if err != nil {
		log.Fatalf("os.Open: %q", err)
	}
	defer f.Close()

	g, err := gif.DecodeAll(f)
	if err != nil {
		log.Fatal(err)
	}
	var buf bytes.Buffer
	if err := gogif.EncodeAll(&buf, g); err != nil {
		log.Fatal(err)
	}
	ioutil.WriteFile("test1.gif", buf.Bytes(), 0660)
}
