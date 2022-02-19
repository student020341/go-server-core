package TWTServer

import "net/http"

// Interface for plugins to implement. Every plugin must export a variable that implements STPlugin.
// The default plugin name will be "TWTPlugin" but can be renamed with the config "plugin-variable".
type STPlugin interface {
	/*
		void function that writes a response to the client. In most cases this can be handled by SubRouter.Handle.

		If your website is www.site.com, your plugin is "misc" and you register a route of "/hello",
		when you visit www.site.com/misc/hello your plugin "misc" will be passed a path of ["hello"].
	*/
	HandleWeb(http.ResponseWriter, *http.Request, []string)

	/*
		void function that will receive config information from the gateway, like what name the application is being served as
		or what directory the files exist in.
	*/
	ReceiveInfo(name string, fileDir string)
}

// a basic / common implementation for the interface
type BasicPlugin struct {
	Name    string
	FileDir string
	Router  SubRouter
}

func (p *BasicPlugin) HandleWeb(w http.ResponseWriter, r *http.Request, path []string) {

	p.Router.Handle(w, r, path)
}

func (p *BasicPlugin) ReceiveInfo(name string, fileDir string) {
	p.Name = name
	p.FileDir = fileDir
}

func (p *BasicPlugin) FilePath(url string, fileRoute string) string {
	return p.FileDir + "/" + url[len("/"+p.Name+"/"+fileRoute+"/"):]
}
