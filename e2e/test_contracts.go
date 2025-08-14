package e2e

import (
	"io/ioutil"
	"path/filepath"
)

// TestContracts contains sample Solidity contracts for testing
var TestContracts = map[string]string{
	"SimpleERC20": `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

contract SimpleERC20 {
    string public name;
    string public symbol;
    uint8 public decimals = 18;
    uint256 public totalSupply;
    
    mapping(address => uint256) public balanceOf;
    mapping(address => mapping(address => uint256)) public allowance;
    
    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
    
    constructor(string memory _name, string memory _symbol, uint256 _totalSupply) {
        name = _name;
        symbol = _symbol;
        totalSupply = _totalSupply * 10**decimals;
        balanceOf[msg.sender] = totalSupply;
        emit Transfer(address(0), msg.sender, totalSupply);
    }
    
    function transfer(address to, uint256 value) public returns (bool) {
        require(balanceOf[msg.sender] >= value, "Insufficient balance");
        balanceOf[msg.sender] -= value;
        balanceOf[to] += value;
        emit Transfer(msg.sender, to, value);
        return true;
    }
    
    function approve(address spender, uint256 value) public returns (bool) {
        allowance[msg.sender][spender] = value;
        emit Approval(msg.sender, spender, value);
        return true;
    }
    
    function transferFrom(address from, address to, uint256 value) public returns (bool) {
        require(allowance[from][msg.sender] >= value, "Allowance exceeded");
        require(balanceOf[from] >= value, "Insufficient balance");
        
        allowance[from][msg.sender] -= value;
        balanceOf[from] -= value;
        balanceOf[to] += value;
        
        emit Transfer(from, to, value);
        return true;
    }
}`,

	"MintableToken": `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

contract MintableToken {
    string public name;
    string public symbol;
    uint8 public decimals = 18;
    uint256 public totalSupply;
    address public owner;
    
    mapping(address => uint256) public balanceOf;
    mapping(address => mapping(address => uint256)) public allowance;
    
    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
    event Mint(address indexed to, uint256 value);
    
    modifier onlyOwner() {
        require(msg.sender == owner, "Only owner can call this function");
        _;
    }
    
    constructor(string memory _name, string memory _symbol, uint256 _initialSupply) {
        name = _name;
        symbol = _symbol;
        owner = msg.sender;
        totalSupply = _initialSupply * 10**decimals;
        balanceOf[msg.sender] = totalSupply;
        emit Transfer(address(0), msg.sender, totalSupply);
    }
    
    function transfer(address to, uint256 value) public returns (bool) {
        require(balanceOf[msg.sender] >= value, "Insufficient balance");
        balanceOf[msg.sender] -= value;
        balanceOf[to] += value;
        emit Transfer(msg.sender, to, value);
        return true;
    }
    
    function approve(address spender, uint256 value) public returns (bool) {
        allowance[msg.sender][spender] = value;
        emit Approval(msg.sender, spender, value);
        return true;
    }
    
    function transferFrom(address from, address to, uint256 value) public returns (bool) {
        require(allowance[from][msg.sender] >= value, "Allowance exceeded");
        require(balanceOf[from] >= value, "Insufficient balance");
        
        allowance[from][msg.sender] -= value;
        balanceOf[from] -= value;
        balanceOf[to] += value;
        
        emit Transfer(from, to, value);
        return true;
    }
    
    function mint(address to, uint256 value) public onlyOwner returns (bool) {
        totalSupply += value;
        balanceOf[to] += value;
        emit Transfer(address(0), to, value);
        emit Mint(to, value);
        return true;
    }
}`,
}

// LoadContractFromFile loads a Solidity contract from the contracts directory
func LoadContractFromFile(filename string) (string, error) {
	contractPath := filepath.Join("e2e", "contracts", filename)
	content, err := ioutil.ReadFile(contractPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// GetSimpleERC20Contract returns a basic ERC20 contract for testing
func GetSimpleERC20Contract() string {
	return TestContracts["SimpleERC20"]
}

// GetMintableTokenContract returns a mintable token contract for testing
func GetMintableTokenContract() string {
	return TestContracts["MintableToken"]
}
