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
	"wallets"
)

var (
	params = env.LocalTestNet()
)

func TestContract(t *testing.T) {
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

	// The user approves the contract to withdraw collateral so it can use it to pay back the
	// flash loan.
	if err := scenarios.ApproveAToken(ctx, client, user, repAddr); err != nil {
		t.Fatalf("Error approving aToken: %v", err)
	}

	// The user signs a delegation certificate allowing the bot to execute the repayment contract.
	cert, err := delegation.New(client.BotAddress())
	if err != nil {
		t.Fatalf("delegation.New(%v) = _, %v, want _, nil", client.BotAddress(), err)
	}
	signature, err := user.Sign(cert.Hash())
	if err != nil {
		t.Fatalf("Error signing certificate: %v", err)
	}

	// Executes the swap.
	loan, err := client.Loan(ctx, user.Address)
	if err != nil {
		t.Fatalf("client.Loan(ctx, %v) = _, %v, want _, nil", user.Address, err)
	}
	exec, err := repayment.NewExecution(ctx, client, loan, repAddr, signature)
	if err != nil {
		t.Fatalf("repayment.NewExecution(...) = _, %v, want _, nil", err)
	}
	if err := exec.Execute(ctx, client, rep); err != nil {
		t.Fatalf("exec.Execute(...) = _, %v, want _, nil", err)
	}

	// Verifies that debts are cleared.
	sDebt, err := client.BalanceOf(ctx, loan.StableDebt, user.Address)
	if err != nil {
		t.Fatalf("client.BalanceOf(%v, %v) = _, %v, want _, nil", loan.StableDebt, user.Address, err)
	}
	if sDebt.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("sDebt=%v, want 0", sDebt)
	}

	vDebt, err := client.BalanceOf(ctx, loan.VariableDebt, user.Address)
	if err != nil {
		t.Fatalf("client.BalanceOf(%v, %v) = _, %v, want _, nil", loan.VariableDebt, user.Address, err)
	}
	if vDebt.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("vDebt=%v, want 0", sDebt)
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
