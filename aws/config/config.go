package awsconfig

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"plugin"

	"gopkg.in/yaml.v2"

	"strings"

	config "github.com/Skarlso/go-furnace/config"
)

// Configuration object with all the properties that AWS needs.
type Configuration struct {
	Main struct {
		Stackname string `yaml:"stackname"`
		Spinner   int    `yaml:"spinner"`
	} `yaml:"main"`
	Aws struct {
		CodeDeployRole     string `yaml:"code_deploy_role"`
		Region             string `yaml:"region"`
		EnablePluginSystem bool   `yaml:"enable_plugin_system"`
		TemplateName       string `yaml:"template_name"`
		CodeDeploy         struct {
			S3Bucket    string `yaml:"code_deploy_s3_bucket, omitempty"`
			S3Key       string `yaml:"code_deploy_s3_key, omitempty"`
			GitAccount  string `yaml:"git_account, omitempty"`
			GitRevision string `yaml:"git_revision, omitempty"`
		} `yaml:"code_deploy"`
	} `yaml:"aws"`
}

const (
	// PRECREATE Event name for plugins
	PRECREATE = "pre_create"
	// POSTCREATE Event name for plugins
	POSTCREATE = "post_create"
	// PREDELETE Event name for plugins
	PREDELETE = "pre_delete"
	// POSTDELETE Event name for plugins
	POSTDELETE = "post_delete"
)

// Config is the loaded configuration entity.
var Config Configuration

// Plugin is a plugin to execute
type Plugin struct {
	Run  interface{}
	Name string
}

// PluginRegistry is a registry of plugins for certain events
var PluginRegistry map[string][]Plugin

var configPath string

func init() {
	configPath = config.Path()
	Config.loadConfiguration()
	PluginRegistry = fillRegistry()
}

func (c *Configuration) loadConfiguration() {
	content, err := ioutil.ReadFile(filepath.Join(configPath, "furnace_config.yaml"))
	config.HandleFatal("unable to load configuration file", err)
	err = yaml.Unmarshal(content, c)
	config.HandleFatal("couldn't unmarshall yaml content", err)
}

func fillRegistry() map[string][]Plugin {
	ret := make(map[string][]Plugin)
	if !Config.Aws.EnablePluginSystem {
		return ret
	}
	// log.Println("Filling plugin registry.")
	files, _ := ioutil.ReadDir(filepath.Join(configPath, "plugins"))
	pluginCount := 0
	for _, f := range files {
		split := strings.Split(f.Name(), ".")
		key := split[len(split)-1]
		fullPath := filepath.Join(configPath, "plugins", f.Name())
		p, err := plugin.Open(fullPath)
		if err != nil {
			log.Printf("Plugin '%s' failed to load. Error: %s\n", fullPath, err.Error())
			continue
		}
		run, err := p.Lookup("RunPlugin")
		if err != nil {
			log.Printf("Plugin '%s' did not have 'RunPlugin' method. Error: %s\n", fullPath, err.Error())
			continue
		}
		plug := Plugin{
			Run:  run,
			Name: f.Name(),
		}
		if p, ok := ret[key]; ok {
			p = append(p, plug)
			ret[key] = p
		} else {
			plugs := make([]Plugin, 0)
			plugs = append(plugs, plug)
			ret[key] = plugs
		}
		pluginCount++
	}
	log.Printf("'%d' plugins loaded successfully.\n", pluginCount)
	return ret
}

// LoadCFStackConfig Load the CF stack configuration file into a []byte.
func LoadCFStackConfig() []byte {
	dat, err := ioutil.ReadFile(filepath.Join(configPath, Config.Aws.TemplateName))
	config.CheckError(err)
	return dat
}
