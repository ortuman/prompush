package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	fs := flag.NewFlagSet("prompush", flag.ExitOnError)
	fs.SetOutput(os.Stdout)

	var url, username, password string
	var exLabelName, exLabelValue string

	fs.StringVar(&url, "push.host", "localhost:8082", "Remote write host.")
	fs.StringVar(&username, "push.username", "", "Set the basic auth user on write requests.")
	fs.StringVar(&password, "push.password", "", "Set the basic auth password on write requests.")
	fs.StringVar(&exLabelName, "push.exemplar.label", "trace_id", "Exemplar label name.")
	fs.StringVar(&exLabelValue, "push.exemplar.value", "1234", "Exemplar label value.")

	_ = fs.Parse(os.Args[1:])

	if err := runPush(url, username, password, exLabelName, exLabelValue); err != nil {
		log.Fatal(err)
	}
}
