/* eslint-disable @typescript-eslint/no-explicit-any */
import { useCallback, useEffect, useState } from "react";
import type { TransactionState, TransactionStatus } from "../types/wallet";

interface UseTransactionProps {
  sessionId?: string;
}

export function useTransaction({ sessionId }: UseTransactionProps = {}) {
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
    async (index: number, signTransaction: (tx: any) => Promise<any>, signMessage?: (message: string) => Promise<string>) => {
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
          value: deployment.value.length > 0 ? deployment.value : "0",
        };

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

        // Generate signature for ownership verification if signMessage is provided
        let signature = "";
        let signedMessage = "";
        if (signMessage) {
          try {
            // Get the signing message from meta tag
            const messageMetaTag = document.querySelector('meta[name="signing-message"]');
            let message = "Default signing message"; // fallback
            if (messageMetaTag) {
              message = messageMetaTag.getAttribute("content") || message;
            }
            signedMessage = message; // Store the message that was signed
            signature = await signMessage(message);
          } catch (signError) {
            console.warn("Failed to generate signature for ownership verification:", signError);
            // Continue without signature - backend will log warning
          }
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
              signature: signature,
              signedMessage: signedMessage,
            }),
          });
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
    async (signTransaction: (tx: any) => Promise<any>, signMessage?: (message: string) => Promise<string>) => {
      if (!state.session) {
        throw new Error("No session loaded");
      }

      setState((prev) => ({ ...prev, isExecuting: true, error: null }));

      try {
        const results = [];
        for (let i = 0; i < state.session.transaction_deployments.length; i++) {
          setState((prev) => ({ ...prev, currentIndex: i }));
          const receipt = await executeTransaction(i, signTransaction, signMessage);
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
    reset,
  };
}
