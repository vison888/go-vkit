package config

import (
	"log"
	"testing"
)

func TestConfigs_Load(t *testing.T) {
	log.Println(GetString("NODE_NAME"))
}
