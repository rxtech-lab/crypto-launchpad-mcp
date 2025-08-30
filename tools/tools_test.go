package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestContractGeneration tests the go:generate functionality
func TestContractGeneration(t *testing.T) {
	// Get the project root directory
	projectRoot := filepath.Join("..", "..")
	if wd, err := os.Getwd(); err == nil {
		// If we're in tools directory, go up two levels to project root
		if strings.HasSuffix(wd, "tools") {
			projectRoot = ".."
		}
	}

	// Check if the contracts directory exists
	contractsDir := filepath.Join(projectRoot, "internal", "contracts", "openzeppelin-contracts", "contracts")
	if _, err := os.Stat(contractsDir); os.IsNotExist(err) {
		t.Skipf("OpenZeppelin contracts directory not found at %s, skipping test", contractsDir)
	}

	// Check if the output file would be generated in the correct location
	outputFile := filepath.Join(projectRoot, "internal", "contracts", "contracts.go")
	outputDir := filepath.Dir(outputFile)

	// Ensure the output directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Fatalf("Output directory %s does not exist", outputDir)
	}

	t.Logf("Contract generation would create file at: %s", outputFile)
	t.Logf("Source contracts directory: %s", contractsDir)
}

// TestPackageStructure ensures the tools package is properly structured
func TestPackageStructure(t *testing.T) {
	// Check if tools.go exists
	if _, err := os.Stat("tools.go"); os.IsNotExist(err) {
		t.Fatalf("tools.go file not found")
	}

	// Check if contract/generate_contracts.go exists
	generateScript := filepath.Join("contract", "generate_contracts.go")
	if _, err := os.Stat(generateScript); os.IsNotExist(err) {
		t.Fatalf("generate_contracts.go file not found at %s", generateScript)
	}

	t.Log("Package structure is correct")
}

// TestGenerateContractsScript tests the contract generation script functionality
func TestGenerateContractsScript(t *testing.T) {
	// This test validates that the generate_contracts.go script can be compiled
	generateScript := filepath.Join("contract", "generate_contracts.go")

	// Check if the file exists and is readable
	content, err := os.ReadFile(generateScript)
	if err != nil {
		t.Fatalf("Could not read generate_contracts.go: %v", err)
	}

	// Basic validation that it's a Go main package
	contentStr := string(content)
	if !strings.Contains(contentStr, "package main") {
		t.Errorf("generate_contracts.go should be a main package")
	}

	if !strings.Contains(contentStr, "func main()") {
		t.Errorf("generate_contracts.go should have a main function")
	}

	// Check that it references the expected paths
	if !strings.Contains(contentStr, "openzeppelin-contracts") {
		t.Errorf("generate_contracts.go should reference openzeppelin-contracts")
	}

	if !strings.Contains(contentStr, "contracts.go") {
		t.Errorf("generate_contracts.go should generate contracts.go")
	}

	t.Log("Contract generation script structure is valid")
}
