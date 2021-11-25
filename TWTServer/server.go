package TWTServer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"plugin"
	"strconv"
	"strings"
)

// Type to receive all of the functions here
type ServerThing struct {
	// handler functions from sub routed applications
	WebRouter map[string]func(http.ResponseWriter, *http.Request, []string)
	ProgArgs  struct {
		Build   bool     `json:"build"`
		Port    int      `json:"port"`
		Include []string `json:"include"`
	}
}

// clear ./modules folder, intended to be used before building
func (st *ServerThing) DeleteBuiltModules() {
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

// step through src folders (excluding lib) and attempt to build .so files
func (st *ServerThing) BuildModules() []string {

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

	// folder names from your src dir
	for _, name := range list {
		// skip lib
		if name == "lib" {
			continue
		}
		// check for exclusion
		if len(st.ProgArgs.Include) > 0 {
			skip := true
			for _, file := range st.ProgArgs.Include {
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

// check ./modules folder for existing modules instead of building them
func (st *ServerThing) GetExistingModules() []string {
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
	if len(st.ProgArgs.Include) > 0 {
		for _, file := range list {
			for _, inc := range st.ProgArgs.Include {
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

// load .so files in ./modules dir, optionally filtered by a config file or program arg
func (st *ServerThing) LoadModules(names []string) {
	fmt.Printf("loading %v modules...\n", len(names))
	// initialize web handler
	st.WebRouter = make(map[string]func(http.ResponseWriter, *http.Request, []string))
	// load plugins
	loaded := 0
	for _, name := range names {
		mod, err := plugin.Open("./modules/" + name)
		if err != nil {
			fmt.Printf("failed to open %s\n", name)
			continue
		}
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
				st.WebRouter[getName()] = handleWeb
			}
		}
		//todo: build concept of & check for internal handler
	}

	fmt.Printf("loaded %v modules\n", loaded)
}

// build args from args & config file
func (st *ServerThing) ArgsAndConfig() {
	// default program options
	st.ProgArgs.Build = false
	st.ProgArgs.Include = []string{}
	st.ProgArgs.Port = 2000

	// get config file
	configRaw, err := ioutil.ReadFile("config.json")
	if err == nil {
		err = json.Unmarshal(configRaw, &st.ProgArgs)
		if err != nil {
			panic(err)
		}
	}

	// override config with args
	for i, arg := range os.Args {
		if arg == "--build" {
			st.ProgArgs.Build = true
		}
		if arg == "--port" && i+1 < len(os.Args) {
			portInt, err := strconv.Atoi(os.Args[i+1])
			if err != nil {
				panic(err)
			}
			st.ProgArgs.Port = portInt
		}
		// get include as comma separated list
		if arg == "--include" && i+1 < len(os.Args) {
			st.ProgArgs.Include = strings.Split(os.Args[i+1], ",")
		}
	}
}

// determine whether we're loading or building modules, then do it!
func (st *ServerThing) DoPluginStuff() {
	// ensure the modules folder exists since a fresh git pull won't have it
	err := exec.Command("mkdir", "-p", "modules").Run()
	if err != nil {
		panic(err)
	}

	// have --build flag
	var files []string
	if st.ProgArgs.Build {
		st.DeleteBuiltModules()
		files = st.BuildModules()
	} else {
		files = st.GetExistingModules()
	}

	st.LoadModules(files)
}

// Handle - main server handler
func (st *ServerThing) Handle(w http.ResponseWriter, r *http.Request) {
	path := fixPath(strings.Split(r.URL.Path, "/"))
	if len(path) == 0 {
		// todo: enable devs to easily do *something* here, but this is primarily an application gateway
		fmt.Fprintf(w, "home")
	} else if path[0] == "favicon.ico" {
		// browsers seem to make this request automatically :)
		// feel free to move this into ./files or serve a different file name
		http.ServeFile(w, r, "./favicon.ico")
	} else if handler, ok := st.WebRouter[path[0]]; ok {
		// ex: if we have a subrouter called misc and the route is /misc/asdf/qwerty
		// then pass ["asdf", "qwerty"] into the misc sub application
		handler(w, r, path[1:])
	} else {
		// todo: enable devs to do something here. Maybe special root & 404 handlers
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "404 or something")
	}
}

func (st *ServerThing) Start() {
	st.ArgsAndConfig()
	st.DoPluginStuff()

	http.HandleFunc("/", st.Handle)
	fmt.Println("serving on port:", st.ProgArgs.Port)
	http.ListenAndServe(fmt.Sprintf(":%d", st.ProgArgs.Port), nil)
}
