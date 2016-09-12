package eupho

type Plugin interface {
	Run(w *Worker, f func())
}

type PluginLoader interface {
	Load(name, args string) Plugin
}

type PluginLoaderFunc func(name, args string) Plugin

func (f PluginLoaderFunc) Load(name, args string) Plugin {
	return f(name, args)
}

var pluginLoaders map[string]PluginLoader = map[string]PluginLoader{}

func AppendPluginLoader(name string, loader PluginLoader) {
	pluginLoaders[name] = loader
}
