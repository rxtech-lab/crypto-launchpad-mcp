package services

import (
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
)

type HookService interface {
	AddHook(hook Hook) error
	OnTransactionConfirmed(txType models.TransactionType, txHash string, contractAddress *string, session models.TransactionSession) error
}

type hookService struct {
	hooks []Hook
}

func NewHookService() HookService {
	return &hookService{
		hooks: []Hook{},
	}
}

func (h *hookService) AddHook(hook Hook) error {
	h.hooks = append(h.hooks, hook)
	return nil
}

func (h *hookService) OnTransactionConfirmed(txType models.TransactionType, txHash string, contractAddress *string, session models.TransactionSession) error {
	for _, hook := range h.hooks {
		if hook.CanHandle(txType) {
			if err := hook.OnTransactionConfirmed(txType, txHash, contractAddress, session); err != nil {
				return err
			}
		}
	}
	return nil
}
