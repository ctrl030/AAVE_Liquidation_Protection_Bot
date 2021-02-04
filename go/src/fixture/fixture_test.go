package test

import (
	"context"
	"encoding/hex"
	"log"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"clients"
	"delegation"
	"env"
	"ports"
	"repayment"
	"scenarios"
	"service"
	"wallets"
)

var (
	params = env.LocalTestNet()
)

func TestFixture(t *testing.T) {
	ctx := context.Background()

	client, err := clients.NewClient(params)
	if err != nil {
		t.Fatalf("client initialization failed: %v", err)
	}

	// Deploys the contract.
	_, repAddr, err := repayment.Deploy(ctx, client)
	if err != nil {
		t.Fatalf("deploying repayment contract failed: %v", err)
	}

	user, err := wallets.NewWallet(params.UserKey())
	if err != nil {
		t.Fatalf("Error creating user wallet: %v", err)
	}

	// Sets up a loan.
	if err := scenarios.SetupLoan(ctx, client, user); err != nil {
		t.Fatalf("Error setting up loan: %v", err)
	}

	// Creates the certificate.
	cert, err := delegation.New(client.BotAddress())
	if err != nil {
		t.Fatalf("delegation.New(%v) = _, %v, want _, nil", client.BotAddress(), err)
	}

	// Computes the signature locally to test it against the one produced by Metamask.
	hash, err := cert.Hash()
	if err != nil {
		t.Fatalf("cert.Hash() = _, %v, want _, nil", err)
	}
	sig, err := user.Sign(hash)
	if err != nil {
		t.Fatalf("Error signing certificate: %v", err)
	}
	want := hex.EncodeToString(sig[:len(sig)-1])

	var wg sync.WaitGroup
	wg.Add(1)
	var got string

	// service.Serve doesn't return, so we start it in a goroutine, then wait on the callback.
	go service.Serve(client, "../../../ui/dist", repAddr, cert, func(sig []byte) {
		got = string(sig[2 : len(sig)-2])
		wg.Done()
	})

	wg.Wait()

	if got != want {
		t.Fatalf("signature mismatch, got=%s, want=%s", got, want)
	}
	t.Logf("done!")
}

func TestMain(m *testing.M) {
	ctx, cancelNode := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "npx", "hardhat", "node")
	cmd.Dir = "../../../hardhat"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start node: %v", err)
	}
	if err := ports.WaitForConnect("127.0.0.1:8545", time.Minute); err != nil {
		log.Fatalf("Couldn't connect to node: %v", err)
	}

	result := m.Run()

	cancelNode()
	cmd.Wait()
	os.Exit(result)
}
