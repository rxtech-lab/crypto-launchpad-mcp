import { useState, useEffect, useCallback } from "react";
import { BrowserProvider, parseEther } from "ethers";
import type { Signer } from "ethers";
import type {
  EIP6963Provider,
  WalletState,
  BlockchainNetwork,
} from "../types/wallet";

export function useWallet() {
  const [state, setState] = useState<WalletState>({
    providers: [],
    selectedProvider: null,
    account: null,
    chainId: null,
    isConnected: false,
    isConnecting: false,
    error: null,
  });

  const [networkSwitchError, setNetworkSwitchError] = useState<Error | null>(
    null
  );

  // Discover wallets via EIP-6963
  const discoverWallets = useCallback(() => {
    const handleAnnouncement = (event: CustomEvent) => {
      const provider = event.detail as EIP6963Provider;
      setState((prev) => ({
        ...prev,
        providers: [
          ...prev.providers.filter((p) => p.info.uuid !== provider.info.uuid),
          provider,
        ],
      }));
    };

    window.addEventListener(
      "eip6963:announceProvider",
      handleAnnouncement as EventListener
    );

    // Request providers to announce
    window.dispatchEvent(new Event("eip6963:requestProvider"));

    return () => {
      window.removeEventListener(
        "eip6963:announceProvider",
        handleAnnouncement as EventListener
      );
    };
  }, []);

  // Read RPC network metadata from page
  const getRPCNetworkMetadata = useCallback((): BlockchainNetwork | null => {
    const metaTag = document.querySelector('meta[name="rpc-network"]');
    if (metaTag) {
      const content = metaTag.getAttribute("content");
      if (content) {
        try {
          return JSON.parse(content) as BlockchainNetwork;
        } catch (e) {
          console.error("Failed to parse RPC network metadata:", e);
        }
      }
    }
    return null;
  }, []);

  // Add network to wallet if not present
  const addNetwork = useCallback(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    async (provider: any, network: BlockchainNetwork) => {
      try {
        await provider.request({
          method: "wallet_addEthereumChain",
          params: [
            {
              chainId: `0x${network.chain_id.toString(16)}`,
              chainName: network.name,
              nativeCurrency: {
                name: "ETH",
                symbol: "ETH",
                decimals: 18,
              },
              rpcUrls: [network.rpc],
              blockExplorerUrls: [],
            },
          ],
        });
        return true;
      } catch (error) {
        console.error("Failed to add network:", error);
        return false;
      }
    },
    []
  );

  // Connect to wallet
  const connectWallet = useCallback(
    async (providerUuid: string) => {
      const provider = state.providers.find(
        (p) => p.info.uuid === providerUuid
      );
      if (!provider) {
        setState((prev) => ({
          ...prev,
          error: new Error("Provider not found"),
        }));
        return;
      }

      setState((prev) => ({ ...prev, isConnecting: true, error: null }));

      try {
        const ethersProvider = new BrowserProvider(provider.provider);
        const accounts = await ethersProvider.send("eth_requestAccounts", []);
        const network = await ethersProvider.getNetwork();

        // Check if we need to switch to RPC network from metadata
        const rpcNetwork = getRPCNetworkMetadata();
        if (rpcNetwork && rpcNetwork.type === "ethereum") {
          const currentChainId = Number(network.chainId);
          if (currentChainId !== rpcNetwork.chain_id) {
            try {
              // Try to switch to the network
              const chainIdHex = `0x${rpcNetwork.chain_id.toString(16)}`;
              await provider.provider.request({
                method: "wallet_switchEthereumChain",
                params: [{ chainId: chainIdHex }],
              });
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
            } catch (switchError: any) {
              // If chain doesn't exist (error code 4902), add it
              if (switchError.code === 4902) {
                const added = await addNetwork(provider.provider, rpcNetwork);
                if (added) {
                  // Try switching again after adding
                  const chainIdHex = `0x${rpcNetwork.chain_id.toString(16)}`;
                  await provider.provider.request({
                    method: "wallet_switchEthereumChain",
                    params: [{ chainId: chainIdHex }],
                  });
                }
              } else {
                console.error("Failed to switch network:", switchError);
              }
            }
            // Get updated network info after switch
            try {
              // Add a small delay to allow the network change to settle
              await new Promise((resolve) => setTimeout(resolve, 100));

              let updatedNetwork;
              let retries = 3;
              while (retries > 0) {
                try {
                  updatedNetwork = await ethersProvider.getNetwork();
                  break;
                } catch (networkError) {
                  retries--;
                  if (retries === 0) throw networkError;
                  // Wait a bit longer on retry
                  await new Promise((resolve) => setTimeout(resolve, 200));
                }
              }

              setState((prev) => ({
                ...prev,
                selectedProvider: provider,
                account: accounts[0],
                chainId: Number(updatedNetwork!.chainId),
                isConnected: true,
                isConnecting: false,
              }));
            } catch (networkError) {
              console.warn(
                "Failed to get updated network info after switch:",
                networkError
              );
              // Fall back to using the expected chain ID
              setState((prev) => ({
                ...prev,
                selectedProvider: provider,
                account: accounts[0],
                chainId: rpcNetwork.chain_id,
                isConnected: true,
                isConnecting: false,
              }));
            }
          } else {
            setState((prev) => ({
              ...prev,
              selectedProvider: provider,
              account: accounts[0],
              chainId: Number(network.chainId),
              isConnected: true,
              isConnecting: false,
            }));
          }
        } else {
          setState((prev) => ({
            ...prev,
            selectedProvider: provider,
            account: accounts[0],
            chainId: Number(network.chainId),
            isConnected: true,
            isConnecting: false,
          }));
        }

        // Listen for account changes
        provider.provider.on("accountsChanged", (accounts: string[]) => {
          if (accounts.length === 0) {
            disconnectWallet();
          } else {
            setState((prev) => ({ ...prev, account: accounts[0] }));
          }
        });

        // Listen for chain changes
        provider.provider.on("chainChanged", (chainId: string) => {
          setState((prev) => ({ ...prev, chainId: parseInt(chainId, 16) }));
          setNetworkSwitchError(null); // Clear network switch error on successful change
        });
      } catch (error) {
        setState((prev) => ({
          ...prev,
          error: error as Error,
          isConnecting: false,
        }));
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [state.providers, getRPCNetworkMetadata, addNetwork]
  );

  // Switch network
  const switchNetwork = useCallback(
    async (targetChainId: number) => {
      if (!state.selectedProvider) {
        const error = new Error("No wallet connected");
        setNetworkSwitchError(error);
        throw error;
      }

      // Check if already on the target network
      const currentChainId = Number(state.chainId);
      const targetId = Number(targetChainId);
      if (currentChainId === targetId) {
        console.log(`Already on chain ${targetId}, no switch needed`);
        setNetworkSwitchError(null);
        return; // Already on the correct network
      }

      try {
        setNetworkSwitchError(null); // Clear any previous errors
        const chainIdHex = `0x${targetChainId.toString(16)}`;
        let retries = 3;
        while (retries > 0) {
          try {
            await state.selectedProvider.provider.request({
              method: "wallet_switchEthereumChain",
              params: [{ chainId: chainIdHex }],
            });
            break;
          } catch (switchError) {
            // Check if the error is because we're already on the chain
            const error = switchError as { code?: number; message?: string };
            if (error.code === -32000 && error.message?.includes("already")) {
              console.log("Wallet reports already on target chain");
              break;
            }
            retries--;
            if (retries === 0) throw switchError;
            // Wait before retry
            await new Promise((resolve) => setTimeout(resolve, 500));
          }
        }
      } catch (error) {
        const networkError =
          error instanceof Error
            ? error
            : new Error("Failed to switch network");
        setNetworkSwitchError(networkError);
        throw networkError;
      }
    },
    [state.selectedProvider, state.chainId]
  );

  // Disconnect wallet
  const disconnectWallet = useCallback(() => {
    setState((prev) => ({
      ...prev,
      selectedProvider: null,
      account: null,
      chainId: null,
      isConnected: false,
      error: null,
    }));
    setNetworkSwitchError(null); // Clear network switch error on disconnect
  }, []);

  // Sign and send transaction
  const signTransaction = useCallback(
    async (transaction: { to?: string; data: string; value: string }) => {
      if (!state.selectedProvider || !state.account) {
        throw new Error("No wallet connected");
      }

      const ethersProvider = new BrowserProvider(
        state.selectedProvider.provider
      );
      const signer = await ethersProvider.getSigner();

      const tx = {
        to: transaction.to,
        data: transaction.data,
        value: parseEther(transaction.value),
      };

      const txResponse = await signer.sendTransaction(tx);
      return txResponse;
    },
    [state.selectedProvider, state.account]
  );

  // Get signer
  const getSigner = useCallback(async (): Promise<Signer | null> => {
    if (!state.selectedProvider) return null;

    const ethersProvider = new BrowserProvider(state.selectedProvider.provider);
    return await ethersProvider.getSigner();
  }, [state.selectedProvider]);

  // Initialize wallet discovery on mount
  useEffect(() => {
    const cleanup = discoverWallets();
    return cleanup;
  }, [discoverWallets]);

  return {
    ...state,
    connectWallet,
    disconnectWallet,
    switchNetwork,
    signTransaction,
    getSigner,
    discoverWallets,
    networkSwitchError,
    getRPCNetworkMetadata,
  };
}
