package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

// Config is loaded from JSON (snake_case). Omitted keys keep defaults from defaultConfig().
type Config struct {
	BehindNginx        bool     `json:"behind_nginx"`
	Port               int      `json:"port"`
	Domain             string   `json:"domain"`
	Tcping             bool     `json:"tcping"`
	Tcping6            bool     `json:"tcping6"`
	Msg                bool     `json:"msg"`
	Tracer             bool     `json:"tracer"`
	EnableView         bool     `json:"enable_view"`
	V4Fn               string   `json:"v4_fn"`
	V6Fn               string   `json:"v6_fn"`
	MaxSingleFileMB    int64    `json:"max_single_file_mb"`
	MaxTotalFileGB     int64    `json:"max_total_file_gb"`
	UdpPort            int      `json:"udp_port"`
	TcpPort            int      `json:"tcp_port"`
	UseQuic            bool     `json:"use_quic"`
	QuicOnly           bool     `json:"quic_only"`
	QuicAddr           string   `json:"quic_addr"`
	QuicCertPath       string   `json:"quic_cert_path"`
	QuicKeyPath        string   `json:"quic_key_path"`
	AgentUpstreamHosts []string `json:"agent_upstream_hosts"`
	MongoURI           string   `json:"mongo_uri"`
	MongoDB            string   `json:"mongo_db"`
	MongoColl          string   `json:"mongo_coll"`
	UploadDir          string   `json:"upload_dir"`
	StaticDir          string   `json:"static_dir"`
	AuthPassword       string   `json:"auth_password"`
	CookiePass         string   `json:"cookie_pass"`
	AuthExpireS        int64    `json:"auth_expire_s"`
}

// Cfg is the process-wide config after LoadConfig.
var Cfg Config

// AgentUpstreamHosts is the parsed lowercase host allowlist for agentproxy.
// Empty means deny all upstreams.
var AgentUpstreamHosts = map[string]struct{}{}

func defaultConfig() Config {
	return Config{
		BehindNginx:        true,
		Port:               10080,
		Domain:             "x.871116.xyz",
		Tcping:             false,
		Tcping6:            false,
		Msg:                false,
		Tracer:             false,
		EnableView:         false,
		V4Fn:               "v4.txt",
		V6Fn:               "v6.txt",
		MaxSingleFileMB:    100,
		MaxTotalFileGB:     10,
		UdpPort:            10081,
		TcpPort:            10082,
		UseQuic:            true,
		QuicOnly:           false,
		QuicAddr:           ":443",
		QuicCertPath:       "/root/sh/cert.pem",
		QuicKeyPath:        "/root/sh/priv.pem",
		AgentUpstreamHosts: []string{"llmapi.qiniu.com", "llmapi.qiniu.io", "heilovehei.com"},
		MongoURI:           "mongodb://localhost:27017",
		MongoDB:            "msglist",
		MongoColl:          "msg",
		UploadDir:          "/root/vps/www/dl",
		StaticDir:          "./public",
		AuthPassword:       "",
		CookiePass:         "",
		AuthExpireS:        604800,
	}
}

func normalizeAgentUpstreamHosts(hosts []string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, part := range hosts {
		host := strings.ToLower(strings.TrimSpace(part))
		if host == "" {
			continue
		}
		out[host] = struct{}{}
	}
	return out
}

func applyDerivedGlobals(cfg Config) {
	MaxHTTPPayload = cfg.MaxSingleFileMB * 1024 * 1024
	MaxTotalFileSize = cfg.MaxTotalFileGB * 1024 * 1024 * 1024
	AgentUpstreamHosts = normalizeAgentUpstreamHosts(cfg.AgentUpstreamHosts)
	initAuth(cfg)
}

func validateConfig(cfg Config) error {
	if cfg.Port < 0 || cfg.Port > 65535 {
		return fmt.Errorf("port out of range: %d", cfg.Port)
	}
	if cfg.UdpPort < 0 || cfg.UdpPort > 65535 {
		return fmt.Errorf("udp_port out of range: %d", cfg.UdpPort)
	}
	if cfg.TcpPort < 0 || cfg.TcpPort > 65535 {
		return fmt.Errorf("tcp_port out of range: %d", cfg.TcpPort)
	}
	if cfg.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	if cfg.MaxSingleFileMB <= 0 {
		return fmt.Errorf("max_single_file_mb must be > 0")
	}
	if cfg.MaxTotalFileGB <= 0 {
		return fmt.Errorf("max_total_file_gb must be > 0")
	}
	if cfg.UploadDir == "" {
		return fmt.Errorf("upload_dir is required")
	}
	if cfg.StaticDir == "" {
		return fmt.Errorf("static_dir is required")
	}
	if cfg.Msg {
		if cfg.MongoURI == "" || cfg.MongoDB == "" || cfg.MongoColl == "" {
			return fmt.Errorf("mongo_uri/mongo_db/mongo_coll are required when msg is true")
		}
	}
	if cfg.UseQuic {
		if cfg.QuicAddr == "" || cfg.QuicCertPath == "" || cfg.QuicKeyPath == "" {
			return fmt.Errorf("quic_addr/quic_cert_path/quic_key_path are required when use_quic is true")
		}
	}
	if cfg.AuthPassword != "" {
		if cfg.CookiePass == "" {
			return fmt.Errorf("cookie_pass is required when auth_password is set")
		}
		if cfg.AuthPassword == cfg.CookiePass {
			return fmt.Errorf("auth_password and cookie_pass must be different")
		}
		if cfg.AuthExpireS < 60 {
			return fmt.Errorf("auth_expire_s must be >= 60")
		}
	}
	return nil
}

// LoadConfig reads JSON from path into Cfg. Omitted keys keep defaults.
func LoadConfig(path string) error {
	cfg := defaultConfig()
	bt, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config %s: %w", path, err)
	}
	if err := json.Unmarshal(bt, &cfg); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}
	if err := validateConfig(cfg); err != nil {
		return fmt.Errorf("invalid config %s: %w", path, err)
	}
	Cfg = cfg
	applyDerivedGlobals(Cfg)
	logConfig(Cfg)
	return nil
}

func logConfig(cfg Config) {
	log.Println("BehindNginx:", cfg.BehindNginx)
	log.Println("Port:", cfg.Port)
	log.Println("Domain:", cfg.Domain)
	log.Println("Tcping:", cfg.Tcping)
	log.Println("Tcping6:", cfg.Tcping6)
	log.Println("Msg:", cfg.Msg)
	log.Println("Tracer:", cfg.Tracer)
	log.Println("EnableView:", cfg.EnableView)
	log.Println("V4Fn:", cfg.V4Fn)
	log.Println("V6Fn:", cfg.V6Fn)
	log.Println("MaxSingleFileMB:", cfg.MaxSingleFileMB)
	log.Println("MaxTotalFileGB:", cfg.MaxTotalFileGB)
	log.Println("UdpPort:", cfg.UdpPort)
	log.Println("TcpPort:", cfg.TcpPort)
	log.Println("UseQuic:", cfg.UseQuic)
	log.Println("QuicOnly:", cfg.QuicOnly)
	log.Println("QuicAddr:", cfg.QuicAddr)
	log.Println("QuicCertPath:", cfg.QuicCertPath)
	log.Println("QuicKeyPath:", cfg.QuicKeyPath)
	log.Println("AgentUpstreamHosts:", strings.Join(cfg.AgentUpstreamHosts, ","))
	log.Println("MongoURI:", cfg.MongoURI)
	log.Println("MongoDB:", cfg.MongoDB)
	log.Println("MongoColl:", cfg.MongoColl)
	log.Println("UploadDir:", cfg.UploadDir)
	log.Println("StaticDir:", cfg.StaticDir)
	log.Println("AuthEnabled:", cfg.AuthPassword != "")
	log.Println("AuthExpireS:", cfg.AuthExpireS)
}

func printUsage() {
	Version()
	fmt.Fprintf(flag.CommandLine.Output(), `
Usage:
  simpleserver [-c config.json]

Options:
  -c string
        path to JSON config file (default "config.json")
  -h, -help
        show this help and exit

Config file (JSON, snake_case). Omitted keys use defaults.
See sample.config.json for a full example with fake data.

Fields:
  behind_nginx          bool     default true
      Trust X-Real-Ip from reverse proxy; if false, set X-Real-Ip from RemoteAddr.
  port                  int      default 10080
      HTTP/1 and HTTP/2 listen port.
  domain                string   default "x.871116.xyz"
      Public domain used in upload/message pages and HTTP/3 SNI check.
  tcping                bool     default false
      Enable tcping workers and expose /metrics.
  tcping6               bool     default false
      Also ping IPv6 targets when tcping is true.
  msg                   bool     default false
      Enable message board routes /t and MongoDB storage.
  tracer                bool     default false
      Enable /tracer endpoint.
  enable_view           bool     default false
      Enable IP geo view (/ip/) using v4_fn/v6_fn databases.
  v4_fn                 string   default "v4.txt"
      IPv4 geo database file path.
  v6_fn                 string   default "v6.txt"
      IPv6 geo database file path.
  max_single_file_mb    int      default 100
      Max multipart memory / single upload size in MB.
  max_total_file_gb     int      default 10
      Max total uploaded bytes tracked in memory, in GB.
  udp_port              int      default 10081
      UDP echo port; 0 disables.
  tcp_port              int      default 10082
      TCP echo port; 0 disables.
  use_quic              bool     default true
      Enable HTTP/3 (QUIC) listener.
  quic_only             bool     default false
      If true, only run HTTP/3 (do not listen HTTP/1-2).
  quic_addr             string   default ":443"
      HTTP/3 listen address.
  quic_cert_path        string   default "/root/sh/cert.pem"
      TLS certificate file for HTTP/3.
  quic_key_path         string   default "/root/sh/priv.pem"
      TLS private key file for HTTP/3.
  agent_upstream_hosts  []string default ["llmapi.qiniu.com","llmapi.qiniu.io","heilovehei.com"]
      Allowlist for /v1/agentproxy/ x-upstream-base Host.
      Matching: exact host or subdomain suffix (heilovehei.com allows cn3.heilovehei.com).
      Only https upstreams are accepted. Empty array denies all upstreams.
  mongo_uri             string   default "mongodb://localhost:27017"
      MongoDB URI used when msg is true.
  mongo_db              string   default "msglist"
      MongoDB database name.
  mongo_coll            string   default "msg"
      MongoDB collection name.
  upload_dir            string   default "/root/vps/www/dl"
      Local directory for uploaded files.
  static_dir            string   default "./public"
      Public static files root for FileServer. Do NOT point this at the repo root.
      Put pages like agent.html under this directory.
  auth_password         string   default ""
      Site login password. Empty disables auth.
      When set, protects POST /upload, POST /t, and /v1/agentproxy/*.
  cookie_pass           string   default ""
      Secret used to sign the login cookie (ss_auth). Required if auth_password is set.
      Must be different from auth_password.
  auth_expire_s         int      default 604800
      Login cookie lifetime in seconds (min 60). Default 7 days.

Example:
  cp sample.config.json config.json
  simpleserver -c config.json
`)
}

// ParseArgs registers -c/-h only and returns the config path.
func ParseArgs() string {
	configPath := flag.String("c", "config.json", "path to JSON config file")
	flag.Usage = printUsage
	flag.Parse()
	return *configPath
}
