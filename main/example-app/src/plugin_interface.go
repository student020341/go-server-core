package main

import (
	"net/http"
)

var Foo ExamplePlugin

// implement receiver functions to meet STPlugin interface defined in TWTServer/server.go
type ExamplePlugin struct {
	Name    string
	FileDir string
}

func (p *ExamplePlugin) HandleWeb(w http.ResponseWriter, r *http.Request, path []string) {

	router.Handle(w, r, path)
}

func (p *ExamplePlugin) ReceiveInfo(name string, fileDir string) {
	Foo.Name = name
	Foo.FileDir = fileDir
}
