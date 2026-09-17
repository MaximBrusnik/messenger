package config

import (
	"os"
	"strconv"
)

type Config struct {
	Env             string
	ServiceName     string
	ServerPort      string
	GRPCPort        string
	HTTPPort        string
	JWTSecret       string
	Postgres        Postgres
	Redis           Redis
	NATS            NATS
	SMTP            SMTP
	App             App
	AIConfig        AI
	PushConfig      Push
	Services        ServiceAddrs
	Calls           CallBackend
	FileStorageDir  string
	MusicStorageDir string
	MusicHTTPAddr   string
}

type ServiceAddrs struct {
	UserAddr     string
	ChatAddr     string
	AuthAddr     string
	RealtimeAddr string
	MusicAddr    string
	AIAddr       string
	PushAddr     string
	RealtimeWS   string
	CallsAddr    string
	CallsWS      string
}

type CallBackend struct {
	StunServers []string
	TurnServers []TurnServer
}

// TurnServer describes a single TURN relay endpoint + credentials.
type TurnServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

type Postgres struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type Redis struct {
	Addr string
	Pass string
}

type NATS struct {
	URL string
}

type SMTP struct {
	Host string
	Port string
	User string
	Pass string
	From string
}

type App struct {
	AppURL                   string
	RequireEmailVerification bool
}

// AI Gemini API settings.
type AI struct {
	GeminiAPIKey string
}

// Push FCM settings.
type Push struct {
	FCMCredentials string
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func Load() *Config {
	return &Config{
		Env:         getEnv("ENV", "development"),
		ServiceName: getEnv("SERVICE_NAME", "unknown"),
		ServerPort:  getEnv("SERVER_PORT", "8080"),
		GRPCPort:    getEnv("GRPC_PORT", "9000"),
		HTTPPort:    getEnv("HTTP_PORT", "8080"),
		JWTSecret:   getEnv("JWT_SECRET", "change-me-in-production"),
		Postgres: Postgres{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "postgres"),
		},
		Redis: Redis{
			Addr: getEnv("REDIS_ADDR", "localhost:6379"),
			Pass: getEnv("REDIS_PASSWORD", ""),
		},
		NATS: NATS{
			URL: getEnv("NATS_URL", "nats://localhost:4222"),
		},
		SMTP: SMTP{
			Host: getEnv("SMTP_HOST", ""),
			Port: getEnv("SMTP_PORT", "465"),
			User: getEnv("SMTP_USER", ""),
			Pass: getEnv("SMTP_PASS", ""),
			From: getEnv("SMTP_FROM", ""),
		},
		App: App{
			AppURL:                   getEnv("APP_URL", "http://localhost:8080"),
			RequireEmailVerification: getEnvBool("REQUIRE_EMAIL_VERIFICATION", false),
		},
		AIConfig: AI{
			GeminiAPIKey: getEnv("GEMINI_API_KEY", ""),
		},
		PushConfig: Push{
			FCMCredentials: getEnv("FCM_CREDENTIALS", ""),
		},
		Services: ServiceAddrs{
			UserAddr:     getEnv("SERVICE_USER_ADDR", "userservice:9000"),
			ChatAddr:     getEnv("SERVICE_CHAT_ADDR", "chatservice:9000"),
			AuthAddr:     getEnv("SERVICE_AUTH_ADDR", "authservice:9000"),
			RealtimeAddr: getEnv("SERVICE_REALTIME_ADDR", "realtimeservice:9000"),
			MusicAddr:    getEnv("SERVICE_MUSIC_ADDR", "musicservice:9000"),
			AIAddr:       getEnv("SERVICE_AI_ADDR", "aiservice:9000"),
			PushAddr:     getEnv("SERVICE_PUSH_ADDR", "pushservice:9000"),
			RealtimeWS:   getEnv("SERVICE_REALTIME_WS_ADDR", "realtimeservice:8080"),
			CallsAddr:    getEnv("SERVICE_CALLS_ADDR", "callsservice:9000"),
			CallsWS:      getEnv("SERVICE_CALLS_WS_ADDR", "callsservice:8080"),
		},
		Calls: CallBackend{
			StunServers: []string{getEnv("CALLS_STUN", "stun:stun.l.google.com:19302")},
		},
		FileStorageDir:  getEnv("FILE_STORAGE_DIR", "uploads"),
		MusicStorageDir: getEnv("MUSIC_STORAGE_DIR", "music"),
		MusicHTTPAddr:   getEnv("SERVICE_MUSIC_HTTP_ADDR", "musicservice:8080"),
	}
}
