package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"plugin"
	"strings"
)

// WebRouter - handler functions from sub routed applications
var WebRouter map[string]func(http.ResponseWriter, *http.Request, []string)

// plugin related
func deleteBuiltModules() {
	builtFiles, err := os.Open("./modules")
	if err != nil {
		panic(err)
	}
	defer builtFiles.Close()

	list, err := builtFiles.Readdirnames(0)
	if err != nil {
		panic(err)
	}
	for _, name := range list {
		err = os.Remove("./modules/" + name)
		if err != nil {
			fmt.Println("error removing file:", err)
		}
	}
}

func buildModules() []string {

	var names []string
	fmt.Println("discovering plugins...")
	// find all plugins
	files, err := os.Open("./src")
	if err != nil {
		panic(err)
	}
	defer files.Close()

	list, err := files.Readdirnames(0)
	if err != nil {
		panic(err)
	}

	// since a default is provided, this should always succeed
	include := progArgs["include"].([]string)

	for _, name := range list {
		// skip lib
		if name == "lib" {
			continue
		}
		// check for exclusion
		if len(include) > 0 {
			skip := true
			for _, file := range include {
				if file == name {
					skip = false
					break
				}
			}
			if skip {
				continue
			}
		}

		fmt.Printf("building %s...\n", name)
		// build plugin
		err := exec.Command("go", "build", "-o", "./modules/"+name+".so", "-buildmode=plugin", "./src/"+name).Run()
		if err != nil {
			panic(err)
		} else {
			names = append(names, name+".so")
		}
	}

	return names
}

func getExistingModules() []string {
	include := progArgs["include"].([]string)
	builtFiles, err := os.Open("./modules")
	if err != nil {
		panic(err)
	}
	defer builtFiles.Close()

	list, err := builtFiles.Readdirnames(0)
	if err != nil {
		panic(err)
	}

	var files []string
	if len(include) > 0 {
		for _, file := range list {
			for _, inc := range include {
				if file == (inc + ".so") {
					files = append(files, file)
					break
				}
			}
		}
	} else {
		files = list
	}

	return files
}

func loadModules(names []string) {
	fmt.Printf("loading %v modules...\n", len(names))
	// initialize web handler
	WebRouter = make(map[string]func(http.ResponseWriter, *http.Request, []string))
	// load plugins
	loaded := 0
	for _, name := range names {
		mod, err := plugin.Open("./modules/" + name)
		// lookup exported router module name
		exportedGetName, err := mod.Lookup("GetName")
		if err != nil {
			fmt.Printf("module '%s' did not provide a name", name)
			continue
		}
		getName, ok := exportedGetName.(func() string)
		if !ok {
			fmt.Printf("GetName failed for module '%s'", name)
			continue
		}
		// check for web handler
		exportedWebHandler, err := mod.Lookup("HandleWeb")
		if err == nil {
			handleWeb, ok := exportedWebHandler.(func(http.ResponseWriter, *http.Request, []string))
			if ok {
				loaded++
				WebRouter[getName()] = handleWeb
			}
		}
		//todo: check for internal handler
	}

	fmt.Printf("loaded %v modules\n", loaded)
}

func argsAndConfig() {
	// default program options
	progArgs = map[string]interface{}{
		"build":   false,
		"include": []string{},
	}

	// get config file
	configRaw, err := ioutil.ReadFile("config.json")
	if err == nil {
		// config file exists
		var obj map[string]interface{}
		err = json.Unmarshal(configRaw, &obj)
		// invalid config file?
		if err != nil {
			panic(err)
		}

		// check for files to include
		files, ok := obj["include"].([]interface{})
		if ok {
			for _, f := range files {
				progArgs["include"] = append(progArgs["include"].([]string), f.(string))
			}
		}
	}

	// get program args second, override configs
	for _, arg := range os.Args {
		if arg == "--build" {
			progArgs["build"] = true
		}
	}
}

func doPluginStuff() {
	// ensure the modules folder exists since a fresh git pull won't have it
	err := exec.Command("mkdir", "-p", "modules").Run()
	if err != nil {
		panic(err)
	}

	// have --build flag
	var files []string
	if progArgs["build"].(bool) {
		deleteBuiltModules()
		files = buildModules()
	} else {
		files = getExistingModules()
	}

	loadModules(files)
}

// Handle - main server handler
func Handle(w http.ResponseWriter, r *http.Request) {
	path := fixPath(strings.Split(r.URL.Path, "/"))
	if len(path) == 0 {
		fmt.Fprintf(w, "home")
	} else if path[0] == "favicon.ico" {
		http.ServeFile(w, r, "./files/avatar.png")
	} else if handler, ok := WebRouter[path[0]]; ok {
		handler(w, r, path[1:])
	} else {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "404 or something")
	}
}

func fixPath(path []string) []string {
	var tmp []string
	for _, value := range path {
		if value != "" {
			tmp = append(tmp, value)
		}
	}
	return tmp
}

var progArgs map[string]interface{}

func main() {

	argsAndConfig()
	doPluginStuff()

	// todo: make server killable via web request
	http.HandleFunc("/", Handle)

	port := "2000"
	fmt.Println("serving on port:", port)

	http.ListenAndServe(":"+port, nil)
}
