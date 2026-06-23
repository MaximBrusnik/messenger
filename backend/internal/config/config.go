package config

import "os"

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	JWTSecret  string
	ServerPort string
	Env        string
	UploadDir  string

	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPass     string
	SMTPFrom     string
	AppURL       string
	GeminiAPIKey string
	MusicDir     string
}

func Load() *Config {
	return &Config{
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "postgres"),
		DBPassword:   getEnv("DB_PASSWORD", "postgres"),
		DBName:       getEnv("DB_NAME", "auth_db"),
		JWTSecret:    getEnv("JWT_SECRET", "your-super-secret-key-change-in-production"),
		ServerPort:   getEnv("SERVER_PORT", "8080"),
		Env:          getEnv("ENV", "development"),
		UploadDir:    getEnv("UPLOAD_DIR", "uploads"),
		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPass:     getEnv("SMTP_PASS", ""),
		SMTPFrom:     getEnv("SMTP_FROM", "MessangerMax <noreply@example.com>"),
		AppURL:       getEnv("APP_URL", "http://localhost:5173"),
		GeminiAPIKey: getEnv("GEMINI_API_KEY", ""),
		MusicDir:     getEnv("MUSIC_DIR", "music"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
