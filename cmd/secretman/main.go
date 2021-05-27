package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/orakurudata/crystal-ball/secrets"
	"os"
)

func main() {
	generateMode := flag.Bool("generate", false, "generate a new key")
	encryptMode := flag.Bool("encrypt", false, "encrypt a message")
	decryptMode := flag.Bool("decrypt", false, "decrypt a message")
	message := flag.String("message", "", "message to encrypt")
	target := flag.String("pubkey", "", "recipient pubkey")
	flag.Parse()
	if !*generateMode && !*encryptMode && !*decryptMode {
		fmt.Println("specify operation mode")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if *generateMode {
		seed, err := secrets.GenerateKey()
		if err != nil {
			fmt.Println("generation error:", err)
			os.Exit(2)
		}
		public, err := secrets.PublicKeyFromSeed(seed)
		if err != nil {
			fmt.Println("conversion error:", err)
			os.Exit(2)
		}
		fmt.Println("Seed:", base64.StdEncoding.EncodeToString(seed))
		fmt.Println("Pubkey:", base64.StdEncoding.EncodeToString(public))
	} else if *encryptMode {
		targetKey, err := base64.StdEncoding.DecodeString(*target)
		if err != nil {
			fmt.Println("pubkey decode failed:", err)
			os.Exit(2)
		}
		key, err := secrets.GenerateKey()
		if err != nil {
			fmt.Println("failed to generate ephemeral key:", err)
			os.Exit(2)
		}
		pub, err := secrets.PublicKeyFromSeed(key)
		if err != nil {
			fmt.Println("conversion failed:", err)
			os.Exit(2)
		}
		encryptedData, err := secrets.Encrypt(key, targetKey, *message)
		if err != nil {
			fmt.Println("data encryption failed:", err)
			os.Exit(2)
		}
		fmt.Println(secrets.Encode(pub, encryptedData))
	}
}
