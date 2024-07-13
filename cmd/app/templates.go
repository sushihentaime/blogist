package main

import "embed"

//go:embed templates/*
var htmlFS embed.FS
