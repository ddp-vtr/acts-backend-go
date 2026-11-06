package main

import (
	"acts-backend-go/internal/database"
	"acts-backend-go/internal/model"
	"acts-backend-go/internal/rag"
	"fmt"
	"os"
)

func main() {
	repo := database.NewRepository()
	endpoint := "https://rlau4p8.abc-tunnel.us/v1/chat/completions"
	apiKey := "sk-5b2c9a975f7f309f-qcuwzl-e7e1385d"
	modelName := "ag/gemini-3.8-flash-high"

	engine := rag.NewEngine(repo, endpoint, apiKey, modelName)

	fmt.Println("=== TEST 1: Filter out-of-stock (BBIA Cushion stock = 0) ===")
	res1, err := engine.ProcessQuery(model.ChatRequest{
		Message: "Tư vấn cushion bbia căng bóng giúp mình",
	})
	if err != nil {
		fmt.Printf("Error 1: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Matched items count: %d\n", len(res1.MatchedItems))
	fmt.Printf("LLM Reply: %s\n\n", res1.Reply)

	fmt.Println("=== TEST 2: Price constraint (son dưới 180k) ===")
	res2, err := engine.ProcessQuery(model.ChatRequest{
		Message: "Tìm cho mình son bbia dưới 180k tone ấm",
	})
	if err != nil {
		fmt.Printf("Error 2: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Matched items count: %d\n", len(res2.MatchedItems))
	for _, it := range res2.MatchedItems {
		fmt.Printf("- %s | %s | Stock: %d\n", it.Name, it.PriceDisplay, it.StockQuantity)
	}
	fmt.Printf("LLM Reply: %s\n", res2.Reply)
}

