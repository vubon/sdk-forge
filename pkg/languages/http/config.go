// Package http provides HTTP library configuration and validation for different programming languages.
//
//nolint:revive // Package name 'http' is intentional and does not conflict with stdlib in this context
package http

import "fmt"

const (
	langPython = "python"
	langPy     = "py"
	langGo     = "go"
	langGolang = "golang"
	langPHP    = "php"
	langJS     = "js"
	langJSFull = "javascript"
	langTS     = "ts"
	langTSFull = "typescript"
	langRuby   = "ruby"
)

// LibraryConfig represents configuration for an HTTP library
type LibraryConfig struct {
	Import      string // How to import it in code
	Dependency  string // Dependency specification for package managers
	ClientClass string // Client class/type name
	Default     bool   // Is this the default for the language?
}

// LanguageConfig maps HTTP library names to their configurations
type LanguageConfig map[string]LibraryConfig

// Config holds HTTP library configurations for all languages
type Config struct {
	Python LanguageConfig
	Go     LanguageConfig
	PHP    LanguageConfig
	JS     LanguageConfig
	Ruby   LanguageConfig
}

// GetConfig returns the HTTP library configuration for all languages
func GetConfig() *Config {
	return &Config{
		Python: LanguageConfig{
			"requests": {
				Import:      "requests",
				Dependency:  "requests>=2.31.0",
				ClientClass: "requests.Session",
				Default:     true,
			},
			"httpx": {
				Import:      "httpx",
				Dependency:  "httpx>=0.24.0",
				ClientClass: "httpx.Client",
				Default:     false,
			},
			"aiohttp": {
				Import:      "aiohttp",
				Dependency:  "aiohttp>=3.9.0",
				ClientClass: "aiohttp.ClientSession",
				Default:     false,
			},
			"urllib3": {
				Import:      "urllib3",
				Dependency:  "urllib3>=2.0.0",
				ClientClass: "urllib3.PoolManager",
				Default:     false,
			},
		},
		Go: LanguageConfig{
			"nethttp": {
				Import:      "net/http",
				Dependency:  "", // Standard library, no dependency needed
				ClientClass: "http.Client",
				Default:     true,
			},
			"resty": {
				Import:      "github.com/go-resty/resty/v2",
				Dependency:  "github.com/go-resty/resty/v2 v2.11.0",
				ClientClass: "*resty.Client",
				Default:     false,
			},
			"gentleman": {
				Import:      "gopkg.in/h2non/gentleman.v2",
				Dependency:  "gopkg.in/h2non/gentleman.v2 v2.0.5",
				ClientClass: "*gentleman.Client",
				Default:     false,
			},
		},
		PHP: LanguageConfig{
			"guzzle": {
				Import:      "GuzzleHttp\\Client",
				Dependency:  "guzzlehttp/guzzle:^7.0",
				ClientClass: "GuzzleHttp\\Client",
				Default:     true,
			},
			"symfony": {
				Import:      "Symfony\\Component\\HttpClient\\HttpClient",
				Dependency:  "symfony/http-client:^6.0",
				ClientClass: "Symfony\\Component\\HttpClient\\HttpClient",
				Default:     false,
			},
			"curl": {
				Import:      "",
				Dependency:  "", // Built-in PHP extension
				ClientClass: "curl",
				Default:     false,
			},
		},
		JS: LanguageConfig{
			"axios": {
				Import:      "axios",
				Dependency:  "axios:^1.6.0",
				ClientClass: "AxiosInstance",
				Default:     true,
			},
			"fetch": {
				Import:      "",
				Dependency:  "", // Built-in browser/Node.js API
				ClientClass: "fetch",
				Default:     false,
			},
			"node-fetch": {
				Import:      "node-fetch",
				Dependency:  "node-fetch:^3.3.0",
				ClientClass: "fetch",
				Default:     false,
			},
			"ky": {
				Import:      "ky",
				Dependency:  "ky:^1.1.0",
				ClientClass: "ky",
				Default:     false,
			},
		},
		Ruby: LanguageConfig{
			"faraday": {
				Import:      "faraday",
				Dependency:  "faraday:~> 2.0",
				ClientClass: "Faraday::Connection",
				Default:     true,
			},
			"net-http": {
				Import:      "net/http",
				Dependency:  "", // Standard library, no dependency needed
				ClientClass: "Net::HTTP",
				Default:     false,
			},
			"httprb": {
				Import:      "http",
				Dependency:  "http:~> 5.0",
				ClientClass: "HTTP::Client",
				Default:     false,
			},
		},
	}
}

// GetDefaultLibrary returns the default HTTP library for a language
func GetDefaultLibrary(language string) string {
	config := GetConfig()

	switch language {
	case langPython, langPy:
		for name, lib := range config.Python {
			if lib.Default {
				return name
			}
		}
	case langGo, langGolang:
		for name, lib := range config.Go {
			if lib.Default {
				return name
			}
		}
	case langPHP:
		for name, lib := range config.PHP {
			if lib.Default {
				return name
			}
		}
	case langJS, langJSFull, langTS, langTSFull:
		for name, lib := range config.JS {
			if lib.Default {
				return name
			}
		}
	case langRuby:
		for name, lib := range config.Ruby {
			if lib.Default {
				return name
			}
		}
	}

	return ""
}

// IsValidLibrary checks if an HTTP library is valid for a given language
func IsValidLibrary(language, httpLib string) bool {
	config := GetConfig()

	switch language {
	case langPython, langPy:
		_, exists := config.Python[httpLib]
		return exists
	case langGo, langGolang:
		_, exists := config.Go[httpLib]
		return exists
	case langPHP:
		_, exists := config.PHP[httpLib]
		return exists
	case langJS, langJSFull, langTS, langTSFull:
		_, exists := config.JS[httpLib]
		return exists
	case langRuby:
		_, exists := config.Ruby[httpLib]
		return exists
	}

	return false
}

// GetValidLibraries returns a list of valid HTTP libraries for a language
func GetValidLibraries(language string) []string {
	config := GetConfig()
	var libraries []string

	switch language {
	case langPython, langPy:
		for name := range config.Python {
			libraries = append(libraries, name)
		}
	case langGo, langGolang:
		for name := range config.Go {
			libraries = append(libraries, name)
		}
	case langPHP:
		for name := range config.PHP {
			libraries = append(libraries, name)
		}
	case langJS, langJSFull, langTS, langTSFull:
		for name := range config.JS {
			libraries = append(libraries, name)
		}
	case langRuby:
		for name := range config.Ruby {
			libraries = append(libraries, name)
		}
	}

	return libraries
}

// GetLibraryConfig returns the configuration for a specific HTTP library
func GetLibraryConfig(language, httpLib string) (*LibraryConfig, error) {
	config := GetConfig()

	var langConfig LanguageConfig
	switch language {
	case langPython, langPy:
		langConfig = config.Python
	case langGo, langGolang:
		langConfig = config.Go
	case langPHP:
		langConfig = config.PHP
	case langJS, langJSFull, langTS, langTSFull:
		langConfig = config.JS
	case langRuby:
		langConfig = config.Ruby
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	libConfig, exists := langConfig[httpLib]
	if !exists {
		return nil, fmt.Errorf("invalid HTTP library '%s' for language '%s'", httpLib, language)
	}

	return &libConfig, nil
}
