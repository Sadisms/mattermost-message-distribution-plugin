package main

import "fmt"

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func removeDuplicates(strList []string) []string {
	list := []string{}
	for _, item := range strList {
		fmt.Println(item)
		if contains(list, item) == false {
			list = append(list, item)
		}
	}
	return list
}
