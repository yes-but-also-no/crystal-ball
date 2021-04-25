package main

import (
	"fmt"
	"github.com/orakurudata/crystal-ball/configuration"
	feed2 "github.com/orakurudata/crystal-ball/executor/feed"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s <configuration file>\n", os.Args[0])
		os.Exit(1)
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("Cannot open file %s: %s\n", os.Args[1], err)
		os.Exit(2)
	}
	feeds, err := configuration.ParseFeeds(f)
	if err != nil {
		fmt.Printf("Cannot parse configuration file: %s\n", err)
		os.Exit(2)
	}
outer:
	for name, feed := range feeds.Feeds {
		fmt.Printf("Executing feed %s (will be commited to %s):\n", name, feed.Name)
		values := make([]float64, 0)
		for _, sourceDef := range feed.Aggregation.Sources {
			source := feeds.Sources[sourceDef.Source]
			value, err := feed2.ExecuteSource(source, sourceDef.Arguments)
			if err != nil {
				fmt.Printf(" Failed to execute source %s: %s\n", sourceDef.Source, err)
				fmt.Printf(" Aborting feed execution")
				continue outer
			}
			fmt.Printf(" Source %s returned value %f\n", sourceDef.Source, value)
			values = append(values, value)
		}
		value := feed2.ExecuteAggregator(feed.Aggregation.Method, values)
		fmt.Printf(" [!] Aggregated value is: %f\n", value)
	}
}
