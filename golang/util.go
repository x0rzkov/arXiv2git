package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

func shuffle(vals []string) []string {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	ret := make([]string, len(vals))
	perm := r.Perm(len(vals))
	for i, randIndex := range perm {
		ret[i] = vals[randIndex]
	}
	return ret
}

func logAndExitIfError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func requiredStringEnvVar(varName string) string {
	v := os.Getenv(varName)
	if v == "" {
		logAndExitIfError(fmt.Errorf("%s environment variable is not defined", varName))
	}
	return v
}

func optionalStringEnvVar(varName string, defaultValue string) string {
	v := os.Getenv(varName)
	if v == "" {
		return defaultValue
	}
	return v
}

func optionalBoolEnvVar(varName string, defaultValue bool) bool {
	v := os.Getenv(varName)
	if v == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(v)
	logAndExitIfError(err)
	return b
}

func parseExclusionList(envVar string) []string {
	if envVar == "" {
		return []string{}
	}
	return strings.Split(envVar, ",")
}
