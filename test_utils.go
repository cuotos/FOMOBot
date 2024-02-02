package main

import "encoding/json"

func prettyPrint(i any) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}

func print(i any) string {
	s, _ := json.Marshal(i)
	return string(s)
}
