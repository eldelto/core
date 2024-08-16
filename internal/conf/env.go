package conf

import (
	"log"
	"os"
	"strconv"
)

func RequireEnvVar(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("required environment variable %q is not set", key)
	}

	return value
}

func RequireIntEnvVar(key string) int64 {
	rawValue := RequireEnvVar(key)

	value, err := strconv.ParseInt(rawValue, 10, 64)
	if err != nil {
		log.Fatalf("value %q of environment variable %q is not a valid int: %v",
			rawValue, key, err)
	}

	return value
}

func EnvVarWithDefault(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		log.Printf("environment variable %q not set - using fallback value %q\n",
			key, fallback)
		return fallback
	}

	return value
}

func IntEnvVarWithDefault(key string, fallback int64) int64 {
	rawValue, ok := os.LookupEnv(key)
	if !ok {
		log.Printf("environment variable %q not set - using fallback value %d\n",
			key, fallback)
		return fallback
	}

	value, err := strconv.ParseInt(rawValue, 10, 64)
	if err != nil {
		log.Fatalf("value %q of environment variable %q is not a valid int: %v",
			rawValue, key, err)
	}

	return value
}
