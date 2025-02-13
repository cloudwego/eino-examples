package env

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

}

func MustHasEnvs(envs ...string) {
	for _, env := range envs {
		if os.Getenv(env) == "" {
			log.Fatalf("❌ [ERROR] env [%s] is required, but is not setted now, please check your .env file", env)
		}
	}
}
