/* eslint-disable @typescript-eslint/no-explicit-any */
import { useCallback, useEffect, useState, useRef } from "react";
import { formatEther, BrowserProvider } from "ethers";
import type { TransactionState, TransactionStatus } from "../types/wallet";

interface UseTransactionProps {
  sessionId?: string;
  walletProvider?: any;
  account?: string | null;
}

export function useTransaction({ sessionId, walletProvider, account }: UseTransactionProps = {}) {
  const [state, setState] = useState<TransactionState>({
    session: null,
    transactionStatuses: new Map(),
    currentIndex: 0,
    error: null,
    isExecuting: false,
  });

  // Store deployed contract addresses
  const [deployedContracts, setDeployedContracts] = useState<
    Map<number, { address: string; txHash: string }>
  >(new Map());

  // Balance refresh interval ref
  const balanceIntervalRef = useRef<NodeJS.Timeout | null>(null);

  // Load session data from meta tag or API
  const loadSession = useCallback(async () => {
    if (!sessionId) {
      // Try to get from meta tag
      const metaTag = document.querySelector(
        'meta[name="transaction-session"]'
      );
      if (metaTag) {
        try {
          const metaTagContent = metaTag.getAttribute("content");
          const sessionData = JSON.parse(metaTagContent || "{}");
          const statusMap = new Map<number, TransactionStatus>();
          sessionData.transaction_deployments?.forEach(
            (_: any, index: number) => {
              statusMap.set(index, "waiting");
            }
          );

          setState((prev) => ({
            ...prev,
            session: sessionData,
            transactionStatuses: statusMap,
          }));
          return;
        } catch (error) {
          console.error("Failed to parse session from meta tag:", error);
        }
      }
    }

    // Load from API if sessionId provided
    if (sessionId) {
      try {
        const response = await fetch(`/api/session/${sessionId}`);
        if (!response.ok) throw new Error("Failed to load session");

        const sessionData = await response.json();
        const statusMap = new Map<number, TransactionStatus>();
        sessionData.transaction_deployments?.forEach(
          (_: any, index: number) => {
            statusMap.set(index, "waiting");
          }
        );

        setState((prev) => ({
          ...prev,
          session: sessionData,
          transactionStatuses: statusMap,
        }));
      } catch (error) {
        setState((prev) => ({
          ...prev,
          error: error as Error,
        }));
      }
    }
  }, [sessionId]);

  // Fetch balances for all tokens in the session
  const fetchBalances = useCallback(async () => {
    if (!state.session || !walletProvider || !account) {
      return;
    }

    try {
      const provider = new BrowserProvider(walletProvider);
      const updatedBalances: Record<string, string | null> = {};

      // Fetch balances for each token address in the session
      for (const [address, currentBalance] of Object.entries(state.session.balances)) {
        try {
          if (!address || address === "0x0" || address.toLowerCase() === "0x0000000000000000000000000000000000000000") {
            // Native token balance (ETH)
            const balance = await provider.getBalance(account);
            updatedBalances[address] = balance.toString();
          } else {
            // ERC-20 token balance - use basic ERC-20 balanceOf call
            const tokenContract = {
              address: address,
              abi: ["function balanceOf(address) view returns (uint256)"]
            };
            
            // Create contract instance and call balanceOf
            const contract = new (await import("ethers")).Contract(
              tokenContract.address, 
              tokenContract.abi, 
              provider
            );
            const balance = await contract.balanceOf(account);
            updatedBalances[address] = balance.toString();
          }
        } catch (error) {
          console.warn(`Failed to fetch balance for ${address}:`, error);
          updatedBalances[address] = currentBalance; // Keep existing balance on error
        }
      }

      // Update session with new balances
      setState(prev => ({
        ...prev,
        session: prev.session ? {
          ...prev.session,
          balances: updatedBalances
        } : null
      }));

    } catch (error) {
      console.error("Failed to fetch balances:", error);
    }
  }, [state.session, walletProvider, account]);

  // Setup balance refresh interval (every 10 seconds)
  useEffect(() => {
    // Clear existing interval
    if (balanceIntervalRef.current) {
      clearInterval(balanceIntervalRef.current);
    }

    // Only start interval if we have session, wallet, and account
    if (state.session && walletProvider && account) {
      // Fetch balances immediately
      fetchBalances();

      // Setup interval for every 10 seconds
      balanceIntervalRef.current = setInterval(() => {
        fetchBalances();
      }, 10000);
    }

    // Cleanup on unmount or dependency changes
    return () => {
      if (balanceIntervalRef.current) {
        clearInterval(balanceIntervalRef.current);
        balanceIntervalRef.current = null;
      }
    };
  }, [fetchBalances, state.session, walletProvider, account]);

  // Update transaction status
  const updateTransactionStatus = useCallback(
    (index: number, status: TransactionStatus) => {
      setState((prev) => {
        const newStatuses = new Map(prev.transactionStatuses);
        newStatuses.set(index, status);
        return {
          ...prev,
          transactionStatuses: newStatuses,
        };
      });
    },
    []
  );

  // Execute transaction
  const executeTransaction = useCallback(
    async (index: number, signTransaction: (tx: any) => Promise<any>) => {
      if (!state.session) {
        throw new Error("No session loaded");
      }

      const deployment = state.session.transaction_deployments[index];
      if (!deployment) {
        throw new Error(`No transaction at index ${index}`);
      }

      updateTransactionStatus(index, "pending");

      try {
        const tx = {
          data: deployment.data,
          value:
            deployment.value.length > 0 && deployment.value != "0"
              ? formatEther(deployment.value)
              : "0",
          to: deployment.receiver ? deployment.receiver : undefined,
        };

        console.log("Constructing transaction:", tx);

        const txResponse = await signTransaction(tx);
        const receipt = await txResponse.wait();

        updateTransactionStatus(
          index,
          receipt.status === 1 ? "confirmed" : "failed"
        );

        // Store contract address if deployment created one
        if (receipt.contractAddress) {
          setDeployedContracts((prev) => {
            const newMap = new Map(prev);
            newMap.set(index, {
              address: receipt.contractAddress,
              txHash: receipt.hash,
            });
            return newMap;
          });
        }
        
        // Update session status on backend
        if (state.session.id) {
          await fetch(`/api/tx/${state.session.id}/transaction/${index}`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              status: receipt.status === 1 ? "confirmed" : "failed",
              transactionHash: receipt.hash,
              contractAddress: receipt.contractAddress,
            }),
          });
        }

        // Refresh balances after successful transaction
        if (receipt.status === 1) {
          setTimeout(() => {
            fetchBalances();
          }, 2000); // Wait 2 seconds for blockchain to update
        }

        return receipt;
      } catch (error) {
        updateTransactionStatus(index, "failed");
        throw error;
      }
    },
    [state.session, updateTransactionStatus]
  );

  // Execute all transactions sequentially
  const executeAllTransactions = useCallback(
    async (signTransaction: (tx: any) => Promise<any>) => {
      if (!state.session) {
        throw new Error("No session loaded");
      }

      setState((prev) => ({ ...prev, isExecuting: true, error: null }));

      try {
        const results = [];
        for (let i = 0; i < state.session.transaction_deployments.length; i++) {
          setState((prev) => ({ ...prev, currentIndex: i }));
          const receipt = await executeTransaction(i, signTransaction);
          results.push(receipt);
        }

        setState((prev) => ({ ...prev, isExecuting: false }));
        return results;
      } catch (error) {
        setState((prev) => ({
          ...prev,
          isExecuting: false,
          error: error as Error,
        }));
        throw error;
      }
    },
    [state.session, executeTransaction]
  );

  // Reset state
  const reset = useCallback(() => {
    setState({
      session: null,
      transactionStatuses: new Map(),
      currentIndex: 0,
      error: null,
      isExecuting: false,
    });
    setDeployedContracts(new Map());
  }, []);

  // Load session on mount
  useEffect(() => {
    loadSession();
  }, [loadSession]);

  return {
    ...state,
    deployedContracts,
    loadSession,
    updateTransactionStatus,
    executeTransaction,
    executeAllTransactions,
    fetchBalances,
    reset,
  };
}
