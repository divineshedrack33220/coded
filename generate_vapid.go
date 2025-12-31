// File: backend/generate_vapid.go

package main

import (
	"fmt"
	"log"

	"github.com/SherClockHolmes/webpush-go"
)

func main() {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		log.Fatal("Failed to generate VAPID keys:", err)
	}

	fmt.Println("========================================")
	fmt.Println("VAPID PUBLIC KEY:")
	fmt.Println(publicKey)
	fmt.Println()
	fmt.Println("VAPID PRIVATE KEY:")
	fmt.Println(privateKey)
	fmt.Println("========================================")
	fmt.Println("Copy these and add them to your .env file:")
	fmt.Println("VAPID_PUBLIC_KEY=", publicKey)
	fmt.Println("VAPID_PRIVATE_KEY=", privateKey)
	fmt.Println("VAPID_EMAIL=mailto:your-email@example.com") // Use your real email
}