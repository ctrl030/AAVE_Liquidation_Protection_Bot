package test

import (
	"context"
	"log"
	"math/big"
	"os"
	"os/exec"
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

// TestFixture sets up the local (forked) test network and the initial loan. The network stays
// up until the loan is repaid, which is determined by polling.
func TestFixture(t *testing.T) {
	ctx := context.Background()

	client, err := clients.NewClient(params)
	if err != nil {
		t.Fatalf("client initialization failed: %v", err)
	}

	// Deploys the contract.
	rep, repAddr, err := repayment.Deploy(ctx, client)
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

	loan, err := client.Loan(ctx, user.Address)
	if err != nil {
		t.Fatalf("clients.Loan(ctx, %v) = _, %v, want _, nil", user.Address, err)
	}

	// Creates the certificate.
	cert, err := delegation.New(client.BotAddress())
	if err != nil {
		t.Fatalf("delegation.New(%v) = _, %v, want _, nil", client.BotAddress(), err)
	}

	s, err := service.New(service.Deps{
		Client:  client,
		RepAddr: repAddr,
		Rep:     rep,
		Root:    "../../../ui/dist",
		Cert:    cert,
	})
	if err != nil {
		t.Fatalf("service.New(...) = _, %v, want _, nil", err)
	}

	// Since Run doesn't return, we start it in a goroutine.
	go s.Run()

	for {
		debt, err := loan.DebtAmount(ctx, client)
		if err != nil {
			t.Fatalf("loan.DebtAmount(...) = _, %v, want _, nil", err)
		}
		if debt.Cmp(big.NewInt(0)) == 0 {
			t.Logf("Debt has been repaid.")
			return
		}
		time.Sleep(time.Second * 2)
	}
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
