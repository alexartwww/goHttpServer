package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexartwww/goHttpServer/pkg/config"
)

const projectName = "goHttpServer"
const defaultHost = "default"

type Config struct {
	Settings `yaml:",inline"`
	Servers  []ServerConfig `yaml:"servers"`
}

// Settings apply to every connection: the virtual host is not known until the request head is read.
type Settings struct {
	ServerName   string   `yaml:"serverName"`
	ReadTimeout  Duration `yaml:"readTimeout"`
	WriteTimeout Duration `yaml:"writeTimeout"`
	MaxRequests  int      `yaml:"maxRequests"`
}

var defaultSettings = Settings{
	ServerName:   "goHttpServer/0.0.1",
	ReadTimeout:  Duration(5 * time.Second),
	WriteTimeout: Duration(30 * time.Second),
	MaxRequests:  1000,
}

var settings = defaultSettings

// Duration accepts only strings like "5s": yaml would read a bare 5 as 5 nanoseconds.
type Duration time.Duration

func (d *Duration) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	value, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(value)
	return nil
}

func (d Duration) String() string { return time.Duration(d).String() }

type ServerConfig struct {
	Listen    string   `yaml:"listen"`
	Host      string   `yaml:"host"`
	Home      string   `yaml:"home"`
	Indexes   []string `yaml:"indexes"`
	LogFormat string   `yaml:"logFormat"`
}

type vhost struct {
	host    string
	hostKey []byte
	home    string
	indexes []string // "" goes first: the requested path itself
	logJSON bool
	cache   fileCache
}

// loadConfig reads the config and groups virtual hosts by listen address.
func loadConfig(file string) (Settings, map[string][]*vhost, error) {
	if file == "" {
		found, err := config.Find("config", projectName)
		if err != nil {
			return Settings{}, nil, err
		}
		file = found
	}
	conf := Config{Settings: defaultSettings}
	if err := config.Load(file, &conf); err != nil {
		return Settings{}, nil, fmt.Errorf("%s: %w", file, err)
	}
	if err := conf.Settings.validate(); err != nil {
		return Settings{}, nil, fmt.Errorf("%s: %w", file, err)
	}
	if len(conf.Servers) == 0 {
		return Settings{}, nil, fmt.Errorf("%s: no servers", file)
	}

	listeners := map[string][]*vhost{}
	for i, s := range conf.Servers {
		vh, err := newVhost(s)
		if err != nil {
			return Settings{}, nil, fmt.Errorf("%s: servers[%d]: %w", file, i, err)
		}
		listen := s.Listen
		if listen == "" {
			listen = "localhost:4321"
		}
		for _, other := range listeners[listen] {
			if other.host == vh.host {
				return Settings{}, nil, fmt.Errorf("%s: servers[%d]: host %q on %s is already defined", file, i, vh.host, listen)
			}
		}
		listeners[listen] = append(listeners[listen], vh)
	}
	return conf.Settings, listeners, nil
}

func (s Settings) validate() error {
	// The name goes into the Server header: CR or LF there would let the config inject headers.
	for i := 0; i < len(s.ServerName); i++ {
		if c := s.ServerName[i]; c < 0x20 || c >= 0x7f {
			return fmt.Errorf("serverName %q: only printable ASCII is allowed", s.ServerName)
		}
	}
	if s.ServerName == "" {
		return fmt.Errorf("serverName is empty")
	}
	if s.ReadTimeout <= 0 || s.WriteTimeout <= 0 {
		return fmt.Errorf("readTimeout and writeTimeout must be positive")
	}
	if s.MaxRequests <= 0 {
		return fmt.Errorf("maxRequests must be positive")
	}
	return nil
}

func newVhost(s ServerConfig) (*vhost, error) {
	vh := &vhost{host: strings.ToLower(s.Host), home: filepath.Clean(s.Home)}
	if vh.host == "" {
		vh.host = defaultHost
	}
	vh.hostKey = []byte(vh.host)
	if s.Home == "" {
		return nil, fmt.Errorf("home is empty")
	}
	if info, err := os.Stat(vh.home); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("home %q is not a directory", s.Home)
	}
	switch s.LogFormat {
	case "", "json":
		vh.logJSON = true
	case "text":
	default:
		return nil, fmt.Errorf("logFormat %q: want json or text", s.LogFormat)
	}
	vh.indexes = append([]string{""}, s.Indexes...)
	if len(s.Indexes) == 0 {
		vh.indexes = append(vh.indexes, "index.html", "index.htm")
	}
	return vh, nil
}

// pickHost finds the virtual host by the Host header, falling back to "default" and then to the first one.
func pickHost(hosts []*vhost, host []byte) *vhost {
	host = stripPort(host)
	for _, vh := range hosts {
		if bytes.EqualFold(host, vh.hostKey) {
			return vh
		}
	}
	for _, vh := range hosts {
		if vh.host == defaultHost {
			return vh
		}
	}
	return hosts[0]
}

func stripPort(host []byte) []byte {
	if len(host) > 0 && host[0] == '[' {
		if i := bytes.IndexByte(host, ']'); i > 0 {
			return host[:i+1]
		}
		return host
	}
	if i := bytes.LastIndexByte(host, ':'); i >= 0 {
		return host[:i]
	}
	return host
}
