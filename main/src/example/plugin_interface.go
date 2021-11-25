package main

import (
	"net/http"
)

// implement receiver functions to meet STPlugin interface defined in TWTServer/server.go
type ExamplePlugin struct{}

func (p *ExamplePlugin) HandleWeb(w http.ResponseWriter, r *http.Request, path []string) {

	router.Handle(w, r, path)
}

func (p *ExamplePlugin) GetName() string {
	return "misc"
}

func (p *ExamplePlugin) GetFilesDir() string {
	return "../files"
}
