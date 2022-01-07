// Safe wrappers around common env var operations.
package env

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// reads env vars from a file.
//
// Failing will exit.
func Import(filename string) {
	err := godotenv.Load(filename)
	if err != nil {
		fmt.Println("Failed to load env vars from file: " + filename)
		fmt.Println(err)
		os.Exit(1)
	}
}

// returns an env var by name as string
//
// Failing will panic.
func Get(name string) string {
	envVar := os.Getenv(name)
	if envVar == "" {
		fmt.Println("Missing env var: " + name)
		os.Exit(1)
	}
	return envVar
}

func GetInt(name string) (num int) {
	envVar := Get(name)
	num, err := strconv.Atoi(envVar)
	if err != nil {
		fmt.Println("Invalid env var: " + name + ", expected int")
		fmt.Println(err)
		os.Exit(1)
	}
	return num
}
