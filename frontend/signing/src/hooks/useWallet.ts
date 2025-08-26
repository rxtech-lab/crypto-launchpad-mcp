import { useState, useEffect, useCallback } from 'react';
import { BrowserProvider, parseEther } from 'ethers';
import type { Signer } from 'ethers';
import type { EIP6963Provider, WalletState } from '../types/wallet';

export function useWallet() {
  const [state, setState] = useState<WalletState>({
    providers: [],
    selectedProvider: null,
    account: null,
    chainId: null,
    isConnected: false,
    isConnecting: false,
    error: null
  });

  // Discover wallets via EIP-6963
  const discoverWallets = useCallback(() => {
    const handleAnnouncement = (event: CustomEvent) => {
      const provider = event.detail as EIP6963Provider;
      setState(prev => ({
        ...prev,
        providers: [...prev.providers.filter(p => p.info.uuid !== provider.info.uuid), provider]
      }));
    };

    window.addEventListener('eip6963:announceProvider', handleAnnouncement as EventListener);
    
    // Request providers to announce
    window.dispatchEvent(new Event('eip6963:requestProvider'));

    return () => {
      window.removeEventListener('eip6963:announceProvider', handleAnnouncement as EventListener);
    };
  }, []);

  // Connect to wallet
  const connectWallet = useCallback(async (providerUuid: string) => {
    const provider = state.providers.find(p => p.info.uuid === providerUuid);
    if (!provider) {
      setState(prev => ({ ...prev, error: new Error('Provider not found') }));
      return;
    }

    setState(prev => ({ ...prev, isConnecting: true, error: null }));

    try {
      const ethersProvider = new BrowserProvider(provider.provider);
      const accounts = await ethersProvider.send('eth_requestAccounts', []);
      const network = await ethersProvider.getNetwork();
      
      setState(prev => ({
        ...prev,
        selectedProvider: provider,
        account: accounts[0],
        chainId: Number(network.chainId),
        isConnected: true,
        isConnecting: false
      }));

      // Listen for account changes
      provider.provider.on('accountsChanged', (accounts: string[]) => {
        if (accounts.length === 0) {
          disconnectWallet();
        } else {
          setState(prev => ({ ...prev, account: accounts[0] }));
        }
      });

      // Listen for chain changes
      provider.provider.on('chainChanged', (chainId: string) => {
        setState(prev => ({ ...prev, chainId: parseInt(chainId, 16) }));
      });
    } catch (error) {
      setState(prev => ({
        ...prev,
        error: error as Error,
        isConnecting: false
      }));
    }
  }, [state.providers]);

  // Disconnect wallet
  const disconnectWallet = useCallback(() => {
    setState(prev => ({
      ...prev,
      selectedProvider: null,
      account: null,
      chainId: null,
      isConnected: false,
      error: null
    }));
  }, []);

  // Switch network
  const switchNetwork = useCallback(async (targetChainId: number) => {
    if (!state.selectedProvider) {
      throw new Error('No wallet connected');
    }

    try {
      const chainIdHex = `0x${targetChainId.toString(16)}`;
      await state.selectedProvider.provider.request({
        method: 'wallet_switchEthereumChain',
        params: [{ chainId: chainIdHex }]
      });
    } catch (error: any) {
      // If the chain hasn't been added, we could add it here
      // For now, just throw the error
      throw error;
    }
  }, [state.selectedProvider]);

  // Sign and send transaction
  const signTransaction = useCallback(async (transaction: {
    to?: string;
    data: string;
    value: string;
  }) => {
    if (!state.selectedProvider || !state.account) {
      throw new Error('No wallet connected');
    }

    const ethersProvider = new BrowserProvider(state.selectedProvider.provider);
    const signer = await ethersProvider.getSigner();
    
    const tx = {
      to: transaction.to,
      data: transaction.data,
      value: parseEther(transaction.value)
    };

    const txResponse = await signer.sendTransaction(tx);
    return txResponse;
  }, [state.selectedProvider, state.account]);

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
    discoverWallets
  };
}