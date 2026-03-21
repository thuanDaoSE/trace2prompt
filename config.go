package main

import (
	"log"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

type RegexRule struct {
	Pattern string `yaml:"pattern"`
	Replace string `yaml:"replace"`
}

type Config struct {
	Masking struct {
		Fields []string    `yaml:"fields"`
		Regex  []RegexRule `yaml:"regex"`
	} `yaml:"masking"`
}

var AppConfig Config

type CompiledRule struct {
	Regex   *regexp.Regexp
	Replace string
}

var CompiledMaskingRules []CompiledRule

func LoadConfig() {
	// 1. Default configuration if the user does not have a yaml file
	AppConfig.Masking.Fields = []string{
		"password", "secret", "token", "api_key", "apikey",
		"credential", "cookie", "authorization", "credit_card", "cvv",
	}
	AppConfig.Masking.Regex = []RegexRule{
		{Pattern: `(?i)(Bearer\s+|token=|api_key=|secret=)[A-Za-z0-9\-\._~+/]+=*`, Replace: "${1}[REDACTED]"},
		{Pattern: `(?i)[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}`, Replace: "[EMAIL_HIDDEN]"},
		{Pattern: `(?i)(password|passwd|secret|token|api_key|cvv|credit_card)[\s=:]+('[^']+'|"[^"]+")`, Replace: "${1} = '[REDACTED]'"},
		{Pattern: `(https?://[^\s"'<>]+)(\?(X-Amz-Algorithm|AWSAccessKeyId|Signature|Expires|X-Goog-Algorithm)[^\s"'<>]*)`, Replace: "$1?[SIGNED_PARAMS_HIDDEN]"},
		{Pattern: `sk-[a-zA-Z0-9]{48}`, Replace: "[REDACTED_API_KEY]"},
	}

	// 2. Attempt to read trace2prompt.yaml file
	data, err := os.ReadFile("trace2prompt.yaml")
	if err == nil {
		err = yaml.Unmarshal(data, &AppConfig)
		if err != nil {
			log.Printf("Error parsing trace2prompt.yaml file: %v", err)
		}
		log.Println("🟢 Masking configuration loaded from trace2prompt.yaml")
	} else {
		log.Println("🟡 trace2prompt.yaml not found, using default Masking configuration")
	}

	// 3. Pre-compile Regexes to optimize performance
	for _, rule := range AppConfig.Masking.Regex {
		compiled, err := regexp.Compile(rule.Pattern)
		if err == nil {
			CompiledMaskingRules = append(CompiledMaskingRules, CompiledRule{Regex: compiled, Replace: rule.Replace})
		} else {
			log.Printf("🔴 Regex error in yaml: %s", rule.Pattern)
		}
	}
}
