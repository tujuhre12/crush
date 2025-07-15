package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/crush/internal/ollama"
)

func main() {
	fmt.Println("🧪 Ollama Test Suite")
	fmt.Println("===================")

	// Test 1: Check if Ollama is installed
	fmt.Print("1. Checking if Ollama is installed... ")
	if ollama.IsInstalled() {
		fmt.Println("✅ PASS")
	} else {
		fmt.Println("❌ FAIL - Ollama is not installed")
		fmt.Println("   Please install Ollama from https://ollama.com")
		os.Exit(1)
	}

	// Test 2: Check if Ollama is running
	fmt.Print("2. Checking if Ollama is running... ")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if ollama.IsRunning(ctx) {
		fmt.Println("✅ PASS")
	} else {
		fmt.Println("❌ FAIL - Ollama is not running")

		// Test 3: Try to start Ollama service
		fmt.Print("3. Attempting to start Ollama service... ")
		ctx2, cancel2 := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel2()

		if err := ollama.StartOllamaService(ctx2); err != nil {
			fmt.Printf("❌ FAIL - %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ PASS")

		// Verify it's now running
		fmt.Print("4. Verifying Ollama is now running... ")
		if ollama.IsRunning(ctx2) {
			fmt.Println("✅ PASS")
		} else {
			fmt.Println("❌ FAIL - Service started but not responding")
			os.Exit(1)
		}
	}

	// Test 4: Get available models
	fmt.Print("5. Getting available models... ")
	ctx3, cancel3 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel3()

	models, err := ollama.GetModels(ctx3)
	if err != nil {
		fmt.Printf("❌ FAIL - %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ PASS (%d models found)\n", len(models))

	// Display models
	if len(models) > 0 {
		fmt.Println("\n📋 Available Models:")
		for i, model := range models {
			fmt.Printf("   %d. %s\n", i+1, model.ID)
			fmt.Printf("      Context: %d tokens, Max: %d tokens\n",
				model.ContextWindow, model.DefaultMaxTokens)
		}
	} else {
		fmt.Println("\n⚠️  No models found. You may need to download some models first.")
		fmt.Println("   Example: ollama pull llama3.2:3b")
	}

	// Test 5: Get provider
	fmt.Print("\n6. Getting Ollama provider... ")
	provider, err := ollama.GetProvider(ctx3)
	if err != nil {
		fmt.Printf("❌ FAIL - %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ PASS (%s with %d models)\n", provider.Name, len(provider.Models))

	// Test 6: Test model loading check
	if len(models) > 0 {
		testModel := models[0].ID
		fmt.Printf("7. Checking if model '%s' is loaded... ", testModel)

		loaded, err := ollama.IsModelLoaded(ctx3, testModel)
		if err != nil {
			fmt.Printf("❌ FAIL - %v\n", err)
		} else if loaded {
			fmt.Println("✅ PASS (model is loaded)")
		} else {
			fmt.Println("⚠️  PASS (model is not loaded)")
		}
	}

	fmt.Println("\n🎉 All tests completed successfully!")
	fmt.Println("\nTo run individual tests:")
	fmt.Println("   go test ./internal/ollama -v")
	fmt.Println("\nTo run benchmarks:")
	fmt.Println("   go test ./internal/ollama -bench=.")
}
