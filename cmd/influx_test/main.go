package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"botik/answer"
	"botik/api"
	"botik/util"
)

func main() {
	client := &http.Client{Timeout: time.Second * 5}

	i := api.NewInfluxApi("192.168.0.1:8086", client)
	r, err := i.GetSingleSeries("bio", "select last(sys), * from pressure where \"name\"='kott'")

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if len(r) == 1 {
		if p, err := answer.MapToPressure(r[0]); err == nil {
			fmt.Printf("Last record: %s\n", util.FormatTime(p.Time))
			fmt.Printf("%d/%d", p.Sys, p.Dia)
		}
	}
}
