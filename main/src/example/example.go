// This is an example plugin that will be available to www.yoursite.com/misc
package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/student020341/go-server-core/TWTServer"
)

var Foo ExamplePlugin

// router included with server core
var router TWTServer.SubRouter

// use init to setup your router
func init() {
	// www.site.com/misc/file/anything.asdf
	// shows route that takes a writer, a request, and args - expected to respond to client directly
	router.Register("/file/*", "GET", func(w http.ResponseWriter, r *http.Request, args map[string]interface{}) {
		// will attempt to serve the given path ex: project-root/files/misc/anything.asdf
		// r.URL.Path[11:] turns /misc/file/anything.asdf into anything.asdf
		// files/misc/hello.html will be included with the base repo, visit www.yoursite.com/misc/hello.html to check it out
		http.ServeFile(w, r, "./files/misc/"+r.URL.Path[11:])
	})

	// server particular file from exact without using file/* strategy
	router.Register("/foo", "GET", func(w http.ResponseWriter, r *http.Request, args map[string]interface{}) {
		http.ServeFile(w, r, "./files/misc/hello.html")
	})

	// www.site.com/misc/code/200
	// shows a route that takes args only - expected to return a json-like interface to be returned to client as json
	// note: needs to be a real status code or the server will throw errors
	router.Register("/code/:code", "*", func(args map[string]interface{}) interface{} {
		// get route arguments
		route := args["route"].(map[string]string)
		// get :code from route arguments
		status, err := strconv.Atoi(route["code"])

		var code int
		var msg string
		if err != nil {
			code = 500
			msg = err.Error()
		} else {
			code = status
			msg = "testing status code"
		}

		// special arg HTTPStatusCode will overwrite the status code returned by the included router
		return map[string]interface{}{
			"HTTPStatusCode": code,
			"status":         msg,
		}
	})

	// 404 route -- register last :)
	router.Register("*", "*", func(args map[string]interface{}) interface{} {
		return map[string]interface{}{
			"HTTPStatusCode": 404,
			"something":      "This is a 404 message as json instead of a file",
		}
	})
}

// use main to test your application outside of the server environment
func main() {
	fmt.Println("hello from example")
}
