package interactors

import (
	"context"
	"errors"
	"time"

	walletapi "code.vegaprotocol.io/vega/wallet/api"
	"go.uber.org/zap"
)

var (
	ErrWalletSelectionDoesNotContainConfiguredOne = errors.New("the wallet selection doesn't contain the configured one")
	ErrRequestedWalletDoesNotMatchConfiguredOne   = errors.New("the requested wallet doesn't match with the configured one")
)

type AlwaysAgreeInteractor struct {
	Logger           *zap.Logger
	ConfiguredWallet string
	WalletPassphrase string
}

func (a *AlwaysAgreeInteractor) NotifyInteractionSessionBegan(_ context.Context, traceID string) error {
	a.Logger.Info("NotifyInteractionSessionBegan does nothing", zap.String("trace-id", traceID))
	return nil
}

func (a *AlwaysAgreeInteractor) NotifyInteractionSessionEnded(_ context.Context, traceID string) {
	a.Logger.Debug("NotifyInteractionSessionEnded does nothing", zap.String("trace-id", traceID))
}

func (a *AlwaysAgreeInteractor) NotifySuccessfulTransaction(_ context.Context, traceID, txHash, _, _ string, sentAt time.Time) {
	a.Logger.Debug("NotifySuccessfulTransaction does nothing", zap.String("trace-id", traceID), zap.String("tx-hash", txHash), zap.Time("sent-at", sentAt))
}

func (a *AlwaysAgreeInteractor) NotifyFailedTransaction(_ context.Context, traceID, _, _ string, err error, sentAt time.Time) {
	a.Logger.Debug("NotifyFailedTransaction does nothing", zap.String("trace-id", traceID), zap.Error(err), zap.Time("sent-at", sentAt))
}

func (a *AlwaysAgreeInteractor) NotifySuccessfulRequest(_ context.Context, traceID string, message string) {
	a.Logger.Debug("NotifySuccessfulRequest does nothing", zap.String("trace-id", traceID), zap.String("message", message))
}

func (a *AlwaysAgreeInteractor) NotifyError(_ context.Context, traceID string, t walletapi.ErrorType, err error) {
	a.Logger.Debug("NotifyError  does nothing", zap.String("trace-id", traceID), zap.String("error-type", string(t)), zap.Error(err))
}

func (a *AlwaysAgreeInteractor) Log(_ context.Context, traceID string, t walletapi.LogType, msg string) {
	a.Logger.Debug("Log does nothing", zap.String("trace-id", traceID), zap.String("log-type", string(t)), zap.String("message", msg))
}

func (a *AlwaysAgreeInteractor) RequestWalletConnectionReview(_ context.Context, traceID, hostname string) (string, error) {
	a.Logger.Debug("RequestWalletConnectionReview called", zap.String("trace-id", traceID), zap.String("hostname", hostname))
	a.Logger.Info("RequestWalletConnectionReview approves the connection request only this time", zap.String("trace-id", traceID), zap.String("hostname", hostname))
	return "APPROVED_ONLY_THIS_TIME", nil
}

func (a *AlwaysAgreeInteractor) RequestWalletSelection(_ context.Context, traceID, hostname string, availableWallets []string) (walletapi.SelectedWallet, error) {
	a.Logger.Debug("RequestWalletSelection called", zap.String("trace-id", traceID), zap.String("hostname", hostname), zap.Strings("wallets", availableWallets))

	matched := false
	for _, wallet := range availableWallets {
		if wallet == a.ConfiguredWallet {
			matched = true
			break
		}
	}
	if !matched {
		a.Logger.Error("RequestWalletSelection has been called with a selection that does not contain the configured wallet, verify you configured an existing wallet", zap.String("trace-id", traceID), zap.Strings("wallets", availableWallets), zap.String("configured-wallet", a.ConfiguredWallet))
		return walletapi.SelectedWallet{}, ErrWalletSelectionDoesNotContainConfiguredOne
	}

	a.Logger.Info("RequestWalletSelection selects the default wallet", zap.String("trace-id", traceID), zap.String("wallet", a.ConfiguredWallet), zap.String("passphrase", a.WalletPassphrase))
	return walletapi.SelectedWallet{
		Wallet:     a.ConfiguredWallet,
		Passphrase: a.WalletPassphrase,
	}, nil
}

func (a *AlwaysAgreeInteractor) RequestPassphrase(_ context.Context, traceID, wallet string) (string, error) {
	a.Logger.Debug("RequestPassphrase called", zap.String("trace-id", traceID), zap.String("wallet", wallet))
	if wallet != a.ConfiguredWallet {
		a.Logger.Error("RequestPassphrase has been called with a different wallet than the one configured, using different wallet is not supported yet", zap.String("configured-wallet", a.ConfiguredWallet), zap.String("requested-wallet", wallet))
		return "", ErrRequestedWalletDoesNotMatchConfiguredOne
	}
	a.Logger.Info("RequestPassphrase returns the passphrase", zap.String("trace-id", traceID), zap.String("wallet", wallet), zap.String("passphrase", wallet))
	return a.WalletPassphrase, nil
}

func (a *AlwaysAgreeInteractor) RequestPermissionsReview(_ context.Context, traceID, hostname, wallet string, perms map[string]string) (bool, error) {
	a.Logger.Debug("RequestPermissionsReview called", zap.String("trace-id", traceID), zap.String("hostname", hostname), zap.Any("permissions", perms))
	if wallet != a.ConfiguredWallet {
		a.Logger.Error("RequestPermissionsReview has been called with a different wallet than the one configured, using different wallet is not supported yet", zap.String("configured-wallet", a.ConfiguredWallet), zap.String("requested-wallet", wallet))
		return false, ErrRequestedWalletDoesNotMatchConfiguredOne
	}
	a.Logger.Info("RequestPermissionsReview approves the permissions", zap.String("trace-id", traceID), zap.String("hostname", hostname), zap.String("wallet", wallet), zap.Any("permissions", perms))
	return true, nil
}

func (a *AlwaysAgreeInteractor) RequestTransactionReviewForSending(_ context.Context, traceID, hostname, wallet, pubKey, _ string, _ time.Time) (bool, error) {
	a.Logger.Debug("RequestTransactionReviewForSending called", zap.String("trace-id", traceID), zap.String("hostname", hostname), zap.String("public-key", pubKey))
	if wallet != a.ConfiguredWallet {
		a.Logger.Error("RequestTransactionReviewForSending has been called with a different wallet than the one configured, using different wallet is not supported yet", zap.String("configured-wallet", a.ConfiguredWallet), zap.String("requested-wallet", wallet))
		return false, ErrRequestedWalletDoesNotMatchConfiguredOne
	}
	a.Logger.Info("RequestTransactionReviewForSending approves the transaction sending", zap.String("trace-id", traceID))
	return true, nil
}

func (a *AlwaysAgreeInteractor) RequestTransactionReviewForSigning(_ context.Context, traceID, hostname, wallet, pubKey, transaction string, receivedAt time.Time) (bool, error) {
	a.Logger.Debug("RequestTransactionReviewForSigning called", zap.String("trace-id", traceID), zap.String("hostname", hostname), zap.String("public-key", pubKey))
	if wallet != a.ConfiguredWallet {
		a.Logger.Error("RequestTransactionReviewForSigning has been called with a different wallet than the one configured, using different wallet is not supported yet", zap.String("configured-wallet", a.ConfiguredWallet), zap.String("requested-wallet", wallet))
		return false, ErrRequestedWalletDoesNotMatchConfiguredOne
	}
	a.Logger.Info("RequestTransactionReviewForSigning approves the transaction sending", zap.String("trace-id", traceID))
	return true, nil
}
