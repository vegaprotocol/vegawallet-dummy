package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/cli"
	"code.vegaprotocol.io/vega/cmd/vegawallet/commands/flags"
	vgzap "code.vegaprotocol.io/vega/libs/zap"
	"code.vegaprotocol.io/vega/paths"
	coreversion "code.vegaprotocol.io/vega/version"
	walletapi "code.vegaprotocol.io/vega/wallet/api"
	nodeapi "code.vegaprotocol.io/vega/wallet/api/node"
	"code.vegaprotocol.io/vega/wallet/network"
	netstore "code.vegaprotocol.io/vega/wallet/network/store/v1"
	"code.vegaprotocol.io/vega/wallet/node"
	"code.vegaprotocol.io/vega/wallet/service"
	svcstore "code.vegaprotocol.io/vega/wallet/service/store/v1"
	walletversion "code.vegaprotocol.io/vega/wallet/version"
	"code.vegaprotocol.io/vega/wallet/wallets"
	"com.github/vegaprotocol/vegawallet-dummy/interactors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var ErrNoHostSpecified = errors.New("no host specified in the configuration")

var (
	ErrProgramIsNotInitialised = errors.New("first, you need initialise the program, using the `init` command")

	runServiceLong = cli.LongDesc(`
		Start a Vega wallet service behind an HTTP server.

		By default, every incoming transactions will be approved.

		Warning:
		This software is insecure by design to ease development and testing.

		USE FOR DEVELOPMENT AND TESTING ONLY.

		To terminate the service, hit ctrl+c.
	`)

	runServiceExample = cli.Examples(`
		# Start the service
		{{.Software}} service run --network NETWORK --wallet WALLET --passphrase-file FILE

		# Start the service with log level to "debug"
		{{.Software}} service run --log-level debug --network NETWORK --wallet WALLET --passphrase-file FILE

		# Start the service with logs as JSON
		{{.Software}} service run --log-format json --network NETWORK --wallet WALLET --passphrase-file FILE
	`)
)

type RunServiceHandler func(*RootFlags, *RunServiceFlags) error

func NewCmdRunService(rf *RootFlags) *cobra.Command {
	return BuildCmdRunService(RunService, rf)
}

func BuildCmdRunService(handler RunServiceHandler, rf *RootFlags) *cobra.Command {
	f := &RunServiceFlags{}

	cmd := &cobra.Command{
		Use:     "run",
		Short:   "Start the Vega wallet service",
		Long:    runServiceLong,
		Example: runServiceExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := f.Validate(); err != nil {
				return err
			}

			if err := handler(rf, f); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&f.Network,
		"network", "n",
		"",
		"Network configuration to use",
	)
	cmd.Flags().StringVarP(&f.Wallet,
		"wallet", "w",
		"",
		"The wallet to use",
	)
	cmd.Flags().StringVarP(&f.PassphraseFile,
		"passphrase-file", "p",
		"",
		"The wallet's passphrase",
	)
	cmd.Flags().StringVar(&f.LogLevel,
		"log-level",
		"info",
		"The minimum log level to display",
	)
	cmd.Flags().StringVar(&f.LogFormat,
		"log-format",
		"console",
		"The format of the logs on the standard output: [console, json]",
	)

	return cmd
}

type RunServiceFlags struct {
	Network        string
	Wallet         string
	PassphraseFile string
	LogLevel       string
	LogFormat      string
}

func (f *RunServiceFlags) Validate() error {
	if len(f.Network) == 0 {
		return flags.MustBeSpecifiedError("network")
	}

	if len(f.Wallet) == 0 {
		return flags.MustBeSpecifiedError("wallet")
	}

	if len(f.PassphraseFile) == 0 {
		return flags.MustBeSpecifiedError("passphrase-file")
	}

	return nil
}

func RunService(rf *RootFlags, f *RunServiceFlags) error {
	passphrase, err := flags.ReadPassphraseFile(f.PassphraseFile)
	if err != nil {
		return err
	}

	var logger *zap.Logger

	if f.LogFormat == "json" {
		l, err := vgzap.BuildStandardJSONLogger(f.LogLevel)
		if err != nil {
			return fmt.Errorf("could not build the service logger: %w", err)
		}
		logger = l
	} else if f.LogFormat == "console" {
		l, err := vgzap.BuildStandardConsoleLogger(f.LogLevel)
		if err != nil {
			return fmt.Errorf("could not build the service logger: %w", err)
		}
		logger = l
	}
	logger = logger.Named("service")

	logger.Debug("Initializing the wallet store...")
	walletStore, err := wallets.InitialiseStore(rf.Home)
	if err != nil {
		return fmt.Errorf("couldn't initialise wallets store: %w", err)
	}
	logger.Debug("The wallet store has been initialized")

	if _, err = walletStore.GetWallet(context.Background(), f.Wallet, passphrase); err != nil {
		return fmt.Errorf("could not retrieve the wallet: %w", err)
	}

	handler := wallets.NewHandler(walletStore)

	vegaPaths := paths.New(rf.Home)

	logger.Debug("Initializing the network store...")
	netStore, err := netstore.InitialiseStore(vegaPaths)
	if err != nil {
		return fmt.Errorf("couldn't initialise network store: %w", err)
	}
	logger.Debug("The network store has been initialized")

	logger.Debug("Verifying the network exist...", zap.String("network", f.Network))
	exists, err := netStore.NetworkExists(f.Network)
	if err != nil {
		return fmt.Errorf("couldn't verify the network existence: %w", err)
	}
	if !exists {
		return network.NewDoesNotExistError(f.Network)
	}
	logger.Debug("The network exists")

	logger.Debug("Retrieving the network configuration...", zap.String("network", f.Network))
	cfg, err := netStore.GetNetwork(f.Network)
	if err != nil {
		return fmt.Errorf("couldn't retrieve the network configuration: %w", err)
	}
	logger.Debug("The network configuration has been retrieved")

	logger.Debug("Ensuring the network configuration has the minimal configuration to connect to the network...")
	if err := cfg.EnsureCanConnectGRPCNode(); err != nil {
		return err
	}
	logger.Debug("The network configuration is ok")

	networkVersion, err := walletversion.GetNetworkVersionThroughGRPC(cfg.API.GRPC.Hosts)
	if err != nil {
		return err
	}
	if networkVersion != coreversion.Get() {
		logger.Warn("This software is not compatible with this network", zap.String("network-version", networkVersion), zap.String("backend-version", coreversion.Get()))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Debug("Initializing the service store...")
	svcStore, err := svcstore.InitialiseStore(paths.New(rf.Home))
	if err != nil {
		return fmt.Errorf("couldn't initialise service store: %w", err)
	}
	logger.Debug("The service store has been initialized")

	logger.Debug("Verifying the service has been initialized...")
	if isInit, err := service.IsInitialised(svcStore); err != nil {
		return fmt.Errorf("couldn't verify service initialisation state: %w", err)
	} else if !isInit {
		return ErrProgramIsNotInitialised
	}
	logger.Debug("The service is properly initialized")

	logger.Debug("Initializing API v1 authentication system...")
	auth, err := service.NewAuth(logger.Named("auth"), svcStore, cfg.TokenExpiry.Get())
	if err != nil {
		return fmt.Errorf("couldn't initialise authentication: %w", err)
	}
	logger.Debug("API v1 authentication system has been initialized")

	logger.Debug("Initializing API v1 node forwarder...")
	forwarder, err := node.NewForwarder(logger.Named("forwarder"), cfg.API.GRPC)
	if err != nil {
		return fmt.Errorf("couldn't initialise the node forwarder: %w", err)
	}
	logger.Debug("API v1 node forwarder has been initialized")

	alwaysAgreeInteractor := &interactors.AlwaysAgreeInteractor{
		Logger:           logger.Named("always-agree-interactor"),
		ConfiguredWallet: f.Wallet,
		WalletPassphrase: passphrase,
	}

	jsonrpcLog := logger.Named("json-rpc")

	logger.Debug("Initializing API v2 node selector...")
	nodeSelector, err := nodeapi.BuildRoundRobinSelectorWithRetryingNodes(jsonrpcLog, cfg.API.GRPC.Hosts, cfg.API.GRPC.Retries)
	if err != nil {
		logger.Error("Couldn't instantiate node API", zap.Error(err))
		return fmt.Errorf("couldn't instantiate the node API: %w", err)
	}
	logger.Debug("API v2 node selector is initialized")

	logger.Debug("Initializing API v2 client...")
	apiV2, err := walletapi.ClientAPI(jsonrpcLog, walletStore, alwaysAgreeInteractor, nodeSelector)
	if err != nil {
		return fmt.Errorf("couldn't instantiate the JSON-RPC API: %w", err)
	}
	logger.Debug("API v2 client is initialized")

	logger.Debug("Initializing the service...")
	srv, err := service.NewService(logger.Named("api"), cfg, apiV2, handler, auth, forwarder, service.NewAutomaticConsentPolicy())
	if err != nil {
		return err
	}
	logger.Debug("The service is initialized")

	go func() {
		defer cancel()
		serviceHost := fmt.Sprintf("http://%v:%v", cfg.Host, cfg.Port)
		logger.Info("Starting HTTP service", zap.String("url", serviceHost))
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Failed to start HTTP server", zap.Error(err))
		}
	}()

	defer func() {
		if err = srv.Stop(); err != nil {
			logger.Error("Failed to stop HTTP server", zap.Error(err))
			return
		}
		logger.Info("HTTP server stopped with success")
	}()

	waitSig(ctx, cancel, logger)

	return nil
}

// waitSig will wait for a sigterm or sigint interrupt.
func waitSig(ctx context.Context, cancelFunc context.CancelFunc, log *zap.Logger) {
	gracefulStop := make(chan os.Signal, 1)
	defer close(gracefulStop)

	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	signal.Notify(gracefulStop, syscall.SIGQUIT)

	for {
		select {
		case sig := <-gracefulStop:
			log.Info("Caught signal", zap.String("signal", fmt.Sprintf("%+v", sig)))
			cancelFunc()
			return
		case <-ctx.Done():
			// nothing to do
			return
		}
	}
}
