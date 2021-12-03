/*
	A simple router to help my plugin application gateway experiment
*/
package TWTServer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// router routes :)
type SubRoute struct {
	Path      []string    // ex: ["misc", "file", "thefile.png"]
	Method    string      // ex: "GET" or "POST" or "CUSTOM" or "*"
	Handler   interface{} // handler for this route
	GlobIndex int         // cached index of glob
	IsDefault bool        // any router or subroute registered as *
}

// a collection of routes for the given application
type SubRouter struct {
	Routes []SubRoute
}

// build args["route"] for given subroute.
/*
	ex: route - /cake/:flavor
	request: /cake/chocolate
	args["route"] = { "flavor": "chocolate" }
*/
func (route *SubRoute) GetRouteParams(path []string) map[string]string {
	args := make(map[string]string)
	for index, routeChunk := range route.Path {
		if string(routeChunk[0]) == ":" {
			args[routeChunk[1:]] = path[index]
		}
	}
	return args
}

// build args["query"] for given subroute
/*
	ex: route /cake
	request: /cake?flavor=chocolate
	args["query"] = { "flavor" : "chocolate" }

	request: /cake?flavor=chocolate&flavor=vanilla
	args["query"] = { "flavor" : ["chocolate", "vanilla"] }
*/
func GetQueryParams(r *http.Request) map[string]interface{} {
	obj := make(map[string]interface{})

	for key, value := range r.URL.Query() {
		if len(value) < 2 {
			obj[key] = value[0]
		} else {
			obj[key] = value
		}
	}

	return obj
}

// build args["body"] for given subroute
/*
	ex: route /cake
	request: (post) /cake
	post body: { "flavor" : "chocolate" }
	args["body"] = { "flavor" : "chocolate" }
	+ similar nesting for args["query"] based on payload, which is why both are map[string]interface
*/
func GetRequestBody(r *http.Request) map[string]interface{} {
	if r.Body == nil {
		return nil
	}

	var obj map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&obj)
	if err != nil {
		if err != io.EOF {
			fmt.Println(err)
		}
		return nil
	}

	return obj
}

// used in the main router's Handle function to decide which route to execute, if any
func (route *SubRoute) MatchPath(path []string, method string) bool {
	// filter out requests by method
	if route.Method != "*" && method != route.Method {
		return false
	}

	// glob is at start, catch all route
	if route.GlobIndex == 0 {
		return true
	}

	// special case, catch undefined root route in a sub router
	if route.IsDefault {
		// ex sub route registered with prefix "sub" will have a default of sub *
		// ex sub route to that one called "foo" will have a default of sub foo *
		// and an incoming root request will be sub foo
		haveMatch := true
		for i := 0; i < len(route.Path)-1; i++ {
			if route.Path[i] != path[i] {
				haveMatch = false
			}
		}

		if haveMatch {
			return true
		}
	}

	if len(path) != len(route.Path) && (route.GlobIndex == -1 || route.GlobIndex >= len(path)) {
		// if there is a glob, the request /shirt/file/img/something.png could match /shirt/file/*
		// can also be used to have another sub router
		return false
	} else if len(path) == 0 && len(route.Path) == 0 {
		// root
		return true
	}
	for index, pathVal := range path {
		routeVal := route.Path[index]
		// can potentially catch everything if someone registered /* first which would be dumb
		// but that can also function as a 404 route or SPA index if registered last
		if routeVal == "*" {
			return true
		}

		// match route or continue matching if a variable is encountered
		if routeVal != pathVal && string(routeVal[0]) != ":" {
			return false
		}
	}
	return true
}

// add route to router
func (router *SubRouter) Register(uri string, method string, handler interface{}) {
	pathSlice := fixPath(strings.Split(uri, "/"))

	globIndex := -1
	for index, value := range pathSlice {
		if value == "*" {
			globIndex = index
			break
		}
	}

	router.Routes = append(router.Routes, SubRoute{
		Path:      pathSlice,
		Method:    method,
		Handler:   handler,
		GlobIndex: globIndex,
		IsDefault: globIndex == 0,
	})
}

// prefix a collection of SubRoutes
func (router *SubRouter) AddSubRoutes(prefix string, routes []SubRoute) {
	prefixed := make([]SubRoute, len(routes))
	for i, v := range routes {
		// shift existing globs up by 1
		adjustedGlobIndex := v.GlobIndex
		if v.GlobIndex != -1 {
			adjustedGlobIndex += 1
		}
		prefixed[i] = SubRoute{
			Path:      append([]string{prefix}, v.Path...),
			Method:    v.Method,
			Handler:   v.Handler,
			GlobIndex: adjustedGlobIndex,
			IsDefault: v.IsDefault, // sub routes registered as * will still be a default route once that sub route is entered
		}
	}

	router.Routes = append(router.Routes, prefixed...)
}

// default handler for application plugin's HandleWeb
func (router *SubRouter) Handle(w http.ResponseWriter, r *http.Request, path []string) {

	var response interface{}
	haveMatch := false
	writeResponse := false

	// find a path match
	for _, sub := range router.Routes {
		if sub.MatchPath(path, r.Method) {
			haveMatch = true
			// get search string, request body, and url params
			args := make(map[string]interface{})
			// map[string]string
			args["route"] = sub.GetRouteParams(path)
			// map[string]interface{}
			args["body"] = GetRequestBody(r)
			// map[string]interface{}
			args["query"] = GetQueryParams(r)
			// identify type of sub route handler
			switch t := sub.Handler.(type) {
			// simplified handler that returns json
			case func(map[string]interface{}) interface{}:
				writeResponse = true
				response = t(args)
			// generic handler that will write its own response to client
			case func(w http.ResponseWriter, r *http.Request, args map[string]interface{}):
				t(w, r, args)
			//
			default:
				fmt.Printf("Unhandled router signature %T\n", t)
			}
			break
		}
	}

	if haveMatch && writeResponse {
		responseStatusCode := 200
		// check for custom status code
		hash, haveMap := response.(map[string]interface{})
		if haveMap {
			if StatusCode, ok := hash["HTTPStatusCode"]; ok {
				intStatusCode, ok := StatusCode.(int)
				// remove 'HTTPStatusCode' from response
				delete(response.(map[string]interface{}), "HTTPStatusCode")
				if ok {
					responseStatusCode = intStatusCode
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Println("error processing response")
					return
				}
			}
		}

		encoded, err := json.Marshal(response)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "failed to encode response")
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(responseStatusCode)
			fmt.Fprintf(w, "%v", string(encoded))
		}
	} else if !haveMatch {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "unhandled request")
	}
	// else: type switch identified handler that will write to client directly
}
