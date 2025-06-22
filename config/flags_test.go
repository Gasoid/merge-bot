package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringVar(t *testing.T) {
	// Reset flag set for testing
	originalFS := fs
	defer func() { fs = originalFS }()

	fs = flag.NewFlagSet("test", flag.ContinueOnError)

	var testString string
	StringVar(&testString, "test-string", "default", "test string variable")

	// Test that the flag was added
	flag := fs.Lookup("test-string")
	assert.NotNil(t, flag)
	assert.Equal(t, "test string variable", flag.Usage)
	assert.Equal(t, "default", flag.DefValue)
}

func TestBoolVar(t *testing.T) {
	// Reset flag set for testing
	originalFS := fs
	defer func() { fs = originalFS }()

	fs = flag.NewFlagSet("test", flag.ContinueOnError)

	var testBool bool
	BoolVar(&testBool, "test-bool", true, "test bool variable")

	// Test that the flag was added
	flag := fs.Lookup("test-bool")
	assert.NotNil(t, flag)
	assert.Equal(t, "test bool variable", flag.Usage)
	assert.Equal(t, "true", flag.DefValue)
}

func TestParse(t *testing.T) {
	// Save original args and restore after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Reset flag set for testing
	originalFS := fs
	defer func() { fs = originalFS }()

	fs = flag.NewFlagSet("test", flag.ContinueOnError)

	var testString string
	var testBool bool

	StringVar(&testString, "test-string", "default", "test string")
	BoolVar(&testBool, "test-bool", false, "test bool")

	// Test successful parsing
	os.Args = []string{"program", "-test-string=hello", "-test-bool=true"}

	// This should not exit or panic
	Parse()

	// Parse again to ensure variables are set correctly
	fs.Parse([]string{"-test-string=hello", "-test-bool=true"})
	assert.Equal(t, "hello", testString)
	assert.Equal(t, true, testBool)
}

func TestParseWithEnvVars(t *testing.T) {
	// Save original env vars and restore after test
	originalTestString := os.Getenv("TEST_STRING")
	originalTestBool := os.Getenv("TEST_BOOL")
	defer func() {
		if originalTestString == "" {
			os.Unsetenv("TEST_STRING")
		} else {
			os.Setenv("TEST_STRING", originalTestString)
		}
		if originalTestBool == "" {
			os.Unsetenv("TEST_BOOL")
		} else {
			os.Setenv("TEST_BOOL", originalTestBool)
		}
	}()

	// Save original args and restore after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Reset flag set for testing
	originalFS := fs
	defer func() { fs = originalFS }()

	fs = flag.NewFlagSet("test", flag.ContinueOnError)

	var testString string
	var testBool bool

	StringVar(&testString, "test-string", "default", "test string")
	BoolVar(&testBool, "test-bool", false, "test bool")

	// Set environment variables
	os.Setenv("TEST_STRING", "env-value")
	os.Setenv("TEST_BOOL", "true")

	// Test parsing with no command line args (should use env vars)
	os.Args = []string{"program"}

	// This should not exit or panic
	Parse()
}
